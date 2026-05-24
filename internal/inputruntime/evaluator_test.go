package inputruntime

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

type testStore struct {
	channel          *inputs.InputChannel
	runtimeState     *InputRuntimeState
	transitions      []inputs.InputTransition
	auditEntries     []audit.AuditTimelineEntry
	transitionAutoID int64
}

func (s *testStore) GetInputChannel(_ context.Context, _ string) (*inputs.InputChannel, error) {
	if s.channel == nil {
		return nil, nil
	}
	copy := *s.channel
	return &copy, nil
}

func (s *testStore) GetInputRuntimeState(_ context.Context, _ string) (*InputRuntimeState, error) {
	if s.runtimeState == nil {
		return nil, nil
	}
	copy := *s.runtimeState
	return &copy, nil
}

func (s *testStore) UpsertInputRuntimeState(_ context.Context, state InputRuntimeState) error {
	copy := state
	s.runtimeState = &copy
	return nil
}

func (s *testStore) RecordInputTransition(_ context.Context, transition inputs.InputTransition) (int64, error) {
	s.transitionAutoID++
	transition.ID = s.transitionAutoID
	s.transitions = append(s.transitions, transition)
	return transition.ID, nil
}

func (s *testStore) RecordAuditTimelineEntry(_ context.Context, entry audit.AuditTimelineEntry) (int64, error) {
	entry.ID = int64(len(s.auditEntries) + 1)
	s.auditEntries = append(s.auditEntries, entry)
	return entry.ID, nil
}

func TestApplySampleFirstObservationInitializesStateNoTransition(t *testing.T) {
	now := time.Now().UTC()
	store := &testStore{channel: &inputs.InputChannel{
		ID:                "in-1",
		Name:              "Regular",
		Enabled:           true,
		NormalState:       inputs.InputStateHigh,
		CurrentState:      inputs.InputStateHigh,
		DerivedState:      inputs.DerivedStateNormal,
		DebounceMS:        30,
		Priority:          1,
		OnTriggerActionID: "action-a",
		LatchingMode:      inputs.LatchingAutoClear,
		GPIOChannel:       "GPIO1",
	}}
	evaluator := &Evaluator{Store: store}

	result, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now})
	if err != nil {
		t.Fatalf("apply sample: %v", err)
	}
	if result.Transition != nil {
		t.Fatal("expected no transition on first observation")
	}
	if result.State.DerivedState != inputs.DerivedStateNormal {
		t.Fatalf("derived state = %s", result.State.DerivedState)
	}
}

func TestApplySampleNormalHighAndRawHighDerivesNormal(t *testing.T) {
	now := time.Now().UTC()
	store := seededStore(now, inputs.InputStateHigh, inputs.DerivedStateNormal)
	evaluator := &Evaluator{Store: store}

	result, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(1 * time.Second)})
	if err != nil {
		t.Fatalf("apply sample: %v", err)
	}
	if result.State.DerivedState != inputs.DerivedStateNormal {
		t.Fatalf("derived state = %s", result.State.DerivedState)
	}
}

func TestApplySampleNormalHighAndRawLowDebouncedToTriggered(t *testing.T) {
	now := time.Now().UTC()
	store := seededStore(now, inputs.InputStateHigh, inputs.DerivedStateNormal)
	evaluator := &Evaluator{Store: store}

	_, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateLow, ObservedAt: now.Add(10 * time.Millisecond)})
	if err != nil {
		t.Fatalf("first low sample: %v", err)
	}
	if len(store.transitions) != 0 {
		t.Fatal("expected debounce to prevent transition")
	}

	result, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateLow, ObservedAt: now.Add(40 * time.Millisecond)})
	if err != nil {
		t.Fatalf("second low sample: %v", err)
	}
	if result.Transition == nil {
		t.Fatal("expected transition after debounce elapsed")
	}
	if result.State.DerivedState != inputs.DerivedStateTriggered {
		t.Fatalf("derived state = %s", result.State.DerivedState)
	}
	if result.ActionQueuedID != "action-trigger" {
		t.Fatalf("action queued = %q", result.ActionQueuedID)
	}
}

func TestApplySampleReturnToNormalQueuesOnNormalAction(t *testing.T) {
	now := time.Now().UTC()
	store := seededStore(now, inputs.InputStateLow, inputs.DerivedStateTriggered)
	store.channel.OnNormalActionID = "action-normal"
	evaluator := &Evaluator{Store: store}

	_, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(10 * time.Millisecond)})
	if err != nil {
		t.Fatalf("first high sample: %v", err)
	}
	result, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(40 * time.Millisecond)})
	if err != nil {
		t.Fatalf("second high sample: %v", err)
	}
	if result.Transition == nil {
		t.Fatal("expected transition")
	}
	if result.ActionQueuedID != "action-normal" {
		t.Fatalf("action queued = %q", result.ActionQueuedID)
	}
	if result.State.DerivedState != inputs.DerivedStateNormal {
		t.Fatalf("derived state = %s", result.State.DerivedState)
	}
}

func TestApplySampleManualClearStaysTriggeredWhenRawReturnsNormal(t *testing.T) {
	now := time.Now().UTC()
	store := seededStore(now, inputs.InputStateLow, inputs.DerivedStateTriggered)
	store.channel.LatchingMode = inputs.LatchingManualClear
	store.runtimeState.LatchActive = true
	store.channel.OnNormalActionID = "action-normal"
	evaluator := &Evaluator{Store: store}

	_, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(10 * time.Millisecond)})
	if err != nil {
		t.Fatalf("first high sample: %v", err)
	}
	result, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(40 * time.Millisecond)})
	if err != nil {
		t.Fatalf("second high sample: %v", err)
	}
	if result.State.DerivedState != inputs.DerivedStateTriggered {
		t.Fatalf("derived state=%s want TRIGGERED", result.State.DerivedState)
	}
	if result.ActionQueuedID != "" {
		t.Fatalf("action queued=%q want empty", result.ActionQueuedID)
	}
	if result.Transition != nil {
		t.Fatalf("expected no transition while manual-clear latched: %+v", result.Transition)
	}
}

func TestApplySampleDisabledChannelDoesNotCreateTransition(t *testing.T) {
	now := time.Now().UTC()
	store := seededStore(now, inputs.InputStateHigh, inputs.DerivedStateNormal)
	store.channel.Enabled = false
	evaluator := &Evaluator{Store: store}

	_, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateLow, ObservedAt: now.Add(10 * time.Millisecond)})
	if err != nil {
		t.Fatalf("first low sample: %v", err)
	}
	result, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateLow, ObservedAt: now.Add(40 * time.Millisecond)})
	if err != nil {
		t.Fatalf("second low sample: %v", err)
	}
	if result.Transition != nil {
		t.Fatal("expected no transition for disabled channel")
	}
}

func TestApplySampleTransitionRecordsAuditEntry(t *testing.T) {
	now := time.Now().UTC()
	store := seededStore(now, inputs.InputStateHigh, inputs.DerivedStateNormal)
	store.channel.Name = "Emergency Broadcast"
	evaluator := &Evaluator{Store: store}

	_, _ = evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateLow, ObservedAt: now.Add(10 * time.Millisecond)})
	result, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateLow, ObservedAt: now.Add(40 * time.Millisecond)})
	if err != nil {
		t.Fatalf("apply sample: %v", err)
	}
	if len(store.auditEntries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(store.auditEntries))
	}
	entry := store.auditEntries[0]
	if entry.EventType != "INPUT_TRANSITION" {
		t.Fatalf("event_type = %q", entry.EventType)
	}
	if entry.Severity != audit.AuditSeverityEmergency {
		t.Fatalf("severity = %s", entry.Severity)
	}
	if !strings.Contains(entry.DetailJSON, "action-trigger") {
		t.Fatalf("detail_json missing action id: %s", entry.DetailJSON)
	}
	if result.Transition == nil || result.Transition.ID == 0 {
		t.Fatal("expected transition id")
	}
}

func TestApplySampleInvalidRawStateReturnsError(t *testing.T) {
	store := seededStore(time.Now().UTC(), inputs.InputStateHigh, inputs.DerivedStateNormal)
	evaluator := &Evaluator{Store: store}
	_, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: "BAD", ObservedAt: time.Now().UTC()})
	if err == nil {
		t.Fatal("expected invalid raw state error")
	}
}

func TestClearLatchedInputFailsWhenStillPhysicallyTriggered(t *testing.T) {
	now := time.Now().UTC()
	store := seededStore(now, inputs.InputStateLow, inputs.DerivedStateTriggered)
	store.channel.LatchingMode = inputs.LatchingManualClear
	store.runtimeState.LatchActive = true
	evaluator := &Evaluator{Store: store, Clock: func() time.Time { return now }}

	_, err := evaluator.ClearLatchedInput(context.Background(), "in-1", "operator-a", "clear attempt")
	if err == nil {
		t.Fatal("expected clear failure while physically triggered")
	}
}

func TestClearLatchedInputSucceedsAfterRawReturnsNormalAndQueuesOnNormal(t *testing.T) {
	now := time.Now().UTC()
	store := seededStore(now, inputs.InputStateLow, inputs.DerivedStateTriggered)
	store.channel.LatchingMode = inputs.LatchingManualClear
	store.runtimeState.LatchActive = true
	store.channel.OnNormalActionID = "action-normal"
	evaluator := &Evaluator{Store: store}

	_, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(10 * time.Millisecond)})
	if err != nil {
		t.Fatalf("first high sample: %v", err)
	}
	_, err = evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(40 * time.Millisecond)})
	if err != nil {
		t.Fatalf("second high sample: %v", err)
	}
	result, err := evaluator.ClearLatchedInput(context.Background(), "in-1", "operator-a", "manual clear")
	if err != nil {
		t.Fatalf("clear latched input: %v", err)
	}
	if !result.Cleared {
		t.Fatalf("expected cleared result: %+v", result)
	}
	if result.ActionQueuedID != "action-normal" {
		t.Fatalf("action queued=%q want action-normal", result.ActionQueuedID)
	}
	if result.Transition == nil || result.Transition.NewDerived != inputs.DerivedStateNormal {
		t.Fatalf("expected transition to NORMAL: %+v", result.Transition)
	}
}

func TestAutoClearInputReturnsToNormalAutomatically(t *testing.T) {
	now := time.Now().UTC()
	store := seededStore(now, inputs.InputStateLow, inputs.DerivedStateTriggered)
	store.channel.LatchingMode = inputs.LatchingAutoClear
	evaluator := &Evaluator{Store: store}

	_, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(10 * time.Millisecond)})
	if err != nil {
		t.Fatalf("first high sample: %v", err)
	}
	result, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(40 * time.Millisecond)})
	if err != nil {
		t.Fatalf("second high sample: %v", err)
	}
	if result.State.DerivedState != inputs.DerivedStateNormal {
		t.Fatalf("derived state=%s want NORMAL", result.State.DerivedState)
	}
}

func TestCooldownSuppressesRapidTransitionsAfterManualClear(t *testing.T) {
	now := time.Now().UTC()
	store := seededStore(now, inputs.InputStateHigh, inputs.DerivedStateNormal)
	store.channel.LatchingMode = inputs.LatchingManualClear
	evaluator := &Evaluator{Store: store, Clock: func() time.Time { return now.Add(70 * time.Millisecond) }}

	_, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateLow, ObservedAt: now.Add(10 * time.Millisecond)})
	if err != nil {
		t.Fatalf("first low sample: %v", err)
	}
	firstTransition, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateLow, ObservedAt: now.Add(40 * time.Millisecond)})
	if err != nil {
		t.Fatalf("second low sample: %v", err)
	}
	if firstTransition.Transition == nil {
		t.Fatal("expected trigger transition")
	}
	_, err = evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(50 * time.Millisecond)})
	if err != nil {
		t.Fatalf("first high sample: %v", err)
	}
	_, err = evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateHigh, ObservedAt: now.Add(80 * time.Millisecond)})
	if err != nil {
		t.Fatalf("second high sample: %v", err)
	}
	if _, err := evaluator.ClearLatchedInput(context.Background(), "in-1", "operator-a", "clear"); err != nil {
		t.Fatalf("clear latched input: %v", err)
	}
	_, err = evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateLow, ObservedAt: now.Add(90 * time.Millisecond)})
	if err != nil {
		t.Fatalf("third low sample: %v", err)
	}
	suppressed, err := evaluator.ApplySample(context.Background(), InputSample{InputID: "in-1", RawState: inputs.InputStateLow, ObservedAt: now.Add(130 * time.Millisecond)})
	if err != nil {
		t.Fatalf("fourth low sample: %v", err)
	}
	if !suppressed.Suppressed {
		t.Fatalf("expected cooldown suppression: %+v", suppressed)
	}
}

func seededStore(base time.Time, stableRaw inputs.DigitalState, derived inputs.DerivedInputState) *testStore {
	pending := base
	_ = pending
	return &testStore{
		channel: &inputs.InputChannel{
			ID:                "in-1",
			Name:              "General Broadcast",
			Enabled:           true,
			NormalState:       inputs.InputStateHigh,
			CurrentState:      stableRaw,
			DerivedState:      derived,
			DebounceMS:        30,
			Priority:          2,
			OnTriggerActionID: "action-trigger",
			OnNormalActionID:  "action-normal",
			LatchingMode:      inputs.LatchingAutoClear,
			GPIOChannel:       "GPIO1",
		},
		runtimeState: &InputRuntimeState{
			InputID:              "in-1",
			StableRawState:       stableRaw,
			DerivedState:         derived,
			StableSince:          base,
			LastObservedRawState: stableRaw,
			LastObservedAt:       base,
			UpdatedAt:            base,
		},
	}
}
