package inputpoller

import (
	"context"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

type defaultsStore struct {
	channels map[string]inputs.InputChannel
}

func newDefaultsStore() *defaultsStore {
	return &defaultsStore{channels: map[string]inputs.InputChannel{}}
}

func (s *defaultsStore) ListInputChannels(context.Context) ([]inputs.InputChannel, error) {
	out := make([]inputs.InputChannel, 0, len(s.channels))
	for _, c := range s.channels {
		out = append(out, c)
	}
	return out, nil
}

func (s *defaultsStore) UpsertInputChannel(_ context.Context, channel inputs.InputChannel) error {
	s.channels[channel.ID] = channel
	return nil
}

func (s *defaultsStore) GetInputChannel(_ context.Context, id string) (*inputs.InputChannel, error) {
	channel, ok := s.channels[id]
	if !ok {
		return nil, nil
	}
	copy := channel
	return &copy, nil
}

func (s *defaultsStore) GetInputRuntimeState(context.Context, string) (*inputruntime.InputRuntimeState, error) {
	return nil, nil
}

func (s *defaultsStore) UpsertInputRuntimeState(context.Context, inputruntime.InputRuntimeState) error {
	return nil
}

func (s *defaultsStore) RecordInputTransition(context.Context, inputs.InputTransition) (int64, error) {
	return 0, nil
}

func (s *defaultsStore) RecordAuditTimelineEntry(context.Context, audit.AuditTimelineEntry) (int64, error) {
	return 0, nil
}

func TestDefaultPi4InputChannelsExpectedShape(t *testing.T) {
	channels := DefaultPi4InputChannels()
	if len(channels) != 4 {
		t.Fatalf("len(channels) = %d, want 4", len(channels))
	}

	tests := []struct {
		id       string
		gpio     string
		priority int
		latch    inputs.InputLatchingMode
	}{
		{id: "regular-operation", gpio: "GPIO16", priority: 100, latch: inputs.LatchingAutoClear},
		{id: "general-broadcast", gpio: "GPIO20", priority: 300, latch: inputs.LatchingAutoClear},
		{id: "emergency-broadcast", gpio: "GPIO21", priority: 400, latch: inputs.LatchingManualClear},
		{id: "local-notice", gpio: "GPIO26", priority: 200, latch: inputs.LatchingAutoClear},
	}

	for i, tc := range tests {
		c := channels[i]
		if c.ID != tc.id {
			t.Fatalf("channel[%d].ID = %q, want %q", i, c.ID, tc.id)
		}
		if c.GPIOChannel != tc.gpio {
			t.Fatalf("channel[%d].GPIOChannel = %q, want %q", i, c.GPIOChannel, tc.gpio)
		}
		if c.Priority != tc.priority {
			t.Fatalf("channel[%d].Priority = %d, want %d", i, c.Priority, tc.priority)
		}
		if c.LatchingMode != tc.latch {
			t.Fatalf("channel[%d].LatchingMode = %q, want %q", i, c.LatchingMode, tc.latch)
		}
		if c.NormalState != inputs.InputStateHigh {
			t.Fatalf("channel[%d].NormalState = %q, want HIGH", i, c.NormalState)
		}
		if c.CurrentState != inputs.InputStateHigh {
			t.Fatalf("channel[%d].CurrentState = %q, want HIGH", i, c.CurrentState)
		}
		if c.DerivedState != inputs.DerivedStateNormal {
			t.Fatalf("channel[%d].DerivedState = %q, want NORMAL", i, c.DerivedState)
		}
		if c.CreatedAt != (time.Time{}) || c.UpdatedAt != (time.Time{}) {
			t.Fatalf("channel[%d] expected zero created_at/updated_at", i)
		}
	}
}

func TestEnsureDefaultPi4InputChannelsNoOverwrite(t *testing.T) {
	ctx := context.Background()
	store := newDefaultsStore()
	store.channels["regular-operation"] = inputs.InputChannel{
		ID:           "regular-operation",
		Name:         "Custom Regular",
		GPIOChannel:  "GPIO99",
		Enabled:      true,
		NormalState:  inputs.InputStateLow,
		CurrentState: inputs.InputStateLow,
		DerivedState: inputs.DerivedStateNormal,
		DebounceMS:   99,
		Priority:     1,
		LatchingMode: inputs.LatchingAutoClear,
	}

	if err := EnsureDefaultPi4InputChannels(ctx, store, false); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}

	regular := store.channels["regular-operation"]
	if regular.GPIOChannel != "GPIO99" {
		t.Fatalf("expected existing channel to remain unchanged, got GPIOChannel=%q", regular.GPIOChannel)
	}
	if len(store.channels) != 4 {
		t.Fatalf("len(store.channels) = %d, want 4", len(store.channels))
	}
}

func TestEnsureDefaultPi4InputChannelsOverwriteTrue(t *testing.T) {
	ctx := context.Background()
	store := newDefaultsStore()
	store.channels["regular-operation"] = inputs.InputChannel{
		ID:           "regular-operation",
		Name:         "Custom Regular",
		GPIOChannel:  "GPIO99",
		Enabled:      true,
		NormalState:  inputs.InputStateLow,
		CurrentState: inputs.InputStateLow,
		DerivedState: inputs.DerivedStateNormal,
		DebounceMS:   99,
		Priority:     1,
		LatchingMode: inputs.LatchingAutoClear,
	}

	if err := EnsureDefaultPi4InputChannels(ctx, store, true); err != nil {
		t.Fatalf("ensure defaults overwrite=true: %v", err)
	}

	regular := store.channels["regular-operation"]
	if regular.GPIOChannel != "GPIO16" {
		t.Fatalf("overwrite should restore default channel GPIO16, got %q", regular.GPIOChannel)
	}
	if regular.Priority != 100 {
		t.Fatalf("overwrite should restore default priority 100, got %d", regular.Priority)
	}
}
