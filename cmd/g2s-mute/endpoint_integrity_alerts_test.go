package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestEndpointIntegrityAlertsLifecycleAndAuth(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		EGMRoster: []config.EGM{
			{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443},
			{EGMID: "EGM-02", IPAddress: "127.0.0.1", Port: 9444},
		},
	}
	eng := engine.New("G2S-MC-ENDPOINT-INTEGRITY", cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)
	now := time.Now().UTC()
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: now})
	eng.Submit(engine.Event{
		Type:       engine.EventKeepAlive,
		EGMID:      "EGM-01",
		At:         now.Add(1 * time.Second),
		SourceIP:   "10.20.30.55",
		SourcePort: 9550,
	})
	eng.Submit(engine.Event{
		Type:       engine.EventKeepAlive,
		EGMID:      "EGM-02",
		At:         now.Add(2 * time.Second),
		SourceIP:   "10.20.30.55",
		SourcePort: 9550,
	})
	waitForLastEvent(t, eng, string(engine.EventKeepAlive))

	listHandler := endpointIntegrityAlertsHandler(eng, auditStore, cfg)
	getReq := httptest.NewRequest(http.MethodGet, "/api/endpoint-integrity/alerts", nil)
	getRec := httptest.NewRecorder()
	listHandler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET alerts status = %d: %s", getRec.Code, getRec.Body.String())
	}

	var listBody endpointIntegrityAlertsResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode GET alerts: %v", err)
	}
	if listBody.Summary.Total == 0 {
		t.Fatalf("expected at least one endpoint integrity alert")
	}
	if len(listBody.ActiveAlerts) == 0 {
		t.Fatalf("expected active alerts in initial response")
	}
	alert := listBody.ActiveAlerts[0]
	if alert.Severity == "" {
		t.Fatalf("expected severity for alert")
	}

	actionHandler := requireMutationAuthForMethods(
		endpointIntegrityAlertActionHandler(eng, auditStore, cfg),
		cfg,
		http.MethodPost,
	)

	unauthorizedAckReq := httptest.NewRequest(http.MethodPost, "/api/endpoint-integrity/alerts/"+alert.ID+"/ack", nil)
	unauthorizedAckRec := httptest.NewRecorder()
	actionHandler(unauthorizedAckRec, unauthorizedAckReq)
	if !deniedByAuth(unauthorizedAckRec.Code) {
		t.Fatalf("ack without token status = %d, want 401/403", unauthorizedAckRec.Code)
	}

	authorizedAckReq := httptest.NewRequest(http.MethodPost, "/api/endpoint-integrity/alerts/"+alert.ID+"/ack", nil)
	authorizedAckReq.Header.Set("Authorization", "Bearer lab-secret")
	authorizedAckRec := httptest.NewRecorder()
	actionHandler(authorizedAckRec, authorizedAckReq)
	if authorizedAckRec.Code != http.StatusOK {
		t.Fatalf("ack with token status = %d: %s", authorizedAckRec.Code, authorizedAckRec.Body.String())
	}
	var ackBody endpointIntegrityAlertsResponse
	if err := json.Unmarshal(authorizedAckRec.Body.Bytes(), &ackBody); err != nil {
		t.Fatalf("decode ack response: %v", err)
	}
	if ackBody.Summary.AckedCount != 1 {
		t.Fatalf("acked_count after ack = %d, want 1", ackBody.Summary.AckedCount)
	}
	if ackBody.Summary.SnoozedCount != 0 {
		t.Fatalf("snoozed_count after ack = %d, want 0", ackBody.Summary.SnoozedCount)
	}

	snoozeReq := httptest.NewRequest(http.MethodPost, "/api/endpoint-integrity/alerts/"+alert.ID+"/snooze", bytes.NewBufferString(`{"minutes":15,"snooze_reason":"known maintenance window"}`))
	snoozeReq.Header.Set("Content-Type", "application/json")
	snoozeReq.Header.Set("Authorization", "Bearer lab-secret")
	snoozeRec := httptest.NewRecorder()
	actionHandler(snoozeRec, snoozeReq)
	if snoozeRec.Code != http.StatusOK {
		t.Fatalf("snooze status = %d: %s", snoozeRec.Code, snoozeRec.Body.String())
	}
	var snoozeBody endpointIntegrityAlertsResponse
	if err := json.Unmarshal(snoozeRec.Body.Bytes(), &snoozeBody); err != nil {
		t.Fatalf("decode snooze response: %v", err)
	}
	if snoozeBody.Summary.SnoozedCount != 1 {
		t.Fatalf("snoozed_count after snooze = %d, want 1", snoozeBody.Summary.SnoozedCount)
	}
	if len(snoozeBody.SnoozedAlerts) != 1 {
		t.Fatalf("snoozed alerts len = %d, want 1", len(snoozeBody.SnoozedAlerts))
	}
	if snoozeBody.SnoozedAlerts[0].SnoozedUntil == nil {
		t.Fatalf("snoozed_until missing in snoozed alert")
	}

	unsnoozeReq := httptest.NewRequest(http.MethodPost, "/api/endpoint-integrity/alerts/"+alert.ID+"/unsnooze", nil)
	unsnoozeReq.Header.Set("Authorization", "Bearer lab-secret")
	unsnoozeRec := httptest.NewRecorder()
	actionHandler(unsnoozeRec, unsnoozeReq)
	if unsnoozeRec.Code != http.StatusOK {
		t.Fatalf("unsnooze status = %d: %s", unsnoozeRec.Code, unsnoozeRec.Body.String())
	}
	var unsnoozeBody endpointIntegrityAlertsResponse
	if err := json.Unmarshal(unsnoozeRec.Body.Bytes(), &unsnoozeBody); err != nil {
		t.Fatalf("decode unsnooze response: %v", err)
	}
	if unsnoozeBody.Summary.SnoozedCount != 0 {
		t.Fatalf("snoozed_count after unsnooze = %d, want 0", unsnoozeBody.Summary.SnoozedCount)
	}
	if unsnoozeBody.Summary.AckedCount != 1 {
		t.Fatalf("acked_count after unsnooze = %d, want 1", unsnoozeBody.Summary.AckedCount)
	}
}

func TestEndpointIntegrityAlertsActionValidation(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		EGMRoster: []config.EGM{
			{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443},
			{EGMID: "EGM-02", IPAddress: "127.0.0.1", Port: 9444},
		},
	}
	eng := engine.New("G2S-MC-ENDPOINT-INTEGRITY", cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)
	now := time.Now().UTC()
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: now})
	eng.Submit(engine.Event{
		Type:       engine.EventKeepAlive,
		EGMID:      "EGM-01",
		At:         now.Add(1 * time.Second),
		SourceIP:   "10.20.30.55",
		SourcePort: 9550,
	})
	eng.Submit(engine.Event{
		Type:       engine.EventKeepAlive,
		EGMID:      "EGM-02",
		At:         now.Add(2 * time.Second),
		SourceIP:   "10.20.30.55",
		SourcePort: 9550,
	})
	waitForLastEvent(t, eng, string(engine.EventKeepAlive))

	listRec := httptest.NewRecorder()
	endpointIntegrityAlertsHandler(eng, auditStore, cfg)(listRec, httptest.NewRequest(http.MethodGet, "/api/endpoint-integrity/alerts", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("GET alerts status = %d: %s", listRec.Code, listRec.Body.String())
	}
	var listBody endpointIntegrityAlertsResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode alerts list: %v", err)
	}
	if len(listBody.Alerts) == 0 {
		t.Fatalf("expected alerts list to contain at least one alert")
	}
	alertID := listBody.Alerts[0].ID

	actionHandler := requireMutationAuthForMethods(
		endpointIntegrityAlertActionHandler(eng, auditStore, cfg),
		cfg,
		http.MethodPost,
	)

	missingReq := httptest.NewRequest(http.MethodPost, "/api/endpoint-integrity/alerts/eia-missing-000001/ack", nil)
	missingReq.Header.Set("Authorization", "Bearer lab-secret")
	missingRec := httptest.NewRecorder()
	actionHandler(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing alert ack status = %d, want 404", missingRec.Code)
	}

	invalidSnoozeReq := httptest.NewRequest(http.MethodPost, "/api/endpoint-integrity/alerts/"+alertID+"/snooze", bytes.NewBufferString(`{"minutes":0}`))
	invalidSnoozeReq.Header.Set("Content-Type", "application/json")
	invalidSnoozeReq.Header.Set("Authorization", "Bearer lab-secret")
	invalidSnoozeRec := httptest.NewRecorder()
	actionHandler(invalidSnoozeRec, invalidSnoozeReq)
	if invalidSnoozeRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid snooze status = %d, want 400", invalidSnoozeRec.Code)
	}

	invalidSnoozeJSONReq := httptest.NewRequest(http.MethodPost, "/api/endpoint-integrity/alerts/"+alertID+"/snooze", bytes.NewBufferString(`{"minutes":"fifteen"}`))
	invalidSnoozeJSONReq.Header.Set("Content-Type", "application/json")
	invalidSnoozeJSONReq.Header.Set("Authorization", "Bearer lab-secret")
	invalidSnoozeJSONRec := httptest.NewRecorder()
	actionHandler(invalidSnoozeJSONRec, invalidSnoozeJSONReq)
	if invalidSnoozeJSONRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid snooze JSON status = %d, want 400", invalidSnoozeJSONRec.Code)
	}

	invalidActionReq := httptest.NewRequest(http.MethodPost, "/api/endpoint-integrity/alerts/"+alertID+"/badaction", nil)
	invalidActionReq.Header.Set("Authorization", "Bearer lab-secret")
	invalidActionRec := httptest.NewRecorder()
	actionHandler(invalidActionRec, invalidActionReq)
	if invalidActionRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid action status = %d, want 400", invalidActionRec.Code)
	}

	maxMinutesReq := httptest.NewRequest(http.MethodPost, "/api/endpoint-integrity/alerts/"+alertID+"/snooze", bytes.NewBufferString(`{"minutes":`+strconv.Itoa(endpointIntegritySnoozeMaxMinutes+1)+`}`))
	maxMinutesReq.Header.Set("Content-Type", "application/json")
	maxMinutesReq.Header.Set("Authorization", "Bearer lab-secret")
	maxMinutesRec := httptest.NewRecorder()
	actionHandler(maxMinutesRec, maxMinutesReq)
	if maxMinutesRec.Code != http.StatusBadRequest {
		t.Fatalf("excessive snooze minutes status = %d, want 400", maxMinutesRec.Code)
	}
}
