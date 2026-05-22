package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func setupBlockerPolicyRuntime(t *testing.T, withToken bool) (context.Context, *store.SQLiteStore, *engine.Engine, config.Config, runtimeInfo) {
	t.Helper()
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		ControllerID: "G2S-MC-BLOCKER-POLICY",
		Database:     config.Database{Path: ":memory:"},
		WebUI:        config.WebUI{BindAddress: "127.0.0.1:8444"},
		G2S: config.G2S{
			HostURL:      "http://127.0.0.1:8444/g2s",
			EndpointPath: "/g2s",
			RequireTLS:   false,
		},
		CabinetProfile: config.CabinetProfile{
			WireHostURL:     "https://lab-cabinet.local:8444/g2s",
			ListenerDNSName: "lab-cabinet.local",
			RequiredSANDNS:  []string{"lab-cabinet.local"},
			HostID:          "HOST-LAB-1001",
			FirstTestEGMIDs: []string{},
		},
		EGMRoster: []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}},
		BlockerPolicy: config.BlockerPolicy{
			ApprovedBlockerIDs: []string{"service_readiness"},
		},
	}
	if withToken {
		cfg.API.AuthToken = "lab-secret"
	}

	eng := engine.New(cfg.ControllerID, cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	eng.Start(runCtx)
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now()})
	waitForLastEvent(t, eng, string(engine.EventBootComplete))

	runtime := runtimeInfo{
		ConfigPath: "/etc/g2s-mute/config.json",
		StartedAt:  time.Now().Add(-10 * time.Second),
	}
	return ctx, auditStore, eng, cfg, runtime
}

func TestBlockerPolicyHandlerCRUD(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		BlockerPolicy: config.BlockerPolicy{
			ApprovedBlockerIDs: []string{
				"service_readiness",
				"cabinet_profile",
			},
		},
	}
	handler := blockerPolicyHandler(auditStore, cfg)

	getReq := httptest.NewRequest(http.MethodGet, "/api/blocker-policy", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}
	var getBody blockerPolicyResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if getBody.PolicySource != blockerPolicySourceFile {
		t.Fatalf("GET policy_source = %q, want %q", getBody.PolicySource, blockerPolicySourceFile)
	}
	wantInitial := []string{"cabinet_profile", "service_readiness"}
	if !reflect.DeepEqual(getBody.Effective.ApprovedBlockerIDs, wantInitial) {
		t.Fatalf("GET approved_blocker_ids = %#v, want %#v", getBody.Effective.ApprovedBlockerIDs, wantInitial)
	}
	if len(getBody.EscalationHistory) != 0 {
		t.Fatalf("expected empty escalation history, got %+v", getBody.EscalationHistory)
	}

	putRaw := []byte(`{"approved_blocker_ids":["service_readiness"]}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/blocker-policy", bytes.NewReader(putRaw))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("X-Operator", "tester")
	putRec := httptest.NewRecorder()
	handler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d: %s", putRec.Code, putRec.Body.String())
	}
	var putBody blockerPolicyResponse
	if err := json.Unmarshal(putRec.Body.Bytes(), &putBody); err != nil {
		t.Fatalf("decode PUT: %v", err)
	}
	if putBody.PolicySource != blockerPolicySourceOverride {
		t.Fatalf("PUT policy_source = %q, want %q", putBody.PolicySource, blockerPolicySourceOverride)
	}
	if !putBody.OverridePresent {
		t.Fatal("expected override_present on PUT")
	}
	wantOverride := []string{"service_readiness"}
	if !reflect.DeepEqual(putBody.Effective.ApprovedBlockerIDs, wantOverride) {
		t.Fatalf("PUT approved_blocker_ids = %#v, want %#v", putBody.Effective.ApprovedBlockerIDs, wantOverride)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/blocker-policy", nil)
	deleteRec := httptest.NewRecorder()
	handler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	var deleteBody blockerPolicyResponse
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &deleteBody); err != nil {
		t.Fatalf("decode DELETE: %v", err)
	}
	if deleteBody.PolicySource != blockerPolicySourceFile {
		t.Fatalf("DELETE policy_source = %q, want %q", deleteBody.PolicySource, blockerPolicySourceFile)
	}
	if deleteBody.OverridePresent {
		t.Fatalf("DELETE override_present = true, want false")
	}
	if !reflect.DeepEqual(deleteBody.Effective.ApprovedBlockerIDs, wantInitial) {
		t.Fatalf("DELETE approved_blocker_ids = %#v, want %#v", deleteBody.Effective.ApprovedBlockerIDs, wantInitial)
	}
}

func TestBlockerPolicySuggestionsHandler(t *testing.T) {
	_, auditStore, eng, cfg, runtime := setupBlockerPolicyRuntime(t, false)
	handler := blockerPolicySuggestionsHandler(eng, auditStore, cfg, runtime)

	req := httptest.NewRequest(http.MethodGet, "/api/blocker-policy/suggestions", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET suggestions status = %d: %s", rec.Code, rec.Body.String())
	}
	var body blockerPolicySuggestionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode suggestions: %v", err)
	}
	if body.GeneratedAt.IsZero() {
		t.Fatal("expected generated_at")
	}
	foundCabinetProfile := false
	for _, row := range body.Suggestions {
		if row.FindingID == "cabinet_profile" {
			foundCabinetProfile = true
			if !row.DowngradedByPolicy {
				t.Fatalf("expected cabinet_profile suggestion downgraded_by_policy=true")
			}
		}
	}
	if !foundCabinetProfile {
		t.Fatalf("expected cabinet_profile in suggestions: %+v", body.Suggestions)
	}
}

func TestBlockerPolicySuggestionsHandlerAllowsGETWithoutToken(t *testing.T) {
	_, auditStore, eng, cfg, runtime := setupBlockerPolicyRuntime(t, true)
	handler := blockerPolicySuggestionsHandler(eng, auditStore, cfg, runtime)

	req := httptest.NewRequest(http.MethodGet, "/api/blocker-policy/suggestions", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET suggestions without token status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestBlockerPolicyApproveAndRevokeWorkflow(t *testing.T) {
	_, auditStore, eng, cfg, runtime := setupBlockerPolicyRuntime(t, true)

	approveHandler := requireMutationAuthForMethods(
		blockerPolicyApproveHandler(eng, auditStore, cfg, runtime),
		cfg,
		http.MethodPost,
	)
	revokeHandler := requireMutationAuthForMethods(
		blockerPolicyRevokeHandler(auditStore, cfg),
		cfg,
		http.MethodPost,
	)
	getPolicyHandler := blockerPolicyHandler(auditStore, cfg)

	unauthReq := httptest.NewRequest(http.MethodPost, "/api/blocker-policy/approve", bytes.NewBufferString(`{"finding_id":"cabinet_profile","rationale":"needed"}`))
	unauthReq.Header.Set("Content-Type", "application/json")
	unauthRec := httptest.NewRecorder()
	approveHandler(unauthRec, unauthReq)
	if !deniedByAuth(unauthRec.Code) {
		t.Fatalf("approve without token status = %d, want 401/403", unauthRec.Code)
	}

	missingRationaleReq := httptest.NewRequest(http.MethodPost, "/api/blocker-policy/approve", bytes.NewBufferString(`{"finding_id":"cabinet_profile"}`))
	missingRationaleReq.Header.Set("Content-Type", "application/json")
	missingRationaleReq.Header.Set("Authorization", "Bearer lab-secret")
	missingRationaleRec := httptest.NewRecorder()
	approveHandler(missingRationaleRec, missingRationaleReq)
	if missingRationaleRec.Code != http.StatusBadRequest {
		t.Fatalf("approve missing rationale status = %d, want 400: %s", missingRationaleRec.Code, missingRationaleRec.Body.String())
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/blocker-policy/approve", bytes.NewBufferString(`{"finding_id":"cabinet_profile","rationale":"required for controlled cabinet rollout"}`))
	approveReq.Header.Set("Content-Type", "application/json")
	approveReq.Header.Set("Authorization", "Bearer lab-secret")
	approveReq.Header.Set("X-Operator", "op-a")
	approveRec := httptest.NewRecorder()
	approveHandler(approveRec, approveReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve status = %d: %s", approveRec.Code, approveRec.Body.String())
	}
	var approveBody blockerPolicyResponse
	if err := json.Unmarshal(approveRec.Body.Bytes(), &approveBody); err != nil {
		t.Fatalf("decode approve body: %v", err)
	}
	if !containsString(approveBody.Effective.ApprovedBlockerIDs, "cabinet_profile") {
		t.Fatalf("expected cabinet_profile approved, got %#v", approveBody.Effective.ApprovedBlockerIDs)
	}

	unauthRevokeReq := httptest.NewRequest(http.MethodPost, "/api/blocker-policy/revoke", bytes.NewBufferString(`{"finding_id":"cabinet_profile"}`))
	unauthRevokeReq.Header.Set("Content-Type", "application/json")
	unauthRevokeRec := httptest.NewRecorder()
	revokeHandler(unauthRevokeRec, unauthRevokeReq)
	if !deniedByAuth(unauthRevokeRec.Code) {
		t.Fatalf("revoke without token status = %d, want 401/403", unauthRevokeRec.Code)
	}

	revokeReq := httptest.NewRequest(http.MethodPost, "/api/blocker-policy/revoke", bytes.NewBufferString(`{"finding_id":"cabinet_profile","rationale":"lab-only exception removed"}`))
	revokeReq.Header.Set("Content-Type", "application/json")
	revokeReq.Header.Set("Authorization", "Bearer lab-secret")
	revokeReq.Header.Set("X-Operator", "op-b")
	revokeRec := httptest.NewRecorder()
	revokeHandler(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d: %s", revokeRec.Code, revokeRec.Body.String())
	}
	var revokeBody blockerPolicyResponse
	if err := json.Unmarshal(revokeRec.Body.Bytes(), &revokeBody); err != nil {
		t.Fatalf("decode revoke body: %v", err)
	}
	if containsString(revokeBody.Effective.ApprovedBlockerIDs, "cabinet_profile") {
		t.Fatalf("expected cabinet_profile revoked, got %#v", revokeBody.Effective.ApprovedBlockerIDs)
	}

	policyReq := httptest.NewRequest(http.MethodGet, "/api/blocker-policy", nil)
	policyRec := httptest.NewRecorder()
	getPolicyHandler(policyRec, policyReq)
	if policyRec.Code != http.StatusOK {
		t.Fatalf("policy GET status = %d: %s", policyRec.Code, policyRec.Body.String())
	}
	var policyBody blockerPolicyResponse
	if err := json.Unmarshal(policyRec.Body.Bytes(), &policyBody); err != nil {
		t.Fatalf("decode policy body: %v", err)
	}
	if len(policyBody.EscalationHistory) < 2 {
		t.Fatalf("expected escalation history entries, got %+v", policyBody.EscalationHistory)
	}
	if policyBody.EscalationHistory[0].Action != blockerPolicyActionRevoke || policyBody.EscalationHistory[1].Action != blockerPolicyActionApprove {
		t.Fatalf("unexpected escalation history order: %+v", policyBody.EscalationHistory)
	}
}

func TestBlockerPolicyApproveValidation(t *testing.T) {
	_, auditStore, eng, cfg, runtime := setupBlockerPolicyRuntime(t, true)
	handler := requireMutationAuthForMethods(
		blockerPolicyApproveHandler(eng, auditStore, cfg, runtime),
		cfg,
		http.MethodPost,
	)

	tests := []struct {
		name string
		body string
	}{
		{name: "missing finding id", body: `{"rationale":"needed"}`},
		{name: "invalid finding id", body: `{"finding_id":"CABINET_PROFILE","rationale":"needed"}`},
		{name: "not current fail", body: `{"finding_id":"profile_source","rationale":"needed"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/blocker-policy/approve", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer lab-secret")
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestBlockerPolicyRouteMutationAuthTokenGuard(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		BlockerPolicy: config.BlockerPolicy{
			ApprovedBlockerIDs: []string{"service_readiness", "cabinet_profile"},
		},
	}
	handler := requireMutationAuthForMethods(
		blockerPolicyHandler(auditStore, cfg),
		cfg,
		http.MethodPut,
		http.MethodDelete,
	)

	getReq := httptest.NewRequest(http.MethodGet, "/api/blocker-policy", nil)
	getRec := httptest.NewRecorder()
	handler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}

	unauthorizedPut := httptest.NewRequest(http.MethodPut, "/api/blocker-policy", bytes.NewBufferString(`{"approved_blocker_ids":["service_readiness"]}`))
	unauthorizedPut.Header.Set("Content-Type", "application/json")
	unauthorizedPutRec := httptest.NewRecorder()
	handler(unauthorizedPutRec, unauthorizedPut)
	if !deniedByAuth(unauthorizedPutRec.Code) {
		t.Fatalf("PUT without token status = %d, want 401/403", unauthorizedPutRec.Code)
	}

	authorizedPut := httptest.NewRequest(http.MethodPut, "/api/blocker-policy", bytes.NewBufferString(`{"approved_blocker_ids":["service_readiness"]}`))
	authorizedPut.Header.Set("Content-Type", "application/json")
	authorizedPut.Header.Set("Authorization", "Bearer lab-secret")
	authorizedPutRec := httptest.NewRecorder()
	handler(authorizedPutRec, authorizedPut)
	if authorizedPutRec.Code != http.StatusOK {
		t.Fatalf("PUT with token status = %d: %s", authorizedPutRec.Code, authorizedPutRec.Body.String())
	}

	unauthorizedDelete := httptest.NewRequest(http.MethodDelete, "/api/blocker-policy", nil)
	unauthorizedDeleteRec := httptest.NewRecorder()
	handler(unauthorizedDeleteRec, unauthorizedDelete)
	if !deniedByAuth(unauthorizedDeleteRec.Code) {
		t.Fatalf("DELETE without token status = %d, want 401/403", unauthorizedDeleteRec.Code)
	}

	authorizedDelete := httptest.NewRequest(http.MethodDelete, "/api/blocker-policy", nil)
	authorizedDelete.Header.Set("Authorization", "Bearer lab-secret")
	authorizedDeleteRec := httptest.NewRecorder()
	handler(authorizedDeleteRec, authorizedDelete)
	if authorizedDeleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE with token status = %d: %s", authorizedDeleteRec.Code, authorizedDeleteRec.Body.String())
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
