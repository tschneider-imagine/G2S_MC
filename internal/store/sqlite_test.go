package store

import (
	"context"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/model"
)

func TestSQLiteStoreMigratesAndRecordsAuditRows(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	incidentID, err := store.RecordIncident(ctx, model.Incident{
		TriggerType:   "SECURITY_LINE_DROP",
		TriggerSource: "test",
		CreatedAt:     time.Now(),
		FinalState:    model.StateEmergencyActive,
	})
	if err != nil {
		t.Fatalf("record incident: %v", err)
	}
	if incidentID == 0 {
		t.Fatal("expected incident id")
	}

	if err := store.RecordEGMStatus(ctx, model.EGMStatusSnapshot{
		EGMID:     "EGM-01",
		Status:    model.EGMGreen,
		EventType: "G2S_SESSION_ONLINE",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("record egm status: %v", err)
	}

	if err := store.RecordStateChange(ctx, model.StateChange{
		OldState:  model.StateBooting,
		NewState:  model.StateHealthy,
		Reason:    "test",
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("record state change: %v", err)
	}

	assertCount(t, store, "incident_records", 1)
	assertCount(t, store, "egm_status_snapshots", 1)
	assertCount(t, store, "controller_state_history", 1)
}

func assertCount(t *testing.T, store *SQLiteStore, table string, want int) {
	t.Helper()
	got, err := store.Count(context.Background(), table)
	if err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("count %s = %d, want %d", table, got, want)
	}
}
