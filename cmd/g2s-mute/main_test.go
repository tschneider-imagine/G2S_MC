package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestStatusHandlerIncludesRuntimeReadiness(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	certificates := []model.CertificateInventory{
		{Role: "g2s_ca_cert", Path: "/tmp/ca.crt", Status: "MISSING", LastCheckedAt: time.Now()},
		{Role: "web_server_cert", Status: "NOT_CONFIGURED", LastCheckedAt: time.Now()},
	}
	if err := auditStore.ReplaceCertificateInventory(ctx, certificates); err != nil {
		t.Fatalf("replace certificate inventory: %v", err)
	}

	cfg := config.Config{
		ControllerID: "G2S-MC-TEST",
		Database:     config.Database{Path: "/var/lib/g2s-mute/controller.db"},
		WebUI:        config.WebUI{BindAddress: "127.0.0.1:8444"},
		G2S: config.G2S{
			HostURL:      "http://127.0.0.1:8444/g2s",
			EndpointPath: "/g2s",
		},
		EGMRoster: []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}},
	}
	eng := engine.New(cfg.ControllerID, cfg.EGMRoster)
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now()})

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()
	statusHandler(eng, auditStore, cfg, runtimeInfo{
		ConfigPath:       "/etc/g2s-mute/config.json",
		StartedAt:        time.Now().Add(-5 * time.Second),
		SimulatedTrigger: true,
	})(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var body applianceStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if body.ControllerID != "G2S-MC-TEST" {
		t.Fatalf("controller_id = %q", body.ControllerID)
	}
	if body.Runtime.InputMode != "SIMULATED_SOFTWARE_ONLY" {
		t.Fatalf("input_mode = %q", body.Runtime.InputMode)
	}
	if body.Readiness.Overall != "READY_LAB" {
		t.Fatalf("overall = %q", body.Readiness.Overall)
	}
	if body.Readiness.CertificateSummary["MISSING"] != 1 {
		t.Fatalf("missing certificate count = %d", body.Readiness.CertificateSummary["MISSING"])
	}
}
