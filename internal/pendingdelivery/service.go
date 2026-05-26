package pendingdelivery

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actionruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

const defaultWaitTimeout = 30 * time.Second

type Store interface {
	ListMessageJournalEntries(ctx context.Context, query store.MessageJournalListQuery) ([]g2sengine.MessageJournalEntry, error)
	GetMessageJournalEntry(ctx context.Context, id int64) (*g2sengine.MessageJournalEntry, error)
	UpdateMessageJournalOffer(ctx context.Context, id int64, offeredAt time.Time, result g2sengine.MessageResult) (bool, error)
	UpdateMessageJournalResult(ctx context.Context, id int64, result g2sengine.MessageResult, errText string, responseExcerpt string, httpStatusCode int, latencyMS int, transportMode string, sentAt *time.Time, completedAt *time.Time) error
	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)

	ListActionRuns(ctx context.Context, query store.ActionRunListQuery) ([]actions.ActionRun, error)
	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	UpdateActionRun(ctx context.Context, run actions.ActionRun) error
	ListActionTargetResults(ctx context.Context, actionRunID string) ([]actions.ActionTargetResult, error)
	UpdateActionTargetResult(ctx context.Context, row actions.ActionTargetResult) error
	ListEGMRecords(ctx context.Context) ([]egms.EGMRecord, error)
	GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error)
	GetEGMGroup(ctx context.Context, id string) (*egms.EGMGroup, error)
	ListEGMGroups(ctx context.Context) ([]egms.EGMGroup, error)
	CreateActionRun(ctx context.Context, run actions.ActionRun) (actions.ActionRun, error)
	CreateActionTargetResult(ctx context.Context, result actions.ActionTargetResult) (actions.ActionTargetResult, error)
}

type ContactRequest struct {
	EGMID     string
	ContactAt time.Time
}

type OfferedMessage struct {
	MessageID       int64     `json:"message_id"`
	ActionRunID     string    `json:"action_run_id,omitempty"`
	ActionStepID    string    `json:"action_step_id,omitempty"`
	TemplateID      string    `json:"template_id,omitempty"`
	TemplateVersion string    `json:"template_version,omitempty"`
	MessageType     string    `json:"message_type,omitempty"`
	RawPayload      string    `json:"raw_payload"`
	OfferedAt       time.Time `json:"offered_at"`
	OfferCount      int       `json:"offer_count"`
}

type ContactResult struct {
	Offered  *OfferedMessage `json:"offered,omitempty"`
	Warnings []string        `json:"warnings,omitempty"`
}

type SweepResult struct {
	CheckedRuns        int      `json:"checked_runs"`
	MessagesExpired    int      `json:"messages_expired"`
	MessagesReprepared int      `json:"messages_reprepared"`
	MessagesSuperseded int      `json:"messages_superseded"`
	TargetsFailed      int      `json:"targets_failed"`
	RunsFailed         int      `json:"runs_failed"`
	RunsEscalated      int      `json:"runs_escalated"`
	Warnings           []string `json:"warnings,omitempty"`
}

type Service struct {
	Store Store
	Clock func() time.Time
}

func (s *Service) HandleClientContact(ctx context.Context, req ContactRequest) (ContactResult, error) {
	if s.Store == nil {
		return ContactResult{}, fmt.Errorf("store is required")
	}
	egmID := strings.TrimSpace(req.EGMID)
	if egmID == "" {
		return ContactResult{Warnings: []string{"egm id is required for pending delivery lookup"}}, nil
	}
	offeredAt := req.ContactAt.UTC()
	if offeredAt.IsZero() {
		offeredAt = s.now()
	}

	rows, err := s.Store.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{
		Limit:     200,
		EGMID:     egmID,
		Direction: g2sengine.DirectionOutbound,
		Results: []g2sengine.MessageResult{
			g2sengine.MessageResultPrepared,
			g2sengine.MessageResultPending,
		},
	})
	if err != nil {
		return ContactResult{}, err
	}
	if len(rows) == 0 {
		return ContactResult{}, nil
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Timestamp.Equal(rows[j].Timestamp) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].Timestamp.Before(rows[j].Timestamp)
	})
	candidate := rows[0]
	updated, err := s.Store.UpdateMessageJournalOffer(ctx, candidate.ID, offeredAt, g2sengine.MessageResultOffered)
	if err != nil {
		return ContactResult{}, err
	}
	if !updated {
		return ContactResult{Warnings: []string{"pending message state changed before offer"}}, nil
	}
	row, err := s.Store.GetMessageJournalEntry(ctx, candidate.ID)
	if err != nil {
		return ContactResult{}, err
	}
	if row == nil {
		return ContactResult{Warnings: []string{"offered message row not found after update"}}, nil
	}

	if _, err := s.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:       offeredAt,
		Severity:         audit.AuditSeverityInfo,
		EventType:        audit.EventTypeMessageOffered,
		Summary:          fmt.Sprintf("Prepared message offered to %s", egmID),
		DetailJSON:       encodeDetail(map[string]any{"message_id": row.ID, "offer_count": row.OfferCount, "result": row.Result}),
		ActionRunID:      strings.TrimSpace(row.ActionRunID),
		MessageJournalID: row.ID,
	}); err != nil {
		return ContactResult{}, err
	}

	return ContactResult{
		Offered: &OfferedMessage{
			MessageID:       row.ID,
			ActionRunID:     strings.TrimSpace(row.ActionRunID),
			ActionStepID:    strings.TrimSpace(row.ActionStepID),
			TemplateID:      strings.TrimSpace(row.TemplateID),
			TemplateVersion: strings.TrimSpace(row.TemplateVersion),
			MessageType:     strings.TrimSpace(row.MessageType),
			RawPayload:      row.RawPayload,
			OfferedAt:       offeredAt,
			OfferCount:      row.OfferCount,
		},
	}, nil
}

func (s *Service) SweepWaitingConfirmations(ctx context.Context, now time.Time) (SweepResult, error) {
	if s.Store == nil {
		return SweepResult{}, fmt.Errorf("store is required")
	}
	when := now.UTC()
	if when.IsZero() {
		when = s.now()
	}
	result := SweepResult{}

	runs, err := s.Store.ListActionRuns(ctx, store.ActionRunListQuery{
		Limit:  400,
		Status: actions.RunStatusWaitingConfirmation,
	})
	if err != nil {
		return result, err
	}
	result.CheckedRuns = len(runs)

	for _, run := range runs {
		definition, err := s.Store.GetActionDefinition(ctx, run.ActionDefinitionID)
		if err != nil {
			return result, err
		}
		if definition == nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("action definition %s not found for run %s", run.ActionDefinitionID, run.ID))
			continue
		}
		retry := parseRetryPolicy(definition.RetryPolicyJSON)
		waitTimeout := time.Duration(retry.DelayMS) * time.Millisecond
		if waitTimeout <= 0 {
			waitTimeout = defaultWaitTimeout
		}
		maxAttempts := retry.Count + 1
		if maxAttempts < 1 {
			maxAttempts = 1
		}

		targets, err := s.Store.ListActionTargetResults(ctx, run.ID)
		if err != nil {
			return result, err
		}
		anyTargetFailed := false
		highestAttemptsUsed := 0
		for i := range targets {
			target := targets[i]
			if target.Status != actions.TargetStatusPending {
				continue
			}
			rows, err := s.Store.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{
				Limit:       200,
				ActionRunID: run.ID,
				EGMID:       target.TargetEGMID,
				Direction:   g2sengine.DirectionOutbound,
				Results: []g2sengine.MessageResult{
					g2sengine.MessageResultPrepared,
					g2sengine.MessageResultPending,
					g2sengine.MessageResultOffered,
				},
			})
			if err != nil {
				return result, err
			}
			if len(rows) == 0 {
				continue
			}
			row := rows[0]
			waitSince := row.Timestamp
			if row.OfferedAt != nil {
				waitSince = row.OfferedAt.UTC()
			}
			if waitSince.IsZero() || when.Sub(waitSince) < waitTimeout {
				continue
			}

			attemptsUsed := row.OfferCount
			if attemptsUsed < 1 {
				attemptsUsed = 1
			}
			if attemptsUsed > highestAttemptsUsed {
				highestAttemptsUsed = attemptsUsed
			}
			if attemptsUsed < maxAttempts {
				if err := s.Store.UpdateMessageJournalResult(ctx, row.ID, g2sengine.MessageResultPrepared, "awaiting re-offer", "", 0, 0, row.TransportMode, nil, nil); err != nil {
					return result, err
				}
				result.MessagesReprepared++
				if _, err := s.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
					OccurredAt:       when,
					Severity:         audit.AuditSeverityInfo,
					EventType:        audit.EventTypeRetry,
					Summary:          "Pending delivery returned to PREPARED for re-offer",
					DetailJSON:       encodeDetail(map[string]any{"message_id": row.ID, "egm_id": target.TargetEGMID, "attempts_used": attemptsUsed, "max_attempts": maxAttempts}),
					ActionRunID:      run.ID,
					MessageJournalID: row.ID,
				}); err != nil {
					return result, err
				}
				continue
			}

			if err := s.Store.UpdateMessageJournalResult(ctx, row.ID, g2sengine.MessageResultExpired, "waiting confirmation timeout", "", 0, 0, row.TransportMode, nil, &when); err != nil {
				return result, err
			}
			result.MessagesExpired++
			target.Status = actions.TargetStatusFailed
			target.LastError = "waiting confirmation timeout"
			target.LastResultAt = &when
			if err := s.Store.UpdateActionTargetResult(ctx, target); err != nil {
				return result, err
			}
			result.TargetsFailed++
			anyTargetFailed = true
			if _, err := s.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
				OccurredAt:       when,
				Severity:         audit.AuditSeverityWarning,
				EventType:        audit.EventTypeMessageExpired,
				Summary:          "Pending delivery expired while waiting confirmation",
				DetailJSON:       encodeDetail(map[string]any{"message_id": row.ID, "egm_id": target.TargetEGMID, "attempts_used": attemptsUsed, "max_attempts": maxAttempts}),
				ActionRunID:      run.ID,
				MessageJournalID: row.ID,
			}); err != nil {
				return result, err
			}
		}

		if !anyTargetFailed {
			continue
		}

		updatedTargets, err := s.Store.ListActionTargetResults(ctx, run.ID)
		if err != nil {
			return result, err
		}
		confirmed := 0
		failed := 0
		for _, row := range updatedTargets {
			switch row.Status {
			case actions.TargetStatusConfirmed:
				confirmed++
			case actions.TargetStatusFailed:
				failed++
			}
		}
		run.TargetCount = len(updatedTargets)
		run.ConfirmedCount = confirmed
		run.FailedCount = failed
		run.CompletedAt = &when
		run.Status = actions.RunStatusFailed
		queuedEscalation := false
		if escalation := parseEscalationPolicy(definition.EscalationJSON); strings.TrimSpace(escalation.ActionID) != "" && (escalation.AfterAttempts <= 0 || highestAttemptsUsed >= escalation.AfterAttempts) {
			queuer := actionruntime.Queuer{Store: s.Store, Clock: s.Clock}
			queueResult, queueErr := queuer.QueueActionRun(ctx, actionruntime.QueueRequest{
				InputTransition: inputs.InputTransition{ID: run.InputTransitionID},
				ActionID:        escalation.ActionID,
				IncidentID:      strings.TrimSpace(run.IncidentID),
				TriggerReason:   fmt.Sprintf("escalation from waiting confirmation timeout for run %s", run.ID),
				Actor:           "pending-delivery-sweep",
				QueuedAt:        when,
			})
			if queueErr != nil {
				return result, queueErr
			}
			if queueResult.Queued && queueResult.ActionRun != nil {
				queuedEscalation = true
				run.EscalatedCount++
				run.Status = actions.RunStatusEscalated
				result.RunsEscalated++
				if _, err := s.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
					OccurredAt:  when,
					Severity:    audit.AuditSeverityWarning,
					EventType:   audit.EventTypeEscalation,
					Summary:     fmt.Sprintf("Escalation action %s queued", escalation.ActionID),
					DetailJSON:  encodeDetail(map[string]any{"action_run_id": run.ID, "escalation_action_id": escalation.ActionID, "escalation_run_id": queueResult.ActionRun.ID}),
					ActionRunID: run.ID,
					Operator:    "pending-delivery-sweep",
				}); err != nil {
					return result, err
				}
			}
		}
		if err := s.Store.UpdateActionRun(ctx, run); err != nil {
			return result, err
		}
		if queuedEscalation {
			result.Warnings = append(result.Warnings, fmt.Sprintf("run %s escalated after waiting confirmation timeout", run.ID))
		} else {
			result.RunsFailed++
		}

		_ = s.supersedeRemainingPrepared(ctx, run.ID, when, &result)
	}

	return result, nil
}

func (s *Service) supersedeRemainingPrepared(ctx context.Context, runID string, when time.Time, result *SweepResult) error {
	rows, err := s.Store.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{
		Limit:       400,
		ActionRunID: strings.TrimSpace(runID),
		Direction:   g2sengine.DirectionOutbound,
		Results: []g2sengine.MessageResult{
			g2sengine.MessageResultPrepared,
			g2sengine.MessageResultPending,
			g2sengine.MessageResultOffered,
		},
	})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := s.Store.UpdateMessageJournalResult(ctx, row.ID, g2sengine.MessageResultSuperseded, "run completed before confirmation", "", 0, 0, row.TransportMode, nil, &when); err != nil {
			return err
		}
		result.MessagesSuperseded++
		if _, err := s.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
			OccurredAt:       when,
			Severity:         audit.AuditSeverityInfo,
			EventType:        audit.EventTypeMessageSuperseded,
			Summary:          "Pending delivery superseded",
			DetailJSON:       encodeDetail(map[string]any{"message_id": row.ID}),
			ActionRunID:      strings.TrimSpace(runID),
			MessageJournalID: row.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock().UTC()
	}
	return time.Now().UTC()
}

type retryPolicy struct {
	Count   int `json:"count"`
	DelayMS int `json:"delay_ms"`
}

type escalationPolicy struct {
	ActionID      string `json:"escalation_action_id"`
	AfterAttempts int    `json:"after_attempts"`
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

func encodeDetail(payload map[string]any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
