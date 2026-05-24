package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
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

func TestGetInputsStateReturnsJSON(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	if err := db.UpsertInputChannel(ctx, validInputChannel()); err != nil {
		t.Fatalf("seed input channel: %v", err)
	}
	if err := db.UpsertInputRuntimeState(ctx, inputruntime.InputRuntimeState{
		InputID:              "input-1",
		StableRawState:       inputs.InputStateHigh,
		DerivedState:         inputs.DerivedStateNormal,
		StableSince:          time.Now().UTC(),
		LastObservedRawState: inputs.InputStateHigh,
		LastObservedAt:       time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed input runtime state: %v", err)
	}

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/inputs/state", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}
	var states []InputStateEnvelope
	if err := json.Unmarshal(res.Body.Bytes(), &states); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(states) != 1 || states[0].Channel.ID != "input-1" || states[0].RuntimeState == nil {
		t.Fatalf("unexpected states: %+v", states)
	}
}

func TestGetInputsTransitionsReturnsJSON(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	if _, err := db.RecordInputTransition(ctx, inputs.InputTransition{
		InputChannelID:  "input-1",
		PreviousDerived: inputs.DerivedStateNormal,
		NewDerived:      inputs.DerivedStateTriggered,
		TransitionAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed transition: %v", err)
	}

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/inputs/transitions", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var transitions []inputs.InputTransition
	if err := json.Unmarshal(res.Body.Bytes(), &transitions); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(transitions) != 1 {
		t.Fatalf("transitions len = %d, want 1", len(transitions))
	}
}

func TestInputsStateMethodNotAllowed(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/inputs/state", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusMethodNotAllowed)
	}
}

func TestInputsTransitionsMethodNotAllowed(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/inputs/transitions", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusMethodNotAllowed)
	}
}

func TestActionPreviewReturnsJSON(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()

	if err := db.UpsertActionDefinition(ctx, actions.ActionDefinition{
		ID:               "action-1",
		Name:             "Emergency Silence",
		Severity:         actions.SeverityEmergency,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps: []actions.ActionStep{{
			ID:                "step-1",
			Name:              "Send mute",
			Sequence:          0,
			TemplateActionKey: "mute_primary",
		}},
		Version: 1,
	}); err != nil {
		t.Fatalf("seed action definition: %v", err)
	}
	if err := db.UpsertG2STemplate(ctx, templates.G2STemplate{ID: "tpl-1", Name: "IGT", Vendor: "IGT", Status: templates.TemplateStatusActive}); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	if err := db.UpsertEGMRecord(ctx, egms.EGMRecord{
		EGMID:              "EGM-001",
		DisplayName:        "Cabinet 1",
		Enabled:            true,
		EmergencyEnabled:   true,
		TemplateID:         "tpl-1",
		CurrentActionState: egms.EGMActionStateNormal,
	}); err != nil {
		t.Fatalf("seed egm: %v", err)
	}

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/actions/action-1/preview", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", res.Code, http.StatusOK, res.Body.String())
	}
	if got := res.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}
	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["action_id"] != "action-1" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestGetActionRunsReturnsJSON(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedActionRunFixtures(t, ctx, db)

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/actions/runs", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var rows []actions.ActionRun
	if err := json.Unmarshal(res.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "run-1" {
		t.Fatalf("unexpected action runs: %+v", rows)
	}
}

func TestGetActionRunByIDReturnsRun(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedActionRunFixtures(t, ctx, db)

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/actions/runs/run-1", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", res.Code, http.StatusOK, res.Body.String())
	}
	var row actions.ActionRun
	if err := json.Unmarshal(res.Body.Bytes(), &row); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if row.ID != "run-1" || row.Status != actions.RunStatusPending {
		t.Fatalf("unexpected action run: %+v", row)
	}
}

func TestGetActionRunTargetsReturnsRows(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedActionRunFixtures(t, ctx, db)

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/actions/runs/run-1/targets", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var rows []actions.ActionTargetResult
	if err := json.Unmarshal(res.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(rows) != 1 || rows[0].TargetEGMID != "EGM-1" {
		t.Fatalf("unexpected target rows: %+v", rows)
	}
}

func TestPostActionRunDispatchDryRunRequiresMutationAuth(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedActionRunFixtures(t, ctx, db)
	seedDispatchFixtures(t, ctx, db)

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

	req := httptest.NewRequest(http.MethodPost, "/api/v2/actions/runs/run-1/dispatch-dry-run", bytes.NewReader([]byte(`{"actor":"tester"}`)))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if calls != 1 {
		t.Fatalf("authorize calls=%d, want 1", calls)
	}
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestPostActionRunDispatchDryRunCreatesMessages(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedActionRunFixtures(t, ctx, db)
	seedDispatchFixtures(t, ctx, db)

	mux := http.NewServeMux()
	server := &Server{Store: db, AuthorizeMutation: allowMutation}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/actions/runs/run-1/dispatch-dry-run", bytes.NewReader([]byte(`{"actor":"tester"}`)))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d body=%s", res.Code, http.StatusOK, res.Body.String())
	}
	messages, err := db.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 50, ActionRunID: "run-1"})
	if err != nil {
		t.Fatalf("list message journal by run: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("message rows=%d, want 1", len(messages))
	}
	if messages[0].Result != g2sengine.MessageResultDryRun {
		t.Fatalf("message result=%q, want %q", messages[0].Result, g2sengine.MessageResultDryRun)
	}
	if !strings.Contains(messages[0].RawPayload, `egm="EGM-1"`) || !strings.Contains(messages[0].RawPayload, `run="run-1"`) {
		t.Fatalf("expected rendered payload markers: %s", messages[0].RawPayload)
	}
	run, err := db.GetActionRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("get action run: %v", err)
	}
	if run == nil || run.Status != actions.RunStatusDispatchPrepared {
		t.Fatalf("unexpected run after dispatch: %+v", run)
	}
}

func TestPostTemplatesRenderPreviewReturnsJSON(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedDispatchFixtures(t, ctx, db)

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	body := []byte(`{
		"template_id":"template-smoke-no-send",
		"template_action_key":"queue_only_no_send",
		"action_id":"emergency-broadcast-trigger",
		"action_run_id":"run-preview-1",
		"egm_id":"EGM-1"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/render-preview", bytes.NewReader(body))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d body=%s", res.Code, http.StatusOK, res.Body.String())
	}
	var payload TemplateRenderPreviewResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.Contains(payload.Rendered.RawPayload, `egm="EGM-1"`) || !strings.Contains(payload.Rendered.RawPayload, `run="run-preview-1"`) {
		t.Fatalf("unexpected rendered payload: %s", payload.Rendered.RawPayload)
	}
}

func TestPostTemplatesRenderPreviewMissingActionKeyReturns400(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedDispatchFixtures(t, ctx, db)

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	body := []byte(`{"template_id":"template-smoke-no-send"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/render-preview", bytes.NewReader(body))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d body=%s", res.Code, http.StatusBadRequest, res.Body.String())
	}
}

func TestPostActionRunDispatchDryRunMissingRunReturns404(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()

	mux := http.NewServeMux()
	server := &Server{Store: db, AuthorizeMutation: allowMutation}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/actions/runs/missing/dispatch-dry-run", bytes.NewReader([]byte(`{"actor":"tester"}`)))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want %d body=%s", res.Code, http.StatusNotFound, res.Body.String())
	}
}

func TestGetActionRunMessagesReturnsJSON(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedActionRunFixtures(t, ctx, db)
	if _, err := db.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:   time.Now().UTC(),
		Direction:   g2sengine.DirectionOutbound,
		EGMID:       "EGM-1",
		ActionRunID: "run-1",
		MessageType: "queue_only_no_send",
		RawPayload:  "DRY_RUN_NO_SEND action=action-1 egm=EGM-1 step=queue_only_no_send",
		Result:      g2sengine.MessageResultDryRun,
	}); err != nil {
		t.Fatalf("seed message journal entry: %v", err)
	}

	mux := http.NewServeMux()
	server := &Server{Store: db}
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/actions/runs/run-1/messages", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", res.Code, http.StatusOK)
	}
	var entries []g2sengine.MessageJournalEntry
	if err := json.Unmarshal(res.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(entries) != 1 || entries[0].ActionRunID != "run-1" {
		t.Fatalf("unexpected action run messages: %+v", entries)
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

func seedActionRunFixtures(t *testing.T, ctx context.Context, db *store.SQLiteStore) {
	t.Helper()
	if err := db.UpsertActionDefinition(ctx, actions.ActionDefinition{
		ID:               "action-1",
		Name:             "Queue Action",
		Severity:         actions.SeverityEmergency,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps: []actions.ActionStep{{
			ID:                "step-1",
			Name:              "Queue only",
			Sequence:          0,
			TemplateActionKey: "queue_only_no_send",
		}},
		Version: 1,
	}); err != nil {
		t.Fatalf("seed action definition: %v", err)
	}
	if _, err := db.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-1",
		ActionDefinitionID: "action-1",
		InputTransitionID:  12,
		StartedAt:          time.Now().UTC(),
		Status:             actions.RunStatusPending,
		TriggerReason:      "input transition 12",
		TargetCount:        1,
		ConfirmedCount:     0,
		FailedCount:        0,
		EscalatedCount:     0,
	}); err != nil {
		t.Fatalf("seed action run: %v", err)
	}
	if _, err := db.CreateActionTargetResult(ctx, actions.ActionTargetResult{
		ActionRunID:  "run-1",
		TargetEGMID:  "EGM-1",
		Status:       actions.TargetStatusPending,
		AttemptCount: 0,
	}); err != nil {
		t.Fatalf("seed action target result: %v", err)
	}
}

func seedDispatchFixtures(t *testing.T, ctx context.Context, db *store.SQLiteStore) {
	t.Helper()
	if err := db.UpsertG2STemplate(ctx, templates.G2STemplate{
		ID:     "template-smoke-no-send",
		Name:   "Smoke",
		Vendor: "SMOKE",
		Status: templates.TemplateStatusActive,
	}); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	if err := db.UpsertG2STemplateVersion(ctx, templates.G2STemplateVersion{
		ID:           "template-smoke-no-send-v1",
		TemplateID:   "template-smoke-no-send",
		VersionLabel: "1",
		ActionsJSON:  `{"actions":{"queue_only_no_send":{"message_type":"DRY_RUN_NO_SEND","content_type":"application/xml","payload_template":"<dryRunG2SMessage noSend=\"true\" action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\" step=\"{{.TemplateActionKey}}\" timestamp=\"{{.TimestampRFC3339}}\"/>"}}}`,
	}); err != nil {
		t.Fatalf("seed template version: %v", err)
	}
	if err := db.SetActiveG2STemplateVersion(ctx, "template-smoke-no-send", 1); err != nil {
		t.Fatalf("set active template version: %v", err)
	}
	if err := db.UpsertEGMRecord(ctx, egms.EGMRecord{
		EGMID:              "EGM-1",
		DisplayName:        "Smoke EGM 1",
		Enabled:            true,
		EmergencyEnabled:   true,
		TemplateID:         "template-smoke-no-send",
		CurrentActionState: egms.EGMActionStateNormal,
	}); err != nil {
		t.Fatalf("seed egm: %v", err)
	}
}
