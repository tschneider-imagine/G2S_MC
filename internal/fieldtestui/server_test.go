package fieldtestui

import (
	"context"
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
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

func TestFieldTestHomeRendersHTML(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedFieldTestData(t, ctx, db)

	mux := http.NewServeMux()
	server := NewServer(db, Options{}, allowMutation)
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/field-test", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("content-type=%q", got)
	}
	if !strings.Contains(res.Body.String(), "REAL SEND IS GATED / DISABLED") {
		t.Fatalf("missing real send gate summary")
	}
	if strings.Contains(res.Body.String(), "G2S_MC Rebuild Project Definition") {
		t.Fatalf("project-definition markdown text should not be embedded")
	}
}

func TestFieldTestInputsRendersConfiguredInputs(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedFieldTestData(t, ctx, db)

	mux := http.NewServeMux()
	server := NewServer(db, Options{}, allowMutation)
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/field-test/inputs", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Configured Inputs") || !strings.Contains(body, "input-1") {
		t.Fatalf("missing input table markers")
	}
}

func TestFieldTestActionsRendersDefinitionsAndPreviewLink(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedFieldTestData(t, ctx, db)

	mux := http.NewServeMux()
	server := NewServer(db, Options{}, allowMutation)
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/field-test/actions", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Action Definitions") || !strings.Contains(body, "action-1") {
		t.Fatalf("missing action table markers")
	}
	if !strings.Contains(body, "/api/v2/actions/action-1/preview") {
		t.Fatalf("missing action preview link")
	}
}

func TestFieldTestEGMsRendersRecords(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedFieldTestData(t, ctx, db)

	mux := http.NewServeMux()
	server := NewServer(db, Options{}, allowMutation)
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/field-test/egms", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "EGM Registry") || !strings.Contains(res.Body.String(), "EGM-1") {
		t.Fatalf("missing egm registry markers")
	}
}

func TestFieldTestTemplatesRendersTemplates(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedFieldTestData(t, ctx, db)

	mux := http.NewServeMux()
	server := NewServer(db, Options{}, allowMutation)
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/field-test/templates", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Templates") || !strings.Contains(body, "template-1") {
		t.Fatalf("missing template markers")
	}
	if !strings.Contains(body, "Render Preview (No Send)") {
		t.Fatalf("missing render preview marker")
	}
}

func TestFieldTestCommsRendersJournal(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedFieldTestData(t, ctx, db)

	mux := http.NewServeMux()
	server := NewServer(db, Options{}, allowMutation)
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/field-test/comms", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "Comms Journal") || !strings.Contains(res.Body.String(), "queue_only_no_send") {
		t.Fatalf("missing comms markers")
	}
}

func TestFieldTestAuditRendersTimeline(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedFieldTestData(t, ctx, db)

	mux := http.NewServeMux()
	server := NewServer(db, Options{}, allowMutation)
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/field-test/audit", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "Emergency Audit Timeline") || !strings.Contains(res.Body.String(), "INPUT_TRANSITION") {
		t.Fatalf("missing audit markers")
	}
}

func TestFieldTestSettingsRendersGatedState(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedFieldTestData(t, ctx, db)

	mux := http.NewServeMux()
	server := NewServer(db, Options{RealSendDefaultDisabled: true}, allowMutation)
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/field-test/settings", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "disabled/gated") {
		t.Fatalf("missing settings gate marker")
	}
}

func TestFieldTestInputMutationRequiresAuth(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedFieldTestData(t, ctx, db)

	calls := 0
	mux := http.NewServeMux()
	server := NewServer(db, Options{}, func(_ http.ResponseWriter, _ *http.Request) bool {
		calls++
		return false
	})
	server.RegisterRoutes(mux)

	body := strings.NewReader("normal_state=LOW&debounce_ms=100")
	req := httptest.NewRequest(http.MethodPost, "/field-test/inputs/input-1", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if calls != 1 {
		t.Fatalf("authorize calls=%d", calls)
	}
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", res.Code)
	}
}

func allowMutation(_ http.ResponseWriter, _ *http.Request) bool { return true }

func newTestStore(t *testing.T, ctx context.Context) *store.SQLiteStore {
	t.Helper()
	s, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return s
}

func seedFieldTestData(t *testing.T, ctx context.Context, db *store.SQLiteStore) {
	t.Helper()
	channel := inputs.InputChannel{
		ID:           "input-1",
		Name:         "Emergency Broadcast",
		GPIOChannel:  "GPIO21",
		Enabled:      true,
		NormalState:  inputs.InputStateHigh,
		CurrentState: inputs.InputStateHigh,
		DerivedState: inputs.DerivedStateNormal,
		DebounceMS:   50,
		Priority:     400,
		LatchingMode: inputs.LatchingManualClear,
	}
	if err := db.UpsertInputChannel(ctx, channel); err != nil {
		t.Fatalf("upsert input channel: %v", err)
	}
	if err := db.UpsertInputRuntimeState(ctx, inputruntime.InputRuntimeState{
		InputID:              "input-1",
		StableRawState:       inputs.InputStateHigh,
		DerivedState:         inputs.DerivedStateTriggered,
		LatchActive:          true,
		StableSince:          time.Now().UTC(),
		LastObservedRawState: inputs.InputStateHigh,
		LastObservedAt:       time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}); err != nil {
		t.Fatalf("upsert input runtime state: %v", err)
	}
	if _, err := db.RecordInputTransition(ctx, inputs.InputTransition{
		InputChannelID:  "input-1",
		PreviousDerived: inputs.DerivedStateNormal,
		NewDerived:      inputs.DerivedStateTriggered,
		TransitionAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("record input transition: %v", err)
	}

	if err := db.UpsertActionDefinition(ctx, actions.ActionDefinition{
		ID:               "action-1",
		Name:             "Emergency Action",
		Severity:         actions.SeverityEmergency,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps: []actions.ActionStep{{
			ID:                "step-1",
			Name:              "Primary",
			Sequence:          0,
			TemplateActionKey: "queue_only_no_send",
		}},
		Version: 1,
	}); err != nil {
		t.Fatalf("upsert action definition: %v", err)
	}
	if _, err := db.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-1",
		ActionDefinitionID: "action-1",
		StartedAt:          time.Now().UTC(),
		Status:             actions.RunStatusDispatchPrepared,
		TargetCount:        1,
	}); err != nil {
		t.Fatalf("create action run: %v", err)
	}

	if err := db.UpsertG2STemplate(ctx, templates.G2STemplate{
		ID:     "template-1",
		Name:   "Template 1",
		Vendor: "SMOKE",
		Status: templates.TemplateStatusActive,
	}); err != nil {
		t.Fatalf("upsert template: %v", err)
	}
	if err := db.UpsertG2STemplateVersion(ctx, templates.G2STemplateVersion{
		ID:           "template-1-v1",
		TemplateID:   "template-1",
		VersionLabel: "1",
		ActionsJSON:  `{"actions":{"queue_only_no_send":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<x>{{.EGMID}}</x>"}}}`,
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}
	if err := db.SetActiveG2STemplateVersion(ctx, "template-1", 1); err != nil {
		t.Fatalf("set active template version: %v", err)
	}

	if err := db.UpsertEGMRecord(ctx, egms.EGMRecord{
		EGMID:              "EGM-1",
		DisplayName:        "Cabinet 1",
		IPAddress:          "127.0.0.1",
		EndpointPath:       "/capture",
		Vendor:             "SMOKE",
		Zone:               "A",
		Enabled:            true,
		EmergencyEnabled:   true,
		TemplateID:         "template-1",
		CurrentActionState: egms.EGMActionStateNormal,
	}); err != nil {
		t.Fatalf("upsert egm record: %v", err)
	}

	if _, err := db.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:       time.Now().UTC(),
		Direction:       g2sengine.DirectionOutbound,
		EGMID:           "EGM-1",
		ActionRunID:     "run-1",
		TemplateID:      "template-1",
		TemplateVersion: "1",
		MessageType:     "queue_only_no_send",
		RawPayload:      "<dryRun/>",
		Result:          g2sengine.MessageResultDryRun,
		TransportMode:   "DRY_RUN",
	}); err != nil {
		t.Fatalf("record message journal entry: %v", err)
	}

	if _, err := db.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:        time.Now().UTC(),
		Severity:          audit.AuditSeverityEmergency,
		EventType:         "INPUT_TRANSITION",
		Summary:           "Input transition recorded",
		InputTransitionID: 1,
		ActionRunID:       "run-1",
		MessageJournalID:  1,
		Operator:          "tester",
		DetailJSON:        `{"note":"seeded"}`,
	}); err != nil {
		t.Fatalf("record audit timeline entry: %v", err)
	}

	if err := db.ReplaceCertificateInventory(ctx, []model.CertificateInventory{
		{
			Role:          "web_server_cert",
			Path:          "/tmp/host.crt",
			Status:        "OK",
			LastCheckedAt: time.Now().UTC(),
		},
		{
			Role:          "g2s_ca",
			Path:          "/tmp/ca.crt",
			Status:        "MISSING",
			LastCheckedAt: time.Now().UTC(),
		},
	}); err != nil {
		t.Fatalf("replace certificate inventory: %v", err)
	}
}
