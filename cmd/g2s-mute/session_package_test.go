package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestSessionPackageExportHandlerShapeAndNoAuthRequirement(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		ControllerID: "CTRL-SESSION-PKG",
		Database:     config.Database{Path: ":memory:"},
		WebUI: config.WebUI{
			BindAddress:                         "127.0.0.1:8444",
			AllowTrustedPrivateNetworkMutations: false,
		},
		G2S: config.G2S{
			HostURL:      "https://cabinet.local:8444/g2s",
			EndpointPath: "/g2s",
		},
		Timeouts: config.Timeouts{
			EGMHeartbeatIntervalMS: 5000,
		},
		API: config.API{AuthToken: "lab-secret"},
	}

	if err := auditStore.UpsertSessionWorkflowProgress(ctx, "run_active", []string{"pre_check", "connect_observe"}, "workflow from test"); err != nil {
		t.Fatalf("upsert workflow: %v", err)
	}
	if err := auditStore.UpsertHeartbeatPolicyOverride(ctx, 6000, 4, 8, "tester"); err != nil {
		t.Fatalf("upsert heartbeat override: %v", err)
	}
	for i := 0; i < 520; i++ {
		_, err := auditStore.RecordOperatorAuditEvent(ctx, model.OperatorAuditEvent{
			Timestamp:  time.Now().UTC().Add(time.Duration(i) * time.Second),
			Action:     "session_workflow.save",
			Result:     "success",
			ActorScope: "authenticated",
			Summary:    "workflow save",
			Detail:     "event index",
		})
		if err != nil {
			t.Fatalf("record operator audit event: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		_, err := auditStore.RecordSessionEvidence(ctx, model.SessionEvidenceRecord{
			CreatedAt:      time.Now().UTC().Add(-time.Duration(i) * time.Minute),
			OverallState:   "READY_LAB",
			ReadyzState:    "READY_LAB",
			PreflightState: "PASS",
			HostID:         "HOST-TSPI4-001",
			WireHostURL:    "https://cabinet.local:8444/g2s",
			OperatorNotes:  "capture note",
			PayloadJSON:    `{"session":{"overall_state":"READY_LAB"}}`,
		})
		if err != nil {
			t.Fatalf("record session evidence: %v", err)
		}
	}

	eng := engine.NewWithAuditSink(cfg.ControllerID, cfg.EGMRoster, auditStore)
	eng.Start(ctx)

	handler := sessionPackageExportHandler(eng, auditStore, cfg, runtimeInfo{
		ConfigPath:       "configs/config.example.json",
		StartedAt:        time.Now().UTC().Add(-2 * time.Minute),
		SimulatedTrigger: false,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/session-package/export", nil)
	req.RemoteAddr = "198.51.100.20:4555"
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type = %q, want application/json", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatalf("content-disposition = %q, want attachment", rec.Header().Get("Content-Disposition"))
	}

	var payload sessionPackageExportPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.SchemaVersion != sessionPackageSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", payload.SchemaVersion, sessionPackageSchemaVersion)
	}
	if payload.GeneratedAt.IsZero() {
		t.Fatalf("generated_at should be set")
	}
	if payload.Status.Runtime.ConfigPath != "configs/config.example.json" {
		t.Fatalf("status.runtime.config_path = %q", payload.Status.Runtime.ConfigPath)
	}
	if payload.SessionWorkflow.CurrentPhase != "run_active" || !payload.SessionWorkflow.Persisted {
		t.Fatalf("session_workflow = %#v, want persisted run_active", payload.SessionWorkflow)
	}
	if payload.HeartbeatPolicy.PolicySource != "override" {
		t.Fatalf("heartbeat_policy.policy_source = %q, want override", payload.HeartbeatPolicy.PolicySource)
	}
	if len(payload.OperatorAudit) != sessionPackageOperatorAuditN {
		t.Fatalf("operator_audit len = %d, want %d", len(payload.OperatorAudit), sessionPackageOperatorAuditN)
	}
	if payload.SessionEvidenceIndex.CaptureCount != 2 {
		t.Fatalf("session_evidence_index.capture_count = %d, want 2", payload.SessionEvidenceIndex.CaptureCount)
	}
	if len(payload.SessionEvidenceIndex.Captures) != 2 {
		t.Fatalf("session_evidence_index.captures len = %d, want 2", len(payload.SessionEvidenceIndex.Captures))
	}
	if len(payload.SavedCapturesMeta) != 2 {
		t.Fatalf("saved_captures_metadata len = %d, want 2", len(payload.SavedCapturesMeta))
	}
}

func TestSessionPackageExportHandlerMethodNotAllowed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{}
	eng := engine.NewWithAuditSink("CTRL", nil, auditStore)
	eng.Start(ctx)

	handler := sessionPackageExportHandler(eng, auditStore, cfg, runtimeInfo{})
	req := httptest.NewRequest(http.MethodPost, "/api/session-package/export", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
