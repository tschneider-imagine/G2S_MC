package actionexecutor

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actionruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

type Executor struct {
	Store  Store
	Sender g2stransport.Sender
	Clock  func() time.Time
	Sleep  func(time.Duration)
}

func (e *Executor) Execute(ctx context.Context, request ExecuteRequest) (ExecuteResult, error) {
	if e.Store == nil {
		return ExecuteResult{}, fmt.Errorf("store is required")
	}
	runID := strings.TrimSpace(request.ActionRunID)
	if runID == "" {
		return ExecuteResult{}, fmt.Errorf("action_run_id is required")
	}

	now := request.RequestedAt
	if now.IsZero() {
		now = e.now()
	}
	delivery := request.Delivery.Normalize()

	run, err := e.Store.GetActionRun(ctx, runID)
	if err != nil {
		return ExecuteResult{}, err
	}
	if run == nil {
		return ExecuteResult{}, fmt.Errorf("action run %q not found", runID)
	}
	if run.Status != actions.RunStatusPending {
		return ExecuteResult{}, fmt.Errorf("action run %q status must be PENDING", runID)
	}

	definition, err := e.Store.GetActionDefinition(ctx, run.ActionDefinitionID)
	if err != nil {
		return ExecuteResult{}, err
	}
	if definition == nil {
		return ExecuteResult{}, fmt.Errorf("action definition %q not found", run.ActionDefinitionID)
	}
	if len(definition.Steps) == 0 {
		return ExecuteResult{}, fmt.Errorf("action definition %q has no steps", definition.ID)
	}

	retryPolicy := parseRetryPolicy(definition.RetryPolicyJSON)
	escalationPolicy := parseEscalationPolicy(definition.EscalationJSON)

	targetRows, err := e.Store.ListActionTargetResults(ctx, run.ID)
	if err != nil {
		return ExecuteResult{}, err
	}
	if request.MaxTargets > 0 && len(targetRows) > request.MaxTargets {
		targetRows = targetRows[:request.MaxTargets]
	}

	// Move run into RUNNING immediately to avoid concurrent double execution.
	run.Status = actions.RunStatusRunning
	if err := e.Store.UpdateActionRun(ctx, *run); err != nil {
		return ExecuteResult{}, err
	}

	auditIDs := make([]int64, 0, 8)
	startAuditID, err := e.recordAudit(ctx, now, mapAuditSeverity(definition.Severity), audit.EventTypeActionStarted, fmt.Sprintf("Action run %s started", run.ID), map[string]any{
		"action_run_id":    run.ID,
		"action_id":        definition.ID,
		"target_count":     len(targetRows),
		"retry_count":      retryPolicy.Count,
		"retry_delay":      retryPolicy.DelayMS,
		"delivery_mode":    delivery.Mode,
		"allow_delivery":   delivery.AllowDelivery,
		"capture_only":     delivery.CaptureOnly,
		"delivery_timeout": delivery.TimeoutMS,
	}, run.ID, strings.TrimSpace(request.Actor), 0)
	if err != nil {
		return ExecuteResult{}, err
	}
	auditIDs = append(auditIDs, startAuditID)

	attemptSummaries := []ExecutionAttemptSummary{}
	warnings := []string{}
	totalConfirmed := 0
	totalFailed := 0
	maxAttemptsUsed := 0

	for i := range targetRows {
		target := targetRows[i]
		targetNow := e.now()
		egmRecord, getErr := e.Store.GetEGMRecord(ctx, target.TargetEGMID)
		if getErr != nil {
			return ExecuteResult{}, getErr
		}
		if egmRecord == nil {
			target.AttemptCount++
			target.Status = actions.TargetStatusFailed
			target.LastError = fmt.Sprintf("egm %s not found", target.TargetEGMID)
			target.LastResultAt = &targetNow
			if err := e.Store.UpdateActionTargetResult(ctx, target); err != nil {
				return ExecuteResult{}, err
			}
			targetRows[i] = target
			totalFailed++
			warnings = append(warnings, target.LastError)
			continue
		}

		templateID := strings.TrimSpace(egmRecord.TemplateID)
		if templateID == "" {
			target.AttemptCount++
			target.Status = actions.TargetStatusFailed
			target.LastError = fmt.Sprintf("egm %s has no template assigned", target.TargetEGMID)
			target.LastResultAt = &targetNow
			if err := e.Store.UpdateActionTargetResult(ctx, target); err != nil {
				return ExecuteResult{}, err
			}
			targetRows[i] = target
			totalFailed++
			warnings = append(warnings, target.LastError)
			continue
		}

		activeVersion, getErr := e.Store.GetActiveG2STemplateVersion(ctx, templateID)
		if getErr != nil {
			return ExecuteResult{}, getErr
		}
		if activeVersion == nil {
			target.AttemptCount++
			target.Status = actions.TargetStatusFailed
			target.LastError = fmt.Sprintf("template %s has no active version", templateID)
			target.LastResultAt = &targetNow
			if err := e.Store.UpdateActionTargetResult(ctx, target); err != nil {
				return ExecuteResult{}, err
			}
			targetRows[i] = target
			totalFailed++
			warnings = append(warnings, target.LastError)
			continue
		}

		templateDoc, parseErr := g2sengine.ParseActionTemplateDocument(activeVersion.ActionsJSON)
		if parseErr != nil {
			target.AttemptCount++
			target.Status = actions.TargetStatusFailed
			target.LastError = fmt.Sprintf("template %s parse failed: %v", templateID, parseErr)
			target.LastResultAt = &targetNow
			if err := e.Store.UpdateActionTargetResult(ctx, target); err != nil {
				return ExecuteResult{}, err
			}
			targetRows[i] = target
			totalFailed++
			warnings = append(warnings, target.LastError)
			continue
		}

		endpointURL := endpointURLFromEGM(egmRecord)
		totalAttempts := retryPolicy.Count + 1
		if totalAttempts < 1 {
			totalAttempts = 1
		}

		confirmed := false
		lastErr := ""
		for attempt := 1; attempt <= totalAttempts; attempt++ {
			maxAttemptsUsed = max(maxAttemptsUsed, attempt)
			stepFailed := false

			for _, step := range definition.Steps {
				stepNow := e.now()
				rendered, renderErr := g2sengine.RenderActionMessage(templateDoc, g2sengine.RenderRequest{
					TemplateID:        templateID,
					TemplateVersion:   parseTemplateVersionInt(activeVersion.VersionLabel),
					TemplateActionKey: step.TemplateActionKey,
					ActionID:          definition.ID,
					ActionRunID:       run.ID,
					ActionStepID:      step.ID,
					EGMID:             target.TargetEGMID,
					HostID:            strings.TrimSpace(request.Actor),
					Timestamp:         stepNow,
					IPAddress:         strings.TrimSpace(egmRecord.IPAddress),
					EndpointPath:      strings.TrimSpace(egmRecord.EndpointPath),
				})
				if renderErr != nil {
					lastErr = fmt.Sprintf("render failed: %v", renderErr)
					stepFailed = true
					attemptSummaries = append(attemptSummaries, ExecutionAttemptSummary{
						EGMID:           target.TargetEGMID,
						ActionStepID:    step.ID,
						TemplateID:      templateID,
						TemplateVersion: activeVersion.VersionLabel,
						Attempt:         attempt,
						DeliveryResult:  string(g2sengine.MessageResultFailed),
						MatchOutcome:    string(g2sengine.MatchOutcomeNoMatch),
						Error:           lastErr,
					})
					_, _ = e.recordAudit(ctx, stepNow, audit.AuditSeverityWarning, audit.EventTypeActionStep, fmt.Sprintf("Render failed for %s step %s", target.TargetEGMID, step.ID), map[string]any{
						"action_run_id": run.ID,
						"egm_id":        target.TargetEGMID,
						"template_id":   templateID,
						"step_id":       step.ID,
						"error":         lastErr,
					}, run.ID, strings.TrimSpace(request.Actor), 0)
					break
				}

				summaryJSON, marshalErr := json.Marshal(map[string]any{
					"action_id":           definition.ID,
					"action_run_id":       run.ID,
					"egm_id":              target.TargetEGMID,
					"step_id":             step.ID,
					"template_id":         templateID,
					"template_version":    activeVersion.VersionLabel,
					"template_action_key": step.TemplateActionKey,
					"message_type":        rendered.MessageType,
				})
				if marshalErr != nil {
					return ExecuteResult{}, fmt.Errorf("marshal message summary: %w", marshalErr)
				}

				messageID, recordErr := e.Store.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
					Timestamp:         stepNow,
					Direction:         g2sengine.DirectionOutbound,
					EGMID:             target.TargetEGMID,
					ActionRunID:       run.ID,
					ActionStepID:      step.ID,
					InputTransitionID: run.InputTransitionID,
					TemplateID:        templateID,
					TemplateVersion:   activeVersion.VersionLabel,
					MessageType:       rendered.MessageType,
					ToEndpoint:        endpointURL,
					RawPayload:        rendered.RawPayload,
					ParsedSummaryJSON: string(summaryJSON),
					Result:            g2sengine.MessageResultSendAttempted,
				})
				if recordErr != nil {
					return ExecuteResult{}, recordErr
				}

				if e.Sender == nil {
					lastErr = "delivery sender is not configured"
					completeAt := e.now()
					if err := e.Store.UpdateMessageJournalResult(
						ctx,
						messageID,
						g2sengine.MessageResultSendFailed,
						lastErr,
						"",
						0,
						0,
						"",
						nil,
						&completeAt,
					); err != nil {
						return ExecuteResult{}, err
					}
					attemptSummaries = append(attemptSummaries, ExecutionAttemptSummary{
						EGMID:           target.TargetEGMID,
						ActionStepID:    step.ID,
						TemplateID:      templateID,
						TemplateVersion: activeVersion.VersionLabel,
						MessageID:       messageID,
						Attempt:         attempt,
						DeliveryResult:  string(g2sengine.MessageResultSendFailed),
						MatchOutcome:    string(g2sengine.MatchOutcomeNoMatch),
						Error:           lastErr,
					})
					stepFailed = true
					break
				}

				sendResult, sendErr := e.Sender.Send(ctx, g2stransport.SendRequest{
					MessageID:       messageID,
					ActionRunID:     run.ID,
					EGMID:           target.TargetEGMID,
					EndpointURL:     endpointURL,
					Method:          httpMethodFromMessageType(rendered.MessageType),
					ContentType:     rendered.ContentType,
					Headers:         rendered.Headers,
					RawPayload:      rendered.RawPayload,
					AllowRealSend:   delivery.AllowDelivery,
					TransportMode:   delivery.TransportMode(),
					RequestedAt:     stepNow,
					CaptureOnlySend: delivery.CaptureOnly,
					TimeoutMS:       delivery.TimeoutMS,
				})
				if sendErr != nil {
					sendResult.Error = sendErr.Error()
				}
				if sendResult.CompletedAt.IsZero() {
					sendResult.CompletedAt = e.now()
				}

				resultType := g2sengine.MessageResultSendFailed
				if sendResult.Blocked {
					resultType = g2sengine.MessageResultSendBlocked
				} else if sendResult.Sent {
					resultType = g2sengine.MessageResultSendSucceeded
				}
				var sentAt *time.Time
				if sendResult.Sent {
					value := sendResult.CompletedAt
					sentAt = &value
				}
				completedAt := sendResult.CompletedAt
				if err := e.Store.UpdateMessageJournalResult(
					ctx,
					messageID,
					resultType,
					sendResult.Error,
					sendResult.ResponseExcerpt,
					sendResult.HTTPStatusCode,
					sendResult.LatencyMS,
					string(sendResult.TransportMode),
					sentAt,
					&completedAt,
				); err != nil {
					return ExecuteResult{}, err
				}

				matchOutcome := g2sengine.MatchOutcomeNoMatch
				matchRuleID := ""
				matchErrText := ""
				if sendResult.Sent {
					matchResult, matchErr := g2sengine.MatchMessage(
						sendResult.ResponseExcerpt,
						"",
						rendered.MessageType,
						activeVersion.ConfirmationRulesJSON,
						activeVersion.FailureRulesJSON,
					)
					if matchErr != nil {
						matchErrText = matchErr.Error()
						lastErr = "matcher error: " + matchErr.Error()
						stepFailed = true
					} else {
						matchOutcome = g2sengine.MatchOutcome(matchResult.Outcome)
						matchRuleID = matchResult.RuleID
						switch matchOutcome {
						case g2sengine.MatchOutcomeExpected:
							// continue to next step
						case g2sengine.MatchOutcomeFailure:
							lastErr = nonEmpty(matchResult.Reason, "failure matcher matched")
							stepFailed = true
						default:
							lastErr = nonEmpty(matchResult.Reason, "no expected matcher match")
							stepFailed = true
						}
					}
				} else {
					lastErr = nonEmpty(sendResult.Error, "delivery did not succeed")
					stepFailed = true
				}

				attemptSummaries = append(attemptSummaries, ExecutionAttemptSummary{
					EGMID:           target.TargetEGMID,
					ActionStepID:    step.ID,
					TemplateID:      templateID,
					TemplateVersion: activeVersion.VersionLabel,
					MessageID:       messageID,
					Attempt:         attempt,
					DeliveryResult:  string(resultType),
					MatchOutcome:    string(matchOutcome),
					Error:           nonEmpty(lastErr, matchErrText),
				})

				stepAuditID, auditErr := e.recordAudit(ctx, e.now(), mapAuditSeverity(definition.Severity), audit.EventTypeMessageSent, fmt.Sprintf("Delivery attempted for %s step %s", target.TargetEGMID, step.ID), map[string]any{
					"action_run_id":    run.ID,
					"egm_id":           target.TargetEGMID,
					"template_id":      templateID,
					"template_version": activeVersion.VersionLabel,
					"step_id":          step.ID,
					"delivery_result":  resultType,
					"match_outcome":    matchOutcome,
					"match_rule_id":    matchRuleID,
					"error":            lastErr,
				}, run.ID, strings.TrimSpace(request.Actor), messageID)
				if auditErr != nil {
					return ExecuteResult{}, auditErr
				}
				auditIDs = append(auditIDs, stepAuditID)

				if stepFailed {
					break
				}
			}

			target.AttemptCount++
			lastResultAt := e.now()
			target.LastResultAt = &lastResultAt
			if stepFailed {
				target.LastError = lastErr
				if attempt < totalAttempts {
					target.Status = actions.TargetStatusPending
					retryAuditID, retryErr := e.recordAudit(ctx, e.now(), audit.AuditSeverityWarning, audit.EventTypeRetry, fmt.Sprintf("Retry scheduled for %s", target.TargetEGMID), map[string]any{
						"action_run_id": run.ID,
						"egm_id":        target.TargetEGMID,
						"attempt":       attempt,
						"max_attempts":  totalAttempts,
						"error":         lastErr,
					}, run.ID, strings.TrimSpace(request.Actor), 0)
					if retryErr != nil {
						return ExecuteResult{}, retryErr
					}
					auditIDs = append(auditIDs, retryAuditID)
					if retryPolicy.DelayMS > 0 {
						e.sleep(time.Duration(retryPolicy.DelayMS) * time.Millisecond)
					}
					if err := e.Store.UpdateActionTargetResult(ctx, target); err != nil {
						return ExecuteResult{}, err
					}
					targetRows[i] = target
					continue
				}
				target.Status = actions.TargetStatusFailed
			} else {
				target.Status = actions.TargetStatusConfirmed
				target.LastError = ""
				confirmed = true
			}
			if err := e.Store.UpdateActionTargetResult(ctx, target); err != nil {
				return ExecuteResult{}, err
			}
			targetRows[i] = target
			break
		}

		if confirmed {
			totalConfirmed++
		} else {
			totalFailed++
		}
	}

	run.ConfirmedCount = totalConfirmed
	run.FailedCount = totalFailed
	run.TargetCount = len(targetRows)
	statusText := "completed"
	if totalFailed == 0 && len(targetRows) > 0 {
		run.Status = actions.RunStatusSucceeded
		statusText = string(actions.RunStatusSucceeded)
	} else {
		run.Status = actions.RunStatusFailed
		statusText = string(actions.RunStatusFailed)
	}

	var escalationRun *actions.ActionRun
	if run.Status == actions.RunStatusFailed {
		shouldEscalate := strings.TrimSpace(escalationPolicy.ActionID) != "" && (escalationPolicy.AfterAttempts <= 0 || maxAttemptsUsed >= escalationPolicy.AfterAttempts)
		if shouldEscalate {
			queuer := &actionruntime.Queuer{Store: e.Store, Clock: e.Clock}
			queueResult, queueErr := queuer.QueueActionRun(ctx, actionruntime.QueueRequest{
				InputTransition: inputs.InputTransition{ID: run.InputTransitionID},
				ActionID:        escalationPolicy.ActionID,
				TriggerReason:   fmt.Sprintf("escalation from action run %s", run.ID),
				Actor:           strings.TrimSpace(request.Actor),
				QueuedAt:        e.now(),
			})
			if queueErr != nil {
				return ExecuteResult{}, queueErr
			}
			if queueResult.Queued && queueResult.ActionRun != nil {
				escalationRun = queueResult.ActionRun
				run.EscalatedCount++
				run.Status = actions.RunStatusEscalated
				statusText = string(actions.RunStatusEscalated)
				escalationAuditID, escalationAuditErr := e.recordAudit(ctx, e.now(), audit.AuditSeverityWarning, audit.EventTypeEscalation, fmt.Sprintf("Escalation action %s queued", escalationPolicy.ActionID), map[string]any{
					"action_run_id":        run.ID,
					"escalation_action_id": escalationPolicy.ActionID,
					"escalation_run_id":    escalationRun.ID,
				}, run.ID, strings.TrimSpace(request.Actor), 0)
				if escalationAuditErr != nil {
					return ExecuteResult{}, escalationAuditErr
				}
				auditIDs = append(auditIDs, escalationAuditID)
			}
		}
	}

	completedAt := e.now()
	run.CompletedAt = &completedAt
	if err := e.Store.UpdateActionRun(ctx, *run); err != nil {
		return ExecuteResult{}, err
	}

	finalAuditID, err := e.recordAudit(ctx, completedAt, mapAuditSeverity(definition.Severity), audit.EventTypeOperatorAction, fmt.Sprintf("Action run %s completed with status %s", run.ID, run.Status), map[string]any{
		"action_run_id":     run.ID,
		"status":            run.Status,
		"confirmed_count":   run.ConfirmedCount,
		"failed_count":      run.FailedCount,
		"escalated_count":   run.EscalatedCount,
		"attempts_recorded": len(attemptSummaries),
	}, run.ID, strings.TrimSpace(request.Actor), 0)
	if err != nil {
		return ExecuteResult{}, err
	}
	auditIDs = append(auditIDs, finalAuditID)

	return ExecuteResult{
		ActionRun:     *run,
		TargetResults: targetRows,
		Attempts:      attemptSummaries,
		AuditEntryIDs: auditIDs,
		Status:        statusText,
		Warnings:      warnings,
		EscalationRun: escalationRun,
	}, nil
}

func (e *Executor) now() time.Time {
	clock := e.Clock
	if clock == nil {
		clock = time.Now
	}
	return clock().UTC()
}

func (e *Executor) sleep(d time.Duration) {
	sleepFn := e.Sleep
	if sleepFn == nil {
		sleepFn = time.Sleep
	}
	sleepFn(d)
}

func (e *Executor) recordAudit(ctx context.Context, occurredAt time.Time, severity audit.AuditSeverity, eventType string, summary string, detail map[string]any, actionRunID string, actor string, messageID int64) (int64, error) {
	detailJSON := ""
	if len(detail) > 0 {
		encoded, err := json.Marshal(detail)
		if err != nil {
			return 0, fmt.Errorf("marshal audit detail: %w", err)
		}
		detailJSON = string(encoded)
	}
	return e.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:       occurredAt,
		Severity:         severity,
		EventType:        eventType,
		Summary:          summary,
		DetailJSON:       detailJSON,
		ActionRunID:      actionRunID,
		MessageJournalID: messageID,
		Operator:         actor,
	})
}

func mapAuditSeverity(severity actions.ActionSeverity) audit.AuditSeverity {
	switch severity {
	case actions.SeverityEmergency:
		return audit.AuditSeverityEmergency
	case actions.SeverityBroadcast:
		return audit.AuditSeverityWarning
	default:
		return audit.AuditSeverityInfo
	}
}

type retryPolicy struct {
	Count   int `json:"count"`
	DelayMS int `json:"delay_ms"`
}

func parseRetryPolicy(raw string) retryPolicy {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return retryPolicy{}
	}
	var policy retryPolicy
	if err := json.Unmarshal([]byte(trimmed), &policy); err != nil {
		return retryPolicy{}
	}
	if policy.Count < 0 {
		policy.Count = 0
	}
	if policy.DelayMS < 0 {
		policy.DelayMS = 0
	}
	return policy
}

type escalationPolicy struct {
	ActionID      string `json:"escalation_action_id"`
	AfterAttempts int    `json:"after_attempts"`
}

func parseEscalationPolicy(raw string) escalationPolicy {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return escalationPolicy{}
	}
	var policy escalationPolicy
	if err := json.Unmarshal([]byte(trimmed), &policy); err != nil {
		return escalationPolicy{}
	}
	policy.ActionID = strings.TrimSpace(policy.ActionID)
	if policy.AfterAttempts < 0 {
		policy.AfterAttempts = 0
	}
	return policy
}

func endpointURLFromEGM(record *egms.EGMRecord) string {
	if record == nil {
		return ""
	}
	endpointPath := strings.TrimSpace(record.EndpointPath)
	if strings.HasPrefix(strings.ToLower(endpointPath), "http://") || strings.HasPrefix(strings.ToLower(endpointPath), "https://") {
		return endpointPath
	}
	ipAddress := strings.TrimSpace(record.IPAddress)
	if ipAddress == "" || endpointPath == "" {
		return ""
	}
	if !strings.HasPrefix(endpointPath, "/") {
		endpointPath = "/" + endpointPath
	}
	return "http://" + ipAddress + endpointPath
}

func parseTemplateVersionInt(versionLabel string) int {
	trimmed := strings.TrimSpace(versionLabel)
	if trimmed == "" {
		return 0
	}
	if value, err := strconv.Atoi(trimmed); err == nil {
		return value
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "v") {
		if value, err := strconv.Atoi(strings.TrimPrefix(lower, "v")); err == nil {
			return value
		}
	}
	return 0
}

func httpMethodFromMessageType(_ string) string {
	return "POST"
}

func nonEmpty(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
