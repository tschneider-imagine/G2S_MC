package fieldtestui

import (
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
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type seedOptions struct {
	emergencyManualClear bool
	missingEGMTemplate   bool
}

func TestFieldTestReadinessPageRendersHTML(t *testing.T) {
	mux := setupFieldTestServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/field-test/readiness", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, "Field-Test Readiness Review") {
		t.Fatalf("missing readiness title")
	}
	if strings.Contains(body, "G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS") {
		t.Fatalf("project-definition markdown text should not be embedded")
	}
}

func TestFieldTestReadinessJSONIsValid(t *testing.T) {
	mux := setupFieldTestServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/field-test/readiness.json", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var report ReadinessReport
	if err := json.Unmarshal(res.Body.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal readiness json: %v", err)
	}
	if report.GeneratedAt.IsZero() || len(report.Sections) == 0 {
		t.Fatalf("unexpected readiness report: %+v", report)
	}
}

func TestReadinessFlagsMissingEmergencyManualClearAsFail(t *testing.T) {
	mux := setupFieldTestServer(t, seedOptions{emergencyManualClear: false})
	report := getReadinessReport(t, mux)
	check := findReadinessCheck(t, report, "INPUT_EMERGENCY_LATCH_MODE")
	if check.Status != ReadinessFail {
		t.Fatalf("status=%s detail=%s", check.Status, check.Detail)
	}
}

func TestReadinessFlagsMissingEGMTemplates(t *testing.T) {
	mux := setupFieldTestServer(t, seedOptions{emergencyManualClear: true, missingEGMTemplate: true})
	report := getReadinessReport(t, mux)
	check := findReadinessCheck(t, report, "EGM_TEMPLATE_ASSIGNMENT")
	if check.Status != ReadinessWarn {
		t.Fatalf("status=%s detail=%s", check.Status, check.Detail)
	}
	if !strings.Contains(check.Detail, "EGM-SMOKE-002") {
		t.Fatalf("missing egm detail: %s", check.Detail)
	}
}

func TestReadinessIncludesRealSendGatedStatus(t *testing.T) {
	mux := setupFieldTestServer(t, seedOptions{emergencyManualClear: true})
	report := getReadinessReport(t, mux)
	check := findReadinessCheck(t, report, "SETTINGS_REAL_SEND_GATED")
	if check.Status != ReadinessPass {
		t.Fatalf("status=%s detail=%s", check.Status, check.Detail)
	}
	if !strings.Contains(strings.ToLower(check.Detail), "transport=http") {
		t.Fatalf("unexpected detail: %s", check.Detail)
	}
}

func TestFieldTestExportReturnsJSONWithoutPrivateKeyMaterial(t *testing.T) {
	mux := setupFieldTestServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/field-test/export", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type=%s", res.Header().Get("Content-Type"))
	}
	body := res.Body.String()
	if strings.Contains(strings.ToUpper(body), "PRIVATE KEY") {
		t.Fatalf("unexpected private key material in export")
	}
	var payload FieldTestExportPackage
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal export: %v", err)
	}
	if payload.GeneratedAt.IsZero() || len(payload.Readiness.Sections) == 0 {
		t.Fatalf("unexpected export payload")
	}
}

func TestFieldTestCommsExportReturnsRows(t *testing.T) {
	mux := setupFieldTestServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/field-test/comms/export", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var rows []g2sengine.MessageJournalEntry
	if err := json.Unmarshal(res.Body.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal comms export: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected comms rows")
	}
}

func TestFieldTestAuditExportReturnsRows(t *testing.T) {
	mux := setupFieldTestServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/field-test/audit/export", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var rows []audit.AuditTimelineEntry
	if err := json.Unmarshal(res.Body.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal audit export: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected audit rows")
	}
}

func TestFieldTestActionsPageIncludesRetryEscalationReturnFields(t *testing.T) {
	mux := setupFieldTestServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/field-test/actions", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "retry_policy_json") {
		t.Fatalf("missing retry field")
	}
	if !strings.Contains(body, "escalation_policy_json") {
		t.Fatalf("missing escalation field")
	}
	if !strings.Contains(body, "return_action_id") {
		t.Fatalf("missing return action field")
	}
}

func TestFieldTestTemplatesPageIncludesRenderPreviewAndMatcherFields(t *testing.T) {
	mux := setupFieldTestServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/field-test/templates", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Render Preview (No Send)") {
		t.Fatalf("missing render preview heading")
	}
	if !strings.Contains(body, "confirmation_rules_json") {
		t.Fatalf("missing expected response matcher field")
	}
	if !strings.Contains(body, "failure_rules_json") {
		t.Fatalf("missing failure matcher field")
	}
}

func TestFieldTestInputMutationRequiresAuth(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedFieldTestData(t, ctx, db, seedOptions{emergencyManualClear: true})

	calls := 0
	mux := http.NewServeMux()
	server := NewServer(db, Options{}, func(_ http.ResponseWriter, _ *http.Request) bool {
		calls++
		return false
	})
	server.RegisterRoutes(mux)

	body := strings.NewReader("normal_state=LOW&debounce_ms=100")
	req := httptest.NewRequest(http.MethodPost, "/field-test/inputs/emergency-broadcast", body)
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

func setupFieldTestServer(t *testing.T, opts seedOptions) *http.ServeMux {
	t.Helper()
	ctx := context.Background()
	db := newTestStore(t, ctx)
	t.Cleanup(func() { db.Close() })
	seedFieldTestData(t, ctx, db, opts)
	mux := http.NewServeMux()
	server := NewServer(db, Options{RealSendDefaultDisabled: true}, allowMutation)
	server.RegisterRoutes(mux)
	return mux
}

func getReadinessReport(t *testing.T, mux *http.ServeMux) ReadinessReport {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/field-test/readiness.json", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var report ReadinessReport
	if err := json.Unmarshal(res.Body.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal readiness: %v", err)
	}
	return report
}

func findReadinessCheck(t *testing.T, report ReadinessReport, code string) ReadinessCheck {
	t.Helper()
	for _, section := range report.Sections {
		for _, check := range section.Checks {
			if check.Code == code {
				return check
			}
		}
	}
	t.Fatalf("readiness check not found: %s", code)
	return ReadinessCheck{}
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

func seedFieldTestData(t *testing.T, ctx context.Context, db *store.SQLiteStore, opts seedOptions) {
	t.Helper()
	now := time.Now().UTC()

	manualMode := inputs.LatchingManualClear
	if !opts.emergencyManualClear {
		manualMode = inputs.LatchingAutoClear
	}

	channels := []inputs.InputChannel{
		{
			ID:                "regular-operation",
			Name:              "Regular Operation",
			GPIOChannel:       "GPIO16",
			Enabled:           true,
			NormalState:       inputs.InputStateHigh,
			CurrentState:      inputs.InputStateHigh,
			DerivedState:      inputs.DerivedStateNormal,
			DebounceMS:        100,
			Priority:          100,
			OnTriggerActionID: "regular-operation-trigger",
			OnNormalActionID:  "regular-operation-normal",
			LatchingMode:      inputs.LatchingAutoClear,
		},
		{
			ID:                "general-broadcast",
			Name:              "General Broadcast",
			GPIOChannel:       "GPIO20",
			Enabled:           true,
			NormalState:       inputs.InputStateHigh,
			CurrentState:      inputs.InputStateHigh,
			DerivedState:      inputs.DerivedStateNormal,
			DebounceMS:        100,
			Priority:          250,
			OnTriggerActionID: "general-broadcast-trigger",
			OnNormalActionID:  "general-broadcast-normal",
			LatchingMode:      inputs.LatchingAutoClear,
		},
		{
			ID:                "emergency-broadcast",
			Name:              "Emergency Broadcast",
			GPIOChannel:       "GPIO21",
			Enabled:           true,
			NormalState:       inputs.InputStateHigh,
			CurrentState:      inputs.InputStateHigh,
			DerivedState:      inputs.DerivedStateTriggered,
			DebounceMS:        150,
			Priority:          400,
			OnTriggerActionID: "emergency-broadcast-trigger",
			OnNormalActionID:  "emergency-broadcast-normal",
			LatchingMode:      manualMode,
		},
		{
			ID:                "local-notice",
			Name:              "Local Notice",
			GPIOChannel:       "GPIO26",
			Enabled:           true,
			NormalState:       inputs.InputStateHigh,
			CurrentState:      inputs.InputStateHigh,
			DerivedState:      inputs.DerivedStateNormal,
			DebounceMS:        80,
			Priority:          200,
			OnTriggerActionID: "local-notice-trigger",
			OnNormalActionID:  "local-notice-normal",
			LatchingMode:      inputs.LatchingAutoClear,
		},
	}
	for _, channel := range channels {
		if err := db.UpsertInputChannel(ctx, channel); err != nil {
			t.Fatalf("upsert input channel %s: %v", channel.ID, err)
		}
	}
	if err := db.UpsertInputRuntimeState(ctx, inputruntime.InputRuntimeState{
		InputID:              "emergency-broadcast",
		StableRawState:       inputs.InputStateLow,
		DerivedState:         inputs.DerivedStateTriggered,
		LatchActive:          true,
		StableSince:          now,
		LastObservedRawState: inputs.InputStateLow,
		LastObservedAt:       now,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("upsert input runtime state: %v", err)
	}
	transitionID, err := db.RecordInputTransition(ctx, inputs.InputTransition{
		InputChannelID:  "emergency-broadcast",
		PreviousDerived: inputs.DerivedStateNormal,
		NewDerived:      inputs.DerivedStateTriggered,
		TransitionAt:    now,
		Reason:          "seed",
	})
	if err != nil {
		t.Fatalf("record input transition: %v", err)
	}

	actionDefs := []actions.ActionDefinition{
		{
			ID:               "regular-operation-trigger",
			Name:             "Regular Operation Trigger",
			Severity:         actions.SeverityNotice,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Step 1", Sequence: 0, TemplateActionKey: "queue_only_no_send"}},
			RetryPolicyJSON:  `{"max_retries":1}`,
			EscalationJSON:   `{"escalate_after_ms":5000}`,
			ReturnActionID:   "regular-operation-normal",
			Version:          1,
		},
		{
			ID:               "regular-operation-normal",
			Name:             "Regular Operation Normal",
			Severity:         actions.SeverityRestore,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Step 1", Sequence: 0, TemplateActionKey: "queue_only_no_send"}},
			Version:          1,
		},
		{
			ID:               "general-broadcast-trigger",
			Name:             "General Broadcast Trigger",
			Severity:         actions.SeverityBroadcast,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Step 1", Sequence: 0, TemplateActionKey: "queue_only_no_send"}},
			ReturnActionID:   "general-broadcast-normal",
			Version:          1,
		},
		{
			ID:               "general-broadcast-normal",
			Name:             "General Broadcast Normal",
			Severity:         actions.SeverityRestore,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Step 1", Sequence: 0, TemplateActionKey: "queue_only_no_send"}},
			Version:          1,
		},
		{
			ID:               "emergency-broadcast-trigger",
			Name:             "Emergency Broadcast Trigger",
			Severity:         actions.SeverityEmergency,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Step 1", Sequence: 0, TemplateActionKey: "queue_only_no_send"}},
			ReturnActionID:   "emergency-broadcast-normal",
			Version:          1,
		},
		{
			ID:               "emergency-broadcast-normal",
			Name:             "Emergency Broadcast Normal",
			Severity:         actions.SeverityRestore,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Step 1", Sequence: 0, TemplateActionKey: "queue_only_no_send"}},
			Version:          1,
		},
		{
			ID:               "local-notice-trigger",
			Name:             "Local Notice Trigger",
			Severity:         actions.SeverityNotice,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Step 1", Sequence: 0, TemplateActionKey: "queue_only_no_send"}},
			ReturnActionID:   "local-notice-normal",
			Version:          1,
		},
		{
			ID:               "local-notice-normal",
			Name:             "Local Notice Normal",
			Severity:         actions.SeverityRestore,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Step 1", Sequence: 0, TemplateActionKey: "queue_only_no_send"}},
			Version:          1,
		},
	}
	for _, definition := range actionDefs {
		if err := db.UpsertActionDefinition(ctx, definition); err != nil {
			t.Fatalf("upsert action definition %s: %v", definition.ID, err)
		}
	}

	if _, err := db.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-1",
		ActionDefinitionID: "emergency-broadcast-trigger",
		InputTransitionID:  transitionID,
		StartedAt:          now,
		Status:             actions.RunStatusDispatchPrepared,
		TargetCount:        2,
	}); err != nil {
		t.Fatalf("create action run: %v", err)
	}

	if err := db.UpsertG2STemplate(ctx, templates.G2STemplate{
		ID:     "template-smoke-no-send",
		Name:   "Smoke No-Send Template",
		Vendor: "SMOKE",
		Status: templates.TemplateStatusActive,
	}); err != nil {
		t.Fatalf("upsert template: %v", err)
	}
	if err := db.UpsertG2STemplateVersion(ctx, templates.G2STemplateVersion{
		ID:                    "template-smoke-no-send-v1",
		TemplateID:            "template-smoke-no-send",
		VersionLabel:          "1",
		ActionsJSON:           `{"actions":{"queue_only_no_send":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<dryRun action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\"/>"}}}`,
		ConfirmationRulesJSON: `{"expected":"placeholder"}`,
		FailureRulesJSON:      `{"failure":"placeholder"}`,
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}
	if err := db.SetActiveG2STemplateVersion(ctx, "template-smoke-no-send", 1); err != nil {
		t.Fatalf("set active template version: %v", err)
	}

	templateForSecond := "template-smoke-no-send"
	if opts.missingEGMTemplate {
		templateForSecond = ""
	}

	egmRows := []egms.EGMRecord{
		{
			EGMID:              "EGM-SMOKE-001",
			DisplayName:        "Cabinet Smoke 1",
			IPAddress:          "127.0.0.1",
			EndpointPath:       "/capture",
			Vendor:             "SMOKE",
			Zone:               "A",
			Enabled:            true,
			EmergencyEnabled:   true,
			TemplateID:         "template-smoke-no-send",
			CurrentActionState: egms.EGMActionStatePending,
		},
		{
			EGMID:              "EGM-SMOKE-002",
			DisplayName:        "Cabinet Smoke 2",
			IPAddress:          "127.0.0.1",
			EndpointPath:       "/capture",
			Vendor:             "SMOKE",
			Zone:               "A",
			Enabled:            true,
			EmergencyEnabled:   true,
			TemplateID:         templateForSecond,
			CurrentActionState: egms.EGMActionStatePending,
		},
	}
	for _, record := range egmRows {
		if err := db.UpsertEGMRecord(ctx, record); err != nil {
			t.Fatalf("upsert egm record %s: %v", record.EGMID, err)
		}
	}

	if _, err := db.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:       now,
		Direction:       g2sengine.DirectionOutbound,
		EGMID:           "EGM-SMOKE-001",
		ActionRunID:     "run-1",
		TemplateID:      "template-smoke-no-send",
		TemplateVersion: "1",
		MessageType:     "queue_only_no_send",
		RawPayload:      "<dryRun action=\"emergency-broadcast-trigger\" run=\"run-1\" egm=\"EGM-SMOKE-001\"/>",
		Result:          g2sengine.MessageResultSendBlocked,
		TransportMode:   "HTTP",
		Error:           "transport_gate",
	}); err != nil {
		t.Fatalf("record message journal entry: %v", err)
	}
	if _, err := db.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:       now,
		Direction:       g2sengine.DirectionOutbound,
		EGMID:           "EGM-SMOKE-002",
		ActionRunID:     "run-1",
		TemplateID:      "template-smoke-no-send",
		TemplateVersion: "1",
		MessageType:     "queue_only_no_send",
		RawPayload:      "<dryRun action=\"emergency-broadcast-trigger\" run=\"run-1\" egm=\"EGM-SMOKE-002\"/>",
		Result:          g2sengine.MessageResultDryRun,
		TransportMode:   "DRY_RUN",
	}); err != nil {
		t.Fatalf("record message journal entry 2: %v", err)
	}

	auditRows := []audit.AuditTimelineEntry{
		{
			OccurredAt:        now,
			Severity:          audit.AuditSeverityEmergency,
			EventType:         audit.EventTypeInputTransition,
			Summary:           "Emergency input transition",
			InputTransitionID: transitionID,
			ActionRunID:       "run-1",
			Operator:          "tester",
		},
		{
			OccurredAt:        now,
			Severity:          audit.AuditSeverityEmergency,
			EventType:         audit.EventTypeActionQueued,
			Summary:           "Action queued from emergency input",
			InputTransitionID: transitionID,
			ActionRunID:       "run-1",
			Operator:          "tester",
		},
		{
			OccurredAt:        now,
			Severity:          audit.AuditSeverityInfo,
			EventType:         audit.EventTypeInputLatchClearSucceeded,
			Summary:           "Manual clear succeeded",
			InputTransitionID: transitionID,
			ActionRunID:       "run-1",
			Operator:          "operator",
		},
		{
			OccurredAt:        now,
			Severity:          audit.AuditSeverityWarning,
			EventType:         audit.EventTypeMessageSendBlocked,
			Summary:           "Send blocked by transport gate",
			InputTransitionID: transitionID,
			ActionRunID:       "run-1",
			Operator:          "tester",
		},
	}
	for _, row := range auditRows {
		if _, err := db.RecordAuditTimelineEntry(ctx, row); err != nil {
			t.Fatalf("record audit timeline entry: %v", err)
		}
	}

	if err := db.ReplaceCertificateInventory(ctx, []model.CertificateInventory{
		{
			Role:          "web_server_cert",
			Path:          "/certs/web.crt",
			Status:        "OK",
			LastCheckedAt: now,
		},
		{
			Role:          "g2s_ca",
			Path:          "/certs/g2s-ca.crt",
			Status:        "MISSING",
			LastCheckedAt: now,
		},
	}); err != nil {
		t.Fatalf("replace certificate inventory: %v", err)
	}
}
