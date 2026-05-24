package actiondispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

type Dispatcher struct {
	Store  Store
	Clock  func() time.Time
	Sender g2stransport.Sender
}

func (d *Dispatcher) Dispatch(ctx context.Context, request DispatchRequest) (DispatchResult, error) {
	if d.Store == nil {
		return DispatchResult{}, fmt.Errorf("store is required")
	}
	actionRunID := strings.TrimSpace(request.ActionRunID)
	if actionRunID == "" {
		return DispatchResult{}, fmt.Errorf("action_run_id is required")
	}
	mode := request.Mode
	if mode != DispatchModeDryRun {
		return DispatchResult{}, fmt.Errorf("unsupported dispatch mode %q", mode)
	}

	now := request.RequestedAt
	if now.IsZero() {
		clock := d.Clock
		if clock == nil {
			clock = time.Now
		}
		now = clock().UTC()
	}

	run, err := d.Store.GetActionRun(ctx, actionRunID)
	if err != nil {
		return DispatchResult{}, err
	}
	if run == nil {
		return DispatchResult{}, fmt.Errorf("action run %q not found", actionRunID)
	}
	if run.Status != actions.RunStatusPending {
		return DispatchResult{}, fmt.Errorf("action run %q status must be PENDING for dry-run dispatch", actionRunID)
	}

	definition, err := d.Store.GetActionDefinition(ctx, run.ActionDefinitionID)
	if err != nil {
		return DispatchResult{}, err
	}
	if definition == nil {
		return DispatchResult{}, fmt.Errorf("action definition %q not found", run.ActionDefinitionID)
	}

	targets, err := d.Store.ListActionTargetResults(ctx, run.ID)
	if err != nil {
		return DispatchResult{}, err
	}

	stepID := ""
	stepKey := "queue_only_no_send"
	if len(definition.Steps) > 0 {
		stepID = definition.Steps[0].ID
		if strings.TrimSpace(definition.Steps[0].TemplateActionKey) != "" {
			stepKey = strings.TrimSpace(definition.Steps[0].TemplateActionKey)
		}
	}

	warnings := []string{}
	prepared := make([]g2sengine.MessageJournalEntry, 0, len(targets))
	for _, target := range targets {
		egm, getErr := d.Store.GetEGMRecord(ctx, target.TargetEGMID)
		if getErr != nil {
			return DispatchResult{}, getErr
		}
		egmID := target.TargetEGMID
		templateID := ""
		ipAddress := ""
		endpointPath := ""
		if egm == nil {
			warnings = append(warnings, fmt.Sprintf("target EGM %s not found", target.TargetEGMID))
		} else {
			egmID = egm.EGMID
			templateID = strings.TrimSpace(egm.TemplateID)
			ipAddress = strings.TrimSpace(egm.IPAddress)
			endpointPath = strings.TrimSpace(egm.EndpointPath)
		}

		rendered := false
		renderError := ""
		messageType := stepKey
		templateVersionLabel := ""
		templateVersionInt := 0
		rawPayload := fmt.Sprintf("DRY_RUN_NO_SEND_RENDER_UNAVAILABLE action=%s run=%s egm=%s step=%s", definition.ID, run.ID, egmID, stepKey)

		if templateID == "" {
			warnings = append(warnings, fmt.Sprintf("EGM %s has no template assigned", egmID))
			renderError = fmt.Sprintf("EGM %s has no template assigned", egmID)
		} else {
			tpl, tplErr := d.Store.GetG2STemplate(ctx, templateID)
			if tplErr != nil {
				return DispatchResult{}, tplErr
			}
			if tpl == nil {
				warnings = append(warnings, fmt.Sprintf("template %s not found for EGM %s", templateID, egmID))
				renderError = fmt.Sprintf("template %s not found", templateID)
			} else {
				activeVersion, activeErr := d.Store.GetActiveG2STemplateVersion(ctx, templateID)
				if activeErr != nil {
					return DispatchResult{}, activeErr
				}
				if activeVersion == nil {
					warnings = append(warnings, fmt.Sprintf("no active template version for template %s", templateID))
					renderError = fmt.Sprintf("no active template version for template %s", templateID)
				} else {
					templateVersionLabel = strings.TrimSpace(activeVersion.VersionLabel)
					templateVersionInt = parseVersionLabel(activeVersion.VersionLabel)

					doc, parseErr := g2sengine.ParseActionTemplateDocument(activeVersion.ActionsJSON)
					if parseErr != nil {
						warnings = append(warnings, fmt.Sprintf("template parse failed for template %s: %v", templateID, parseErr))
						renderError = parseErr.Error()
					} else {
						renderedMessage, renderErr := g2sengine.RenderActionMessage(doc, g2sengine.RenderRequest{
							TemplateID:        templateID,
							TemplateVersion:   templateVersionInt,
							TemplateActionKey: stepKey,
							ActionID:          definition.ID,
							ActionRunID:       run.ID,
							ActionStepID:      stepID,
							EGMID:             egmID,
							HostID:            strings.TrimSpace(request.Actor),
							Timestamp:         now,
							IPAddress:         ipAddress,
							EndpointPath:      endpointPath,
						})
						if renderErr != nil {
							warnings = append(warnings, fmt.Sprintf("template render failed for template %s action_key %s: %v", templateID, stepKey, renderErr))
							renderError = renderErr.Error()
						} else {
							rendered = true
							rawPayload = renderedMessage.RawPayload
							messageType = renderedMessage.MessageType
						}
					}
				}
			}
		}

		parsedSummary, marshalErr := json.Marshal(map[string]any{
			"dry_run":             true,
			"no_send":             true,
			"rendered":            rendered,
			"action_id":           definition.ID,
			"action_run_id":       run.ID,
			"egm_id":              egmID,
			"template_id":         templateID,
			"template_version":    templateVersionInt,
			"template_action_key": stepKey,
			"message_type":        messageType,
		})
		if marshalErr != nil {
			return DispatchResult{}, fmt.Errorf("marshal dry-run summary: %w", marshalErr)
		}

		entry := g2sengine.MessageJournalEntry{
			Timestamp:         now,
			Direction:         g2sengine.DirectionOutbound,
			EGMID:             egmID,
			ActionRunID:       run.ID,
			ActionStepID:      stepID,
			TemplateID:        templateID,
			TemplateVersion:   templateVersionLabel,
			MessageType:       messageType,
			RawPayload:        rawPayload,
			ParsedSummaryJSON: string(parsedSummary),
			Result:            g2sengine.MessageResultDryRun,
			Error:             renderError,
		}
		id, recordErr := d.Store.RecordMessageJournalEntry(ctx, entry)
		if recordErr != nil {
			return DispatchResult{}, recordErr
		}
		entry.ID = id
		prepared = append(prepared, entry)
	}

	if len(targets) == 0 {
		warnings = append(warnings, "action run has no target rows")
	}

	run.Status = actions.RunStatusDispatchPrepared
	if err := d.Store.UpdateActionRun(ctx, *run); err != nil {
		return DispatchResult{}, err
	}

	detailJSON, err := json.Marshal(map[string]any{
		"mode":              mode,
		"target_row_count":  len(targets),
		"prepared_messages": len(prepared),
		"warnings":          warnings,
	})
	if err != nil {
		return DispatchResult{}, fmt.Errorf("marshal dispatch audit detail: %w", err)
	}

	auditID, err := d.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:  now,
		Severity:    mapSeverity(definition.Severity, len(warnings)),
		EventType:   audit.EventTypeActionDispatchPrepared,
		Summary:     fmt.Sprintf("Action run %s prepared in DRY_RUN mode", run.ID),
		DetailJSON:  string(detailJSON),
		ActionRunID: run.ID,
		Operator:    strings.TrimSpace(request.Actor),
	})
	if err != nil {
		return DispatchResult{}, err
	}

	return DispatchResult{
		ActionRunID:      run.ID,
		Mode:             mode,
		PreparedMessages: prepared,
		TargetCount:      len(targets),
		WarningCount:     len(warnings),
		AuditEntryID:     auditID,
	}, nil
}

func parseVersionLabel(versionLabel string) int {
	trimmed := strings.TrimSpace(versionLabel)
	if trimmed == "" {
		return 0
	}
	if value, err := strconv.Atoi(trimmed); err == nil {
		return value
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "v") {
		if value, err := strconv.Atoi(strings.TrimPrefix(strings.ToLower(trimmed), "v")); err == nil {
			return value
		}
	}
	return 0
}

func mapSeverity(severity actions.ActionSeverity, warnings int) audit.AuditSeverity {
	if warnings > 0 {
		return audit.AuditSeverityWarning
	}
	switch severity {
	case actions.SeverityEmergency:
		return audit.AuditSeverityEmergency
	case actions.SeverityBroadcast:
		return audit.AuditSeverityWarning
	default:
		return audit.AuditSeverityInfo
	}
}

func (d *Dispatcher) SendPreparedMessages(ctx context.Context, request SendPreparedMessagesRequest) (SendPreparedMessagesResult, error) {
	if d.Store == nil {
		return SendPreparedMessagesResult{}, fmt.Errorf("store is required")
	}
	actionRunID := strings.TrimSpace(request.ActionRunID)
	if actionRunID == "" {
		return SendPreparedMessagesResult{}, fmt.Errorf("action_run_id is required")
	}

	now := request.RequestedAt
	if now.IsZero() {
		clock := d.Clock
		if clock == nil {
			clock = time.Now
		}
		now = clock().UTC()
	}

	run, err := d.Store.GetActionRun(ctx, actionRunID)
	if err != nil {
		return SendPreparedMessagesResult{}, err
	}
	if run == nil {
		return SendPreparedMessagesResult{}, fmt.Errorf("action run %q not found", actionRunID)
	}

	entries, err := d.Store.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{
		ActionRunID: actionRunID,
		Direction:   g2sengine.DirectionOutbound,
		Limit:       500,
	})
	if err != nil {
		return SendPreparedMessagesResult{}, err
	}

	sender := d.Sender
	if sender == nil {
		sender = g2stransport.NewSender(request.TransportMode)
	}

	sentCount := 0
	failedCount := 0
	blockedCount := 0
	processed := 0
	for _, entry := range entries {
		switch entry.Result {
		case g2sengine.MessageResultDryRun, g2sengine.MessageResultSendBlocked, g2sengine.MessageResultSendFailed, g2sengine.MessageResultSendAttempted:
		default:
			continue
		}
		processed++
		egmRecord, err := d.Store.GetEGMRecord(ctx, entry.EGMID)
		if err != nil {
			return SendPreparedMessagesResult{}, err
		}
		endpointURL := strings.TrimSpace(entry.ToEndpoint)
		if endpointURL == "" {
			endpointURL = endpointURLFromEGM(egmRecord)
		}

		sendResult, sendErr := sender.Send(ctx, g2stransport.SendRequest{
			MessageID:     entry.ID,
			ActionRunID:   actionRunID,
			EGMID:         entry.EGMID,
			EndpointURL:   endpointURL,
			Method:        "POST",
			ContentType:   "application/soap+xml",
			RawPayload:    entry.RawPayload,
			TimeoutMS:     request.DefaultTimeout,
			AllowRealSend: request.AllowRealSend,
			TransportMode: request.TransportMode,
			RequestedAt:   now,
		})
		if sendErr != nil {
			sendResult.Error = sendErr.Error()
			sendResult.CompletedAt = now
		}

		resultType := g2sengine.MessageResultSendFailed
		if sendResult.Blocked {
			resultType = g2sengine.MessageResultSendBlocked
			blockedCount++
		} else if sendResult.Sent {
			resultType = g2sengine.MessageResultSendSucceeded
			sentCount++
		} else {
			failedCount++
		}
		completedAt := sendResult.CompletedAt
		var sentAt *time.Time
		if sendResult.Sent {
			value := sendResult.CompletedAt
			sentAt = &value
		}
		if err := d.Store.UpdateMessageJournalResult(
			ctx,
			entry.ID,
			resultType,
			sendResult.Error,
			sendResult.ResponseExcerpt,
			sendResult.HTTPStatusCode,
			sendResult.LatencyMS,
			string(sendResult.TransportMode),
			sentAt,
			&completedAt,
		); err != nil {
			return SendPreparedMessagesResult{}, err
		}
	}

	eventType := audit.EventTypeMessageSendAttempted
	severity := audit.AuditSeverityInfo
	if blockedCount > 0 && sentCount == 0 && failedCount == 0 {
		eventType = audit.EventTypeMessageSendBlocked
		severity = audit.AuditSeverityWarning
	}
	if failedCount > 0 {
		eventType = audit.EventTypeMessageSendFailed
		severity = audit.AuditSeverityWarning
	}
	if sentCount > 0 && failedCount == 0 && blockedCount == 0 {
		eventType = audit.EventTypeMessageSendSucceeded
		severity = audit.AuditSeverityInfo
	}

	detailJSON, err := json.Marshal(map[string]any{
		"action_run_id":   actionRunID,
		"transport_mode":  request.TransportMode,
		"allow_real_send": request.AllowRealSend,
		"processed":       processed,
		"sent":            sentCount,
		"failed":          failedCount,
		"blocked":         blockedCount,
	})
	if err != nil {
		return SendPreparedMessagesResult{}, fmt.Errorf("marshal send-prepared detail: %w", err)
	}
	auditID, err := d.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:  now,
		Severity:    severity,
		EventType:   eventType,
		Summary:     fmt.Sprintf("Prepared messages processed for action run %s", actionRunID),
		DetailJSON:  string(detailJSON),
		ActionRunID: actionRunID,
		Operator:    strings.TrimSpace(request.Actor),
	})
	if err != nil {
		return SendPreparedMessagesResult{}, err
	}

	return SendPreparedMessagesResult{
		ActionRunID:   actionRunID,
		TransportMode: request.TransportMode,
		SentCount:     sentCount,
		FailedCount:   failedCount,
		BlockedCount:  blockedCount,
		AuditEntryID:  auditID,
	}, nil
}

func endpointURLFromEGM(egmRecord *egms.EGMRecord) string {
	if egmRecord == nil {
		return ""
	}
	endpointPath := strings.TrimSpace(egmRecord.EndpointPath)
	if strings.HasPrefix(strings.ToLower(endpointPath), "http://") || strings.HasPrefix(strings.ToLower(endpointPath), "https://") {
		return endpointPath
	}
	ipAddress := strings.TrimSpace(egmRecord.IPAddress)
	if ipAddress == "" || endpointPath == "" {
		return ""
	}
	if !strings.HasPrefix(endpointPath, "/") {
		endpointPath = "/" + endpointPath
	}
	return "http://" + ipAddress + endpointPath
}
