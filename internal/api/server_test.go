package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestGetInputsReturnsJSON(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	if err := db.UpsertInputChannel(ctx, validInputChannel()); err != nil {
		t.Fatalf("seed input channel: %v", err)
	}

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/inputs", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}
	var channels []inputs.InputChannel
	if err := json.Unmarshal(res.Body.Bytes(), &channels); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(channels) != 1 || channels[0].ID != "input-1" {
		t.Fatalf("unexpected channels: %+v", channels)
	}
}

func TestPutInputValidatesAndStores(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()

	calls := 0
	mux := http.NewServeMux()
	server := &Server{
		Store: db,
		AuthorizeMutation: func(_ http.ResponseWriter, _ *http.Request) bool {
			calls++
			return true
		},
	}
	server.RegisterRoutes(mux)

	payload := validInputChannel()
	payload.ID = ""
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/v2/inputs/input-1", bytes.NewReader(raw))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", res.Code, http.StatusOK, res.Body.String())
	}
	if calls != 1 {
		t.Fatalf("authorize calls = %d, want 1", calls)
	}
	stored, err := db.GetInputChannel(ctx, "input-1")
	if err != nil {
		t.Fatalf("get input channel: %v", err)
	}
	if stored == nil || stored.Name != payload.Name {
		t.Fatalf("unexpected stored channel: %+v", stored)
	}
}

func TestPutInputPathBodyIDMismatchReturns400(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()

	mux := http.NewServeMux()
	server := &Server{Store: db, AuthorizeMutation: allowMutation}
	server.RegisterRoutes(mux)

	payload := validInputChannel()
	payload.ID = "different-id"
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/v2/inputs/input-1", bytes.NewReader(raw))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestPutInputInvalidJSONReturns400(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()

	mux := http.NewServeMux()
	server := &Server{Store: db, AuthorizeMutation: allowMutation}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPut, "/api/v2/inputs/input-1", bytes.NewReader([]byte("{")))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestMutationAuthHookCalledForPut(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()

	calls := 0
	mux := http.NewServeMux()
	server := &Server{
		Store: db,
		AuthorizeMutation: func(_ http.ResponseWriter, _ *http.Request) bool {
			calls++
			return false
		},
	}
	server.RegisterRoutes(mux)

	raw, _ := json.Marshal(validInputChannel())
	req := httptest.NewRequest(http.MethodPut, "/api/v2/inputs/input-1", bytes.NewReader(raw))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if calls != 1 {
		t.Fatalf("authorize calls = %d, want 1", calls)
	}
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestGetCommsMessagesReturnsEntries(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	if _, err := db.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:  time.Now().UTC(),
		Direction:  g2sengine.DirectionOutbound,
		EGMID:      "EGM-1",
		RawPayload: "<mute/>",
		Result:     g2sengine.MessageResultSent,
	}); err != nil {
		t.Fatalf("seed message journal entry: %v", err)
	}

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/comms/messages", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var entries []g2sengine.MessageJournalEntry
	if err := json.Unmarshal(res.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(entries))
	}
}

func TestGetAuditTimelineReturnsEntries(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	if _, err := db.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt: time.Now().UTC(),
		Severity:   audit.AuditSeverityInfo,
		EventType:  "ACTION_START",
		Summary:    "Action started",
	}); err != nil {
		t.Fatalf("seed audit timeline entry: %v", err)
	}

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/audit/timeline", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var entries []audit.AuditTimelineEntry
	if err := json.Unmarshal(res.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(entries))
	}
}

func allowMutation(_ http.ResponseWriter, _ *http.Request) bool { return true }

func validInputChannel() inputs.InputChannel {
	return inputs.InputChannel{
		ID:           "input-1",
		Name:         "Emergency Broadcast",
		GPIOChannel:  "GPIO17",
		Enabled:      true,
		NormalState:  inputs.InputStateHigh,
		CurrentState: inputs.InputStateLow,
		DerivedState: inputs.DerivedStateTriggered,
		DebounceMS:   25,
		Priority:     4,
		LatchingMode: inputs.LatchingManualClear,
	}
}

func newTestStore(t *testing.T, ctx context.Context) *store.SQLiteStore {
	t.Helper()
	s, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return s
}
