package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestRuntimeOverridesSnapshotAndRestoreFlow(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		ControllerID: "G2S-MC-RUNTIME-OVERRIDES",
		CabinetProfile: config.CabinetProfile{
			WireHostURL:     "https://file.example/g2s",
			ListenerDNSName: "file.example",
			RequiredSANDNS:  []string{"file.example"},
			HostID:          "HOST-FILE",
			FirstTestEGMIDs: []string{"EGM-01"},
		},
		Timeouts: config.Timeouts{
			EGMHeartbeatIntervalMS:         5000,
			EGMHeartbeatWarningAfterMissed: 3,
			EGMHeartbeatBlockAfterMissed:   6,
		},
		EGMRoster: []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}},
	}
	eng := engine.New(cfg.ControllerID, cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)
	now := time.Now().UTC()
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: now})
	eng.Submit(engine.Event{Type: engine.EventKeepAlive, EGMID: "EGM-02", At: now.Add(time.Second)})
	waitForLastEvent(t, eng, string(engine.EventKeepAlive))

	if err := auditStore.UpsertCabinetProfileOverride(ctx, config.CabinetProfile{
		WireHostURL:     "https://seed.example/g2s",
		ListenerDNSName: "seed.example",
		RequiredSANDNS:  []string{"seed.example"},
		HostID:          "HOST-SEED",
		FirstTestEGMIDs: []string{"EGM-02"},
	}, "seed"); err != nil {
		t.Fatalf("seed cabinet profile override: %v", err)
	}
	if err := auditStore.UpsertHeartbeatPolicyOverride(ctx, 6000, 4, 8, "seed"); err != nil {
		t.Fatalf("seed heartbeat policy override: %v", err)
	}
	if err := auditStore.UpsertBlockerPolicyOverrideWithMeta(ctx, []string{"service_readiness"}, "seed", "save_override", "seed", "token"); err != nil {
		t.Fatalf("seed blocker policy override: %v", err)
	}
	if _, err := auditStore.RecordBlockerPolicyEscalationEvent(ctx, store.BlockerPolicyEscalationEvent{
		CreatedAt:  now,
		Action:     "approve",
		FindingID:  "service_readiness",
		Rationale:  "seed",
		ActorScope: "token",
		UpdatedBy:  "seed",
	}); err != nil {
		t.Fatalf("seed blocker policy escalation event: %v", err)
	}
	if err := auditStore.UpsertEGMRegistryOverride(ctx, store.EGMRegistryOverride{
		EGMID:       "EGM-02",
		DisplayName: "Cabinet 2",
		UpdatedBy:   "seed",
	}); err != nil {
		t.Fatalf("seed egm registry override: %v", err)
	}

	snapshotHandler := runtimeOverridesSnapshotHandler(auditStore, cfg)
	snapshotReq := httptest.NewRequest(http.MethodGet, "/api/runtime-overrides/snapshot", nil)
	snapshotRec := httptest.NewRecorder()
	snapshotHandler(snapshotRec, snapshotReq)
	if snapshotRec.Code != http.StatusOK {
		t.Fatalf("snapshot status = %d: %s", snapshotRec.Code, snapshotRec.Body.String())
	}
	var snapshotBody runtimeOverridesSnapshotResponse
	if err := json.Unmarshal(snapshotRec.Body.Bytes(), &snapshotBody); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshotBody.CabinetProfileOverride == nil || snapshotBody.CabinetProfileOverride.Profile.HostID != "HOST-SEED" {
		t.Fatalf("unexpected cabinet_profile_override in snapshot: %+v", snapshotBody.CabinetProfileOverride)
	}
	if snapshotBody.HeartbeatPolicyOverride == nil || snapshotBody.HeartbeatPolicyOverride.IntervalMS != 6000 {
		t.Fatalf("unexpected heartbeat_policy_override in snapshot: %+v", snapshotBody.HeartbeatPolicyOverride)
	}
	if snapshotBody.BlockerPolicyOverride == nil || len(snapshotBody.BlockerPolicyOverride.ApprovedBlockerIDs) != 1 {
		t.Fatalf("unexpected blocker_policy_override in snapshot: %+v", snapshotBody.BlockerPolicyOverride)
	}
	if len(snapshotBody.BlockerPolicyEscalationHistorySummary) == 0 {
		t.Fatalf("expected blocker policy escalation history summary in snapshot")
	}
	if len(snapshotBody.EGMRegistryOverrides) != 1 || snapshotBody.EGMRegistryOverrides[0].EGMID != "EGM-02" {
		t.Fatalf("unexpected egm_registry_overrides in snapshot: %+v", snapshotBody.EGMRegistryOverrides)
	}

	restoreHandler := runtimeOverridesRestoreHandler(eng, auditStore, cfg)
	restoreReq := httptest.NewRequest(http.MethodPost, "/api/runtime-overrides/restore", bytes.NewBufferString(`{
		"cabinet_profile_override":{
			"wire_host_url":"https://restore.example/g2s",
			"listener_dns_name":"restore.example",
			"required_san_dns":["restore.example"],
			"host_id":"HOST-RESTORE",
			"first_test_egm_ids":["EGM-02","EGM-03"]
		},
		"heartbeat_policy_override":{
			"interval_ms":7000,
			"warning_after_missed":5,
			"block_after_missed":9
		},
		"blocker_policy_override":{
			"approved_blocker_ids":["service_readiness","cabinet_profile"]
		},
		"egm_registry_overrides":[
			{"egm_id":"EGM-02","display_name":"Cabinet 2A","vendor":"Acme"},
			{"egm_id":"EGM-03","display_name":"Cabinet 3","vendor":"Acme"}
		]
	}`))
	restoreReq.Header.Set("Content-Type", "application/json")
	restoreRec := httptest.NewRecorder()
	restoreHandler(restoreRec, restoreReq)
	if restoreRec.Code != http.StatusOK {
		t.Fatalf("restore status = %d: %s", restoreRec.Code, restoreRec.Body.String())
	}
	var restoreBody runtimeOverridesRestoreResponse
	if err := json.Unmarshal(restoreRec.Body.Bytes(), &restoreBody); err != nil {
		t.Fatalf("decode restore response: %v", err)
	}
	if restoreBody.CabinetProfile.Effective.HostID != "HOST-RESTORE" {
		t.Fatalf("cabinet_profile.effective.host_id = %q, want HOST-RESTORE", restoreBody.CabinetProfile.Effective.HostID)
	}
	if restoreBody.HeartbeatPolicy.Effective.IntervalMS != 7000 {
		t.Fatalf("heartbeat_policy.effective.interval_ms = %d, want 7000", restoreBody.HeartbeatPolicy.Effective.IntervalMS)
	}
	if len(restoreBody.BlockerPolicy.Effective.ApprovedBlockerIDs) != 2 {
		t.Fatalf("blocker_policy.effective.approved_blocker_ids len = %d, want 2", len(restoreBody.BlockerPolicy.Effective.ApprovedBlockerIDs))
	}
	if len(restoreBody.EGMRegistry.Overrides) != 2 {
		t.Fatalf("egm_registry.overrides len = %d, want 2", len(restoreBody.EGMRegistry.Overrides))
	}
}

func TestRuntimeOverridesRestoreValidation(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		CabinetProfile: config.CabinetProfile{
			WireHostURL:     "https://file.example/g2s",
			ListenerDNSName: "file.example",
			RequiredSANDNS:  []string{"file.example"},
			HostID:          "HOST-FILE",
			FirstTestEGMIDs: []string{"EGM-01"},
		},
		EGMRoster: []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}},
	}
	eng := engine.New("G2S-MC-RUNTIME-OVERRIDES-VALIDATE", cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now().UTC()})
	waitForLastEvent(t, eng, string(engine.EventBootComplete))

	handler := runtimeOverridesRestoreHandler(eng, auditStore, cfg)

	badHeartbeatReq := httptest.NewRequest(http.MethodPost, "/api/runtime-overrides/restore", bytes.NewBufferString(`{"heartbeat_policy_override":{"interval_ms":0,"warning_after_missed":1,"block_after_missed":1}}`))
	badHeartbeatReq.Header.Set("Content-Type", "application/json")
	badHeartbeatRec := httptest.NewRecorder()
	handler(badHeartbeatRec, badHeartbeatReq)
	if badHeartbeatRec.Code != http.StatusBadRequest {
		t.Fatalf("bad heartbeat restore status = %d, want 400", badHeartbeatRec.Code)
	}

	badBlockerReq := httptest.NewRequest(http.MethodPost, "/api/runtime-overrides/restore", bytes.NewBufferString(`{"blocker_policy_override":{"approved_blocker_ids":["BAD-ID"]}}`))
	badBlockerReq.Header.Set("Content-Type", "application/json")
	badBlockerRec := httptest.NewRecorder()
	handler(badBlockerRec, badBlockerReq)
	if badBlockerRec.Code != http.StatusBadRequest {
		t.Fatalf("bad blocker restore status = %d, want 400", badBlockerRec.Code)
	}

	badRegistryReq := httptest.NewRequest(http.MethodPost, "/api/runtime-overrides/restore", bytes.NewBufferString(`{"egm_registry_overrides":[{"egm_id":""}]}`))
	badRegistryReq.Header.Set("Content-Type", "application/json")
	badRegistryRec := httptest.NewRecorder()
	handler(badRegistryRec, badRegistryReq)
	if badRegistryRec.Code != http.StatusBadRequest {
		t.Fatalf("bad registry restore status = %d, want 400", badRegistryRec.Code)
	}
}

func TestRuntimeOverridesRestoreAuthMatrix(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		CabinetProfile: config.CabinetProfile{
			WireHostURL:     "https://file.example/g2s",
			ListenerDNSName: "file.example",
			RequiredSANDNS:  []string{"file.example"},
			HostID:          "HOST-FILE",
			FirstTestEGMIDs: []string{"EGM-01"},
		},
		EGMRoster: []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}},
	}
	eng := engine.New("G2S-MC-RUNTIME-OVERRIDES-AUTH", cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now().UTC()})
	waitForLastEvent(t, eng, string(engine.EventBootComplete))

	snapshot := runtimeOverridesSnapshotHandler(auditStore, cfg)
	restore := requireMutationAuthForMethods(runtimeOverridesRestoreHandler(eng, auditStore, cfg), cfg, http.MethodPost)

	getReq := httptest.NewRequest(http.MethodGet, "/api/runtime-overrides/snapshot", nil)
	getRec := httptest.NewRecorder()
	snapshot(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("snapshot get status = %d, want 200", getRec.Code)
	}

	unauthRestoreReq := httptest.NewRequest(http.MethodPost, "/api/runtime-overrides/restore", bytes.NewBufferString(`{}`))
	unauthRestoreReq.Header.Set("Content-Type", "application/json")
	unauthRestoreRec := httptest.NewRecorder()
	restore(unauthRestoreRec, unauthRestoreReq)
	if !deniedByAuth(unauthRestoreRec.Code) {
		t.Fatalf("restore without token status = %d, want 401/403", unauthRestoreRec.Code)
	}

	authRestoreReq := httptest.NewRequest(http.MethodPost, "/api/runtime-overrides/restore", bytes.NewBufferString(`{}`))
	authRestoreReq.Header.Set("Content-Type", "application/json")
	authRestoreReq.Header.Set("Authorization", "Bearer lab-secret")
	authRestoreRec := httptest.NewRecorder()
	restore(authRestoreRec, authRestoreReq)
	if authRestoreRec.Code != http.StatusOK {
		t.Fatalf("restore with token status = %d: %s", authRestoreRec.Code, authRestoreRec.Body.String())
	}
}
