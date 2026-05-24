package inputpoller

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

type pollerStore struct {
	channels      map[string]inputs.InputChannel
	channelOrder  []string
	runtimeStates map[string]inputruntime.InputRuntimeState
	transitions   []inputs.InputTransition
	auditEntries  []audit.AuditTimelineEntry
	nextID        int64
}

func newPollerStore(channels []inputs.InputChannel) *pollerStore {
	m := make(map[string]inputs.InputChannel, len(channels))
	order := make([]string, 0, len(channels))
	for _, channel := range channels {
		m[channel.ID] = channel
		order = append(order, channel.ID)
	}
	return &pollerStore{
		channels:      m,
		channelOrder:  order,
		runtimeStates: map[string]inputruntime.InputRuntimeState{},
	}
}

func (s *pollerStore) ListInputChannels(_ context.Context) ([]inputs.InputChannel, error) {
	out := make([]inputs.InputChannel, 0, len(s.channels))
	for _, id := range s.channelOrder {
		if channel, ok := s.channels[id]; ok {
			out = append(out, channel)
		}
	}
	return out, nil
}

func (s *pollerStore) UpsertInputChannel(_ context.Context, channel inputs.InputChannel) error {
	if _, exists := s.channels[channel.ID]; !exists {
		s.channelOrder = append(s.channelOrder, channel.ID)
	}
	s.channels[channel.ID] = channel
	return nil
}

func (s *pollerStore) GetInputChannel(_ context.Context, id string) (*inputs.InputChannel, error) {
	channel, ok := s.channels[id]
	if !ok {
		return nil, nil
	}
	copy := channel
	return &copy, nil
}

func (s *pollerStore) GetInputRuntimeState(_ context.Context, inputID string) (*inputruntime.InputRuntimeState, error) {
	state, ok := s.runtimeStates[inputID]
	if !ok {
		return nil, nil
	}
	copy := state
	return &copy, nil
}

func (s *pollerStore) UpsertInputRuntimeState(_ context.Context, state inputruntime.InputRuntimeState) error {
	s.runtimeStates[state.InputID] = state
	return nil
}

func (s *pollerStore) RecordInputTransition(_ context.Context, transition inputs.InputTransition) (int64, error) {
	s.nextID++
	transition.ID = s.nextID
	s.transitions = append(s.transitions, transition)
	return transition.ID, nil
}

func (s *pollerStore) RecordAuditTimelineEntry(_ context.Context, entry audit.AuditTimelineEntry) (int64, error) {
	entry.ID = int64(len(s.auditEntries) + 1)
	s.auditEntries = append(s.auditEntries, entry)
	return entry.ID, nil
}

type readerOutcome struct {
	state inputs.DigitalState
	err   error
}

type scriptedReader struct {
	sequences map[string][]readerOutcome
	indexes   map[string]int
}

func newScriptedReader(sequences map[string][]readerOutcome) *scriptedReader {
	return &scriptedReader{sequences: sequences, indexes: map[string]int{}}
}

func (r *scriptedReader) Read(_ context.Context, gpioChannel string) (inputs.DigitalState, error) {
	seq, ok := r.sequences[gpioChannel]
	if !ok || len(seq) == 0 {
		return "", fmt.Errorf("no scripted read for %s", gpioChannel)
	}
	idx := r.indexes[gpioChannel]
	if idx >= len(seq) {
		idx = len(seq) - 1
	}
	outcome := seq[idx]
	r.indexes[gpioChannel] = r.indexes[gpioChannel] + 1
	if outcome.err != nil {
		return "", outcome.err
	}
	return outcome.state, nil
}

func buildChannel(id, gpio string, priority int) inputs.InputChannel {
	return inputs.InputChannel{
		ID:                id,
		Name:              id,
		GPIOChannel:       gpio,
		Enabled:           true,
		NormalState:       inputs.InputStateHigh,
		CurrentState:      inputs.InputStateHigh,
		DerivedState:      inputs.DerivedStateNormal,
		DebounceMS:        30,
		Priority:          priority,
		OnTriggerActionID: "",
		OnNormalActionID:  "",
		LatchingMode:      inputs.LatchingAutoClear,
	}
}

func TestPollOnceInitializesRuntimeStateWithoutTransitions(t *testing.T) {
	ctx := context.Background()
	store := newPollerStore([]inputs.InputChannel{
		buildChannel("regular-operation", "GPIO16", 100),
		buildChannel("local-notice", "GPIO26", 200),
	})
	reader := newScriptedReader(map[string][]readerOutcome{
		"GPIO16": {{state: inputs.InputStateHigh}},
		"GPIO26": {{state: inputs.InputStateHigh}},
	})

	now := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
	poller := &Poller{Store: store, Reader: reader, Evaluator: &inputruntime.Evaluator{Store: store}, Clock: func() time.Time { return now }}

	result, err := poller.PollOnce(ctx)
	if err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if len(result.Samples) != 2 {
		t.Fatalf("len(samples) = %d, want 2", len(result.Samples))
	}
	for _, sample := range result.Samples {
		if sample.Error != "" {
			t.Fatalf("sample %s unexpected error: %s", sample.InputID, sample.Error)
		}
		if sample.DerivedState != inputs.DerivedStateNormal {
			t.Fatalf("sample %s derived state = %s, want NORMAL", sample.InputID, sample.DerivedState)
		}
		if sample.Transitioned {
			t.Fatalf("sample %s transitioned unexpectedly", sample.InputID)
		}
	}
	if len(store.transitions) != 0 {
		t.Fatalf("transitions recorded = %d, want 0", len(store.transitions))
	}
	if len(store.runtimeStates) != 2 {
		t.Fatalf("runtime state count = %d, want 2", len(store.runtimeStates))
	}
}

func TestPollOnceRecordsTransitionAfterDebounce(t *testing.T) {
	ctx := context.Background()
	channel := buildChannel("regular-operation", "GPIO16", 100)
	channel.OnTriggerActionID = "action-trigger"
	store := newPollerStore([]inputs.InputChannel{channel})
	reader := newScriptedReader(map[string][]readerOutcome{
		"GPIO16": {
			{state: inputs.InputStateHigh},
			{state: inputs.InputStateLow},
			{state: inputs.InputStateLow},
		},
	})

	now := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
	poller := &Poller{Store: store, Reader: reader, Evaluator: &inputruntime.Evaluator{Store: store}, Clock: func() time.Time { return now }}

	if _, err := poller.PollOnce(ctx); err != nil {
		t.Fatalf("initial poll: %v", err)
	}
	now = now.Add(10 * time.Millisecond)
	if _, err := poller.PollOnce(ctx); err != nil {
		t.Fatalf("pending poll: %v", err)
	}
	now = now.Add(40 * time.Millisecond)
	result, err := poller.PollOnce(ctx)
	if err != nil {
		t.Fatalf("debounced poll: %v", err)
	}

	if len(store.transitions) != 1 {
		t.Fatalf("transitions recorded = %d, want 1", len(store.transitions))
	}
	if len(store.auditEntries) != 1 {
		t.Fatalf("audit entries recorded = %d, want 1", len(store.auditEntries))
	}
	if len(result.Samples) != 1 {
		t.Fatalf("len(samples) = %d, want 1", len(result.Samples))
	}
	sample := result.Samples[0]
	if !sample.Transitioned {
		t.Fatal("expected transitioned sample")
	}
	if sample.TransitionID == 0 {
		t.Fatal("expected non-zero transition id")
	}
	if sample.DerivedState != inputs.DerivedStateTriggered {
		t.Fatalf("derived state = %s, want TRIGGERED", sample.DerivedState)
	}
	if sample.ActionQueuedID != "action-trigger" {
		t.Fatalf("action queued id = %q, want action-trigger", sample.ActionQueuedID)
	}
}

func TestPollOnceContinuesWhenOneReadFails(t *testing.T) {
	ctx := context.Background()
	store := newPollerStore([]inputs.InputChannel{
		buildChannel("regular-operation", "GPIO16", 100),
		buildChannel("local-notice", "GPIO26", 200),
	})
	reader := newScriptedReader(map[string][]readerOutcome{
		"GPIO16": {{err: errors.New("gpio read failed")}},
		"GPIO26": {{state: inputs.InputStateHigh}},
	})
	poller := &Poller{Store: store, Reader: reader, Evaluator: &inputruntime.Evaluator{Store: store}, Clock: time.Now}

	result, err := poller.PollOnce(ctx)
	if err != nil {
		t.Fatalf("poll once: %v", err)
	}
	if len(result.Samples) != 2 {
		t.Fatalf("len(samples) = %d, want 2", len(result.Samples))
	}
	if len(result.Errors) != 1 {
		t.Fatalf("len(errors) = %d, want 1", len(result.Errors))
	}

	byID := map[string]PollSampleResult{}
	for _, sample := range result.Samples {
		byID[sample.InputID] = sample
	}
	if byID["regular-operation"].Error == "" {
		t.Fatal("expected read error on regular-operation")
	}
	if byID["local-notice"].Error != "" {
		t.Fatalf("unexpected local-notice error: %s", byID["local-notice"].Error)
	}
	if byID["local-notice"].DerivedState != inputs.DerivedStateNormal {
		t.Fatalf("local-notice derived state = %s", byID["local-notice"].DerivedState)
	}
}

func TestPollOnceActiveInputReflectsHighestPriorityTriggered(t *testing.T) {
	ctx := context.Background()
	store := newPollerStore([]inputs.InputChannel{
		buildChannel("regular-operation", "GPIO16", 100),
		buildChannel("emergency-broadcast", "GPIO21", 400),
	})
	reader := newScriptedReader(map[string][]readerOutcome{
		"GPIO16": {
			{state: inputs.InputStateHigh},
			{state: inputs.InputStateLow},
			{state: inputs.InputStateLow},
		},
		"GPIO21": {
			{state: inputs.InputStateHigh},
			{state: inputs.InputStateLow},
			{state: inputs.InputStateLow},
		},
	})

	now := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
	poller := &Poller{Store: store, Reader: reader, Evaluator: &inputruntime.Evaluator{Store: store}, Clock: func() time.Time { return now }}

	if _, err := poller.PollOnce(ctx); err != nil {
		t.Fatalf("initial poll: %v", err)
	}
	now = now.Add(10 * time.Millisecond)
	if _, err := poller.PollOnce(ctx); err != nil {
		t.Fatalf("pending poll: %v", err)
	}
	now = now.Add(40 * time.Millisecond)
	result, err := poller.PollOnce(ctx)
	if err != nil {
		t.Fatalf("debounced poll: %v", err)
	}

	if result.Active == nil {
		t.Fatal("expected active input")
	}
	if result.Active.InputID != "emergency-broadcast" {
		t.Fatalf("active input id = %q, want emergency-broadcast", result.Active.InputID)
	}
	if result.Active.Priority != 400 {
		t.Fatalf("active input priority = %d, want 400", result.Active.Priority)
	}
}

func TestPollOnceQueuesActionOnlyInResultNoExecution(t *testing.T) {
	ctx := context.Background()
	channel := buildChannel("general-broadcast", "GPIO20", 300)
	channel.OnTriggerActionID = "broadcast-action"
	store := newPollerStore([]inputs.InputChannel{channel})
	reader := newScriptedReader(map[string][]readerOutcome{
		"GPIO20": {
			{state: inputs.InputStateHigh},
			{state: inputs.InputStateLow},
			{state: inputs.InputStateLow},
		},
	})

	now := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
	poller := &Poller{Store: store, Reader: reader, Evaluator: &inputruntime.Evaluator{Store: store}, Clock: func() time.Time { return now }}

	if _, err := poller.PollOnce(ctx); err != nil {
		t.Fatalf("initial poll: %v", err)
	}
	now = now.Add(10 * time.Millisecond)
	if _, err := poller.PollOnce(ctx); err != nil {
		t.Fatalf("pending poll: %v", err)
	}
	now = now.Add(40 * time.Millisecond)
	result, err := poller.PollOnce(ctx)
	if err != nil {
		t.Fatalf("debounced poll: %v", err)
	}

	if len(result.Samples) != 1 {
		t.Fatalf("len(samples) = %d, want 1", len(result.Samples))
	}
	if result.Samples[0].ActionQueuedID != "broadcast-action" {
		t.Fatalf("action queued id = %q, want broadcast-action", result.Samples[0].ActionQueuedID)
	}

	ids := make([]int64, 0, len(store.transitions))
	for _, tr := range store.transitions {
		ids = append(ids, tr.ID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) != 1 || ids[0] == 0 {
		t.Fatalf("expected one persisted transition id, got %+v", ids)
	}
}
