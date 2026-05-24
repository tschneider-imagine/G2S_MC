package actiondispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
)

type Dispatcher struct {
	Store Store
	Clock func() time.Time
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
		if egm == nil {
			warnings = append(warnings, fmt.Sprintf("target EGM %s not found", target.TargetEGMID))
			continue
		}

		templateID := strings.TrimSpace(egm.TemplateID)
		if templateID != "" {
			tpl, tplErr := d.Store.GetG2STemplate(ctx, templateID)
			if tplErr != nil {
				return DispatchResult{}, tplErr
			}
			if tpl == nil {
				warnings = append(warnings, fmt.Sprintf("template %s not found for EGM %s", templateID, egm.EGMID))
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("EGM %s has no template assigned", egm.EGMID))
		}

		parsedSummary, marshalErr := json.Marshal(map[string]any{
			"dry_run":   true,
			"no_send":   true,
			"action_id": definition.ID,
			"egm_id":    egm.EGMID,
			"step_key":  stepKey,
		})
		if marshalErr != nil {
			return DispatchResult{}, fmt.Errorf("marshal dry-run summary: %w", marshalErr)
		}

		entry := g2sengine.MessageJournalEntry{
			Timestamp:         now,
			Direction:         g2sengine.DirectionOutbound,
			EGMID:             egm.EGMID,
			ActionRunID:       run.ID,
			ActionStepID:      stepID,
			TemplateID:        templateID,
			MessageType:       stepKey,
			RawPayload:        fmt.Sprintf("DRY_RUN_NO_SEND action=%s egm=%s step=%s", definition.ID, egm.EGMID, stepKey),
			ParsedSummaryJSON: string(parsedSummary),
			Result:            g2sengine.MessageResultDryRun,
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
