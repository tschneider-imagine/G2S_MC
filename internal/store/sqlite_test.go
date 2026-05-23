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

func TestEGMRegistryOverrideCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	records, err := store.ListEGMRegistryOverrides(ctx)
	if err != nil {
		t.Fatalf("list empty overrides: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected empty registry override list, got %d", len(records))
	}

	row := EGMRegistryOverride{
		EGMID:           "EGM-02",
		DisplayName:     "Cabinet 2",
		Vendor:          "ACME",
		CabinetFamily:   "Family B",
		GameTitle:       "Test Game",
		SoftwareVersion: "1.2.3",
		Notes:           "promoted from discovery",
		UpdatedBy:       "tester-a",
	}
	if err := store.UpsertEGMRegistryOverride(ctx, row); err != nil {
		t.Fatalf("upsert override: %v", err)
	}
	assertCount(t, store, "egm_registry_overrides", 1)

	saved, err := store.GetEGMRegistryOverride(ctx, "EGM-02")
	if err != nil {
		t.Fatalf("get override: %v", err)
	}
	if saved == nil {
		t.Fatalf("expected override row")
	}
	if saved.DisplayName != "Cabinet 2" || saved.Notes != "promoted from discovery" {
		t.Fatalf("unexpected override row: %+v", saved)
	}

	row.DisplayName = "Cabinet 2A"
	row.Notes = "updated note"
	row.UpdatedBy = "tester-b"
	if err := store.UpsertEGMRegistryOverride(ctx, row); err != nil {
		t.Fatalf("update override: %v", err)
	}
	records, err = store.ListEGMRegistryOverrides(ctx)
	if err != nil {
		t.Fatalf("list overrides: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 override row after update, got %d", len(records))
	}
	if records[0].DisplayName != "Cabinet 2A" || records[0].UpdatedBy != "tester-b" {
		t.Fatalf("unexpected updated override: %+v", records[0])
	}

	deleted, err := store.DeleteEGMRegistryOverride(ctx, "EGM-02")
	if err != nil {
		t.Fatalf("delete override: %v", err)
	}
	if !deleted {
		t.Fatalf("expected deleted=true for existing row")
	}
	deleted, err = store.DeleteEGMRegistryOverride(ctx, "EGM-02")
	if err != nil {
		t.Fatalf("delete missing override: %v", err)
	}
	if deleted {
		t.Fatalf("expected deleted=false for missing row")
	}
	assertCount(t, store, "egm_registry_overrides", 0)
}

func TestBlockerPolicyOverrideCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	override, err := store.GetBlockerPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get empty blocker policy override: %v", err)
	}
	if override != nil {
		t.Fatalf("expected no blocker policy override row")
	}

	if err := store.UpsertBlockerPolicyOverride(ctx, []string{"service_readiness", "cabinet_profile"}, "tester"); err != nil {
		t.Fatalf("upsert blocker policy override: %v", err)
	}
	assertCount(t, store, "blocker_policy_overrides", 1)

	override, err = store.GetBlockerPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get blocker policy override: %v", err)
	}
	if override == nil {
		t.Fatalf("expected blocker policy override row")
	}
	if len(override.ApprovedBlockerIDs) != 2 {
		t.Fatalf("approved_blocker_ids len = %d, want 2", len(override.ApprovedBlockerIDs))
	}
	if override.UpdatedBy != "tester" {
		t.Fatalf("updated_by = %q, want tester", override.UpdatedBy)
	}
	if override.LastChangeAction != "" || override.LastChangeRationale != "" || override.LastChangeActorScope != "" {
		t.Fatalf("expected empty last-change metadata from legacy upsert, got %+v", override)
	}

	if err := store.UpsertBlockerPolicyOverrideWithMeta(ctx, []string{"service_readiness"}, "tester2", "approve", "required for deployment", "token"); err != nil {
		t.Fatalf("update blocker policy override: %v", err)
	}
	override, err = store.GetBlockerPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get updated blocker policy override: %v", err)
	}
	if len(override.ApprovedBlockerIDs) != 1 || override.ApprovedBlockerIDs[0] != "service_readiness" {
		t.Fatalf("unexpected updated approved_blocker_ids: %+v", override.ApprovedBlockerIDs)
	}
	if override.UpdatedBy != "tester2" {
		t.Fatalf("updated_by = %q, want tester2", override.UpdatedBy)
	}
	if override.LastChangeAction != "approve" {
		t.Fatalf("last_change_action = %q, want approve", override.LastChangeAction)
	}
	if override.LastChangeRationale != "required for deployment" {
		t.Fatalf("last_change_rationale = %q, want required for deployment", override.LastChangeRationale)
	}
	if override.LastChangeActorScope != "token" {
		t.Fatalf("last_change_actor_scope = %q, want token", override.LastChangeActorScope)
	}

	if err := store.ClearBlockerPolicyOverride(ctx); err != nil {
		t.Fatalf("clear blocker policy override: %v", err)
	}
	assertCount(t, store, "blocker_policy_overrides", 0)
	override, err = store.GetBlockerPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get cleared blocker policy override: %v", err)
	}
	if override != nil {
		t.Fatalf("expected cleared blocker policy override to be nil")
	}
}

func TestBlockerPolicyEscalationHistoryCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	_, err = store.RecordBlockerPolicyEscalationEvent(ctx, BlockerPolicyEscalationEvent{
		CreatedAt:  time.Now().UTC().Add(-2 * time.Minute),
		Action:     "approve",
		FindingID:  "cabinet_profile",
		Rationale:  "needed for controlled rollout",
		ActorScope: "token",
		EGMFocus:   "EGM-02",
		UpdatedBy:  "operator-a",
	})
	if err != nil {
		t.Fatalf("record approve escalation event: %v", err)
	}
	_, err = store.RecordBlockerPolicyEscalationEvent(ctx, BlockerPolicyEscalationEvent{
		CreatedAt:  time.Now().UTC().Add(-1 * time.Minute),
		Action:     "revoke",
		FindingID:  "cabinet_profile",
		Rationale:  "no longer needed",
		ActorScope: "trusted",
		EGMFocus:   "",
		UpdatedBy:  "operator-b",
	})
	if err != nil {
		t.Fatalf("record revoke escalation event: %v", err)
	}
	assertCount(t, store, "blocker_policy_escalation_events", 2)

	events, err := store.ListBlockerPolicyEscalationEvents(ctx, 10)
	if err != nil {
		t.Fatalf("list blocker policy escalation events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events len = %d, want 2", len(events))
	}
	if events[0].Action != "revoke" || events[0].FindingID != "cabinet_profile" || events[0].Rationale != "no longer needed" || events[0].ActorScope != "trusted" || events[0].UpdatedBy != "operator-b" {
		t.Fatalf("unexpected first history row: %+v", events[0])
	}
	if events[1].Action != "approve" || events[1].EGMFocus != "EGM-02" {
		t.Fatalf("unexpected second history row: %+v", events[1])
	}
}

func TestReplaceRuntimeOverridesAtomic(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.UpsertCabinetProfileOverride(ctx, config.CabinetProfile{
		WireHostURL:     "https://before.example/g2s",
		ListenerDNSName: "before.example",
		RequiredSANDNS:  []string{"before.example"},
		HostID:          "HOST-BEFORE",
		FirstTestEGMIDs: []string{"EGM-BEFORE"},
	}, "before"); err != nil {
		t.Fatalf("seed cabinet profile override: %v", err)
	}
	if err := store.UpsertHeartbeatPolicyOverride(ctx, 5000, 3, 6, "before"); err != nil {
		t.Fatalf("seed heartbeat policy override: %v", err)
	}
	if err := store.UpsertBlockerPolicyOverrideWithMeta(ctx, []string{"service_readiness"}, "before", "save_override", "seed", "token"); err != nil {
		t.Fatalf("seed blocker policy override: %v", err)
	}
	if err := store.UpsertEGMRegistryOverride(ctx, EGMRegistryOverride{
		EGMID:       "EGM-OLD",
		DisplayName: "Old",
		UpdatedBy:   "before",
	}); err != nil {
		t.Fatalf("seed egm registry override: %v", err)
	}

	input := RuntimeOverridesReplaceInput{
		CabinetProfileOverride: &CabinetProfileOverride{
			Profile: config.CabinetProfile{
				WireHostURL:     "https://after.example/g2s",
				ListenerDNSName: "after.example",
				ListenerIP:      "10.20.30.40",
				RequiredSANDNS:  []string{"after.example"},
				RequiredSANIPs:  []string{"10.20.30.40"},
				HostID:          "HOST-AFTER",
				FirstTestEGMIDs: []string{"EGM-02", "EGM-03"},
			},
			UpdatedBy: "restore-op",
		},
		HeartbeatPolicyOverride: &HeartbeatPolicyOverride{
			IntervalMS:         7000,
			WarningAfterMissed: 4,
			BlockAfterMissed:   8,
			UpdatedBy:          "restore-op",
		},
		BlockerPolicyOverride: &BlockerPolicyOverride{
			ApprovedBlockerIDs:   []string{"service_readiness", "cabinet_profile"},
			UpdatedBy:            "restore-op",
			LastChangeAction:     "restore_snapshot",
			LastChangeRationale:  "restore",
			LastChangeActorScope: "token",
		},
		EGMRegistryOverrides: []EGMRegistryOverride{
			{EGMID: "EGM-02", DisplayName: "Cabinet 2", Vendor: "Acme", UpdatedBy: "restore-op"},
			{EGMID: "EGM-03", DisplayName: "Cabinet 3", Vendor: "Acme", UpdatedBy: "restore-op"},
		},
	}
	if err := store.ReplaceRuntimeOverrides(ctx, input); err != nil {
		t.Fatalf("replace runtime overrides: %v", err)
	}

	profile, err := store.GetCabinetProfileOverride(ctx)
	if err != nil {
		t.Fatalf("get cabinet profile override: %v", err)
	}
	if profile == nil || profile.Profile.HostID != "HOST-AFTER" {
		t.Fatalf("unexpected cabinet profile override: %+v", profile)
	}
	hb, err := store.GetHeartbeatPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get heartbeat policy override: %v", err)
	}
	if hb == nil || hb.IntervalMS != 7000 || hb.WarningAfterMissed != 4 || hb.BlockAfterMissed != 8 {
		t.Fatalf("unexpected heartbeat policy override: %+v", hb)
	}
	blocker, err := store.GetBlockerPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get blocker policy override: %v", err)
	}
	if blocker == nil || len(blocker.ApprovedBlockerIDs) != 2 {
		t.Fatalf("unexpected blocker policy override: %+v", blocker)
	}
	records, err := store.ListEGMRegistryOverrides(ctx)
	if err != nil {
		t.Fatalf("list egm registry overrides: %v", err)
	}
	if len(records) != 2 || records[0].EGMID != "EGM-02" || records[1].EGMID != "EGM-03" {
		t.Fatalf("unexpected egm registry overrides: %+v", records)
	}

	clearInput := RuntimeOverridesReplaceInput{}
	if err := store.ReplaceRuntimeOverrides(ctx, clearInput); err != nil {
		t.Fatalf("replace runtime overrides clear: %v", err)
	}
	profile, err = store.GetCabinetProfileOverride(ctx)
	if err != nil {
		t.Fatalf("get cleared cabinet profile override: %v", err)
	}
	if profile != nil {
		t.Fatalf("expected cleared cabinet profile override")
	}
	hb, err = store.GetHeartbeatPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get cleared heartbeat policy override: %v", err)
	}
	if hb != nil {
		t.Fatalf("expected cleared heartbeat policy override")
	}
	blocker, err = store.GetBlockerPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get cleared blocker policy override: %v", err)
	}
	if blocker != nil {
		t.Fatalf("expected cleared blocker policy override")
	}
	records, err = store.ListEGMRegistryOverrides(ctx)
	if err != nil {
		t.Fatalf("list cleared egm registry overrides: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected cleared egm registry overrides, got %+v", records)
	}
}

func TestReplaceRuntimeOverridesAtomicRejectsDuplicateRegistryIDs(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	if err := store.UpsertHeartbeatPolicyOverride(ctx, 5000, 3, 6, "before"); err != nil {
		t.Fatalf("seed heartbeat policy override: %v", err)
	}

	err = store.ReplaceRuntimeOverrides(ctx, RuntimeOverridesReplaceInput{
		HeartbeatPolicyOverride: &HeartbeatPolicyOverride{
			IntervalMS:         7000,
			WarningAfterMissed: 4,
			BlockAfterMissed:   8,
			UpdatedBy:          "restore-op",
		},
		EGMRegistryOverrides: []EGMRegistryOverride{
			{EGMID: "EGM-02", DisplayName: "A"},
			{EGMID: "EGM-02", DisplayName: "B"},
		},
	})
	if err == nil {
		t.Fatalf("expected duplicate egm_id error")
	}

	hb, err := store.GetHeartbeatPolicyOverride(ctx)
	if err != nil {
		t.Fatalf("get heartbeat policy override after failure: %v", err)
	}
	if hb == nil || hb.IntervalMS != 5000 || hb.WarningAfterMissed != 3 || hb.BlockAfterMissed != 6 {
		t.Fatalf("expected original heartbeat override to remain unchanged, got %+v", hb)
	}
}

func TestRuntimeOverridePresetCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	presets, err := store.ListRuntimeOverridePresets(ctx)
	if err != nil {
		t.Fatalf("list empty presets: %v", err)
	}
	if len(presets) != 0 {
		t.Fatalf("expected no presets initially, got %d", len(presets))
	}

	payload := `{"generated_at":"2026-05-22T00:00:00Z","egm_registry_overrides":[]}`
	if err := store.UpsertRuntimeOverridePreset(ctx, RuntimeOverridePreset{
		Name:        "lab-a",
		Note:        "first preset",
		PayloadJSON: payload,
	}); err != nil {
		t.Fatalf("upsert preset: %v", err)
	}
	assertCount(t, store, "runtime_override_presets", 1)

	row, err := store.GetRuntimeOverridePreset(ctx, "lab-a")
	if err != nil {
		t.Fatalf("get preset: %v", err)
	}
	if row == nil {
		t.Fatalf("expected preset row")
	}
	if row.Note != "first preset" || row.PayloadJSON != payload {
		t.Fatalf("unexpected preset row: %+v", row)
	}
	createdAt := row.CreatedAt

	if err := store.UpsertRuntimeOverridePreset(ctx, RuntimeOverridePreset{
		Name:        "lab-a",
		Note:        "updated preset",
		PayloadJSON: `{"generated_at":"2026-05-22T01:00:00Z","egm_registry_overrides":[{"egm_id":"EGM-02"}]}`,
	}); err != nil {
		t.Fatalf("update preset: %v", err)
	}
	row, err = store.GetRuntimeOverridePreset(ctx, "lab-a")
	if err != nil {
		t.Fatalf("get updated preset: %v", err)
	}
	if row == nil {
		t.Fatalf("expected updated preset row")
	}
	if row.Note != "updated preset" {
		t.Fatalf("note = %q, want updated preset", row.Note)
	}
	if row.CreatedAt.Before(createdAt) {
		t.Fatalf("created_at moved backwards: before=%s after=%s", createdAt, row.CreatedAt)
	}

	if err := store.UpsertRuntimeOverridePreset(ctx, RuntimeOverridePreset{
		Name:        "lab-b",
		Note:        "",
		PayloadJSON: `{"generated_at":"2026-05-22T02:00:00Z","egm_registry_overrides":[]}`,
	}); err != nil {
		t.Fatalf("upsert second preset: %v", err)
	}
	presets, err = store.ListRuntimeOverridePresets(ctx)
	if err != nil {
		t.Fatalf("list presets: %v", err)
	}
	if len(presets) != 2 {
		t.Fatalf("expected 2 presets, got %d", len(presets))
	}

	deleted, err := store.DeleteRuntimeOverridePreset(ctx, "lab-a")
	if err != nil {
		t.Fatalf("delete preset: %v", err)
	}
	if !deleted {
		t.Fatalf("expected delete preset true for existing row")
	}
	deleted, err = store.DeleteRuntimeOverridePreset(ctx, "lab-a")
	if err != nil {
		t.Fatalf("delete missing preset: %v", err)
	}
	if deleted {
		t.Fatalf("expected delete preset false for missing row")
	}
	assertCount(t, store, "runtime_override_presets", 1)
}

func TestSessionWorkflowProgressCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	progress, err := store.GetSessionWorkflowProgress(ctx)
	if err != nil {
		t.Fatalf("get empty session workflow progress: %v", err)
	}
	if progress != nil {
		t.Fatalf("expected no session workflow progress row")
	}

	completed := []string{"pre_check", "connect_observe"}
	if err := store.UpsertSessionWorkflowProgress(ctx, "run_active", completed, "ready for first live test"); err != nil {
		t.Fatalf("upsert session workflow progress: %v", err)
	}
	assertCount(t, store, "session_workflow_progress", 1)

	progress, err = store.GetSessionWorkflowProgress(ctx)
	if err != nil {
		t.Fatalf("get session workflow progress: %v", err)
	}
	if progress == nil {
		t.Fatalf("expected session workflow progress row")
	}
	if progress.CurrentPhase != "run_active" {
		t.Fatalf("current_phase = %q, want run_active", progress.CurrentPhase)
	}
	if len(progress.CompletedSteps) != len(completed) {
		t.Fatalf("completed_steps len = %d, want %d", len(progress.CompletedSteps), len(completed))
	}
	if progress.CompletedSteps[0] != completed[0] || progress.CompletedSteps[1] != completed[1] {
		t.Fatalf("completed_steps = %#v, want %#v", progress.CompletedSteps, completed)
	}
	if progress.OperatorNotes != "ready for first live test" {
		t.Fatalf("operator_notes = %q", progress.OperatorNotes)
	}
	if progress.LastUpdatedAt.IsZero() {
		t.Fatalf("expected last_updated_at to be set")
	}

	if err := store.UpsertSessionWorkflowProgress(ctx, "capture_evidence", []string{"pre_check", "connect_observe", "run_active"}, "updated notes"); err != nil {
		t.Fatalf("update session workflow progress: %v", err)
	}
	progress, err = store.GetSessionWorkflowProgress(ctx)
	if err != nil {
		t.Fatalf("get updated session workflow progress: %v", err)
	}
	if progress.CurrentPhase != "capture_evidence" {
		t.Fatalf("current_phase = %q, want capture_evidence", progress.CurrentPhase)
	}
	if progress.OperatorNotes != "updated notes" {
		t.Fatalf("operator_notes = %q, want updated notes", progress.OperatorNotes)
	}
	if len(progress.CompletedSteps) != 3 {
		t.Fatalf("completed_steps len = %d, want 3", len(progress.CompletedSteps))
	}

	if err := store.ClearSessionWorkflowProgress(ctx); err != nil {
		t.Fatalf("clear session workflow progress: %v", err)
	}
	assertCount(t, store, "session_workflow_progress", 0)
	progress, err = store.GetSessionWorkflowProgress(ctx)
	if err != nil {
		t.Fatalf("get cleared session workflow progress: %v", err)
	}
	if progress != nil {
		t.Fatalf("expected cleared session workflow progress to be nil")
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

	deleted, err := store.DeleteSessionEvidenceByID(ctx, id)
	if err != nil {
		t.Fatalf("delete session evidence by id: %v", err)
	}
	if !deleted {
		t.Fatalf("delete session evidence by id returned deleted=false")
	}
	deleted, err = store.DeleteSessionEvidenceByID(ctx, id)
	if err != nil {
		t.Fatalf("delete missing session evidence by id: %v", err)
	}
	if deleted {
		t.Fatalf("delete missing session evidence by id returned deleted=true")
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

func TestOperatorAuditEventRecordListAndFilters(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	base := time.Now().UTC().Truncate(time.Second)
	rows := []model.OperatorAuditEvent{
		{
			Timestamp:  base.Add(-2 * time.Minute),
			Action:     "cabinet_profile.save",
			Result:     "success",
			ActorScope: "token",
			EGMFocus:   "EGM-02",
			Summary:    "Cabinet profile saved",
			Detail:     "host_id updated",
		},
		{
			Timestamp:  base.Add(-1 * time.Minute),
			Action:     "certificate.import",
			Result:     "fail",
			ActorScope: "trusted",
			Summary:    "Certificate import failed",
			Detail:     "invalid certificate pem",
		},
		{
			Timestamp:  base,
			Action:     "heartbeat_policy.clear",
			Result:     "success",
			ActorScope: "local",
			Summary:    "Heartbeat policy override cleared",
			Detail:     "reverted to file policy",
		},
	}
	for _, row := range rows {
		if _, err := store.RecordOperatorAuditEvent(ctx, row); err != nil {
			t.Fatalf("record operator audit event: %v", err)
		}
	}
	assertCount(t, store, "operator_audit_events", 3)

	listed, err := store.ListOperatorAuditEvents(ctx, model.OperatorAuditQuery{Limit: 2})
	if err != nil {
		t.Fatalf("list operator audit events: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("list len = %d, want 2", len(listed))
	}
	if listed[0].Action != "heartbeat_policy.clear" || listed[1].Action != "certificate.import" {
		t.Fatalf("unexpected list ordering/actions: %+v", listed)
	}

	filteredAction, err := store.ListOperatorAuditEvents(ctx, model.OperatorAuditQuery{Limit: 10, Action: "cabinet_profile.save"})
	if err != nil {
		t.Fatalf("list action filter: %v", err)
	}
	if len(filteredAction) != 1 || filteredAction[0].EGMFocus != "EGM-02" {
		t.Fatalf("unexpected action-filtered rows: %+v", filteredAction)
	}

	filteredResult, err := store.ListOperatorAuditEvents(ctx, model.OperatorAuditQuery{Limit: 10, Result: "fail"})
	if err != nil {
		t.Fatalf("list result filter: %v", err)
	}
	if len(filteredResult) != 1 || filteredResult[0].Action != "certificate.import" {
		t.Fatalf("unexpected result-filtered rows: %+v", filteredResult)
	}

	filteredSearch, err := store.ListOperatorAuditEvents(ctx, model.OperatorAuditQuery{Limit: 10, Search: "invalid certificate"})
	if err != nil {
		t.Fatalf("list search filter: %v", err)
	}
	if len(filteredSearch) != 1 || filteredSearch[0].Result != "fail" {
		t.Fatalf("unexpected search-filtered rows: %+v", filteredSearch)
	}
}

func TestOperatorAuditEventPruning(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	total := operatorAuditRetentionLimit + 7
	for i := 0; i < total; i++ {
		if _, err := store.RecordOperatorAuditEvent(ctx, model.OperatorAuditEvent{
			Timestamp:  time.Now().UTC().Add(time.Duration(i) * time.Second),
			Action:     "heartbeat_policy.save",
			Result:     "success",
			ActorScope: "local",
			Summary:    "row " + time.Now().UTC().Add(time.Duration(i)*time.Second).Format(time.RFC3339Nano),
		}); err != nil {
			t.Fatalf("record event %d: %v", i, err)
		}
	}

	assertCount(t, store, "operator_audit_events", operatorAuditRetentionLimit)
}

func TestEndpointIntegrityAlertStateAckSnoozeAndExpiration(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Second)
	alertID := "eia-test-001"

	if err := store.UpsertEndpointIntegrityAlertAck(ctx, alertID, now, "token", "operator-a"); err != nil {
		t.Fatalf("upsert alert ack: %v", err)
	}
	if err := store.UpsertEndpointIntegrityAlertSnooze(ctx, alertID, now.Add(10*time.Minute), "known maintenance window", "operator-b"); err != nil {
		t.Fatalf("upsert alert snooze: %v", err)
	}

	assertCount(t, store, "endpoint_integrity_alert_states", 1)
	rows, err := store.ListEndpointIntegrityAlertStates(ctx)
	if err != nil {
		t.Fatalf("list alert states: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("list len = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.AlertID != alertID {
		t.Fatalf("alert_id = %q, want %q", row.AlertID, alertID)
	}
	if row.AckedAt == nil {
		t.Fatalf("acked_at is nil, want value")
	}
	if row.AckedByScope != "token" {
		t.Fatalf("acked_by_scope = %q, want token", row.AckedByScope)
	}
	if row.SnoozedUntil == nil {
		t.Fatalf("snoozed_until is nil, want value")
	}
	if row.SnoozeReason != "known maintenance window" {
		t.Fatalf("snooze_reason = %q", row.SnoozeReason)
	}

	cleared, err := store.ClearExpiredEndpointIntegrityAlertSnoozes(ctx, now.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("clear expired snoozes (before expiry): %v", err)
	}
	if cleared != 0 {
		t.Fatalf("cleared rows before expiry = %d, want 0", cleared)
	}

	cleared, err = store.ClearExpiredEndpointIntegrityAlertSnoozes(ctx, now.Add(11*time.Minute))
	if err != nil {
		t.Fatalf("clear expired snoozes: %v", err)
	}
	if cleared != 1 {
		t.Fatalf("cleared rows = %d, want 1", cleared)
	}

	rows, err = store.ListEndpointIntegrityAlertStates(ctx)
	if err != nil {
		t.Fatalf("list alert states after expiry clear: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("list len after clear = %d, want 1", len(rows))
	}
	row = rows[0]
	if row.SnoozedUntil != nil {
		t.Fatalf("snoozed_until = %v, want nil", row.SnoozedUntil)
	}
	if row.SnoozeReason != "" {
		t.Fatalf("snooze_reason = %q, want empty", row.SnoozeReason)
	}
	if row.AckedAt == nil {
		t.Fatalf("acked_at was unexpectedly cleared")
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
