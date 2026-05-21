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
			name: "no EGMs remains DEGRADED even when TLS is disabled",
			cfg:  baseCfg,
			snapshot: engine.Snapshot{
				State: model.StateHealthy,
				EGMs:  []model.EGM{},
			},
			wantOverall: "DEGRADED",
			wantIssue:   "no EGMs configured",
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
				return
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

func TestHeartbeatPolicyHandlerCRUD(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		Timeouts: config.Timeouts{
			EGMHeartbeatIntervalMS:          5000,
			EGMHeartbeatWarningAfterMissed:  3,
			EGMHeartbeatBlockAfterMissed:    6,
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
	if getBody.Effective.WarningAfterMissed != 3 || getBody.Effective.BlockAfterMissed != 6 {
		t.Fatalf("unexpected GET effective policy: %+v", getBody.Effective)
	}

	raw := []byte(`{"warning_after_missed":4,"block_after_missed":9}`)
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
	if putBody.Effective.WarningAfterMissed != 4 || putBody.Effective.BlockAfterMissed != 9 {
		t.Fatalf("unexpected PUT effective policy: %+v", putBody.Effective)
	}
	if !putBody.OverridePresent {
		t.Fatal("expected override_present on PUT")
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
