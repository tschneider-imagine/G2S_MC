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

func TestSQLiteStoreListsAuditRows(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	createdAt := time.Now().UTC().Truncate(time.Second)
	incidentID, err := store.RecordIncident(ctx, model.Incident{
		TriggerType:   "SECURITY_LINE_DROP",
		TriggerSource: "test",
		CreatedAt:     createdAt,
		FinalState:    model.StateEmergencyActive,
	})
	if err != nil {
		t.Fatalf("record incident: %v", err)
	}
	if err := store.RecordEGMStatus(ctx, model.EGMStatusSnapshot{
		EGMID:     "EGM-01",
		Status:    model.EGMGreen,
		EventType: "G2S_KEEPALIVE",
		Detail:    "keepalive",
		CreatedAt: createdAt,
	}); err != nil {
		t.Fatalf("record status: %v", err)
	}
	if err := store.RecordEGMComplianceLog(ctx, model.EGMComplianceLog{
		IncidentID:   incidentID,
		EGMID:        "EGM-01",
		IPAddress:    "127.0.0.1",
		ActionSent:   "EGM_RESULT",
		StatusResult: "SUCCESS",
		CreatedAt:    createdAt,
	}); err != nil {
		t.Fatalf("record compliance: %v", err)
	}
	if err := store.RecordStateChange(ctx, model.StateChange{
		OldState:  model.StateHealthy,
		NewState:  model.StateEmergencyActive,
		Reason:    "SECURITY_LINE_DROP",
		CreatedAt: createdAt,
	}); err != nil {
		t.Fatalf("record state change: %v", err)
	}

	incidents, err := store.ListIncidents(ctx, 10)
	if err != nil {
		t.Fatalf("list incidents: %v", err)
	}
	if len(incidents) != 1 || incidents[0].ID != incidentID {
		t.Fatalf("unexpected incidents: %+v", incidents)
	}

	statuses, err := store.ListEGMStatus(ctx, model.HistoryLimits{Limit: 10, EGMID: "EGM-01"})
	if err != nil {
		t.Fatalf("list statuses: %v", err)
	}
	if len(statuses) != 1 || statuses[0].EventType != "G2S_KEEPALIVE" {
		t.Fatalf("unexpected statuses: %+v", statuses)
	}

	logs, err := store.ListEGMComplianceLogs(ctx, 10)
	if err != nil {
		t.Fatalf("list compliance: %v", err)
	}
	if len(logs) != 1 || logs[0].IncidentID != incidentID {
		t.Fatalf("unexpected logs: %+v", logs)
	}

	changes, err := store.ListStateChanges(ctx, 10)
	if err != nil {
		t.Fatalf("list state changes: %v", err)
	}
	if len(changes) != 1 || changes[0].Reason != "SECURITY_LINE_DROP" {
		t.Fatalf("unexpected changes: %+v", changes)
	}
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
