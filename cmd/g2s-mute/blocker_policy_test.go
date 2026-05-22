package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

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

func TestBlockerPolicyHandlerValidation(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	handler := blockerPolicyHandler(auditStore, config.Config{})

	tests := []struct {
		name string
		body string
	}{
		{
			name: "empty id",
			body: `{"approved_blocker_ids":[""]}`,
		},
		{
			name: "invalid id format",
			body: `{"approved_blocker_ids":["CABINET_PROFILE"]}`,
		},
		{
			name: "duplicate id",
			body: `{"approved_blocker_ids":["service_readiness","service_readiness"]}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/blocker-policy", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
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
