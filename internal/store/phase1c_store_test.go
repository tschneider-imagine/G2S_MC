package store

import (
	"context"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

func TestPhase1CMigrationIdempotent(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	for i := 0; i < 3; i++ {
		if err := store.Migrate(ctx); err != nil {
			t.Fatalf("migrate pass %d: %v", i+1, err)
		}
	}
	assertCount(t, store, "input_runtime_states", 0)
}

func TestInputRuntimeStateUpsertGet(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Second)
	pending := now.Add(5 * time.Millisecond)
	state := inputruntime.InputRuntimeState{
		InputID:              "input-1",
		StableRawState:       inputs.InputStateHigh,
		DerivedState:         inputs.DerivedStateNormal,
		StableSince:          now,
		LastObservedRawState: inputs.InputStateLow,
		LastObservedAt:       now.Add(1 * time.Second),
		PendingRawState:      inputs.InputStateLow,
		PendingSince:         &pending,
		LastTransitionID:     33,
		UpdatedAt:            now.Add(2 * time.Second),
	}
	if err := store.UpsertInputRuntimeState(ctx, state); err != nil {
		t.Fatalf("upsert input runtime state: %v", err)
	}
	fetched, err := store.GetInputRuntimeState(ctx, "input-1")
	if err != nil {
		t.Fatalf("get input runtime state: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected runtime state")
	}
	if fetched.StableRawState != inputs.InputStateHigh || fetched.PendingRawState != inputs.InputStateLow {
		t.Fatalf("unexpected fetched state: %+v", fetched)
	}
}

func TestRecordListInputTransitions(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	id, err := store.RecordInputTransition(ctx, inputs.InputTransition{
		InputChannelID:  "input-1",
		PreviousDerived: inputs.DerivedStateNormal,
		NewDerived:      inputs.DerivedStateTriggered,
		TransitionAt:    time.Now().UTC(),
		Reason:          "test",
	})
	if err != nil {
		t.Fatalf("record transition: %v", err)
	}
	if id == 0 {
		t.Fatal("expected transition id")
	}
	transitions, err := store.ListInputTransitions(ctx, 10)
	if err != nil {
		t.Fatalf("list transitions: %v", err)
	}
	if len(transitions) != 1 || transitions[0].ID != id {
		t.Fatalf("unexpected transitions: %+v", transitions)
	}
}
