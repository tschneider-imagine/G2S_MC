package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestEGMRegistryHandlersPromoteEditDeleteFlow(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		ControllerID: "G2S-MC-REGISTRY",
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
	now := time.Now().UTC()
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: now})
	eng.Submit(engine.Event{
		Type:       engine.EventKeepAlive,
		EGMID:      "EGM-02",
		At:         now.Add(time.Second),
		SourceIP:   "10.20.30.42",
		SourcePort: 9555,
	})
	waitForLastEvent(t, eng, string(engine.EventKeepAlive))

	listHandler := egmRegistryHandler(eng, auditStore, cfg)
	promoteHandler := egmRegistryPromoteHandler(eng, auditStore, cfg)
	byIDHandler := egmRegistryByIDHandler(eng, auditStore, cfg)

	getReq := httptest.NewRequest(http.MethodGet, "/api/egm-registry", nil)
	getRec := httptest.NewRecorder()
	listHandler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}
	var listBody egmRegistryResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if len(listBody.Overrides) != 0 {
		t.Fatalf("expected no overrides initially, got %d", len(listBody.Overrides))
	}

	promoteReq := httptest.NewRequest(http.MethodPost, "/api/egm-registry/promote", bytes.NewBufferString(`{"egm_id":"EGM-02","display_name":"Cabinet 2","notes":"promoted"}`))
	promoteReq.Header.Set("Content-Type", "application/json")
	promoteRec := httptest.NewRecorder()
	promoteHandler(promoteRec, promoteReq)
	if promoteRec.Code != http.StatusOK {
		t.Fatalf("PROMOTE status = %d: %s", promoteRec.Code, promoteRec.Body.String())
	}
	var promoteBody egmRegistryResponse
	if err := json.Unmarshal(promoteRec.Body.Bytes(), &promoteBody); err != nil {
		t.Fatalf("decode PROMOTE: %v", err)
	}
	if len(promoteBody.Overrides) != 1 || promoteBody.Overrides[0].EGMID != "EGM-02" {
		t.Fatalf("unexpected promote overrides: %+v", promoteBody.Overrides)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/egm-registry/EGM-02", bytes.NewBufferString(`{"display_name":"Cabinet 2A","vendor":"Acme","cabinet_family":"Lab","game_title":"Test","software_version":"2.0.0","notes":"updated"}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	byIDHandler(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d: %s", updateRec.Code, updateRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	statusRec := httptest.NewRecorder()
	statusHandler(eng, auditStore, cfg, runtimeInfo{
		ConfigPath: "/etc/g2s-mute/config.json",
		StartedAt:  now.Add(-5 * time.Second),
	})(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status handler = %d: %s", statusRec.Code, statusRec.Body.String())
	}
	var statusBody applianceStatus
	if err := json.Unmarshal(statusRec.Body.Bytes(), &statusBody); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	found := false
	for _, row := range statusBody.EGMs {
		if row.ID != "EGM-02" {
			continue
		}
		found = true
		if row.Source != "CONFIGURED" {
			t.Fatalf("EGM-02 source = %q, want CONFIGURED", row.Source)
		}
		if !row.RegistryOverride {
			t.Fatalf("EGM-02 registry_override = false, want true")
		}
		if row.DisplayName != "Cabinet 2A" {
			t.Fatalf("EGM-02 display_name = %q, want Cabinet 2A", row.DisplayName)
		}
		if row.Vendor != "Acme" {
			t.Fatalf("EGM-02 vendor = %q, want Acme", row.Vendor)
		}
	}
	if !found {
		t.Fatalf("EGM-02 missing from status payload")
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/egm-registry/EGM-02", nil)
	deleteRec := httptest.NewRecorder()
	byIDHandler(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	getReq = httptest.NewRequest(http.MethodGet, "/api/egm-registry", nil)
	getRec = httptest.NewRecorder()
	listHandler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", getRec.Code, getRec.Body.String())
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode GET after delete: %v", err)
	}
	if len(listBody.Overrides) != 0 {
		t.Fatalf("expected no overrides after delete, got %d", len(listBody.Overrides))
	}
}

func TestEGMRegistryMutationAuthMatrix(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API:       config.API{AuthToken: "lab-secret"},
		EGMRoster: []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}},
	}
	eng := engine.New("G2S-MC-REGISTRY-AUTH", cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)
	now := time.Now().UTC()
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: now})
	eng.Submit(engine.Event{Type: engine.EventKeepAlive, EGMID: "EGM-02", At: now.Add(time.Second)})
	waitForLastEvent(t, eng, string(engine.EventKeepAlive))

	promote := requireMutationAuthForMethods(egmRegistryPromoteHandler(eng, auditStore, cfg), cfg, http.MethodPost)
	byID := requireMutationAuthForMethods(egmRegistryByIDHandler(eng, auditStore, cfg), cfg, http.MethodPut, http.MethodDelete)
	list := egmRegistryHandler(eng, auditStore, cfg)

	getReq := httptest.NewRequest(http.MethodGet, "/api/egm-registry", nil)
	getRec := httptest.NewRecorder()
	list(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d", getRec.Code)
	}

	unauthPromote := httptest.NewRequest(http.MethodPost, "/api/egm-registry/promote", bytes.NewBufferString(`{"egm_id":"EGM-02"}`))
	unauthPromote.Header.Set("Content-Type", "application/json")
	unauthPromoteRec := httptest.NewRecorder()
	promote(unauthPromoteRec, unauthPromote)
	if !deniedByAuth(unauthPromoteRec.Code) {
		t.Fatalf("promote without token status = %d, want 401/403", unauthPromoteRec.Code)
	}

	authPromote := httptest.NewRequest(http.MethodPost, "/api/egm-registry/promote", bytes.NewBufferString(`{"egm_id":"EGM-02"}`))
	authPromote.Header.Set("Content-Type", "application/json")
	authPromote.Header.Set("Authorization", "Bearer lab-secret")
	authPromoteRec := httptest.NewRecorder()
	promote(authPromoteRec, authPromote)
	if authPromoteRec.Code != http.StatusOK {
		t.Fatalf("promote with token status = %d: %s", authPromoteRec.Code, authPromoteRec.Body.String())
	}

	unauthPut := httptest.NewRequest(http.MethodPut, "/api/egm-registry/EGM-02", bytes.NewBufferString(`{"display_name":"Cab 2"}`))
	unauthPut.Header.Set("Content-Type", "application/json")
	unauthPutRec := httptest.NewRecorder()
	byID(unauthPutRec, unauthPut)
	if !deniedByAuth(unauthPutRec.Code) {
		t.Fatalf("put without token status = %d, want 401/403", unauthPutRec.Code)
	}

	authPut := httptest.NewRequest(http.MethodPut, "/api/egm-registry/EGM-02", bytes.NewBufferString(`{"display_name":"Cab 2"}`))
	authPut.Header.Set("Content-Type", "application/json")
	authPut.Header.Set("Authorization", "Bearer lab-secret")
	authPutRec := httptest.NewRecorder()
	byID(authPutRec, authPut)
	if authPutRec.Code != http.StatusOK {
		t.Fatalf("put with token status = %d: %s", authPutRec.Code, authPutRec.Body.String())
	}

	unauthDelete := httptest.NewRequest(http.MethodDelete, "/api/egm-registry/EGM-02", nil)
	unauthDeleteRec := httptest.NewRecorder()
	byID(unauthDeleteRec, unauthDelete)
	if !deniedByAuth(unauthDeleteRec.Code) {
		t.Fatalf("delete without token status = %d, want 401/403", unauthDeleteRec.Code)
	}

	authDelete := httptest.NewRequest(http.MethodDelete, "/api/egm-registry/EGM-02", nil)
	authDelete.Header.Set("Authorization", "Bearer lab-secret")
	authDeleteRec := httptest.NewRecorder()
	byID(authDeleteRec, authDelete)
	if authDeleteRec.Code != http.StatusOK {
		t.Fatalf("delete with token status = %d: %s", authDeleteRec.Code, authDeleteRec.Body.String())
	}
}

func TestEGMRegistryValidationAndPathHandling(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		EGMRoster: []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}},
	}
	eng := engine.New("G2S-MC-REGISTRY-VALIDATION", cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	eng.Start(runCtx)
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now().UTC()})
	waitForLastEvent(t, eng, string(engine.EventBootComplete))

	promoteHandler := egmRegistryPromoteHandler(eng, auditStore, cfg)
	byIDHandler := egmRegistryByIDHandler(eng, auditStore, cfg)

	emptyIDReq := httptest.NewRequest(http.MethodPost, "/api/egm-registry/promote", bytes.NewBufferString(`{"egm_id":""}`))
	emptyIDReq.Header.Set("Content-Type", "application/json")
	emptyIDRec := httptest.NewRecorder()
	promoteHandler(emptyIDRec, emptyIDReq)
	if emptyIDRec.Code != http.StatusBadRequest {
		t.Fatalf("promote empty id status = %d, want 400", emptyIDRec.Code)
	}

	unknownReq := httptest.NewRequest(http.MethodPost, "/api/egm-registry/promote", bytes.NewBufferString(`{"egm_id":"EGM-99"}`))
	unknownReq.Header.Set("Content-Type", "application/json")
	unknownRec := httptest.NewRecorder()
	promoteHandler(unknownRec, unknownReq)
	if unknownRec.Code != http.StatusNotFound {
		t.Fatalf("promote unknown id status = %d, want 404", unknownRec.Code)
	}

	tooLong := strings.Repeat("A", 121)
	invalidPutReq := httptest.NewRequest(http.MethodPut, "/api/egm-registry/EGM-01", bytes.NewBufferString(`{"display_name":"`+tooLong+`"}`))
	invalidPutReq.Header.Set("Content-Type", "application/json")
	invalidPutRec := httptest.NewRecorder()
	byIDHandler(invalidPutRec, invalidPutReq)
	if invalidPutRec.Code != http.StatusBadRequest {
		t.Fatalf("put invalid display_name status = %d, want 400", invalidPutRec.Code)
	}

	invalidPathReq := httptest.NewRequest(http.MethodPut, "/api/egm-registry/EGM-01/extra", bytes.NewBufferString(`{"display_name":"Cabinet"}`))
	invalidPathReq.Header.Set("Content-Type", "application/json")
	invalidPathRec := httptest.NewRecorder()
	byIDHandler(invalidPathRec, invalidPathReq)
	if invalidPathRec.Code != http.StatusBadRequest {
		t.Fatalf("put invalid path status = %d, want 400", invalidPathRec.Code)
	}
}
