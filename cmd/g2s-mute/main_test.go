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

func TestReadinessHandlerPolicy(t *testing.T) {
	baseCfg := config.Config{
		ControllerID: "G2S-MC-TEST",
		Database:     config.Database{Path: "/var/lib/g2s-mute/controller.db"},
		WebUI:        config.WebUI{BindAddress: "127.0.0.1:8444"},
		G2S: config.G2S{
			HostURL:      "http://127.0.0.1:8444/g2s",
			EndpointPath: "/g2s",
		},
		EGMRoster: []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}},
	}

	tests := []struct {
		name       string
		cfg        config.Config
		seedEvents []engine.Event
		wantCode   int
		wantState  string
	}{
		{
			name: "READY returns 200",
			cfg: func() config.Config {
				cfg := baseCfg
				cfg.G2S.RequireTLS = true
				return cfg
			}(),
			seedEvents: []engine.Event{{Type: engine.EventBootComplete, At: time.Now()}},
			wantCode:   http.StatusOK,
			wantState:  "READY",
		},
		{
			name: "READY_LAB returns 200",
			cfg:  baseCfg,
			seedEvents: []engine.Event{
				{Type: engine.EventBootComplete, At: time.Now()},
			},
			wantCode:  http.StatusOK,
			wantState: "READY_LAB",
		},
		{
			name: "DEGRADED returns 503",
			cfg: func() config.Config {
				cfg := baseCfg
				cfg.G2S.RequireTLS = true
				return cfg
			}(),
			seedEvents: []engine.Event{
				{Type: engine.EventBootComplete, At: time.Now()},
				{Type: engine.EventDegraded, At: time.Now().Add(time.Millisecond)},
			},
			wantCode:  http.StatusServiceUnavailable,
			wantState: "DEGRADED",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			auditStore, err := store.Open(ctx, ":memory:")
			if err != nil {
				t.Fatalf("open store: %v", err)
			}
			t.Cleanup(func() { _ = auditStore.Close() })

			eng := engine.New(tc.cfg.ControllerID, tc.cfg.EGMRoster)
			runCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			eng.Start(runCtx)
			for _, event := range tc.seedEvents {
				eng.Submit(event)
			}
			waitForLastEvent(t, eng, string(tc.seedEvents[len(tc.seedEvents)-1].Type))

			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rr := httptest.NewRecorder()
			readinessHandler(eng, auditStore, tc.cfg, runtimeInfo{
				ConfigPath: "/etc/g2s-mute/config.json",
				StartedAt:  time.Now().Add(-2 * time.Second),
			})(rr, req)

			if rr.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d: %s", rr.Code, tc.wantCode, rr.Body.String())
			}
			var body readinessResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Overall != tc.wantState {
				t.Fatalf("overall = %q, want %q", body.Overall, tc.wantState)
			}
		})
	}
}

func waitForLastEvent(t *testing.T, eng *engine.Engine, event string) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if eng.Snapshot().LastEvent == event {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for last event %q (got %q)", event, eng.Snapshot().LastEvent)
}
