package inbound

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func (s *Service) Process(ctx context.Context, message InboundMessage) (ProcessResult, error) {
	if s.Store == nil {
		return ProcessResult{}, fmt.Errorf("store is required")
	}

	now := message.ReceivedAt
	if now.IsZero() {
		now = s.now()
	}

	resolved := resolveMetadata(message)
	correlation, err := s.correlate(ctx, resolved)
	if err != nil {
		return ProcessResult{}, err
	}
	if correlation.Run != nil && strings.TrimSpace(resolved.ActionRunID) == "" {
		resolved.ActionRunID = correlation.Run.ID
	}
	if correlation.Target != nil && strings.TrimSpace(resolved.EGMID) == "" {
		resolved.EGMID = correlation.Target.TargetEGMID
	}

	summaryJSON := encodeSummaryJSON(map[string]any{
		"egm_id":        resolved.EGMID,
		"action_run_id": resolved.ActionRunID,
		"message_type":  resolved.MessageType,
		"remote_addr":   strings.TrimSpace(message.RemoteAddr),
		"warnings":      resolved.Warnings,
		"correlated":    correlation.HasTarget,
	})

	journal := g2sengine.MessageJournalEntry{
		Timestamp:         now,
		Direction:         g2sengine.DirectionInbound,
		FromEndpoint:      firstNonEmpty(message.FromEndpoint, message.RemoteAddr),
		ToEndpoint:        message.ToEndpoint,
		EGMID:             resolved.EGMID,
		ActionRunID:       resolved.ActionRunID,
		HandlerRuleID:     strings.TrimSpace(message.HandlerRuleID),
		MessageType:       resolved.MessageType,
		RawPayload:        message.RawPayload,
		ParsedSummaryJSON: summaryJSON,
		Result:            g2sengine.MessageResultReceived,
	}
	messageID, err := s.Store.RecordMessageJournalEntry(ctx, journal)
	if err != nil {
		return ProcessResult{}, err
	}

	result := ProcessResult{
		MessageID:   messageID,
		EGMID:       resolved.EGMID,
		ActionRunID: resolved.ActionRunID,
		Warnings:    append([]string{}, resolved.Warnings...),
	}

	receiveAuditID, err := s.recordAudit(ctx, audit.AuditTimelineEntry{
		OccurredAt:       now,
		Severity:         audit.AuditSeverityInfo,
		EventType:        audit.EventTypeMessageReceived,
		Summary:          "Inbound message received",
		DetailJSON:       encodeSummaryJSON(map[string]any{"message_type": resolved.MessageType, "remote_addr": message.RemoteAddr}),
		ActionRunID:      resolved.ActionRunID,
		MessageJournalID: messageID,
	})
	if err != nil {
		return result, err
	}
	result.AuditEntryIDs = append(result.AuditEntryIDs, receiveAuditID)
	if !correlation.HasTarget {
		if correlation.Warning != "" {
			result.Warnings = append(result.Warnings, correlation.Warning)
			auditID, auditErr := s.recordAudit(ctx, audit.AuditTimelineEntry{
				OccurredAt:       now,
				Severity:         audit.AuditSeverityWarning,
				EventType:        audit.EventTypeSystemWarning,
				Summary:          "Inbound message not correlated to a target",
				DetailJSON:       encodeSummaryJSON(map[string]any{"warning": correlation.Warning}),
				ActionRunID:      resolved.ActionRunID,
				MessageJournalID: messageID,
			})
			if auditErr != nil {
				return result, auditErr
			}
			result.AuditEntryIDs = append(result.AuditEntryIDs, auditID)
		}
		return result, nil
	}

	result.Correlated = true
	result.CorrelationRef = correlation.Target.TargetEGMID
	if result.EGMID == "" {
		result.EGMID = correlation.Target.TargetEGMID
	}
	if result.ActionRunID == "" {
		result.ActionRunID = correlation.Run.ID
	}
	correlationAuditID, err := s.recordAudit(ctx, audit.AuditTimelineEntry{
		OccurredAt:       now,
		Severity:         audit.AuditSeverityInfo,
		EventType:        audit.EventTypeConfirmation,
		Summary:          "Inbound message correlated to action target",
		DetailJSON:       encodeSummaryJSON(map[string]any{"target_egm_id": correlation.Target.TargetEGMID}),
		ActionRunID:      correlation.Run.ID,
		MessageJournalID: messageID,
	})
	if err != nil {
		return result, err
	}
	result.AuditEntryIDs = append(result.AuditEntryIDs, correlationAuditID)

	matched, err := s.applyMatcher(ctx, now, messageID, resolved, correlation, message.RawPayload, summaryJSON)
	if err != nil {
		return result, err
	}
	result.MatchOutcome = matched.Outcome
	result.TargetUpdated = matched.TargetUpdated
	result.TargetStatus = matched.TargetStatus
	result.AuditEntryIDs = append(result.AuditEntryIDs, matched.AuditEntryIDs...)
	result.Warnings = append(result.Warnings, matched.Warnings...)
	return result, nil
}

type resolvedMetadata struct {
	EGMID       string
	ActionRunID string
	MessageType string
	Warnings    []string
}

func resolveMetadata(message InboundMessage) resolvedMetadata {
	body := strings.TrimSpace(message.RawPayload)
	query := normalizeLookupMap(message.QueryParams)
	headers := normalizeLookupMap(message.Headers)
	jsonBody := map[string]any{}
	_ = json.Unmarshal([]byte(body), &jsonBody)

	egmID := firstNonEmpty(
		strings.TrimSpace(message.EGMID),
		query["egm_id"],
		query["egmid"],
		headers["x-egm-id"],
		headers["egm-id"],
		findJSONValue(jsonBody, "egm_id"),
		findJSONValue(jsonBody, "egmId"),
		findXMLAttribute(body, "egmId"),
		findXMLElement(body, "egmId"),
	)
	actionRunID := firstNonEmpty(
		strings.TrimSpace(message.ActionRunID),
		query["action_run_id"],
		query["actionrunid"],
		headers["x-action-run-id"],
		headers["action-run-id"],
		findJSONValue(jsonBody, "action_run_id"),
		findJSONValue(jsonBody, "actionRunId"),
		findXMLAttribute(body, "actionRunId"),
		findXMLElement(body, "actionRunId"),
	)
	messageType := firstNonEmpty(
		strings.TrimSpace(message.MessageType),
		query["message_type"],
		query["type"],
		headers["x-message-type"],
		findJSONValue(jsonBody, "message_type"),
		findJSONValue(jsonBody, "messageType"),
		detectMessageTypeFromXML(body),
	)

	warnings := []string{}
	if egmID == "" {
		warnings = append(warnings, "egm id not detected")
	}
	if messageType == "" {
		warnings = append(warnings, "message type not detected")
	}
	return resolvedMetadata{
		EGMID:       egmID,
		ActionRunID: actionRunID,
		MessageType: messageType,
		Warnings:    warnings,
	}
}

type correlation struct {
	Run       *actions.ActionRun
	Target    *actions.ActionTargetResult
	HasTarget bool
	Warning   string
}

func (s *Service) correlate(ctx context.Context, resolved resolvedMetadata) (correlation, error) {
	if resolved.ActionRunID != "" {
		run, err := s.Store.GetActionRun(ctx, resolved.ActionRunID)
		if err != nil {
			return correlation{}, err
		}
		if run == nil {
			return correlation{Warning: "action run not found"}, nil
		}
		target, warn, err := s.selectTargetForRun(ctx, *run, resolved.EGMID)
		if err != nil {
			return correlation{}, err
		}
		if target == nil {
			return correlation{Run: run, Warning: warn}, nil
		}
		return correlation{Run: run, Target: target, HasTarget: true}, nil
	}
	if resolved.EGMID == "" {
		return correlation{}, nil
	}

	candidates, err := s.findActiveRunCandidates(ctx, resolved.EGMID)
	if err != nil {
		return correlation{}, err
	}
	if len(candidates) == 0 {
		return correlation{Warning: "no active run found for egm"}, nil
	}
	if len(candidates) > 1 {
		return correlation{Warning: "ambiguous action run correlation for egm"}, nil
	}
	target, warn, err := s.selectTargetForRun(ctx, candidates[0], resolved.EGMID)
	if err != nil {
		return correlation{}, err
	}
	if target == nil {
		return correlation{Run: &candidates[0], Warning: warn}, nil
	}
	return correlation{Run: &candidates[0], Target: target, HasTarget: true}, nil
}

func (s *Service) findActiveRunCandidates(ctx context.Context, egmID string) ([]actions.ActionRun, error) {
	statuses := []actions.ActionRunStatus{actions.RunStatusRunning, actions.RunStatusWaitingConfirmation}
	matches := []actions.ActionRun{}
	for _, status := range statuses {
		runs, err := s.Store.ListActionRuns(ctx, store.ActionRunListQuery{Status: status, Limit: 200})
		if err != nil {
			return nil, err
		}
		for _, run := range runs {
			targets, err := s.Store.ListActionTargetResults(ctx, run.ID)
			if err != nil {
				return nil, err
			}
			for _, target := range targets {
				if strings.EqualFold(strings.TrimSpace(target.TargetEGMID), strings.TrimSpace(egmID)) {
					matches = append(matches, run)
					break
				}
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].StartedAt.Equal(matches[j].StartedAt) {
			return matches[i].ID > matches[j].ID
		}
		return matches[i].StartedAt.After(matches[j].StartedAt)
	})
	if len(matches) > 0 {
		latest := matches[0]
		uniq := []actions.ActionRun{latest}
		for _, run := range matches[1:] {
			if run.ID != latest.ID {
				uniq = append(uniq, run)
			}
		}
		return uniq, nil
	}
	return matches, nil
}

func (s *Service) selectTargetForRun(ctx context.Context, run actions.ActionRun, egmID string) (*actions.ActionTargetResult, string, error) {
	targets, err := s.Store.ListActionTargetResults(ctx, run.ID)
	if err != nil {
		return nil, "", err
	}
	if len(targets) == 0 {
		return nil, "no target rows for action run", nil
	}
	if egmID == "" {
		if len(targets) == 1 {
			t := targets[0]
			return &t, "", nil
		}
		return nil, "egm id required for multi-target correlation", nil
	}
	for _, target := range targets {
		if strings.EqualFold(strings.TrimSpace(target.TargetEGMID), strings.TrimSpace(egmID)) {
			t := target
			return &t, "", nil
		}
	}
	return nil, "target egm not found in action run", nil
}

type matcherResult struct {
	Outcome       string
	TargetUpdated bool
	TargetStatus  string
	AuditEntryIDs []int64
	Warnings      []string
}

func (s *Service) applyMatcher(ctx context.Context, occurredAt time.Time, messageID int64, resolved resolvedMetadata, correlation correlation, rawPayload string, summaryJSON string) (matcherResult, error) {
	result := matcherResult{
		Outcome: "NO_MATCH",
	}
	run := correlation.Run
	target := correlation.Target
	if run == nil || target == nil {
		return result, nil
	}

	definition, err := s.Store.GetActionDefinition(ctx, run.ActionDefinitionID)
	if err != nil {
		return result, err
	}
	if definition == nil {
		result.Warnings = append(result.Warnings, "action definition not found")
		return result, nil
	}

	egmRecord, err := s.Store.GetEGMRecord(ctx, target.TargetEGMID)
	if err != nil {
		return result, err
	}
	if egmRecord == nil || strings.TrimSpace(egmRecord.TemplateID) == "" {
		result.Warnings = append(result.Warnings, "template assignment not found for target egm")
		return result, nil
	}

	version, err := s.Store.GetActiveG2STemplateVersion(ctx, egmRecord.TemplateID)
	if err != nil {
		return result, err
	}
	if version == nil {
		result.Warnings = append(result.Warnings, "active template version not found")
		return result, nil
	}

	match, err := g2sengine.MatchMessage(
		rawPayload,
		summaryJSON,
		resolved.MessageType,
		version.ConfirmationRulesJSON,
		version.FailureRulesJSON,
	)
	if err != nil {
		result.Warnings = append(result.Warnings, "matcher error: "+err.Error())
		return result, nil
	}
	result.Outcome = match.Outcome

	switch match.Outcome {
	case string(g2sengine.MatchOutcomeExpected):
		now := s.now()
		target.Status = actions.TargetStatusConfirmed
		target.LastError = ""
		target.LastResultAt = &now
		if err := s.Store.UpdateActionTargetResult(ctx, *target); err != nil {
			return result, err
		}
		result.TargetUpdated = true
		result.TargetStatus = string(target.Status)
		if err := s.refreshRunState(ctx, run, occurredAt); err != nil {
			return result, err
		}
		id, err := s.recordAudit(ctx, audit.AuditTimelineEntry{
			OccurredAt:       occurredAt,
			Severity:         audit.AuditSeverityInfo,
			EventType:        audit.EventTypeConfirmation,
			Summary:          "Inbound matcher confirmed target result",
			DetailJSON:       encodeSummaryJSON(map[string]any{"rule_id": match.RuleID, "rule_label": match.RuleLabel, "reason": match.Reason}),
			ActionRunID:      run.ID,
			MessageJournalID: messageID,
		})
		if err != nil {
			return result, err
		}
		result.AuditEntryIDs = append(result.AuditEntryIDs, id)
	case string(g2sengine.MatchOutcomeFailure):
		now := s.now()
		target.Status = actions.TargetStatusFailed
		target.LastError = defaultString(match.Reason, "failure matcher matched")
		target.LastResultAt = &now
		if err := s.Store.UpdateActionTargetResult(ctx, *target); err != nil {
			return result, err
		}
		result.TargetUpdated = true
		result.TargetStatus = string(target.Status)
		if err := s.refreshRunState(ctx, run, occurredAt); err != nil {
			return result, err
		}
		id, err := s.recordAudit(ctx, audit.AuditTimelineEntry{
			OccurredAt:       occurredAt,
			Severity:         audit.AuditSeverityWarning,
			EventType:        audit.EventTypeConfirmation,
			Summary:          "Inbound matcher reported failure",
			DetailJSON:       encodeSummaryJSON(map[string]any{"rule_id": match.RuleID, "rule_label": match.RuleLabel, "reason": match.Reason}),
			ActionRunID:      run.ID,
			MessageJournalID: messageID,
		})
		if err != nil {
			return result, err
		}
		result.AuditEntryIDs = append(result.AuditEntryIDs, id)
	default:
		id, err := s.recordAudit(ctx, audit.AuditTimelineEntry{
			OccurredAt:       occurredAt,
			Severity:         audit.AuditSeverityWarning,
			EventType:        audit.EventTypeSystemWarning,
			Summary:          "Inbound matcher did not match expected or failure rules",
			DetailJSON:       encodeSummaryJSON(map[string]any{"reason": match.Reason}),
			ActionRunID:      run.ID,
			MessageJournalID: messageID,
		})
		if err != nil {
			return result, err
		}
		result.AuditEntryIDs = append(result.AuditEntryIDs, id)
	}
	return result, nil
}

func (s *Service) refreshRunState(ctx context.Context, run *actions.ActionRun, occurredAt time.Time) error {
	targets, err := s.Store.ListActionTargetResults(ctx, run.ID)
	if err != nil {
		return err
	}
	confirmed := 0
	failed := 0
	pending := 0
	for _, target := range targets {
		switch target.Status {
		case actions.TargetStatusConfirmed:
			confirmed++
		case actions.TargetStatusFailed:
			failed++
		default:
			pending++
		}
	}
	run.ConfirmedCount = confirmed
	run.FailedCount = failed
	run.TargetCount = len(targets)
	if failed > 0 {
		run.Status = actions.RunStatusFailed
		value := occurredAt
		run.CompletedAt = &value
	} else if len(targets) > 0 && confirmed == len(targets) {
		run.Status = actions.RunStatusSucceeded
		value := occurredAt
		run.CompletedAt = &value
	} else if pending > 0 && run.Status == actions.RunStatusPending {
		run.Status = actions.RunStatusWaitingConfirmation
	}
	return s.Store.UpdateActionRun(ctx, *run)
}

func (s *Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) recordAudit(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error) {
	return s.Store.RecordAuditTimelineEntry(ctx, entry)
}

func normalizeLookupMap(raw map[string]string) map[string]string {
	result := map[string]string{}
	for key, value := range raw {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result[strings.ToLower(strings.TrimSpace(key))] = trimmed
	}
	return result
}

func findJSONValue(raw map[string]any, key string) string {
	for k, value := range raw {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}

func findXMLAttribute(raw string, name string) string {
	needle := name + `="`
	index := strings.Index(raw, needle)
	if index < 0 {
		return ""
	}
	start := index + len(needle)
	end := strings.Index(raw[start:], `"`)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(raw[start : start+end])
}

func findXMLElement(raw string, name string) string {
	open := "<" + name + ">"
	close := "</" + name + ">"
	index := strings.Index(raw, open)
	if index < 0 {
		return ""
	}
	start := index + len(open)
	end := strings.Index(raw[start:], close)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(raw[start : start+end])
}

func detectMessageTypeFromXML(raw string) string {
	decoder := xml.NewDecoder(strings.NewReader(raw))
	for {
		token, err := decoder.Token()
		if err != nil {
			return ""
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		name := strings.TrimSpace(start.Name.Local)
		if name == "" {
			continue
		}
		lower := strings.ToLower(name)
		if lower == "envelope" || lower == "body" || lower == "g2sbody" || lower == "g2sresponse" {
			continue
		}
		return name
	}
}

func encodeSummaryJSON(payload map[string]any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func defaultString(primary string, fallback string) string {
	if strings.TrimSpace(primary) == "" {
		return fallback
	}
	return primary
}
