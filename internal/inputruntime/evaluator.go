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
		initial := InputRuntimeState{
			InputID:              sample.InputID,
			StableRawState:       sample.RawState,
			DerivedState:         derived,
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
	newDerived, err := inputs.DeriveState(sample.RawState, channel.NormalState)
	if err != nil {
		return EvaluationResult{}, err
	}

	working.StableRawState = sample.RawState
	working.StableSince = sample.ObservedAt
	working.DerivedState = newDerived
	working.PendingRawState = ""
	working.PendingSince = nil

	result := EvaluationResult{State: working, Changed: true}
	if !channel.Enabled || previousDerived == newDerived {
		if err := e.Store.UpsertInputRuntimeState(ctx, working); err != nil {
			return EvaluationResult{}, err
		}
		return result, nil
	}

	actionQueuedID := ""
	if previousDerived == inputs.DerivedStateNormal && newDerived == inputs.DerivedStateTriggered {
		actionQueuedID = strings.TrimSpace(channel.OnTriggerActionID)
	}
	if previousDerived == inputs.DerivedStateTriggered && newDerived == inputs.DerivedStateNormal {
		actionQueuedID = strings.TrimSpace(channel.OnNormalActionID)
	}

	transition := inputs.InputTransition{
		InputChannelID:  channel.ID,
		PreviousDerived: previousDerived,
		NewDerived:      newDerived,
		TransitionAt:    sample.ObservedAt,
		Reason:          "debounced input transition",
	}
	transitionID, err := e.Store.RecordInputTransition(ctx, transition)
	if err != nil {
		return EvaluationResult{}, err
	}
	transition.ID = transitionID
	working.LastTransitionID = transitionID

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
