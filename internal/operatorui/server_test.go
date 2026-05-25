package operatorui

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

func TestOperatorSystemCheckPageRendersHTML(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/settings/system-check", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, "System Check") {
		t.Fatalf("missing system check title")
	}
	if strings.Contains(body, "G2S_MC_REBUILD_PROJECT_DEFINITION_AND_GUARDRAILS") {
		t.Fatalf("project-definition markdown text should not be embedded")
	}
}

func TestOperatorSystemCheckJSONIsValid(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/settings/system-check.json", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var report SystemCheckReport
	if err := json.Unmarshal(res.Body.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal system check json: %v", err)
	}
	if report.GeneratedAt.IsZero() || len(report.Sections) == 0 {
		t.Fatalf("unexpected system check report: %+v", report)
	}
}

func TestSystemCheckFlagsMissingEmergencyManualClearAsFail(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: false})
	report := getSystemCheckReport(t, mux)
	check := findSystemCheckItem(t, report, "INPUT_EMERGENCY_LATCH_MODE")
	if check.Status != SystemCheckFail {
		t.Fatalf("status=%s detail=%s", check.Status, check.Detail)
	}
}

func TestSystemCheckFlagsMissingEGMTemplates(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true, missingEGMTemplate: true})
	report := getSystemCheckReport(t, mux)
	check := findSystemCheckItem(t, report, "EGM_TEMPLATE_ASSIGNMENT")
	if check.Status != SystemCheckWarn {
		t.Fatalf("status=%s detail=%s", check.Status, check.Detail)
	}
	if !strings.Contains(check.Detail, "EGM-002") {
		t.Fatalf("missing egm detail: %s", check.Detail)
	}
}

func TestSystemCheckIncludesRealSendGatedStatus(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	report := getSystemCheckReport(t, mux)
	check := findSystemCheckItem(t, report, "SETTINGS_REAL_SEND_GATED")
	if check.Status != SystemCheckPass {
		t.Fatalf("status=%s detail=%s", check.Status, check.Detail)
	}
	if !strings.Contains(strings.ToLower(check.Detail), "transport=http") {
		t.Fatalf("unexpected detail: %s", check.Detail)
	}
}

func TestOperatorExportReturnsJSONWithoutPrivateKeyMaterial(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/export", nil)
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
	var payload OperatorExportPackage
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal export: %v", err)
	}
	if payload.GeneratedAt.IsZero() || len(payload.SystemCheck.Sections) == 0 {
		t.Fatalf("unexpected export payload")
	}
}

func TestOperatorCommsExportReturnsRows(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/comms/export", nil)
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

func TestOperatorAuditExportReturnsRows(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/audit/export", nil)
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

func TestOperatorActionsPageIncludesRetryEscalationReturnFields(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/actions", nil)
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

func TestOperatorTemplatesPageIncludesRenderPreviewAndMatcherFields(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/templates", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	body := res.Body.String()
	if !strings.Contains(body, "Render Preview") {
		t.Fatalf("missing render preview heading")
	}
	if !strings.Contains(body, "confirmation_rules_json") {
		t.Fatalf("missing expected response matcher field")
	}
	if !strings.Contains(body, "failure_rules_json") {
		t.Fatalf("missing failure matcher field")
	}
}

func TestOperatorRoutesAndNavLabels(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	requiredRoutes := []string{
		"/operator",
		"/operator/inputs",
		"/operator/actions",
		"/operator/comms",
		"/operator/egms",
		"/operator/templates",
		"/operator/audit",
		"/operator/settings",
	}
	for _, route := range requiredRoutes {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("route %s status=%d body=%s", route, res.Code, res.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/operator", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	body := res.Body.String()
	navStart := strings.Index(body, "<nav>")
	navEnd := strings.Index(body, "</nav>")
	if navStart < 0 || navEnd < 0 || navEnd <= navStart {
		t.Fatalf("nav section not found")
	}
	nav := body[navStart:navEnd]
	if strings.Count(nav, "<a href=") != 8 {
		t.Fatalf("nav link count=%d want=8 nav=%s", strings.Count(nav, "<a href="), nav)
	}
	if strings.Contains(nav, ">Readiness</a>") {
		t.Fatalf("nav unexpectedly contains Readiness")
	}
	labels := []string{"Live", "Inputs", "Actions", "Comms", "EGMs", "Templates", "Audit", "Settings"}
	cursor := 0
	for _, label := range labels {
		index := strings.Index(nav[cursor:], ">"+label+"</a>")
		if index < 0 {
			t.Fatalf("missing nav label=%s nav=%s", label, nav)
		}
		cursor += index + len(label)
	}
}

func TestOperatorDeprecatedRoutesReturnNotFound(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	for _, route := range []string{"/operator/readiness", "/operator/readiness.json", "/field-test"} {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != http.StatusNotFound {
			t.Fatalf("route %s status=%d want=%d body=%s", route, res.Code, http.StatusNotFound, res.Body.String())
		}
	}
}

func TestOperatorSystemCheckRouteReturnsOK(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/settings/system-check", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestOperatorRuntimeHTMLExcludesProjectAndTestLanguage(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	routes := []string{
		"/operator",
		"/operator/inputs",
		"/operator/actions",
		"/operator/comms",
		"/operator/egms",
		"/operator/templates",
		"/operator/audit",
		"/operator/settings",
		"/operator/settings/system-check",
	}
	bannedTerms := []string{
		"field-test",
		"fieldtest",
		"readiness",
		"phase 2g",
		"phase 3",
		"project-definition",
		"project plan",
		"smoke",
		"queue only",
		"queue_only_no_send",
		"demo",
		"fake",
		"simulator",
		"test harness",
	}
	for _, route := range routes {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("route %s status=%d body=%s", route, res.Code, res.Body.String())
		}
		lowered := strings.ToLower(res.Body.String())
		for _, term := range bannedTerms {
			if strings.Contains(lowered, term) {
				t.Fatalf("route %s contains banned term %q", route, term)
			}
		}
	}
}

func TestOperatorActionsPageUsesProductSemanticNames(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/actions", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	expected := []string{
		"Emergency Broadcast Trigger",
		"Emergency Broadcast Restore",
		"General Broadcast Trigger",
		"General Broadcast Restore",
		"Local Notice Trigger",
		"Local Notice Restore",
		"Regular Operation Trigger",
	}
	for _, term := range expected {
		if !strings.Contains(body, term) {
			t.Fatalf("actions page missing %q", term)
		}
	}
}

func TestOperatorEGMPageUsesProductNeutralSeedLabels(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/egms", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if strings.Contains(body, "Smoke EGM") {
		t.Fatalf("egm page should not include smoke labels")
	}
	if !strings.Contains(body, "Cabinet 001") || !strings.Contains(body, "Cabinet 002") {
		t.Fatalf("egm page missing product-neutral cabinet labels")
	}
}

func TestOperatorTemplatesPageUsesProductNeutralSeedLabels(t *testing.T) {
	mux := setupOperatorServer(t, seedOptions{emergencyManualClear: true})
	req := httptest.NewRequest(http.MethodGet, "/operator/templates", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if strings.Contains(body, "SMOKE") || strings.Contains(body, "Template Smoke No Send") {
		t.Fatalf("template page should not include smoke labels")
	}
	if !strings.Contains(body, "Generic G2S Action Template") || !strings.Contains(body, "Generic") {
		t.Fatalf("template page missing product-neutral labels")
	}
}

func TestOperatorInputMutationRequiresAuth(t *testing.T) {
	ctx := context.Background()
	db := newTestStore(t, ctx)
	defer db.Close()
	seedOperatorData(t, ctx, db, seedOptions{emergencyManualClear: true})

	calls := 0
	mux := http.NewServeMux()
	server := NewServer(db, Options{}, func(_ http.ResponseWriter, _ *http.Request) bool {
		calls++
		return false
	})
	server.RegisterRoutes(mux)

	body := strings.NewReader("normal_state=LOW&debounce_ms=100")
	req := httptest.NewRequest(http.MethodPost, "/operator/inputs/emergency-broadcast", body)
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

func setupOperatorServer(t *testing.T, opts seedOptions) *http.ServeMux {
	t.Helper()
	ctx := context.Background()
	db := newTestStore(t, ctx)
	t.Cleanup(func() { db.Close() })
	seedOperatorData(t, ctx, db, opts)
	mux := http.NewServeMux()
	server := NewServer(db, Options{RealSendDefaultDisabled: true}, allowMutation)
	server.RegisterRoutes(mux)
	return mux
}

func getSystemCheckReport(t *testing.T, mux *http.ServeMux) SystemCheckReport {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/operator/settings/system-check.json", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var report SystemCheckReport
	if err := json.Unmarshal(res.Body.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal system check: %v", err)
	}
	return report
}

func findSystemCheckItem(t *testing.T, report SystemCheckReport, code string) SystemCheckItem {
	t.Helper()
	for _, section := range report.Sections {
		for _, check := range section.Checks {
			if check.Code == code {
				return check
			}
		}
	}
	t.Fatalf("system check item not found: %s", code)
	return SystemCheckItem{}
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

func seedOperatorData(t *testing.T, ctx context.Context, db *store.SQLiteStore, opts seedOptions) {
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
			OnNormalActionID:  "",
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
			OnNormalActionID:  "general-broadcast-restore",
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
			OnNormalActionID:  "emergency-broadcast-restore",
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
			OnNormalActionID:  "local-notice-restore",
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
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary message", Sequence: 0, TemplateActionKey: "regular_operation_notice"}},
			RetryPolicyJSON:  `{"max_retries":1}`,
			EscalationJSON:   `{"escalate_after_ms":5000}`,
			Version:          1,
		},
		{
			ID:               "general-broadcast-trigger",
			Name:             "General Broadcast Trigger",
			Severity:         actions.SeverityBroadcast,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary message", Sequence: 0, TemplateActionKey: "general_broadcast_notice"}},
			ReturnActionID:   "general-broadcast-restore",
			Version:          1,
		},
		{
			ID:               "general-broadcast-restore",
			Name:             "General Broadcast Restore",
			Severity:         actions.SeverityRestore,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary message", Sequence: 0, TemplateActionKey: "general_broadcast_restore"}},
			Version:          1,
		},
		{
			ID:               "emergency-broadcast-trigger",
			Name:             "Emergency Broadcast Trigger",
			Severity:         actions.SeverityEmergency,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary message", Sequence: 0, TemplateActionKey: "emergency_broadcast_silence"}},
			ReturnActionID:   "emergency-broadcast-restore",
			Version:          1,
		},
		{
			ID:               "emergency-broadcast-restore",
			Name:             "Emergency Broadcast Restore",
			Severity:         actions.SeverityRestore,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary message", Sequence: 0, TemplateActionKey: "emergency_broadcast_restore"}},
			Version:          1,
		},
		{
			ID:               "local-notice-trigger",
			Name:             "Local Notice Trigger",
			Severity:         actions.SeverityNotice,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary message", Sequence: 0, TemplateActionKey: "local_notice"}},
			ReturnActionID:   "local-notice-restore",
			Version:          1,
		},
		{
			ID:               "local-notice-restore",
			Name:             "Local Notice Restore",
			Severity:         actions.SeverityRestore,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary message", Sequence: 0, TemplateActionKey: "local_notice_restore"}},
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
		ID:     "template-generic-g2s-action",
		Name:   "Generic G2S Action Template",
		Vendor: "Generic",
		Status: templates.TemplateStatusActive,
	}); err != nil {
		t.Fatalf("upsert template: %v", err)
	}
	if err := db.UpsertG2STemplateVersion(ctx, templates.G2STemplateVersion{
		ID:                    "template-generic-g2s-action-v1",
		TemplateID:            "template-generic-g2s-action",
		VersionLabel:          "1",
		ActionsJSON:           `{"actions":{"emergency_broadcast_silence":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<dryRun action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\"/>"},"emergency_broadcast_restore":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<dryRun action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\"/>"},"general_broadcast_notice":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<dryRun action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\"/>"},"general_broadcast_restore":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<dryRun action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\"/>"},"local_notice":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<dryRun action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\"/>"},"local_notice_restore":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<dryRun action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\"/>"},"regular_operation_notice":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<dryRun action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\"/>"}}}`,
		ConfirmationRulesJSON: `{"expected":"placeholder"}`,
		FailureRulesJSON:      `{"failure":"placeholder"}`,
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}
	if err := db.SetActiveG2STemplateVersion(ctx, "template-generic-g2s-action", 1); err != nil {
		t.Fatalf("set active template version: %v", err)
	}

	templateForSecond := "template-generic-g2s-action"
	if opts.missingEGMTemplate {
		templateForSecond = ""
	}

	egmRows := []egms.EGMRecord{
		{
			EGMID:              "EGM-001",
			DisplayName:        "Cabinet 001",
			IPAddress:          "127.0.0.1",
			EndpointPath:       "/capture",
			Vendor:             "Generic",
			Zone:               "A",
			Enabled:            true,
			EmergencyEnabled:   true,
			TemplateID:         "template-generic-g2s-action",
			CurrentActionState: egms.EGMActionStatePending,
		},
		{
			EGMID:              "EGM-002",
			DisplayName:        "Cabinet 002",
			IPAddress:          "127.0.0.1",
			EndpointPath:       "/capture",
			Vendor:             "Generic",
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
		EGMID:           "EGM-001",
		ActionRunID:     "run-1",
		TemplateID:      "template-generic-g2s-action",
		TemplateVersion: "1",
		MessageType:     "emergency_broadcast_silence",
		RawPayload:      "<dryRun action=\"emergency-broadcast-trigger\" run=\"run-1\" egm=\"EGM-001\"/>",
		Result:          g2sengine.MessageResultSendBlocked,
		TransportMode:   "HTTP",
		Error:           "transport_gate",
	}); err != nil {
		t.Fatalf("record message journal entry: %v", err)
	}
	if _, err := db.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:       now,
		Direction:       g2sengine.DirectionOutbound,
		EGMID:           "EGM-002",
		ActionRunID:     "run-1",
		TemplateID:      "template-generic-g2s-action",
		TemplateVersion: "1",
		MessageType:     "emergency_broadcast_silence",
		RawPayload:      "<dryRun action=\"emergency-broadcast-trigger\" run=\"run-1\" egm=\"EGM-002\"/>",
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
