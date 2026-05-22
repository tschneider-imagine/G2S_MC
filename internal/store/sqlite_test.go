package store

import (
	"context"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
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
	if err := store.ReplaceCertificateInventory(ctx, []model.CertificateInventory{{
		Role:          "g2s_client_cert",
		Path:          "./certs/client.crt",
		Status:        "MISSING",
		LastCheckedAt: time.Now(),
	}}); err != nil {
		t.Fatalf("replace cert inventory: %v", err)
	}

	assertCount(t, store, "incident_records", 1)
	assertCount(t, store, "egm_status_snapshots", 1)
	assertCount(t, store, "controller_state_history", 1)
	assertCount(t, store, "certificate_inventory", 1)

	certs, err := store.ListCertificateInventory(ctx)
	if err != nil {
		t.Fatalf("list cert inventory: %v", err)
	}
	if len(certs) != 1 || certs[0].Role != "g2s_client_cert" {
		t.Fatalf("unexpected cert inventory: %+v", certs)
	}
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

func TestCabinetProfileOverrideCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	override, err := store.GetCabinetProfileOverride(ctx)
	if err != nil {
		t.Fatalf("get empty override: %v", err)
	}
	if override != nil {
		t.Fatalf("expected no override row")
	}

	profile := config.CabinetProfile{
		WireHostURL:     "https://cabinet-a.example/g2s",
		ListenerDNSName: "cabinet-a.example",
		ListenerIP:      "10.20.30.40",
		RequiredSANDNS:  []string{"cabinet-a.example"},
		RequiredSANIPs:  []string{"10.20.30.40"},
		HostID:          "HOST-CAB-001",
		FirstTestEGMIDs: []string{"EGM-100", "EGM-101"},
	}
	if err := store.UpsertCabinetProfileOverride(ctx, profile, "tester"); err != nil {
		t.Fatalf("upsert override: %v", err)
	}
	assertCount(t, store, "cabinet_profile_overrides", 1)

	override, err = store.GetCabinetProfileOverride(ctx)
	if err != nil {
		t.Fatalf("get override: %v", err)
	}
	if override == nil {
		t.Fatalf("expected override row")
	}
	if override.Profile.WireHostURL != profile.WireHostURL {
		t.Fatalf("wire_host_url = %q, want %q", override.Profile.WireHostURL, profile.WireHostURL)
	}
	if len(override.Profile.FirstTestEGMIDs) != 2 {
		t.Fatalf("unexpected first_test_egm_ids: %+v", override.Profile.FirstTestEGMIDs)
	}
	if override.UpdatedBy != "tester" {
		t.Fatalf("updated_by = %q, want tester", override.UpdatedBy)
	}

	profile.HostID = "HOST-CAB-002"
	if err := store.UpsertCabinetProfileOverride(ctx, profile, "tester2"); err != nil {
		t.Fatalf("update override: %v", err)
	}
	override, err = store.GetCabinetProfileOverride(ctx)
	if err != nil {
		t.Fatalf("get updated override: %v", err)
	}
	if override.Profile.HostID != "HOST-CAB-002" {
		t.Fatalf("host_id = %q, want HOST-CAB-002", override.Profile.HostID)
	}
	if override.UpdatedBy != "tester2" {
		t.Fatalf("updated_by = %q, want tester2", override.UpdatedBy)
	}

	if err := store.ClearCabinetProfileOverride(ctx); err != nil {
		t.Fatalf("clear override: %v", err)
	}
	assertCount(t, store, "cabinet_profile_overrides", 0)
	override, err = store.GetCabinetProfileOverride(ctx)
	if err != nil {
		t.Fatalf("get cleared override: %v", err)
	}
	if override != nil {
		t.Fatalf("expected cleared override to be nil")
	}
}

func TestHeartbeatPolicyOverrideCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	override, err := store.GetHeartbeatPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get empty heartbeat policy override: %v", err)
	}
	if override != nil {
		t.Fatalf("expected no heartbeat policy override row")
	}

	if err := store.UpsertHeartbeatPolicyOverride(ctx, 6000, 4, 8, "tester"); err != nil {
		t.Fatalf("upsert heartbeat policy override: %v", err)
	}
	assertCount(t, store, "heartbeat_policy_overrides", 1)

	override, err = store.GetHeartbeatPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get heartbeat policy override: %v", err)
	}
	if override == nil {
		t.Fatalf("expected heartbeat policy override row")
	}
	if override.IntervalMS != 6000 || override.WarningAfterMissed != 4 || override.BlockAfterMissed != 8 {
		t.Fatalf("unexpected heartbeat policy override: %+v", override)
	}
	if override.UpdatedBy != "tester" {
		t.Fatalf("updated_by = %q, want tester", override.UpdatedBy)
	}

	if err := store.UpsertHeartbeatPolicyOverride(ctx, 7000, 5, 9, "tester2"); err != nil {
		t.Fatalf("update heartbeat policy override: %v", err)
	}
	override, err = store.GetHeartbeatPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get updated heartbeat policy override: %v", err)
	}
	if override.IntervalMS != 7000 || override.WarningAfterMissed != 5 || override.BlockAfterMissed != 9 {
		t.Fatalf("unexpected updated heartbeat policy override: %+v", override)
	}
	if override.UpdatedBy != "tester2" {
		t.Fatalf("updated_by = %q, want tester2", override.UpdatedBy)
	}

	if err := store.ClearHeartbeatPolicyOverride(ctx); err != nil {
		t.Fatalf("clear heartbeat policy override: %v", err)
	}
	assertCount(t, store, "heartbeat_policy_overrides", 0)
	override, err = store.GetHeartbeatPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get cleared heartbeat policy override: %v", err)
	}
	if override != nil {
		t.Fatalf("expected cleared heartbeat policy override to be nil")
	}
}

func TestSessionEvidenceCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	record := model.SessionEvidenceRecord{
		CreatedAt:      time.Now().UTC().Truncate(time.Second),
		OverallState:   "LAB_READY",
		ReadyzState:    "READY_LAB",
		PreflightState: "PASS",
		HostID:         "HOST-TSPI4-001",
		WireHostURL:    "https://tspi4.local:8444/g2s",
		OperatorNotes:  "first pass looked clean",
		PayloadJSON:    `{"session":{"overall_state":"LAB_READY"}}`,
	}
	id, err := store.RecordSessionEvidence(ctx, record)
	if err != nil {
		t.Fatalf("record session evidence: %v", err)
	}
	if id == 0 {
		t.Fatal("expected session evidence id")
	}
	assertCount(t, store, "session_evidence_records", 1)

	records, err := store.ListSessionEvidence(ctx, 10)
	if err != nil {
		t.Fatalf("list session evidence: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 evidence record, got %d", len(records))
	}
	if records[0].HostID != record.HostID {
		t.Fatalf("host_id = %q, want %q", records[0].HostID, record.HostID)
	}
	if records[0].PayloadJSON != record.PayloadJSON {
		t.Fatalf("payload_json = %q, want %q", records[0].PayloadJSON, record.PayloadJSON)
	}

	if err := store.DeleteSessionEvidence(ctx, id); err != nil {
		t.Fatalf("delete session evidence: %v", err)
	}
	assertCount(t, store, "session_evidence_records", 0)
}

func TestRunMarkerCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	record := model.RunMarker{
		CreatedAt:   time.Now().UTC().Truncate(time.Second),
		MarkerType:  "start",
		Title:       "Cabinet session started",
		Notes:       "first live cabinet attempt",
		HostID:      "HOST-TSPI4-001",
		WireHostURL: "https://tspi4.local:8444/g2s",
		Operator:    "lab-ui",
	}
	id, err := store.RecordRunMarker(ctx, record)
	if err != nil {
		t.Fatalf("record run marker: %v", err)
	}
	if id == 0 {
		t.Fatal("expected run marker id")
	}
	assertCount(t, store, "run_markers", 1)

	records, err := store.ListRunMarkers(ctx, 10)
	if err != nil {
		t.Fatalf("list run markers: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 run marker, got %d", len(records))
	}
	if records[0].Title != record.Title {
		t.Fatalf("title = %q, want %q", records[0].Title, record.Title)
	}
	if records[0].MarkerType != record.MarkerType {
		t.Fatalf("marker_type = %q, want %q", records[0].MarkerType, record.MarkerType)
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
