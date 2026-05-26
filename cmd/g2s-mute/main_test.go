package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/pendingdeliveryruntime"
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
		Timeouts:     config.Timeouts{EGMHeartbeatIntervalMS: 5000},
		CabinetProfile: config.CabinetProfile{
			WireHostURL:     "https://host-a.example/g2s",
			ListenerDNSName: "host-a.example",
			RequiredSANDNS:  []string{"host-a.example"},
			HostID:          "HOST-TEST-001",
			FirstTestEGMIDs: []string{"EGM-01"},
		},
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
	if body.Runtime.APIMutationAuthRequired {
		t.Fatalf("api_mutation_auth_required = %t, want false", body.Runtime.APIMutationAuthRequired)
	}
	if body.Runtime.AllowPrivateKeyExport {
		t.Fatalf("allow_private_key_export = %t, want false", body.Runtime.AllowPrivateKeyExport)
	}
	if body.Runtime.EGMHeartbeatIntervalMS != 5000 {
		t.Fatalf("egm_heartbeat_interval_ms = %d, want 5000", body.Runtime.EGMHeartbeatIntervalMS)
	}
	if body.ProfileSource != "file" {
		t.Fatalf("profile_source = %q, want file", body.ProfileSource)
	}
	if body.CabinetProfile.WireHostURL != "https://host-a.example/g2s" {
		t.Fatalf("wire_host_url = %q", body.CabinetProfile.WireHostURL)
	}
}

func TestStatusHandlerIncludesEndpointMetadataAndDriftWarning(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		ControllerID: "G2S-MC-ENDPOINT-TEST",
		Database:     config.Database{Path: "/tmp/g2s-mute.db"},
		WebUI:        config.WebUI{BindAddress: "127.0.0.1:8444"},
		Timeouts:     config.Timeouts{EGMHeartbeatIntervalMS: 5000},
		CabinetProfile: config.CabinetProfile{
			WireHostURL:     "https://host-a.example/g2s",
			ListenerDNSName: "host-a.example",
			RequiredSANDNS:  []string{"host-a.example"},
			HostID:          "HOST-TEST-001",
			FirstTestEGMIDs: []string{"EGM-01"},
		},
		G2S: config.G2S{
			HostURL:      "http://127.0.0.1:8444/g2s",
			EndpointPath: "/g2s",
		},
		EGMRoster: []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}},
	}
	eng := engine.New(cfg.ControllerID, cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)
	now := time.Now()
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: now})
	eng.Submit(engine.Event{
		Type:       engine.EventKeepAlive,
		EGMID:      "EGM-01",
		At:         now.Add(time.Second),
		SourceIP:   "10.20.30.40",
		SourcePort: 9443,
	})
	eng.Submit(engine.Event{
		Type:       engine.EventKeepAlive,
		EGMID:      "EGM-01",
		At:         now.Add(2 * time.Second),
		SourceIP:   "10.20.30.41",
		SourcePort: 9443,
	})
	waitForLastEvent(t, eng, string(engine.EventKeepAlive))

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()
	statusHandler(eng, auditStore, cfg, runtimeInfo{
		ConfigPath: "/etc/g2s-mute/config.json",
		StartedAt:  now.Add(-10 * time.Second),
	})(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var body applianceStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	var egm model.EGM
	found := false
	for _, row := range body.EGMs {
		if row.ID == "EGM-01" {
			egm = row
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected EGM-01 in status payload")
	}
	if egm.LastEndpointIP != "10.20.30.41" {
		t.Fatalf("last_endpoint_ip = %q, want 10.20.30.41", egm.LastEndpointIP)
	}
	if egm.LastEndpointPort != 9443 {
		t.Fatalf("last_endpoint_port = %d, want 9443", egm.LastEndpointPort)
	}
	if egm.LastEndpointSeenAt.IsZero() {
		t.Fatalf("expected last_endpoint_seen_at to be set")
	}
	if !egm.EndpointDrift {
		t.Fatalf("endpoint_drift_warning = false, want true")
	}
	if !egm.EndpointCollisionWarning {
		t.Fatalf("endpoint_collision_warning = false, want true")
	}
	if len(egm.EndpointCollisionTypes) == 0 {
		t.Fatalf("endpoint_collision_types empty, want ID_ENDPOINT_DRIFT")
	}
	if len(egm.EndpointDriftIPs) != 2 {
		t.Fatalf("endpoint_drift_ips len = %d, want 2", len(egm.EndpointDriftIPs))
	}
	if len(egm.RecentEndpoints) != 2 {
		t.Fatalf("recent_endpoints len = %d, want 2", len(egm.RecentEndpoints))
	}
	if egm.RecentEndpoints[0].IPAddress != "10.20.30.41" {
		t.Fatalf("recent_endpoints[0].ip = %q, want 10.20.30.41", egm.RecentEndpoints[0].IPAddress)
	}
	if egm.RecentEndpoints[0].SeenCount != 1 {
		t.Fatalf("recent_endpoints[0].seen_count = %d, want 1", egm.RecentEndpoints[0].SeenCount)
	}
	if body.EndpointCollisionSummary.Total == 0 {
		t.Fatalf("endpoint_collision_summary.total = %d, want >0", body.EndpointCollisionSummary.Total)
	}
	if body.EndpointCollisionSummary.IDEndpointDriftCount == 0 {
		t.Fatalf("endpoint_collision_summary.id_endpoint_drift_count = %d, want >0", body.EndpointCollisionSummary.IDEndpointDriftCount)
	}
}

func TestStatusHandlerIncludesSharedEndpointCollisionSummary(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		ControllerID: "G2S-MC-SHARED-ENDPOINT",
		Database:     config.Database{Path: "/tmp/g2s-mute.db"},
		WebUI:        config.WebUI{BindAddress: "127.0.0.1:8444"},
		Timeouts:     config.Timeouts{EGMHeartbeatIntervalMS: 5000},
		CabinetProfile: config.CabinetProfile{
			WireHostURL:     "https://host-a.example/g2s",
			ListenerDNSName: "host-a.example",
			RequiredSANDNS:  []string{"host-a.example"},
			HostID:          "HOST-TEST-001",
			FirstTestEGMIDs: []string{"EGM-01"},
		},
		G2S: config.G2S{
			HostURL:      "http://127.0.0.1:8444/g2s",
			EndpointPath: "/g2s",
		},
		EGMRoster: []config.EGM{
			{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443},
			{EGMID: "EGM-02", IPAddress: "127.0.0.1", Port: 9444},
		},
	}
	eng := engine.New(cfg.ControllerID, cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)
	now := time.Now()
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: now})
	eng.Submit(engine.Event{
		Type:       engine.EventKeepAlive,
		EGMID:      "EGM-01",
		At:         now.Add(time.Second),
		SourceIP:   "10.20.30.50",
		SourcePort: 9550,
	})
	eng.Submit(engine.Event{
		Type:       engine.EventKeepAlive,
		EGMID:      "EGM-02",
		At:         now.Add(2 * time.Second),
		SourceIP:   "10.20.30.50",
		SourcePort: 9550,
	})
	waitForLastEvent(t, eng, string(engine.EventKeepAlive))

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()
	statusHandler(eng, auditStore, cfg, runtimeInfo{
		ConfigPath: "/etc/g2s-mute/config.json",
		StartedAt:  now.Add(-10 * time.Second),
	})(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var body applianceStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if body.EndpointCollisionSummary.SharedEndpointCount != 1 {
		t.Fatalf("shared endpoint count = %d, want 1", body.EndpointCollisionSummary.SharedEndpointCount)
	}
	if body.EndpointCollisionSummary.Total != 1 {
		t.Fatalf("collision total = %d, want 1", body.EndpointCollisionSummary.Total)
	}
	if len(body.EndpointCollisions) != 1 {
		t.Fatalf("endpoint collisions len = %d, want 1", len(body.EndpointCollisions))
	}
	if body.EndpointCollisions[0].CollisionType != model.EndpointCollisionSharedEndpoint {
		t.Fatalf("collision type = %q, want %q", body.EndpointCollisions[0].CollisionType, model.EndpointCollisionSharedEndpoint)
	}
}

func TestBuildRuntimeStatusIncludesAPIMutationAuthFlag(t *testing.T) {
	cfg := config.Config{
		Database: config.Database{Path: "/tmp/controller.db"},
		WebUI: config.WebUI{
			BindAddress: "127.0.0.1:8444",
		},
		Timeouts: config.Timeouts{
			EGMHeartbeatIntervalMS: 7000,
		},
		G2S: config.G2S{
			HostURL:           "http://127.0.0.1:8444/g2s",
			EndpointPath:      "/g2s",
			RequireTLS:        false,
			RequireClientCert: false,
		},
		API: config.API{
			AuthToken: "lab-secret",
		},
	}
	status := buildRuntimeStatus(cfg, runtimeInfo{ConfigPath: "configs/config.example.json", StartedAt: time.Now()}, nil)
	if !status.APIMutationAuthRequired {
		t.Fatalf("api_mutation_auth_required = %t, want true", status.APIMutationAuthRequired)
	}
	if status.AllowPrivateKeyExport {
		t.Fatalf("allow_private_key_export = %t, want false", status.AllowPrivateKeyExport)
	}
	if status.EGMHeartbeatIntervalMS != 7000 {
		t.Fatalf("egm_heartbeat_interval_ms = %d, want 7000", status.EGMHeartbeatIntervalMS)
	}
}

func TestBuildRuntimeStatusTrustedPrivateNetworkBypassForRequest(t *testing.T) {
	cfg := config.Config{
		Database: config.Database{Path: "/tmp/controller.db"},
		WebUI: config.WebUI{
			BindAddress:                         "0.0.0.0:8444",
			RequireLogin:                        false,
			AllowTrustedPrivateNetworkMutations: true,
			AllowPrivateKeyExport:               true,
		},
		Timeouts: config.Timeouts{
			EGMHeartbeatIntervalMS: 5000,
		},
		G2S: config.G2S{
			HostURL:           "http://127.0.0.1:8444/g2s",
			EndpointPath:      "/g2s",
			RequireTLS:        false,
			RequireClientCert: false,
		},
		API: config.API{
			AuthToken: "lab-secret",
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	req.RemoteAddr = "192.168.10.50:5151"
	status := buildRuntimeStatus(cfg, runtimeInfo{ConfigPath: "configs/config.pi.example.json", StartedAt: time.Now()}, req)
	if status.APIMutationAuthRequired {
		t.Fatalf("api_mutation_auth_required = %t, want false for trusted private network request", status.APIMutationAuthRequired)
	}
	if !status.TrustedMutationBypassActive {
		t.Fatal("expected trusted_mutation_bypass_active to be true")
	}
	if !status.AllowPrivateKeyExport {
		t.Fatal("expected allow_private_key_export to be true")
	}
	if status.EGMHeartbeatIntervalMS != 5000 {
		t.Fatalf("egm_heartbeat_interval_ms = %d, want 5000", status.EGMHeartbeatIntervalMS)
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

func TestBuildReadinessStatusPrecedence(t *testing.T) {
	baseCfg := config.Config{
		G2S: config.G2S{
			RequireTLS:        false,
			RequireClientCert: false,
		},
		WebUI: config.WebUI{
			RequireLogin: true,
		},
	}
	healthySnapshot := engine.Snapshot{
		State: model.StateHealthy,
		EGMs:  []model.EGM{{ID: "EGM-01"}},
	}

	tests := []struct {
		name         string
		cfg          config.Config
		snapshot     engine.Snapshot
		certificates []model.CertificateInventory
		wantOverall  string
		wantIssue    string
		wantWarning  string
	}{
		{
			name:        "READY_LAB when TLS is disabled without degraded conditions",
			cfg:         baseCfg,
			snapshot:    healthySnapshot,
			wantOverall: "READY_LAB",
		},
		{
			name: "audit error remains DEGRADED even when TLS is disabled",
			cfg:  baseCfg,
			snapshot: engine.Snapshot{
				State:      model.StateHealthy,
				AuditError: "audit store unavailable",
				EGMs:       []model.EGM{{ID: "EGM-01"}},
			},
			wantOverall: "DEGRADED",
			wantIssue:   "audit store unavailable",
		},
		{
			name: "no EGMs becomes a warning while lab readiness remains READY_LAB",
			cfg:  baseCfg,
			snapshot: engine.Snapshot{
				State: model.StateHealthy,
				EGMs:  []model.EGM{},
			},
			wantOverall: "READY_LAB",
			wantWarning: "No EGM traffic has been observed yet",
		},
		{
			name: "blocking certificate remains DEGRADED even when TLS is disabled",
			cfg: func() config.Config {
				cfg := baseCfg
				cfg.G2S.RequireClientCert = true
				return cfg
			}(),
			snapshot: healthySnapshot,
			certificates: []model.CertificateInventory{
				{
					Role:          "g2s_client_cert",
					Status:        "MISSING",
					LastCheckedAt: time.Now(),
				},
			},
			wantOverall: "DEGRADED",
			wantIssue:   "g2s_client_cert certificate is MISSING",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status := buildReadinessStatus(tc.snapshot, tc.cfg, tc.certificates, "")
			if status.Overall != tc.wantOverall {
				t.Fatalf("overall = %q, want %q", status.Overall, tc.wantOverall)
			}
			if tc.wantIssue == "" {
				if tc.wantWarning == "" {
					return
				}
				for _, warning := range status.Warnings {
					if warning == tc.wantWarning {
						return
					}
				}
				t.Fatalf("warnings = %v, want %q", status.Warnings, tc.wantWarning)
			}
			for _, issue := range status.Issues {
				if issue == tc.wantIssue {
					return
				}
			}
			t.Fatalf("issues = %v, want %q", status.Issues, tc.wantIssue)
		})
	}
}

func TestResolveCabinetProfileUsesOverrideAndSourceMetadata(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	fileProfile := config.CabinetProfile{
		WireHostURL:     "https://file.example/g2s",
		ListenerDNSName: "file.example",
		RequiredSANDNS:  []string{"file.example"},
		HostID:          "HOST-FILE",
		FirstTestEGMIDs: []string{"EGM-01"},
	}

	resolved, err := resolveCabinetProfile(ctx, auditStore, fileProfile)
	if err != nil {
		t.Fatalf("resolve file profile: %v", err)
	}
	if resolved.ProfileSource != "file" {
		t.Fatalf("profile_source = %q, want file", resolved.ProfileSource)
	}

	override := config.CabinetProfile{
		WireHostURL:     "https://override.example/g2s",
		ListenerDNSName: "override.example",
		ListenerIP:      "10.20.30.40",
		RequiredSANDNS:  []string{"override.example"},
		RequiredSANIPs:  []string{"10.20.30.40"},
		HostID:          "HOST-OVERRIDE",
		FirstTestEGMIDs: []string{"EGM-99"},
	}
	if err := auditStore.UpsertCabinetProfileOverride(ctx, override, "tester"); err != nil {
		t.Fatalf("upsert override: %v", err)
	}
	resolved, err = resolveCabinetProfile(ctx, auditStore, fileProfile)
	if err != nil {
		t.Fatalf("resolve override profile: %v", err)
	}
	if resolved.ProfileSource != "override" {
		t.Fatalf("profile_source = %q, want override", resolved.ProfileSource)
	}
	if resolved.Effective.HostID != "HOST-OVERRIDE" {
		t.Fatalf("effective host_id = %q", resolved.Effective.HostID)
	}
	if resolved.ProfileLastUpdatedAt == nil {
		t.Fatal("expected profile_last_updated_at to be set")
	}
	if !resolved.ProfileDiffersFromFile {
		t.Fatal("expected profile_differs_from_file to be true")
	}
}

func TestResolveCabinetProfileInvalidOverrideFallsBackWithWarning(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	fileProfile := config.CabinetProfile{
		WireHostURL:     "https://file.example/g2s",
		ListenerDNSName: "file.example",
		RequiredSANDNS:  []string{"file.example"},
		HostID:          "HOST-FILE",
		FirstTestEGMIDs: []string{"EGM-01"},
	}

	invalidOverride := config.CabinetProfile{
		WireHostURL:     "https://override.example/g2s",
		ListenerDNSName: "override.example",
		ListenerIP:      "not-an-ip",
		RequiredSANDNS:  []string{"override.example"},
		HostID:          "HOST-OVERRIDE",
		FirstTestEGMIDs: []string{"EGM-99"},
	}
	if err := auditStore.UpsertCabinetProfileOverride(ctx, invalidOverride, "tester"); err != nil {
		t.Fatalf("upsert invalid override: %v", err)
	}
	resolved, err := resolveCabinetProfile(ctx, auditStore, fileProfile)
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}
	if resolved.Warning == "" {
		t.Fatal("expected warning for invalid override")
	}
	if resolved.Effective.WireHostURL != fileProfile.WireHostURL {
		t.Fatalf("expected fallback to file profile, got %q", resolved.Effective.WireHostURL)
	}
}

func TestCabinetProfileHandlerCRUD(t *testing.T) {
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
	}
	handler := cabinetProfileHandler(auditStore, cfg)

	getReq := httptest.NewRequest(http.MethodGet, "/api/cabinet-profile", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}
	var getBody cabinetProfileResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if getBody.ProfileSource != "file" {
		t.Fatalf("GET profile_source = %q", getBody.ProfileSource)
	}
	if getBody.OverridePresent {
		t.Fatal("expected no override on GET")
	}

	override := config.CabinetProfile{
		WireHostURL:     "https://override.example/g2s",
		ListenerDNSName: "override.example",
		ListenerIP:      "10.20.30.40",
		RequiredSANDNS:  []string{"override.example"},
		RequiredSANIPs:  []string{"10.20.30.40"},
		HostID:          "HOST-OVERRIDE",
		FirstTestEGMIDs: []string{"EGM-99"},
	}
	raw, _ := json.Marshal(override)
	putReq := httptest.NewRequest(http.MethodPut, "/api/cabinet-profile", bytes.NewReader(raw))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("X-Operator", "tester")
	putRec := httptest.NewRecorder()
	handler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d: %s", putRec.Code, putRec.Body.String())
	}
	var putBody cabinetProfileResponse
	if err := json.Unmarshal(putRec.Body.Bytes(), &putBody); err != nil {
		t.Fatalf("decode PUT: %v", err)
	}
	if putBody.ProfileSource != "override" {
		t.Fatalf("PUT profile_source = %q", putBody.ProfileSource)
	}
	if !putBody.OverridePresent {
		t.Fatal("expected override_present on PUT")
	}
	if putBody.Effective.HostID != "HOST-OVERRIDE" {
		t.Fatalf("PUT host_id = %q", putBody.Effective.HostID)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/cabinet-profile", nil)
	deleteRec := httptest.NewRecorder()
	handler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	var deleteBody cabinetProfileResponse
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &deleteBody); err != nil {
		t.Fatalf("decode DELETE: %v", err)
	}
	if deleteBody.ProfileSource != "file" || deleteBody.OverridePresent {
		t.Fatalf("DELETE response unexpected: %+v", deleteBody)
	}
}

func TestCabinetProfileSuggestionsHandlerReturnsObservedRecommendations(t *testing.T) {
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
		EGMRoster: []config.EGM{
			{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443},
			{EGMID: "EGM-02", IPAddress: "127.0.0.1", Port: 9444},
			{EGMID: "EGM-03", IPAddress: "127.0.0.1", Port: 9445},
		},
	}
	eng := engine.New("G2S-MC-SUGGESTIONS", cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)
	now := time.Now().UTC()
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: now})
	eng.Submit(engine.Event{Type: engine.EventKeepAlive, EGMID: "EGM-02", At: now.Add(1 * time.Second)})
	eng.Submit(engine.Event{Type: engine.EventKeepAlive, EGMID: "EGM-01", At: now.Add(2 * time.Second)})
	eng.Submit(engine.Event{Type: engine.EventKeepAlive, EGMID: "EGM-03", At: now.Add(3 * time.Second)})
	waitForLastEvent(t, eng, string(engine.EventKeepAlive))

	handler := cabinetProfileSuggestionsHandler(eng, auditStore, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/cabinet-profile/suggestions", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", rec.Code, rec.Body.String())
	}

	var body cabinetProfileSuggestionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode suggestions response: %v", err)
	}

	wantObserved := []string{"EGM-03", "EGM-01", "EGM-02"}
	if !reflect.DeepEqual(body.ObservedEGMIDs, wantObserved) {
		t.Fatalf("observed_egm_ids = %#v, want %#v", body.ObservedEGMIDs, wantObserved)
	}
	if !reflect.DeepEqual(body.RecommendedFirstTestEGMIDs, wantObserved) {
		t.Fatalf("recommended_first_test_egm_ids = %#v, want %#v", body.RecommendedFirstTestEGMIDs, wantObserved)
	}
	if !body.PlaceholderDetected {
		t.Fatal("placeholder_detected = false, want true")
	}
	if body.Reason == "" {
		t.Fatal("expected non-empty reason")
	}
	if len(body.Messages) == 0 {
		t.Fatal("expected at least one operator message")
	}
}

func TestBuildCabinetProfileSuggestionsLimitsRecommendationsToThree(t *testing.T) {
	base := time.Now().UTC()
	snapshot := engine.Snapshot{
		EGMs: []model.EGM{
			{ID: "EGM-01", LastSeen: base.Add(1 * time.Second)},
			{ID: "EGM-02", LastSeen: base.Add(2 * time.Second)},
			{ID: "EGM-03", LastSeen: base.Add(3 * time.Second)},
			{ID: "EGM-04", LastSeen: base.Add(4 * time.Second)},
		},
	}

	got := buildCabinetProfileSuggestions(snapshot, []string{"CAB-101"})
	wantObserved := []string{"EGM-04", "EGM-03", "EGM-02", "EGM-01"}
	wantRecommended := []string{"EGM-04", "EGM-03", "EGM-02"}
	if !reflect.DeepEqual(got.ObservedEGMIDs, wantObserved) {
		t.Fatalf("observed_egm_ids = %#v, want %#v", got.ObservedEGMIDs, wantObserved)
	}
	if !reflect.DeepEqual(got.RecommendedFirstTestEGMIDs, wantRecommended) {
		t.Fatalf("recommended_first_test_egm_ids = %#v, want %#v", got.RecommendedFirstTestEGMIDs, wantRecommended)
	}
	if got.PlaceholderDetected {
		t.Fatal("placeholder_detected = true, want false")
	}
}

func TestHeartbeatPolicyHandlerCRUD(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		Timeouts: config.Timeouts{
			EGMHeartbeatIntervalMS:         5000,
			EGMHeartbeatWarningAfterMissed: 3,
			EGMHeartbeatBlockAfterMissed:   6,
		},
	}
	handler := heartbeatPolicyHandler(auditStore, cfg)

	getReq := httptest.NewRequest(http.MethodGet, "/api/heartbeat-policy", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}
	var getBody heartbeatPolicyResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if getBody.PolicySource != "file" {
		t.Fatalf("GET policy_source = %q", getBody.PolicySource)
	}
	if getBody.Effective.IntervalMS != 5000 || getBody.Effective.WarningAfterMissed != 3 || getBody.Effective.BlockAfterMissed != 6 {
		t.Fatalf("unexpected GET effective policy: %+v", getBody.Effective)
	}

	raw := []byte(`{"interval_ms":7000,"warning_after_missed":4,"block_after_missed":9}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/heartbeat-policy", bytes.NewReader(raw))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("X-Operator", "tester")
	putRec := httptest.NewRecorder()
	handler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d: %s", putRec.Code, putRec.Body.String())
	}
	var putBody heartbeatPolicyResponse
	if err := json.Unmarshal(putRec.Body.Bytes(), &putBody); err != nil {
		t.Fatalf("decode PUT: %v", err)
	}
	if putBody.PolicySource != "override" {
		t.Fatalf("PUT policy_source = %q", putBody.PolicySource)
	}
	if putBody.Effective.IntervalMS != 7000 || putBody.Effective.WarningAfterMissed != 4 || putBody.Effective.BlockAfterMissed != 9 {
		t.Fatalf("unexpected PUT effective policy: %+v", putBody.Effective)
	}
	if !putBody.OverridePresent {
		t.Fatal("expected override_present on PUT")
	}
	if putBody.Override == nil || putBody.Override.IntervalMS != 7000 {
		t.Fatalf("expected override interval 7000, got %+v", putBody.Override)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/heartbeat-policy", nil)
	deleteRec := httptest.NewRecorder()
	handler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	var deleteBody heartbeatPolicyResponse
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &deleteBody); err != nil {
		t.Fatalf("decode DELETE: %v", err)
	}
	if deleteBody.PolicySource != "file" || deleteBody.OverridePresent {
		t.Fatalf("unexpected DELETE response: %+v", deleteBody)
	}
	if deleteBody.Effective.IntervalMS != 5000 || deleteBody.Effective.WarningAfterMissed != 3 || deleteBody.Effective.BlockAfterMissed != 6 {
		t.Fatalf("unexpected DELETE effective policy: %+v", deleteBody.Effective)
	}
}

func TestHeartbeatPolicyHandlerValidation(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		Timeouts: config.Timeouts{
			EGMHeartbeatIntervalMS:         5000,
			EGMHeartbeatWarningAfterMissed: 3,
			EGMHeartbeatBlockAfterMissed:   6,
		},
	}
	handler := heartbeatPolicyHandler(auditStore, cfg)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "interval must be positive",
			body: `{"interval_ms":0,"warning_after_missed":3,"block_after_missed":6}`,
		},
		{
			name: "warning must be >=1",
			body: `{"interval_ms":5000,"warning_after_missed":0,"block_after_missed":6}`,
		},
		{
			name: "block must be >= warning",
			body: `{"interval_ms":5000,"warning_after_missed":6,"block_after_missed":5}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/heartbeat-policy", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHeartbeatPolicyRouteMutationAuthTokenGuard(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		Timeouts: config.Timeouts{
			EGMHeartbeatIntervalMS:         5000,
			EGMHeartbeatWarningAfterMissed: 3,
			EGMHeartbeatBlockAfterMissed:   6,
		},
	}
	handler := requireMutationAuthForMethods(
		heartbeatPolicyHandler(auditStore, cfg),
		cfg,
		http.MethodPut,
		http.MethodDelete,
	)

	getReq := httptest.NewRequest(http.MethodGet, "/api/heartbeat-policy", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}

	unauthorizedPut := httptest.NewRequest(http.MethodPut, "/api/heartbeat-policy", bytes.NewBufferString(`{"interval_ms":6000,"warning_after_missed":4,"block_after_missed":7}`))
	unauthorizedPut.Header.Set("Content-Type", "application/json")
	unauthorizedPutRec := httptest.NewRecorder()
	handler(unauthorizedPutRec, unauthorizedPut)
	if !deniedByAuth(unauthorizedPutRec.Code) {
		t.Fatalf("PUT without token status = %d, want 401/403", unauthorizedPutRec.Code)
	}

	authorizedPut := httptest.NewRequest(http.MethodPut, "/api/heartbeat-policy", bytes.NewBufferString(`{"interval_ms":6000,"warning_after_missed":4,"block_after_missed":7}`))
	authorizedPut.Header.Set("Content-Type", "application/json")
	authorizedPut.Header.Set("Authorization", "Bearer lab-secret")
	authorizedPutRec := httptest.NewRecorder()
	handler(authorizedPutRec, authorizedPut)
	if authorizedPutRec.Code != http.StatusOK {
		t.Fatalf("PUT with token status = %d: %s", authorizedPutRec.Code, authorizedPutRec.Body.String())
	}

	unauthorizedDelete := httptest.NewRequest(http.MethodDelete, "/api/heartbeat-policy", nil)
	unauthorizedDeleteRec := httptest.NewRecorder()
	handler(unauthorizedDeleteRec, unauthorizedDelete)
	if !deniedByAuth(unauthorizedDeleteRec.Code) {
		t.Fatalf("DELETE without token status = %d, want 401/403", unauthorizedDeleteRec.Code)
	}

	authorizedDelete := httptest.NewRequest(http.MethodDelete, "/api/heartbeat-policy", nil)
	authorizedDelete.Header.Set("Authorization", "Bearer lab-secret")
	authorizedDeleteRec := httptest.NewRecorder()
	handler(authorizedDeleteRec, authorizedDelete)
	if authorizedDeleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE with token status = %d: %s", authorizedDeleteRec.Code, authorizedDeleteRec.Body.String())
	}
}

func TestSessionWorkflowHandlerCRUDAndValidation(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	handler := sessionWorkflowHandler(auditStore, config.Config{})

	getReq := httptest.NewRequest(http.MethodGet, "/api/session-workflow", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}
	var getBody sessionWorkflowResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if getBody.CurrentPhase != "pre_check" {
		t.Fatalf("GET current_phase = %q, want pre_check", getBody.CurrentPhase)
	}
	if getBody.Persisted {
		t.Fatalf("GET persisted = %t, want false", getBody.Persisted)
	}

	validPut := `{"current_phase":"run_active","completed_steps":["pre_check","connect_observe"],"operator_notes":"session running"}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/session-workflow", bytes.NewBufferString(validPut))
	putReq.Header.Set("Content-Type", "application/json")
	putRec := httptest.NewRecorder()
	handler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d: %s", putRec.Code, putRec.Body.String())
	}
	var putBody sessionWorkflowResponse
	if err := json.Unmarshal(putRec.Body.Bytes(), &putBody); err != nil {
		t.Fatalf("decode PUT: %v", err)
	}
	if putBody.CurrentPhase != "run_active" {
		t.Fatalf("PUT current_phase = %q", putBody.CurrentPhase)
	}
	if !reflect.DeepEqual(putBody.CompletedSteps, []string{"pre_check", "connect_observe"}) {
		t.Fatalf("PUT completed_steps = %#v", putBody.CompletedSteps)
	}
	if !putBody.Persisted {
		t.Fatalf("PUT persisted = %t, want true", putBody.Persisted)
	}
	if putBody.LastUpdatedAt == nil || putBody.LastUpdatedAt.IsZero() {
		t.Fatalf("expected last_updated_at in PUT response")
	}

	invalidCases := []string{
		`{"current_phase":"","completed_steps":[],"operator_notes":""}`,
		`{"current_phase":"not_a_phase","completed_steps":[],"operator_notes":""}`,
		`{"current_phase":"pre_check","completed_steps":["pre_check","pre_check"],"operator_notes":""}`,
		`{"current_phase":"pre_check","completed_steps":["bad_step"],"operator_notes":""}`,
	}
	for _, body := range invalidCases {
		req := httptest.NewRequest(http.MethodPut, "/api/session-workflow", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid PUT status = %d, want 400: %s", rec.Code, rec.Body.String())
		}
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/session-workflow", nil)
	deleteRec := httptest.NewRecorder()
	handler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	var deleteBody sessionWorkflowResponse
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &deleteBody); err != nil {
		t.Fatalf("decode DELETE: %v", err)
	}
	if deleteBody.Persisted {
		t.Fatalf("DELETE persisted = %t, want false", deleteBody.Persisted)
	}
}

func TestSessionWorkflowRouteMutationAuthTokenGuard(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
	}
	handler := requireMutationAuthForMethods(
		sessionWorkflowHandler(auditStore, cfg),
		cfg,
		http.MethodPut,
		http.MethodDelete,
	)

	getReq := httptest.NewRequest(http.MethodGet, "/api/session-workflow", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}

	unauthorizedPut := httptest.NewRequest(http.MethodPut, "/api/session-workflow", bytes.NewBufferString(`{"current_phase":"pre_check","completed_steps":[],"operator_notes":""}`))
	unauthorizedPut.Header.Set("Content-Type", "application/json")
	unauthorizedPutRec := httptest.NewRecorder()
	handler(unauthorizedPutRec, unauthorizedPut)
	if !deniedByAuth(unauthorizedPutRec.Code) {
		t.Fatalf("PUT without token status = %d, want 401/403", unauthorizedPutRec.Code)
	}

	authorizedPut := httptest.NewRequest(http.MethodPut, "/api/session-workflow", bytes.NewBufferString(`{"current_phase":"connect_observe","completed_steps":["pre_check"],"operator_notes":"auth ok"}`))
	authorizedPut.Header.Set("Content-Type", "application/json")
	authorizedPut.Header.Set("Authorization", "Bearer lab-secret")
	authorizedPutRec := httptest.NewRecorder()
	handler(authorizedPutRec, authorizedPut)
	if authorizedPutRec.Code != http.StatusOK {
		t.Fatalf("PUT with token status = %d: %s", authorizedPutRec.Code, authorizedPutRec.Body.String())
	}

	unauthorizedDelete := httptest.NewRequest(http.MethodDelete, "/api/session-workflow", nil)
	unauthorizedDeleteRec := httptest.NewRecorder()
	handler(unauthorizedDeleteRec, unauthorizedDelete)
	if !deniedByAuth(unauthorizedDeleteRec.Code) {
		t.Fatalf("DELETE without token status = %d, want 401/403", unauthorizedDeleteRec.Code)
	}

	authorizedDelete := httptest.NewRequest(http.MethodDelete, "/api/session-workflow", nil)
	authorizedDelete.Header.Set("Authorization", "Bearer lab-secret")
	authorizedDeleteRec := httptest.NewRecorder()
	handler(authorizedDeleteRec, authorizedDelete)
	if authorizedDeleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE with token status = %d: %s", authorizedDeleteRec.Code, authorizedDeleteRec.Body.String())
	}
}

func TestSessionWorkflowRouteAllowsTrustedPrivateNetworkWithoutToken(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		WebUI: config.WebUI{
			RequireLogin:                        false,
			AllowTrustedPrivateNetworkMutations: true,
		},
	}
	handler := requireMutationAuthForMethods(
		sessionWorkflowHandler(auditStore, cfg),
		cfg,
		http.MethodPut,
		http.MethodDelete,
	)

	privateReq := httptest.NewRequest(http.MethodPut, "/api/session-workflow", bytes.NewBufferString(`{"current_phase":"connect_observe","completed_steps":["pre_check"],"operator_notes":"trusted network update"}`))
	privateReq.Header.Set("Content-Type", "application/json")
	privateReq.RemoteAddr = "192.168.10.70:4455"
	privateRec := httptest.NewRecorder()
	handler(privateRec, privateReq)
	if privateRec.Code != http.StatusOK {
		t.Fatalf("PUT trusted private network status = %d: %s", privateRec.Code, privateRec.Body.String())
	}

	publicReq := httptest.NewRequest(http.MethodPut, "/api/session-workflow", bytes.NewBufferString(`{"current_phase":"connect_observe","completed_steps":["pre_check"],"operator_notes":"public blocked"}`))
	publicReq.Header.Set("Content-Type", "application/json")
	publicReq.RemoteAddr = "198.51.100.80:4455"
	publicRec := httptest.NewRecorder()
	handler(publicRec, publicReq)
	if !deniedByAuth(publicRec.Code) {
		t.Fatalf("PUT public network without token status = %d, want 401/403", publicRec.Code)
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

func deniedByAuth(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}

func TestCabinetProfileRouteMutationAuthTokenGuard(t *testing.T) {
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
	}
	handler := requireMutationAuthForMethods(
		cabinetProfileHandler(auditStore, cfg),
		cfg,
		http.MethodPut,
		http.MethodDelete,
	)

	getReq := httptest.NewRequest(http.MethodGet, "/api/cabinet-profile", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}

	override := config.CabinetProfile{
		WireHostURL:     "https://override.example/g2s",
		ListenerDNSName: "override.example",
		ListenerIP:      "10.20.30.40",
		RequiredSANDNS:  []string{"override.example"},
		RequiredSANIPs:  []string{"10.20.30.40"},
		HostID:          "HOST-OVERRIDE",
		FirstTestEGMIDs: []string{"EGM-99"},
	}
	raw, _ := json.Marshal(override)

	putReqUnauthorized := httptest.NewRequest(http.MethodPut, "/api/cabinet-profile", bytes.NewReader(raw))
	putReqUnauthorized.Header.Set("Content-Type", "application/json")
	putRecUnauthorized := httptest.NewRecorder()
	handler(putRecUnauthorized, putReqUnauthorized)
	if !deniedByAuth(putRecUnauthorized.Code) {
		t.Fatalf("PUT without token status = %d, want 401/403", putRecUnauthorized.Code)
	}

	putReqInvalid := httptest.NewRequest(http.MethodPut, "/api/cabinet-profile", bytes.NewReader(raw))
	putReqInvalid.Header.Set("Content-Type", "application/json")
	putReqInvalid.Header.Set("Authorization", "Bearer wrong-token")
	putRecInvalid := httptest.NewRecorder()
	handler(putRecInvalid, putReqInvalid)
	if !deniedByAuth(putRecInvalid.Code) {
		t.Fatalf("PUT with invalid token status = %d, want 401/403", putRecInvalid.Code)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/cabinet-profile", bytes.NewReader(raw))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("Authorization", "Bearer lab-secret")
	putRec := httptest.NewRecorder()
	handler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d: %s", putRec.Code, putRec.Body.String())
	}

	deleteReqUnauthorized := httptest.NewRequest(http.MethodDelete, "/api/cabinet-profile", nil)
	deleteRecUnauthorized := httptest.NewRecorder()
	handler(deleteRecUnauthorized, deleteReqUnauthorized)
	if !deniedByAuth(deleteRecUnauthorized.Code) {
		t.Fatalf("DELETE without token status = %d, want 401/403", deleteRecUnauthorized.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/cabinet-profile", nil)
	deleteReq.Header.Set("Authorization", "Bearer lab-secret")
	deleteRec := httptest.NewRecorder()
	handler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestCabinetProfileRouteAllowsTrustedPrivateNetworkWithoutToken(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		WebUI: config.WebUI{
			RequireLogin:                        false,
			AllowTrustedPrivateNetworkMutations: true,
		},
		CabinetProfile: config.CabinetProfile{
			WireHostURL:     "https://file.example/g2s",
			ListenerDNSName: "file.example",
			RequiredSANDNS:  []string{"file.example"},
			HostID:          "HOST-FILE",
			FirstTestEGMIDs: []string{"EGM-01"},
		},
	}
	handler := requireMutationAuthForMethods(
		cabinetProfileHandler(auditStore, cfg),
		cfg,
		http.MethodPut,
		http.MethodDelete,
	)

	override := config.CabinetProfile{
		WireHostURL:     "https://override.example/g2s",
		ListenerDNSName: "override.example",
		ListenerIP:      "10.20.30.40",
		RequiredSANDNS:  []string{"override.example"},
		RequiredSANIPs:  []string{"10.20.30.40"},
		HostID:          "HOST-OVERRIDE",
		FirstTestEGMIDs: []string{"EGM-99"},
	}
	raw, _ := json.Marshal(override)

	privateReq := httptest.NewRequest(http.MethodPut, "/api/cabinet-profile", bytes.NewReader(raw))
	privateReq.RemoteAddr = "192.168.10.99:5544"
	privateReq.Header.Set("Content-Type", "application/json")
	privateRec := httptest.NewRecorder()
	handler(privateRec, privateReq)
	if privateRec.Code != http.StatusOK {
		t.Fatalf("PUT from trusted private network without token status = %d: %s", privateRec.Code, privateRec.Body.String())
	}

	publicReq := httptest.NewRequest(http.MethodPut, "/api/cabinet-profile", bytes.NewReader(raw))
	publicReq.RemoteAddr = "198.51.100.25:5544"
	publicReq.Header.Set("Content-Type", "application/json")
	publicRec := httptest.NewRecorder()
	handler(publicRec, publicReq)
	if !deniedByAuth(publicRec.Code) {
		t.Fatalf("PUT from public network without token status = %d, want 401/403", publicRec.Code)
	}
}

func TestSessionEvidenceHandlerCRUDAndAuth(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		WebUI: config.WebUI{
			RequireLogin:                        false,
			AllowTrustedPrivateNetworkMutations: true,
		},
	}
	handler := sessionEvidenceHandler(auditStore, cfg)

	payload := `{"captured_at":"2026-05-20T21:00:00Z","operator_notes":"clean run","session":{"overall_state":"LAB_READY","readyz_state":"READY_LAB","preflight_state":"PASS"},"cabinet_profile":{"host_id":"HOST-TSPI4-001","wire_host_url":"https://tspi4.local:8444/g2s"}}`

	postReq := httptest.NewRequest(http.MethodPost, "/api/session-evidence", bytes.NewBufferString(payload))
	postReq.RemoteAddr = "192.168.10.70:4555"
	postRec := httptest.NewRecorder()
	handler(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("POST trusted private network status = %d: %s", postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/session-evidence?limit=10", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}
	var records []model.SessionEvidenceRecord
	if err := json.Unmarshal(getRec.Body.Bytes(), &records); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 session evidence record, got %d", len(records))
	}
	if records[0].OverallState != "LAB_READY" {
		t.Fatalf("overall_state = %q", records[0].OverallState)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/session-evidence?id=1", nil)
	deleteReq.RemoteAddr = "192.168.10.70:4555"
	deleteRec := httptest.NewRecorder()
	handler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE trusted private network status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	strictCfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
	}
	strictHandler := sessionEvidenceHandler(auditStore, strictCfg)
	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/api/session-evidence", bytes.NewBufferString(payload))
	unauthorizedReq.RemoteAddr = "198.51.100.40:4555"
	unauthorizedRec := httptest.NewRecorder()
	strictHandler(unauthorizedRec, unauthorizedReq)
	if !deniedByAuth(unauthorizedRec.Code) {
		t.Fatalf("POST public network without token status = %d, want 401/403", unauthorizedRec.Code)
	}

	unauthorizedDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/session-evidence?id=1", nil)
	unauthorizedDeleteReq.RemoteAddr = "198.51.100.40:4555"
	unauthorizedDeleteRec := httptest.NewRecorder()
	strictHandler(unauthorizedDeleteRec, unauthorizedDeleteReq)
	if !deniedByAuth(unauthorizedDeleteRec.Code) {
		t.Fatalf("DELETE public network without token status = %d, want 401/403", unauthorizedDeleteRec.Code)
	}
}

func TestSessionEvidenceByIDDeleteRouteAndAuth(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	record := model.SessionEvidenceRecord{
		CreatedAt:      time.Now().UTC().Truncate(time.Second),
		OverallState:   "LAB_READY",
		ReadyzState:    "READY_LAB",
		PreflightState: "PASS",
		HostID:         "HOST-TSPI4-001",
		WireHostURL:    "https://tspi4.local:8444/g2s",
		OperatorNotes:  "route test",
		PayloadJSON:    `{"session":{"overall_state":"LAB_READY"}}`,
	}
	id, err := auditStore.RecordSessionEvidence(ctx, record)
	if err != nil {
		t.Fatalf("record session evidence: %v", err)
	}

	cfg := config.Config{API: config.API{AuthToken: "lab-secret"}}
	handler := requireMutationAuthForMethods(
		sessionEvidenceByIDHandler(auditStore, cfg),
		cfg,
		http.MethodDelete,
	)

	unauthorizedReq := httptest.NewRequest(http.MethodDelete, "/api/session-evidence/"+strconv.FormatInt(id, 10), nil)
	unauthorizedReq.RemoteAddr = "198.51.100.10:9999"
	unauthorizedRec := httptest.NewRecorder()
	handler(unauthorizedRec, unauthorizedReq)
	if !deniedByAuth(unauthorizedRec.Code) {
		t.Fatalf("DELETE without token status = %d, want 401/403", unauthorizedRec.Code)
	}

	authorizedReq := httptest.NewRequest(http.MethodDelete, "/api/session-evidence/"+strconv.FormatInt(id, 10), nil)
	authorizedReq.RemoteAddr = "198.51.100.10:9999"
	authorizedReq.Header.Set("Authorization", "Bearer lab-secret")
	authorizedRec := httptest.NewRecorder()
	handler(authorizedRec, authorizedReq)
	if authorizedRec.Code != http.StatusOK {
		t.Fatalf("DELETE with token status = %d: %s", authorizedRec.Code, authorizedRec.Body.String())
	}

	missingReq := httptest.NewRequest(http.MethodDelete, "/api/session-evidence/"+strconv.FormatInt(id, 10), nil)
	missingReq.RemoteAddr = "198.51.100.10:9999"
	missingReq.Header.Set("Authorization", "Bearer lab-secret")
	missingRec := httptest.NewRecorder()
	handler(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("DELETE missing status = %d, want 404", missingRec.Code)
	}

	invalidReq := httptest.NewRequest(http.MethodDelete, "/api/session-evidence/not-a-number", nil)
	invalidReq.RemoteAddr = "198.51.100.10:9999"
	invalidReq.Header.Set("Authorization", "Bearer lab-secret")
	invalidRec := httptest.NewRecorder()
	handler(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("DELETE invalid id status = %d, want 400", invalidRec.Code)
	}
}

func TestSessionEvidenceExportAllHandlerShapeAndNoAuthRequirement(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	records := []model.SessionEvidenceRecord{
		{
			CreatedAt:      time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Second),
			OverallState:   "LAB_READY",
			ReadyzState:    "READY_LAB",
			PreflightState: "PASS",
			HostID:         "HOST-TSPI4-001",
			WireHostURL:    "https://tspi4.local:8444/g2s",
			OperatorNotes:  "first capture",
			PayloadJSON:    `{"session":{"overall_state":"LAB_READY"},"operator_notes":"first capture"}`,
		},
		{
			CreatedAt:      time.Now().UTC().Add(-1 * time.Minute).Truncate(time.Second),
			OverallState:   "ACTION_NEEDED",
			ReadyzState:    "DEGRADED",
			PreflightState: "FAIL",
			HostID:         "HOST-TSPI4-001",
			WireHostURL:    "https://tspi4.local:8444/g2s",
			OperatorNotes:  "second capture",
			PayloadJSON:    `{"session":{"overall_state":"ACTION_NEEDED"},"operator_notes":"second capture"}`,
		},
	}
	for _, record := range records {
		if _, err := auditStore.RecordSessionEvidence(ctx, record); err != nil {
			t.Fatalf("record session evidence: %v", err)
		}
	}

	handler := sessionEvidenceExportAllHandler(auditStore, config.Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/session-evidence/export-all", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET export-all status = %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type = %q, want application/json", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatalf("content-disposition = %q, want attachment", rec.Header().Get("Content-Disposition"))
	}

	var archive sessionEvidenceArchive
	if err := json.Unmarshal(rec.Body.Bytes(), &archive); err != nil {
		t.Fatalf("decode export-all payload: %v", err)
	}
	if archive.SummaryIndex.CaptureCount != 2 {
		t.Fatalf("capture_count = %d, want 2", archive.SummaryIndex.CaptureCount)
	}
	if len(archive.SummaryIndex.Captures) != 2 {
		t.Fatalf("summary captures len = %d, want 2", len(archive.SummaryIndex.Captures))
	}
	if len(archive.CaptureFiles) != 2 {
		t.Fatalf("capture files len = %d, want 2", len(archive.CaptureFiles))
	}
	if archive.CaptureFiles[0].JSONFileName == "" || archive.CaptureFiles[0].MarkdownFileName == "" {
		t.Fatalf("missing archive filenames in capture files")
	}
	if archive.CaptureFiles[0].MarkdownReport == "" {
		t.Fatalf("missing markdown report in export-all payload")
	}
}

func TestRunMarkersHandlerCRUDAndAuth(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		WebUI: config.WebUI{
			RequireLogin:                        false,
			AllowTrustedPrivateNetworkMutations: true,
		},
	}
	handler := runMarkersHandler(auditStore, cfg)

	payload := `{"created_at":"2026-05-20T22:00:00Z","marker_type":"start","title":"Cabinet session started","notes":"operator attached cabinet","host_id":"HOST-TSPI4-001","wire_host_url":"https://tspi4.local:8444/g2s","operator":"lab-ui"}`

	postReq := httptest.NewRequest(http.MethodPost, "/api/run-markers", bytes.NewBufferString(payload))
	postReq.RemoteAddr = "192.168.10.70:4555"
	postRec := httptest.NewRecorder()
	handler(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("POST trusted private network status = %d: %s", postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/run-markers?limit=10", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}
	var records []model.RunMarker
	if err := json.Unmarshal(getRec.Body.Bytes(), &records); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 run marker, got %d", len(records))
	}
	if records[0].MarkerType != "start" {
		t.Fatalf("marker_type = %q", records[0].MarkerType)
	}

	strictCfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
	}
	strictHandler := runMarkersHandler(auditStore, strictCfg)
	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/api/run-markers", bytes.NewBufferString(payload))
	unauthorizedReq.RemoteAddr = "198.51.100.40:4555"
	unauthorizedRec := httptest.NewRecorder()
	strictHandler(unauthorizedRec, unauthorizedReq)
	if !deniedByAuth(unauthorizedRec.Code) {
		t.Fatalf("POST public network without token status = %d, want 401/403", unauthorizedRec.Code)
	}
}

func TestOperatorDrillHandlerGETAndPOST(t *testing.T) {
	ctx := context.Background()
	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		WebUI: config.WebUI{
			RequireLogin:                        false,
			AllowTrustedPrivateNetworkMutations: true,
		},
		EGMRoster: []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}},
		Timeouts: config.Timeouts{
			EGMHeartbeatIntervalMS: 5000,
		},
	}
	eng := engine.New("G2S-MC-TEST", cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)

	manager := newOperatorDrillManager(eng, cfg.EGMRoster, cfg.Timeouts.EGMHeartbeatIntervalMS)
	t.Cleanup(func() { manager.shutdown(context.Background()) })
	handler := operatorDrillHandler(manager, cfg)

	getReq := httptest.NewRequest(http.MethodGet, "/api/operator-drill", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}
	var initial operatorDrillState
	if err := json.Unmarshal(getRec.Body.Bytes(), &initial); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if initial.SelectedEGMID != "EGM-01" {
		t.Fatalf("selected_egm_id = %q, want EGM-01", initial.SelectedEGMID)
	}
	if initial.IntervalMS != 5000 {
		t.Fatalf("interval_ms = %d, want 5000", initial.IntervalMS)
	}

	unauthorizedReq := httptest.NewRequest(http.MethodPost, "/api/operator-drill", bytes.NewBufferString(`{"action":"comms_online"}`))
	unauthorizedReq.RemoteAddr = "198.51.100.25:4000"
	unauthorizedRec := httptest.NewRecorder()
	handler(unauthorizedRec, unauthorizedReq)
	if !deniedByAuth(unauthorizedRec.Code) {
		t.Fatalf("POST public network without token status = %d, want 401/403", unauthorizedRec.Code)
	}

	privateReq := httptest.NewRequest(http.MethodPost, "/api/operator-drill", bytes.NewBufferString(`{"action":"comms_online"}`))
	privateReq.RemoteAddr = "192.168.10.70:4000"
	privateReq.Header.Set("Content-Type", "application/json")
	privateRec := httptest.NewRecorder()
	handler(privateRec, privateReq)
	if privateRec.Code != http.StatusOK {
		t.Fatalf("POST trusted private network status = %d: %s", privateRec.Code, privateRec.Body.String())
	}
	var privateBody operatorDrillState
	if err := json.Unmarshal(privateRec.Body.Bytes(), &privateBody); err != nil {
		t.Fatalf("decode POST: %v", err)
	}
	if privateBody.LastAction != "comms_online" {
		t.Fatalf("last_action = %q, want comms_online", privateBody.LastAction)
	}
	waitForLastEvent(t, eng, string(engine.EventG2SSessionOnline))

	resumeReq := httptest.NewRequest(http.MethodPost, "/api/operator-drill", bytes.NewBufferString(`{"action":"resume","interval_ms":50}`))
	resumeReq.RemoteAddr = "192.168.10.70:4000"
	resumeReq.Header.Set("Content-Type", "application/json")
	resumeRec := httptest.NewRecorder()
	handler(resumeRec, resumeReq)
	if resumeRec.Code != http.StatusOK {
		t.Fatalf("POST resume status = %d: %s", resumeRec.Code, resumeRec.Body.String())
	}
	var resumeBody operatorDrillState
	if err := json.Unmarshal(resumeRec.Body.Bytes(), &resumeBody); err != nil {
		t.Fatalf("decode resume: %v", err)
	}
	if !resumeBody.AutoHeartbeatRunning {
		t.Fatal("expected auto_heartbeat_running after resume")
	}

	pauseReq := httptest.NewRequest(http.MethodPost, "/api/operator-drill", bytes.NewBufferString(`{"action":"pause"}`))
	pauseReq.RemoteAddr = "192.168.10.70:4000"
	pauseReq.Header.Set("Content-Type", "application/json")
	pauseRec := httptest.NewRecorder()
	handler(pauseRec, pauseReq)
	if pauseRec.Code != http.StatusOK {
		t.Fatalf("POST pause status = %d: %s", pauseRec.Code, pauseRec.Body.String())
	}
	var pauseBody operatorDrillState
	if err := json.Unmarshal(pauseRec.Body.Bytes(), &pauseBody); err != nil {
		t.Fatalf("decode pause: %v", err)
	}
	if pauseBody.AutoHeartbeatRunning {
		t.Fatal("expected auto_heartbeat_running to be false after pause")
	}
	if !pauseBody.AutoHeartbeatPaused {
		t.Fatal("expected auto_heartbeat_paused to be true after pause")
	}
}

func TestRollupHeartbeatHistoryCollapsesConsecutiveKeepAliveBuckets(t *testing.T) {
	base := time.Date(2026, time.May, 22, 12, 0, 0, 0, time.UTC)
	history := []model.EGMStatusSnapshot{
		{EGMID: "EGM-02", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(7 * time.Second)},
		{EGMID: "EGM-02", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(6 * time.Second)},
		{EGMID: "EGM-01", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(5 * time.Second)},
		{EGMID: "EGM-01", Status: model.EGMGreen, EventType: "G2S_SESSION_ONLINE", CreatedAt: base.Add(4 * time.Second)},
		{EGMID: "EGM-01", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(3 * time.Second)},
		{EGMID: "EGM-01", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(2 * time.Second)},
		{EGMID: "EGM-01", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(1 * time.Second)},
	}

	rolled := rollupHeartbeatHistory(history)
	if len(rolled) != 4 {
		t.Fatalf("rolled len = %d, want 4", len(rolled))
	}

	if !rolled[0].HeartbeatRollup || rolled[0].HeartbeatRollupCount != 2 {
		t.Fatalf("bucket[0] rollup = %+v, want keepAlive x2", rolled[0])
	}
	if rolled[0].HeartbeatRollupFirstSeenAt == nil || !rolled[0].HeartbeatRollupFirstSeenAt.Equal(base.Add(6*time.Second)) {
		t.Fatalf("bucket[0] first_seen = %v, want %v", rolled[0].HeartbeatRollupFirstSeenAt, base.Add(6*time.Second))
	}
	if rolled[0].HeartbeatRollupLastSeenAt == nil || !rolled[0].HeartbeatRollupLastSeenAt.Equal(base.Add(7*time.Second)) {
		t.Fatalf("bucket[0] last_seen = %v, want %v", rolled[0].HeartbeatRollupLastSeenAt, base.Add(7*time.Second))
	}

	if !rolled[1].HeartbeatRollup || rolled[1].HeartbeatRollupCount != 1 {
		t.Fatalf("bucket[1] rollup = %+v, want keepAlive x1", rolled[1])
	}
	if rolled[2].EventType != "G2S_SESSION_ONLINE" || rolled[2].HeartbeatRollup {
		t.Fatalf("bucket[2] = %+v, want non-rollup session event", rolled[2])
	}
	if !rolled[3].HeartbeatRollup || rolled[3].HeartbeatRollupCount != 3 {
		t.Fatalf("bucket[3] rollup = %+v, want keepAlive x3", rolled[3])
	}
	if rolled[3].HeartbeatRollupFirstSeenAt == nil || !rolled[3].HeartbeatRollupFirstSeenAt.Equal(base.Add(1*time.Second)) {
		t.Fatalf("bucket[3] first_seen = %v, want %v", rolled[3].HeartbeatRollupFirstSeenAt, base.Add(1*time.Second))
	}
	if rolled[3].HeartbeatRollupLastSeenAt == nil || !rolled[3].HeartbeatRollupLastSeenAt.Equal(base.Add(3*time.Second)) {
		t.Fatalf("bucket[3] last_seen = %v, want %v", rolled[3].HeartbeatRollupLastSeenAt, base.Add(3*time.Second))
	}
}

func TestEGMHistoryHandlerRollupHeartbeatQueryParam(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	base := time.Date(2026, time.May, 22, 13, 0, 0, 0, time.UTC)
	rows := []model.EGMStatusSnapshot{
		{EGMID: "EGM-01", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(1 * time.Second)},
		{EGMID: "EGM-01", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(2 * time.Second)},
		{EGMID: "EGM-01", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(3 * time.Second)},
		{EGMID: "EGM-01", Status: model.EGMGreen, EventType: "G2S_SESSION_ONLINE", CreatedAt: base.Add(4 * time.Second)},
		{EGMID: "EGM-01", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(5 * time.Second)},
		{EGMID: "EGM-02", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(6 * time.Second)},
		{EGMID: "EGM-02", Status: model.EGMGreen, EventType: "G2S_KEEPALIVE", CreatedAt: base.Add(7 * time.Second)},
	}
	for _, row := range rows {
		if err := auditStore.RecordEGMStatus(ctx, row); err != nil {
			t.Fatalf("record status: %v", err)
		}
	}

	handler := egmHistoryHandler(auditStore)

	rawReq := httptest.NewRequest(http.MethodGet, "/api/egms/history?limit=20", nil)
	rawRec := httptest.NewRecorder()
	handler(rawRec, rawReq)
	if rawRec.Code != http.StatusOK {
		t.Fatalf("raw status = %d: %s", rawRec.Code, rawRec.Body.String())
	}
	var raw []model.EGMStatusSnapshot
	if err := json.Unmarshal(rawRec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw history: %v", err)
	}
	if len(raw) != len(rows) {
		t.Fatalf("raw len = %d, want %d", len(raw), len(rows))
	}

	rolledReq := httptest.NewRequest(http.MethodGet, "/api/egms/history?limit=20&rollup_heartbeat=true", nil)
	rolledRec := httptest.NewRecorder()
	handler(rolledRec, rolledReq)
	if rolledRec.Code != http.StatusOK {
		t.Fatalf("rolled status = %d: %s", rolledRec.Code, rolledRec.Body.String())
	}
	var rolled []model.EGMStatusSnapshot
	if err := json.Unmarshal(rolledRec.Body.Bytes(), &rolled); err != nil {
		t.Fatalf("decode rolled history: %v", err)
	}
	if len(rolled) != 4 {
		t.Fatalf("rolled len = %d, want 4", len(rolled))
	}
	if !rolled[0].HeartbeatRollup || rolled[0].HeartbeatRollupCount != 2 || rolled[0].EGMID != "EGM-02" {
		t.Fatalf("rolled[0] = %+v, want EGM-02 keepAlive x2 bucket", rolled[0])
	}
	if !rolled[1].HeartbeatRollup || rolled[1].HeartbeatRollupCount != 1 || rolled[1].EGMID != "EGM-01" {
		t.Fatalf("rolled[1] = %+v, want EGM-01 keepAlive x1 bucket", rolled[1])
	}
	if rolled[2].EventType != "G2S_SESSION_ONLINE" || rolled[2].HeartbeatRollup {
		t.Fatalf("rolled[2] = %+v, want non-rollup session row", rolled[2])
	}
	if !rolled[3].HeartbeatRollup || rolled[3].HeartbeatRollupCount != 3 || rolled[3].EGMID != "EGM-01" {
		t.Fatalf("rolled[3] = %+v, want EGM-01 keepAlive x3 bucket", rolled[3])
	}
}

func TestOperatorEntryHandlerRedirectsRootToOperator(t *testing.T) {
	handler := operatorEntryHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if location := rec.Header().Get("Location"); location != "/operator" {
		t.Fatalf("location = %q, want /operator", location)
	}
}

func TestOperatorEntryHandlerReturnsNotFoundForLegacyDashboardRoutes(t *testing.T) {
	handler := operatorEntryHandler()
	for _, path := range []string{"/dashboard", "/static/dashboard.css", "/static/dashboard.js", "/field-test"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want %d", path, rec.Code, http.StatusNotFound)
		}
	}
}

func TestParseDeliveryModeFlag(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    g2stransport.DeliveryMode
		wantErr bool
	}{
		{name: "default disabled", raw: "", want: g2stransport.DeliveryModeDisabled},
		{name: "disabled uppercase", raw: "DISABLED", want: g2stransport.DeliveryModeDisabled},
		{name: "http lowercase", raw: "http", want: g2stransport.DeliveryModeHTTP},
		{name: "invalid", raw: "dry-run", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDeliveryModeFlag(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDeliveryModeFlag(%q): %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("mode=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateInputRuntimeFlags(t *testing.T) {
	if err := validateInputRuntimeFlags(100*time.Millisecond, g2stransport.DeliveryModeDisabled, false, 5000); err != nil {
		t.Fatalf("expected valid flags: %v", err)
	}
	if err := validateInputRuntimeFlags(0, g2stransport.DeliveryModeDisabled, false, 5000); err == nil {
		t.Fatal("expected interval validation error")
	}
	if err := validateInputRuntimeFlags(100*time.Millisecond, g2stransport.DeliveryModeDisabled, false, -1); err == nil {
		t.Fatal("expected timeout validation error")
	}
	if err := validateInputRuntimeFlags(100*time.Millisecond, g2stransport.DeliveryModeDisabled, true, 5000); err == nil {
		t.Fatal("expected allow-delivery/mode validation error")
	}
}

func TestRuntimeOptionsFromConfigAndFlagsUsesConfigDefaults(t *testing.T) {
	cfg := config.Config{
		Timeouts: config.Timeouts{G2SRequestTimeoutMS: 7000},
		Runtime: config.Runtime{
			InputRuntimeEnabled:        true,
			InputRuntimeSeedDefaults:   true,
			InputRuntimeExecuteActions: true,
			InputRuntimeIntervalMS:     125,
			DeliveryTopology:           "HOST_LISTENER",
		},
	}
	options, topology, err := runtimeOptionsFromConfigAndFlags(cfg, map[string]bool{}, runtimeFlagValues{})
	if err != nil {
		t.Fatalf("runtimeOptionsFromConfigAndFlags: %v", err)
	}
	if !options.Enabled || !options.SeedDefaultInputs || !options.ExecuteActions {
		t.Fatalf("runtime booleans did not honor config defaults: %+v", options)
	}
	if options.PollInterval != 125*time.Millisecond {
		t.Fatalf("poll interval = %s, want 125ms", options.PollInterval)
	}
	if topology != g2stransport.DeliveryTopologyHostListener {
		t.Fatalf("topology = %q, want HOST_LISTENER", topology)
	}
	if options.DeliverySettings.Mode != g2stransport.DeliveryModeDisabled || options.DeliverySettings.AllowDelivery {
		t.Fatalf("delivery defaults changed unexpectedly: %+v", options.DeliverySettings)
	}
	if options.DeliverySettings.TimeoutMS != 7000 {
		t.Fatalf("delivery timeout = %d, want 7000", options.DeliverySettings.TimeoutMS)
	}
}

func TestRuntimeOptionsFromConfigAndFlagsExplicitFlagsOverrideConfig(t *testing.T) {
	cfg := config.Config{
		Timeouts: config.Timeouts{G2SRequestTimeoutMS: 5000},
		Runtime: config.Runtime{
			InputRuntimeEnabled:        false,
			InputRuntimeSeedDefaults:   false,
			InputRuntimeExecuteActions: false,
			InputRuntimeIntervalMS:     100,
			DeliveryTopology:           "HOST_LISTENER",
		},
	}
	explicit := map[string]bool{
		"input-runtime":                 true,
		"input-runtime-seed-defaults":   true,
		"input-runtime-execute-actions": true,
		"input-runtime-interval":        true,
		"delivery-mode":                 true,
		"delivery-topology":             true,
		"allow-delivery":                true,
		"delivery-timeout-ms":           true,
	}
	options, topology, err := runtimeOptionsFromConfigAndFlags(cfg, explicit, runtimeFlagValues{
		InputRuntimeEnabled:        true,
		InputRuntimeSeedDefaults:   true,
		InputRuntimeExecuteActions: true,
		InputRuntimeInterval:       225 * time.Millisecond,
		DeliveryModeRaw:            "http",
		DeliveryTopologyRaw:        "OUTBOUND_ENDPOINT",
		AllowDelivery:              true,
		DeliveryTimeoutMS:          9000,
	})
	if err != nil {
		t.Fatalf("runtimeOptionsFromConfigAndFlags: %v", err)
	}
	if !options.Enabled || !options.SeedDefaultInputs || !options.ExecuteActions {
		t.Fatalf("runtime booleans did not honor explicit flags: %+v", options)
	}
	if options.PollInterval != 225*time.Millisecond {
		t.Fatalf("poll interval = %s, want 225ms", options.PollInterval)
	}
	if topology != g2stransport.DeliveryTopologyOutboundEndpoint {
		t.Fatalf("topology = %q, want OUTBOUND_ENDPOINT", topology)
	}
	if options.DeliverySettings.Mode != g2stransport.DeliveryModeHTTP || !options.DeliverySettings.AllowDelivery {
		t.Fatalf("delivery settings did not honor explicit flags: %+v", options.DeliverySettings)
	}
	if options.DeliverySettings.TimeoutMS != 9000 {
		t.Fatalf("delivery timeout = %d, want 9000", options.DeliverySettings.TimeoutMS)
	}
}

func TestSystemdUnitDoesNotEncodeHiddenRuntimeFlags(t *testing.T) {
	unitPath := filepath.Join("..", "..", "packaging", "systemd", "g2s-mute.service")
	raw, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("read systemd unit: %v", err)
	}
	text := string(raw)
	for _, forbidden := range []string{
		"-input-runtime",
		"-input-runtime-seed-defaults",
		"-input-runtime-interval",
		"-input-runtime-execute-actions",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("systemd unit still contains hidden runtime flag %q", forbidden)
		}
	}
	if !strings.Contains(text, "-config /etc/g2s-mute/config.json") {
		t.Fatalf("systemd unit missing config path")
	}
}

func TestPendingDeliverySweepOptionsFromConfigEnabled(t *testing.T) {
	cfg := config.Config{
		Runtime: config.Runtime{
			PendingDeliverySweepEnabled:    true,
			PendingDeliverySweepIntervalMS: 2200,
		},
	}
	options := pendingDeliverySweepOptionsFromConfig(cfg)
	if !options.Enabled {
		t.Fatalf("enabled=%t, want true", options.Enabled)
	}
	if options.Interval != 2200*time.Millisecond {
		t.Fatalf("interval=%s, want 2200ms", options.Interval)
	}
}

func TestPendingDeliverySweepOptionsFromConfigDefaultsInterval(t *testing.T) {
	cfg := config.Config{
		Runtime: config.Runtime{
			PendingDeliverySweepEnabled:    false,
			PendingDeliverySweepIntervalMS: 0,
		},
	}
	options := pendingDeliverySweepOptionsFromConfig(cfg)
	if options.Enabled {
		t.Fatalf("enabled=%t, want false", options.Enabled)
	}
	if options.Interval != pendingdeliveryruntime.DefaultSweepInterval {
		t.Fatalf("interval=%s, want %s", options.Interval, pendingdeliveryruntime.DefaultSweepInterval)
	}
}
