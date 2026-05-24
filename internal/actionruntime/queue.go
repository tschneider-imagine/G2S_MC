package actionruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actionplanner"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
)

type Queuer struct {
	Store Store
	Clock func() time.Time
}

func (q *Queuer) QueueActionRun(ctx context.Context, request QueueRequest) (QueueResult, error) {
	if q.Store == nil {
		return QueueResult{}, fmt.Errorf("store is required")
	}

	actionID := strings.TrimSpace(request.ActionID)
	if actionID == "" {
		return QueueResult{Queued: false, Reason: "no action id"}, nil
	}

	queuedAt := request.QueuedAt
	if queuedAt.IsZero() {
		clock := q.Clock
		if clock == nil {
			clock = time.Now
		}
		queuedAt = clock().UTC()
	}

	definition, err := q.Store.GetActionDefinition(ctx, actionID)
	if err != nil {
		return QueueResult{}, err
	}
	if definition == nil {
		_, auditErr := q.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
			OccurredAt:        queuedAt,
			Severity:          audit.AuditSeverityWarning,
			EventType:         audit.EventTypeSystemWarning,
			Summary:           fmt.Sprintf("Action queue skipped: action %s not found", actionID),
			InputTransitionID: request.InputTransition.ID,
			Operator:          strings.TrimSpace(request.Actor),
		})
		if auditErr != nil {
			return QueueResult{}, auditErr
		}
		return QueueResult{
			Queued: false,
			Reason: fmt.Sprintf("action %s not found", actionID),
		}, nil
	}

	planner := actionplanner.Planner{Store: q.Store}
	plan, err := planner.BuildPlanForDefinition(ctx, *definition)
	if err != nil {
		return QueueResult{}, err
	}

	run := actions.ActionRun{
		ID:                 newActionRunID(queuedAt),
		ActionDefinitionID: definition.ID,
		InputTransitionID:  request.InputTransition.ID,
		StartedAt:          queuedAt,
		Status:             actions.RunStatusPending,
		TriggerReason:      buildTriggerReason(request.TriggerReason, request.InputTransition.ID),
		TargetCount:        len(plan.Targets),
		ConfirmedCount:     0,
		FailedCount:        0,
		EscalatedCount:     0,
	}
	run, err = q.Store.CreateActionRun(ctx, run)
	if err != nil {
		return QueueResult{}, err
	}

	targetResults := make([]actions.ActionTargetResult, 0, len(plan.Targets))
	targetIDs := make([]string, 0, len(plan.Targets))
	for _, target := range plan.Targets {
		targetIDs = append(targetIDs, target.EGMID)
		row, createErr := q.Store.CreateActionTargetResult(ctx, actions.ActionTargetResult{
			ActionRunID:  run.ID,
			TargetEGMID:  target.EGMID,
			Status:       actions.TargetStatusPending,
			AttemptCount: 0,
		})
		if createErr != nil {
			return QueueResult{}, createErr
		}
		targetResults = append(targetResults, row)
	}

	detail, err := json.Marshal(map[string]any{
		"action_id":      definition.ID,
		"target_count":   len(plan.Targets),
		"target_egm_ids": targetIDs,
		"plan_warnings":  plan.Warnings,
	})
	if err != nil {
		return QueueResult{}, fmt.Errorf("marshal action queue audit metadata: %w", err)
	}

	auditID, err := q.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:        queuedAt,
		Severity:          mapSeverity(definition.Severity),
		EventType:         audit.EventTypeActionQueued,
		Summary:           fmt.Sprintf("Action %s queued from input transition %d", definition.ID, request.InputTransition.ID),
		DetailJSON:        string(detail),
		ActionRunID:       run.ID,
		InputTransitionID: request.InputTransition.ID,
		Operator:          strings.TrimSpace(request.Actor),
	})
	if err != nil {
		return QueueResult{}, err
	}

	return QueueResult{
		Queued:        true,
		ActionRun:     &run,
		TargetResults: targetResults,
		PlanWarnings:  plan.Warnings,
		AuditEntryID:  auditID,
		Reason:        "queued",
	}, nil
}

func mapSeverity(severity actions.ActionSeverity) audit.AuditSeverity {
	switch severity {
	case actions.SeverityEmergency:
		return audit.AuditSeverityEmergency
	case actions.SeverityBroadcast:
		return audit.AuditSeverityWarning
	default:
		return audit.AuditSeverityInfo
	}
}

func buildTriggerReason(raw string, inputTransitionID int64) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" {
		return trimmed
	}
	if inputTransitionID > 0 {
		return fmt.Sprintf("input transition %d", inputTransitionID)
	}
	return "queued"
}

func newActionRunID(now time.Time) string {
	var token [8]byte
	if _, err := rand.Read(token[:]); err != nil {
		return fmt.Sprintf("run-%d", now.UTC().UnixNano())
	}
	return fmt.Sprintf("run-%d-%s", now.UTC().UnixNano(), hex.EncodeToString(token[:]))
}

