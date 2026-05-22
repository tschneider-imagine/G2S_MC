package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestOperatorAuditHandlerListFilterAndLimit(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	now := time.Now().UTC()
	seed := []model.OperatorAuditEvent{
		{
			Timestamp:  now.Add(-2 * time.Minute),
			Action:     "cabinet_profile.save",
			Result:     "success",
			ActorScope: "token",
			EGMFocus:   "EGM-01",
			Summary:    "Cabinet profile override saved",
			Detail:     "host_id changed",
		},
		{
			Timestamp:  now.Add(-1 * time.Minute),
			Action:     "heartbeat_policy.save",
			Result:     "fail",
			ActorScope: "trusted",
			Summary:    "Heartbeat policy save failed",
			Detail:     "interval_ms must be greater than zero",
		},
		{
			Timestamp:  now,
			Action:     "session_workflow.save",
			Result:     "success",
			ActorScope: "local",
			Summary:    "Session workflow progress saved",
			Detail:     "current_phase=run_active",
		},
	}
	for _, row := range seed {
		if _, err := auditStore.RecordOperatorAuditEvent(ctx, row); err != nil {
			t.Fatalf("record operator audit seed: %v", err)
		}
	}

	handler := operatorAuditHandler(auditStore)

	limitReq := httptest.NewRequest(http.MethodGet, "/api/operator-audit?limit=2", nil)
	limitRec := httptest.NewRecorder()
	handler(limitRec, limitReq)
	if limitRec.Code != http.StatusOK {
		t.Fatalf("limit status = %d: %s", limitRec.Code, limitRec.Body.String())
	}
	var limitRows []model.OperatorAuditEvent
	if err := json.Unmarshal(limitRec.Body.Bytes(), &limitRows); err != nil {
		t.Fatalf("decode limit rows: %v", err)
	}
	if len(limitRows) != 2 {
		t.Fatalf("limit rows len = %d, want 2", len(limitRows))
	}
	if limitRows[0].Action != "session_workflow.save" {
		t.Fatalf("latest action = %q, want session_workflow.save", limitRows[0].Action)
	}

	actionReq := httptest.NewRequest(http.MethodGet, "/api/operator-audit?action=heartbeat_policy.save", nil)
	actionRec := httptest.NewRecorder()
	handler(actionRec, actionReq)
	if actionRec.Code != http.StatusOK {
		t.Fatalf("action filter status = %d: %s", actionRec.Code, actionRec.Body.String())
	}
	var actionRows []model.OperatorAuditEvent
	if err := json.Unmarshal(actionRec.Body.Bytes(), &actionRows); err != nil {
		t.Fatalf("decode action rows: %v", err)
	}
	if len(actionRows) != 1 || actionRows[0].Result != "fail" {
		t.Fatalf("unexpected action rows: %+v", actionRows)
	}

	resultReq := httptest.NewRequest(http.MethodGet, "/api/operator-audit?result=success", nil)
	resultRec := httptest.NewRecorder()
	handler(resultRec, resultReq)
	if resultRec.Code != http.StatusOK {
		t.Fatalf("result filter status = %d: %s", resultRec.Code, resultRec.Body.String())
	}
	var resultRows []model.OperatorAuditEvent
	if err := json.Unmarshal(resultRec.Body.Bytes(), &resultRows); err != nil {
		t.Fatalf("decode result rows: %v", err)
	}
	if len(resultRows) != 2 {
		t.Fatalf("result rows len = %d, want 2", len(resultRows))
	}

	searchReq := httptest.NewRequest(http.MethodGet, "/api/operator-audit?q=interval_ms", nil)
	searchRec := httptest.NewRecorder()
	handler(searchRec, searchReq)
	if searchRec.Code != http.StatusOK {
		t.Fatalf("search filter status = %d: %s", searchRec.Code, searchRec.Body.String())
	}
	var searchRows []model.OperatorAuditEvent
	if err := json.Unmarshal(searchRec.Body.Bytes(), &searchRows); err != nil {
		t.Fatalf("decode search rows: %v", err)
	}
	if len(searchRows) != 1 || searchRows[0].Action != "heartbeat_policy.save" {
		t.Fatalf("unexpected search rows: %+v", searchRows)
	}

	invalidResultReq := httptest.NewRequest(http.MethodGet, "/api/operator-audit?result=maybe", nil)
	invalidResultRec := httptest.NewRecorder()
	handler(invalidResultRec, invalidResultReq)
	if invalidResultRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid result status = %d, want 400", invalidResultRec.Code)
	}
}

func TestOperatorAuditSensitiveActionRecordingNonCertificatePaths(t *testing.T) {
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
		Timeouts: config.Timeouts{
			EGMHeartbeatIntervalMS:         5000,
			EGMHeartbeatWarningAfterMissed: 3,
			EGMHeartbeatBlockAfterMissed:   6,
		},
		Crypto: config.Crypto{
			G2SCAPath: filepath.Join(t.TempDir(), "ca.crt"),
		},
	}

	cabinetHandler := cabinetProfileHandler(auditStore, cfg)
	overrideRaw := `{"wire_host_url":"https://override.example/g2s","listener_dns_name":"override.example","listener_ip":"10.20.30.40","required_san_dns":["override.example"],"required_san_ips":["10.20.30.40"],"host_id":"HOST-OVERRIDE","first_test_egm_ids":["EGM-99"]}`
	cabPutReq := httptest.NewRequest(http.MethodPut, "/api/cabinet-profile", bytes.NewBufferString(overrideRaw))
	cabPutReq.Header.Set("Content-Type", "application/json")
	cabPutReq.Header.Set("X-EGM-Focus", "EGM-99")
	cabPutRec := httptest.NewRecorder()
	cabinetHandler(cabPutRec, cabPutReq)
	if cabPutRec.Code != http.StatusOK {
		t.Fatalf("cabinet PUT status = %d: %s", cabPutRec.Code, cabPutRec.Body.String())
	}
	cabDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/cabinet-profile", nil)
	cabDeleteRec := httptest.NewRecorder()
	cabinetHandler(cabDeleteRec, cabDeleteReq)
	if cabDeleteRec.Code != http.StatusOK {
		t.Fatalf("cabinet DELETE status = %d: %s", cabDeleteRec.Code, cabDeleteRec.Body.String())
	}

	heartbeatHandler := heartbeatPolicyHandler(auditStore, cfg)
	hbPutReq := httptest.NewRequest(http.MethodPut, "/api/heartbeat-policy", bytes.NewBufferString(`{"interval_ms":6000,"warning_after_missed":4,"block_after_missed":8}`))
	hbPutReq.Header.Set("Content-Type", "application/json")
	hbPutRec := httptest.NewRecorder()
	heartbeatHandler(hbPutRec, hbPutReq)
	if hbPutRec.Code != http.StatusOK {
		t.Fatalf("heartbeat PUT status = %d: %s", hbPutRec.Code, hbPutRec.Body.String())
	}
	hbDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/heartbeat-policy", nil)
	hbDeleteRec := httptest.NewRecorder()
	heartbeatHandler(hbDeleteRec, hbDeleteReq)
	if hbDeleteRec.Code != http.StatusOK {
		t.Fatalf("heartbeat DELETE status = %d: %s", hbDeleteRec.Code, hbDeleteRec.Body.String())
	}

	workflowHandler := sessionWorkflowHandler(auditStore, cfg)
	wfPutReq := httptest.NewRequest(http.MethodPut, "/api/session-workflow", bytes.NewBufferString(`{"current_phase":"run_active","completed_steps":["pre_check","connect_observe"],"operator_notes":"tracking run"}`))
	wfPutReq.Header.Set("Content-Type", "application/json")
	wfPutRec := httptest.NewRecorder()
	workflowHandler(wfPutRec, wfPutReq)
	if wfPutRec.Code != http.StatusOK {
		t.Fatalf("workflow PUT status = %d: %s", wfPutRec.Code, wfPutRec.Body.String())
	}
	wfDeleteReq := httptest.NewRequest(http.MethodDelete, "/api/session-workflow", nil)
	wfDeleteRec := httptest.NewRecorder()
	workflowHandler(wfDeleteRec, wfDeleteReq)
	if wfDeleteRec.Code != http.StatusOK {
		t.Fatalf("workflow DELETE status = %d: %s", wfDeleteRec.Code, wfDeleteRec.Body.String())
	}

	evidenceHandler := sessionEvidenceHandler(auditStore, cfg)
	payload := `{"captured_at":"2026-05-20T21:00:00Z","operator_notes":"clean run","session":{"overall_state":"LAB_READY","readyz_state":"READY_LAB","preflight_state":"PASS"},"cabinet_profile":{"host_id":"HOST-TSPI4-001","wire_host_url":"https://tspi4.local:8444/g2s"}}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/session-evidence", bytes.NewBufferString(payload))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	evidenceHandler(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("session evidence POST status = %d: %s", postRec.Code, postRec.Body.String())
	}
	deleteByIDHandler := sessionEvidenceByIDHandler(auditStore, cfg)
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/session-evidence/1", nil)
	deleteRec := httptest.NewRecorder()
	deleteByIDHandler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("session evidence by-id DELETE status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	exportAllHandler := sessionEvidenceExportAllHandler(auditStore, cfg)
	exportReq := httptest.NewRequest(http.MethodGet, "/api/session-evidence/export-all", nil)
	exportRec := httptest.NewRecorder()
	exportAllHandler(exportRec, exportReq)
	if exportRec.Code != http.StatusOK {
		t.Fatalf("session evidence export-all status = %d: %s", exportRec.Code, exportRec.Body.String())
	}

	events, err := auditStore.ListOperatorAuditEvents(ctx, model.OperatorAuditQuery{Limit: 100})
	if err != nil {
		t.Fatalf("list operator audit events: %v", err)
	}
	actions := map[string]bool{}
	focusSeen := false
	for _, row := range events {
		actions[row.Action] = true
		if row.Action == "cabinet_profile.save" && row.EGMFocus == "EGM-99" {
			focusSeen = true
		}
	}
	for _, action := range []string{
		"cabinet_profile.save",
		"cabinet_profile.clear",
		"heartbeat_policy.save",
		"heartbeat_policy.clear",
		"session_workflow.save",
		"session_workflow.clear",
		"session_evidence.delete",
		"session_evidence.export_all",
	} {
		if !actions[action] {
			t.Fatalf("expected audit action %q to be recorded; actions=%v", action, actions)
		}
	}
	if !focusSeen {
		t.Fatalf("expected cabinet_profile.save event to include egm_focus header")
	}

	failRows, err := auditStore.ListOperatorAuditEvents(ctx, model.OperatorAuditQuery{
		Limit:  10,
		Result: "fail",
	})
	if err != nil {
		t.Fatalf("list fail events: %v", err)
	}
	for _, row := range failRows {
		if strings.TrimSpace(row.Result) != "fail" {
			t.Fatalf("unexpected fail filter row result=%q", row.Result)
		}
	}
}
