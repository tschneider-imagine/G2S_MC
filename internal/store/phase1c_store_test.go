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
	assertInputRuntimeLatchColumns(t, store)
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
		LatchActive:          true,
		StableSince:          now,
		LastObservedRawState: inputs.InputStateLow,
		LastObservedAt:       now.Add(1 * time.Second),
		PendingRawState:      inputs.InputStateLow,
		PendingSince:         &pending,
		LastTransitionID:     33,
		LastTransitionAt:     &now,
		LatchClearedAt:       &now,
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
	if !fetched.LatchActive {
		t.Fatalf("expected latch_active true: %+v", fetched)
	}
	if fetched.LastTransitionAt == nil || fetched.LatchClearedAt == nil {
		t.Fatalf("expected latch timestamps: %+v", fetched)
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

func assertInputRuntimeLatchColumns(t *testing.T, store *SQLiteStore) {
	t.Helper()
	rows, err := store.db.Query(`PRAGMA table_info(input_runtime_states)`)
	if err != nil {
		t.Fatalf("table_info(input_runtime_states): %v", err)
	}
	defer rows.Close()

	found := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table_info row: %v", err)
		}
		found[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("table_info rows: %v", err)
	}
	for _, column := range []string{"latch_active", "latch_cleared_at", "last_transition_at"} {
		if !found[column] {
			t.Fatalf("expected input_runtime_states column %q", column)
		}
	}
}
