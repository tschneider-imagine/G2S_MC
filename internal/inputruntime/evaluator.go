package inputruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

type Store interface {
	GetInputChannel(ctx context.Context, id string) (*inputs.InputChannel, error)
	GetInputRuntimeState(ctx context.Context, inputID string) (*InputRuntimeState, error)
	UpsertInputRuntimeState(ctx context.Context, state InputRuntimeState) error
	RecordInputTransition(ctx context.Context, transition inputs.InputTransition) (int64, error)
	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)
}

type Evaluator struct {
	Store Store
	Clock func() time.Time
}

func (e *Evaluator) ApplySample(ctx context.Context, sample InputSample) (EvaluationResult, error) {
	if e.Store == nil {
		return EvaluationResult{}, fmt.Errorf("store is required")
	}
	if strings.TrimSpace(sample.InputID) == "" {
		return EvaluationResult{}, fmt.Errorf("input_id is required")
	}
	if sample.ObservedAt.IsZero() {
		return EvaluationResult{}, fmt.Errorf("observed_at is required")
	}
	if sample.RawState != inputs.InputStateHigh && sample.RawState != inputs.InputStateLow {
		return EvaluationResult{}, fmt.Errorf("raw_state must be HIGH or LOW")
	}

	channel, err := e.Store.GetInputChannel(ctx, sample.InputID)
	if err != nil {
		return EvaluationResult{}, err
	}
	if channel == nil {
		return EvaluationResult{}, fmt.Errorf("input channel %q not found", sample.InputID)
	}

	state, err := e.Store.GetInputRuntimeState(ctx, sample.InputID)
	if err != nil {
		return EvaluationResult{}, err
	}
	if state == nil {
		derived, err := inputs.DeriveState(sample.RawState, channel.NormalState)
		if err != nil {
			return EvaluationResult{}, err
		}
		latchActive := channel.LatchingMode == inputs.LatchingManualClear && derived == inputs.DerivedStateTriggered
		initial := InputRuntimeState{
			InputID:              sample.InputID,
			StableRawState:       sample.RawState,
			DerivedState:         derived,
			LatchActive:          latchActive,
			StableSince:          sample.ObservedAt,
			LastObservedRawState: sample.RawState,
			LastObservedAt:       sample.ObservedAt,
			UpdatedAt:            sample.ObservedAt,
		}
		if err := e.Store.UpsertInputRuntimeState(ctx, initial); err != nil {
			return EvaluationResult{}, err
		}
		return EvaluationResult{State: initial, Changed: true}, nil
	}

	working := *state
	working.LastObservedRawState = sample.RawState
	working.LastObservedAt = sample.ObservedAt
	working.UpdatedAt = sample.ObservedAt

	if sample.RawState == working.StableRawState {
		working.PendingRawState = ""
		working.PendingSince = nil
		if err := e.Store.UpsertInputRuntimeState(ctx, working); err != nil {
			return EvaluationResult{}, err
		}
		return EvaluationResult{State: working, Changed: true}, nil
	}

	if working.PendingRawState == "" || working.PendingRawState != sample.RawState || working.PendingSince == nil {
		pendingSince := sample.ObservedAt
		working.PendingRawState = sample.RawState
		working.PendingSince = &pendingSince
		if err := e.Store.UpsertInputRuntimeState(ctx, working); err != nil {
			return EvaluationResult{}, err
		}
		return EvaluationResult{State: working, Changed: true}, nil
	}

	debounceElapsed := sample.ObservedAt.Sub(*working.PendingSince)
	if debounceElapsed < time.Duration(channel.DebounceMS)*time.Millisecond {
		if err := e.Store.UpsertInputRuntimeState(ctx, working); err != nil {
			return EvaluationResult{}, err
		}
		return EvaluationResult{State: working, Changed: true}, nil
	}

	previousDerived := working.DerivedState
	candidateDerived, err := inputs.DeriveState(sample.RawState, channel.NormalState)
	if err != nil {
		return EvaluationResult{}, err
	}

	// MANUAL_CLEAR channels stay derived TRIGGERED until an explicit clear-latch operation.
	if channel.LatchingMode == inputs.LatchingManualClear &&
		previousDerived == inputs.DerivedStateTriggered &&
		working.LatchActive &&
		candidateDerived == inputs.DerivedStateNormal {
		working.StableRawState = sample.RawState
		working.StableSince = sample.ObservedAt
		working.PendingRawState = ""
		working.PendingSince = nil
		if err := e.Store.UpsertInputRuntimeState(ctx, working); err != nil {
			return EvaluationResult{}, err
		}
		return EvaluationResult{State: working, Changed: true}, nil
	}

	cooldownMS := minTransitionIntervalMS(channel)
	if previousDerived != candidateDerived && cooldownMS > 0 && working.LastTransitionAt != nil {
		if sample.ObservedAt.Sub(*working.LastTransitionAt) < time.Duration(cooldownMS)*time.Millisecond {
			if err := e.Store.UpsertInputRuntimeState(ctx, working); err != nil {
				return EvaluationResult{}, err
			}
			return EvaluationResult{
				State:      working,
				Changed:    true,
				Suppressed: true,
			}, nil
		}
	}

	working.StableRawState = sample.RawState
	working.StableSince = sample.ObservedAt
	working.DerivedState = candidateDerived
	working.PendingRawState = ""
	working.PendingSince = nil

	result := EvaluationResult{State: working, Changed: true}
	if !channel.Enabled || previousDerived == candidateDerived {
		if err := e.Store.UpsertInputRuntimeState(ctx, working); err != nil {
			return EvaluationResult{}, err
		}
		return result, nil
	}

	actionQueuedID := ""
	if previousDerived == inputs.DerivedStateNormal && candidateDerived == inputs.DerivedStateTriggered {
		actionQueuedID = strings.TrimSpace(channel.OnTriggerActionID)
	}
	if previousDerived == inputs.DerivedStateTriggered && candidateDerived == inputs.DerivedStateNormal {
		actionQueuedID = strings.TrimSpace(channel.OnNormalActionID)
	}

	if channel.LatchingMode == inputs.LatchingManualClear {
		if candidateDerived == inputs.DerivedStateTriggered {
			working.LatchActive = true
		}
		if candidateDerived == inputs.DerivedStateNormal {
			working.LatchActive = false
			working.LatchClearedAt = &sample.ObservedAt
		}
	}

	transition := inputs.InputTransition{
		InputChannelID:  channel.ID,
		PreviousDerived: previousDerived,
		NewDerived:      candidateDerived,
		TransitionAt:    sample.ObservedAt,
		Reason:          "debounced input transition",
	}
	transitionID, err := e.Store.RecordInputTransition(ctx, transition)
	if err != nil {
		return EvaluationResult{}, err
	}
	transition.ID = transitionID
	working.LastTransitionID = transitionID
	working.LastTransitionAt = &sample.ObservedAt

	auditEntry, err := buildTransitionAuditEntry(channel, transition, sample, actionQueuedID, debounceElapsed)
	if err != nil {
		return EvaluationResult{}, err
	}
	auditID, err := e.Store.RecordAuditTimelineEntry(ctx, auditEntry)
	if err != nil {
		return EvaluationResult{}, err
	}
	auditEntry.ID = auditID

	if err := e.Store.UpsertInputRuntimeState(ctx, working); err != nil {
		return EvaluationResult{}, err
	}

	result.State = working
	result.Transition = &transition
	result.AuditEntry = &auditEntry
	result.ActionQueuedID = actionQueuedID
	return result, nil
}

func (e *Evaluator) ClearLatchedInput(ctx context.Context, inputID string, actor string, reason string) (ClearLatchResult, error) {
	if e.Store == nil {
		return ClearLatchResult{}, fmt.Errorf("store is required")
	}
	id := strings.TrimSpace(inputID)
	if id == "" {
		return ClearLatchResult{}, fmt.Errorf("input_id is required")
	}
	now := e.now().UTC()
	result := ClearLatchResult{
		InputID: id,
	}

	channel, err := e.Store.GetInputChannel(ctx, id)
	if err != nil {
		return result, err
	}
	if channel == nil {
		return result, fmt.Errorf("input channel %q not found", id)
	}

	state, err := e.Store.GetInputRuntimeState(ctx, id)
	if err != nil {
		return result, err
	}
	if state == nil {
		return result, fmt.Errorf("input runtime state %q not found", id)
	}

	attemptEntry := audit.AuditTimelineEntry{
		OccurredAt: now,
		Severity:   audit.AuditSeverityInfo,
		EventType:  audit.EventTypeInputLatchClearAttempted,
		Summary:    fmt.Sprintf("Manual clear requested for input %s (%s)", channel.Name, channel.ID),
		DetailJSON: mustJSON(map[string]any{
			"input_id": id,
			"reason":   strings.TrimSpace(reason),
			"actor":    strings.TrimSpace(actor),
		}),
		Operator: strings.TrimSpace(actor),
	}
	if auditID, err := e.Store.RecordAuditTimelineEntry(ctx, attemptEntry); err == nil {
		attemptEntry.ID = auditID
		result.AuditEntries = append(result.AuditEntries, attemptEntry)
	}

	if channel.LatchingMode != inputs.LatchingManualClear {
		return e.recordClearFailure(ctx, result, now, actor, channel, "input is not MANUAL_CLEAR")
	}
	if !state.LatchActive || state.DerivedState != inputs.DerivedStateTriggered {
		return e.recordClearFailure(ctx, result, now, actor, channel, "input is not currently latched")
	}
	if state.StableRawState != channel.NormalState || state.PendingRawState != "" {
		return e.recordClearFailure(ctx, result, now, actor, channel, "cannot clear while input is physically triggered")
	}

	transition := inputs.InputTransition{
		InputChannelID:  channel.ID,
		PreviousDerived: inputs.DerivedStateTriggered,
		NewDerived:      inputs.DerivedStateNormal,
		TransitionAt:    now,
		Reason:          strings.TrimSpace("manual clear: " + strings.TrimSpace(reason)),
	}
	transitionID, err := e.Store.RecordInputTransition(ctx, transition)
	if err != nil {
		return result, err
	}
	transition.ID = transitionID

	working := *state
	working.DerivedState = inputs.DerivedStateNormal
	working.LatchActive = false
	working.LatchClearedAt = &now
	working.LastTransitionID = transitionID
	working.LastTransitionAt = &now
	working.UpdatedAt = now
	if err := e.Store.UpsertInputRuntimeState(ctx, working); err != nil {
		return result, err
	}

	actionQueuedID := strings.TrimSpace(channel.OnNormalActionID)
	successEntry := audit.AuditTimelineEntry{
		OccurredAt:        now,
		Severity:          audit.AuditSeverityInfo,
		EventType:         audit.EventTypeInputLatchClearSucceeded,
		Summary:           fmt.Sprintf("Manual clear succeeded for input %s (%s)", channel.Name, channel.ID),
		DetailJSON:        mustJSON(map[string]any{"input_id": id, "action_queued_id": actionQueuedID}),
		InputTransitionID: transition.ID,
		Operator:          strings.TrimSpace(actor),
	}
	if auditID, err := e.Store.RecordAuditTimelineEntry(ctx, successEntry); err == nil {
		successEntry.ID = auditID
		result.AuditEntries = append(result.AuditEntries, successEntry)
	}

	result.Cleared = true
	result.Transition = &transition
	result.ActionQueuedID = actionQueuedID
	result.Reason = "manual clear succeeded"
	return result, nil
}

func (e *Evaluator) recordClearFailure(ctx context.Context, result ClearLatchResult, occurredAt time.Time, actor string, channel *inputs.InputChannel, reason string) (ClearLatchResult, error) {
	result.Cleared = false
	result.Reason = reason
	entry := audit.AuditTimelineEntry{
		OccurredAt: occurredAt,
		Severity:   audit.AuditSeverityWarning,
		EventType:  audit.EventTypeInputLatchClearFailed,
		Summary:    fmt.Sprintf("Manual clear failed for input %s (%s): %s", channel.Name, channel.ID, reason),
		DetailJSON: mustJSON(map[string]any{
			"input_id": channel.ID,
			"reason":   reason,
		}),
		Operator: strings.TrimSpace(actor),
	}
	if auditID, err := e.Store.RecordAuditTimelineEntry(ctx, entry); err == nil {
		entry.ID = auditID
		result.AuditEntries = append(result.AuditEntries, entry)
	}
	return result, fmt.Errorf("%s", reason)
}

func (e *Evaluator) now() time.Time {
	if e.Clock != nil {
		return e.Clock()
	}
	return time.Now()
}

func buildTransitionAuditEntry(channel *inputs.InputChannel, transition inputs.InputTransition, sample InputSample, actionQueuedID string, debounceElapsed time.Duration) (audit.AuditTimelineEntry, error) {
	severity := audit.AuditSeverityInfo
	if transition.NewDerived == inputs.DerivedStateTriggered {
		severity = audit.AuditSeverityWarning
	}
	if emergencyLike(channel, actionQueuedID) {
		severity = audit.AuditSeverityEmergency
	}
	metadata := map[string]any{
		"input_id":               channel.ID,
		"input_name":             channel.Name,
		"raw_state":              sample.RawState,
		"previous_derived_state": transition.PreviousDerived,
		"new_derived_state":      transition.NewDerived,
		"action_queued_id":       actionQueuedID,
		"debounce_ms":            channel.DebounceMS,
		"debounce_elapsed_ms":    debounceElapsed.Milliseconds(),
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return audit.AuditTimelineEntry{}, err
	}
	return audit.AuditTimelineEntry{
		OccurredAt:        transition.TransitionAt,
		Severity:          severity,
		EventType:         "INPUT_TRANSITION",
		Summary:           fmt.Sprintf("Input %s (%s) transitioned %s -> %s", channel.Name, channel.ID, transition.PreviousDerived, transition.NewDerived),
		DetailJSON:        string(metadataJSON),
		InputTransitionID: transition.ID,
	}, nil
}

func emergencyLike(channel *inputs.InputChannel, actionQueuedID string) bool {
	if strings.Contains(strings.ToLower(strings.TrimSpace(channel.Name)), "emergency") {
		return true
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(actionQueuedID)), "emergency") {
		return true
	}
	return false
}

func minTransitionIntervalMS(channel *inputs.InputChannel) int {
	if channel == nil {
		return 0
	}
	if channel.LatchingMode == inputs.LatchingManualClear {
		return 1000
	}
	return 0
}

func mustJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
