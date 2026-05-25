package operatorui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
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

var forbiddenRuntimeTerms = []string{
	"Transport Status",
	"SENDING DISABLED",
	"controlled transport settings",
	"Capture endpoint behavior",
	"Evidence Export (JSON)",
	"Readiness",
	"System Check",
	"system-check",
	"Gate",
	"gate",
	"Safety Gate",
	"Send Gate",
	"transport gate",
	"field-test",
	"Field-Test",
	"fieldtest",
	"Phase",
	"project-definition",
	"project plan",
	"Smoke",
	"Queue Only",
	"queue_only_no_send",
	"Demo",
	"fake",
	"simulator",
	"test harness",
	"dashboard",
}

func TestOperatorNavLabelsExact(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	body := res.Body.String()
	nav := []string{"Live", "Inputs", "Actions", "Comms", "EGMs", "Templates", "Audit", "Settings"}
	prev := -1
	for _, label := range nav {
		idx := strings.Index(body, ">"+label+"<")
		if idx < 0 {
			t.Fatalf("missing nav label %q", label)
		}
		if idx <= prev {
			t.Fatalf("nav label %q is out of order", label)
		}
		prev = idx
	}
	if strings.Contains(body, ">Home<") {
		t.Fatal("unexpected Home nav label")
	}
}

func TestOperatorPagesExcludeForbiddenTerms(t *testing.T) {
	mux := setupOperatorServer(t)
	pages := []string{
		"/operator",
		"/operator/inputs",
		"/operator/actions",
		"/operator/comms",
		"/operator/egms",
		"/operator/templates",
		"/operator/audit",
		"/operator/settings",
	}
	for _, path := range pages {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, res.Code, res.Body.String())
		}
		body := res.Body.String()
		for _, term := range forbiddenRuntimeTerms {
			if strings.Contains(body, term) {
				t.Fatalf("%s contains forbidden term %q", path, term)
			}
		}
	}
}

func TestRemovedRoutesReturnNotFound(t *testing.T) {
	mux := setupOperatorServer(t)
	for _, path := range []string{
		"/field-test",
		"/operator/export",
		"/operator/readiness",
		"/operator/readiness.json",
		"/operator/settings/system-check",
		"/operator/settings/system-check.json",
		"/dashboard",
		"/static/dashboard.css",
		"/static/dashboard.js",
	} {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(res, req)
		if res.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d want=404", path, res.Code)
		}
	}
}

func TestCommsExportStillWorks(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/comms/export", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var payload commsExportPayload
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.GeneratedAt.IsZero() {
		t.Fatal("expected generated_at")
	}
	if payload.Count != len(payload.Rows) {
		t.Fatalf("count=%d rows=%d", payload.Count, len(payload.Rows))
	}
	if len(payload.Rows) == 0 {
		t.Fatal("expected comms rows")
	}
}

func TestAuditExportStillWorks(t *testing.T) {
	mux := setupOperatorServer(t)

	auditRes := httptest.NewRecorder()
	auditReq := httptest.NewRequest(http.MethodGet, "/operator/audit/export", nil)
	mux.ServeHTTP(auditRes, auditReq)
	if auditRes.Code != http.StatusOK {
		t.Fatalf("audit export status=%d body=%s", auditRes.Code, auditRes.Body.String())
	}
	var payload auditExportPayload
	if err := json.Unmarshal(auditRes.Body.Bytes(), &payload); err != nil {
		t.Fatalf("audit unmarshal: %v", err)
	}
	if payload.GeneratedAt.IsZero() {
		t.Fatal("expected generated_at")
	}
	if payload.Count != len(payload.Rows) {
		t.Fatalf("count=%d rows=%d", payload.Count, len(payload.Rows))
	}
	if len(payload.Rows) == 0 {
		t.Fatal("expected audit rows")
	}
}

func TestOperatorExportRouteRemoved(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/export", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("status=%d want=404", res.Code)
	}
}

func TestCommsPageRendersMessageEvidence(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/comms", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Message Journal",
		"/operator/comms/export",
		"from://controller",
		"to://cabinet/EGM-001",
		"run-1",
		"template-generic-g2s-action@1",
		"emergency_broadcast_silence",
		"SEND_BLOCKED",
		"dry-run rendered",
		"emergency-broadcast-trigger",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in comms page", expected)
		}
	}
}

func TestAuditPageRendersTimelineEvidence(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Audit Timeline",
		"/operator/audit/export",
		"INPUT_TRANSITION",
		"Input transition recorded",
		"run-1",
		"EGM-001",
		"operator",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in audit page", expected)
		}
	}
}

func TestActionsPageRendersActionRowsAndFields(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/actions", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Action Definitions",
		"regular-operation-trigger",
		"Emergency Broadcast Trigger",
		"Return Action",
		"Retry Count",
		"Retry Delay (ms)",
		"Escalation Action",
		"Escalation After Attempts",
		"Target Selector Type",
		"Target Preview",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in actions page", expected)
		}
	}
}

func TestActionsPageShowsInlineTargetPreviewAndWarnings(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/actions", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"targets=2",
		"EMPTY_TARGET_SET",
		"MISSING_TEMPLATE",
		"EMERGENCY_RETURN_MISSING",
		"Cabinet 001 (EGM-001)",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in actions page", expected)
		}
	}
}

func TestPostActionsSavesRetryAndEscalationJSON(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	body := url.Values{
		"id":                        {"action-new"},
		"name":                      {"Action New"},
		"severity":                  {"NOTICE"},
		"enabled":                   {"true"},
		"target_selector_type":      {targetSelectorTypeEGMIDs},
		"target_selector_value":     {"EGM-001"},
		"template_selector":         {"template-by-egm"},
		"step_template_action_key":  {"regular_operation_notice"},
		"return_action_id":          {"regular-operation-trigger"},
		"retry_count":               {"3"},
		"retry_delay_ms":            {"1500"},
		"escalation_action_id":      {"emergency-broadcast-trigger"},
		"escalation_after_attempts": {"2"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/actions", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	definition, err := st.GetActionDefinition(context.Background(), "action-new")
	if err != nil {
		t.Fatalf("get action: %v", err)
	}
	if definition == nil {
		t.Fatal("expected action definition saved")
	}
	if definition.TargetSelector != "EGM_IDS:EGM-001" {
		t.Fatalf("target_selector=%q", definition.TargetSelector)
	}
	retryPolicy := parseRetryPolicyJSON(definition.RetryPolicyJSON)
	if !reflect.DeepEqual(retryPolicy, retryPolicyConfig{Count: 3, DelayMS: 1500}) {
		t.Fatalf("retry_policy=%+v", retryPolicy)
	}
	escalationPolicy := parseEscalationPolicyJSON(definition.EscalationJSON)
	if !reflect.DeepEqual(escalationPolicy, escalationPolicyConfig{ActionID: "emergency-broadcast-trigger", AfterAttempts: 2}) {
		t.Fatalf("escalation_policy=%+v", escalationPolicy)
	}
}

func TestPostActionsRejectsInvalidRetryCount(t *testing.T) {
	mux := setupOperatorServer(t)
	body := url.Values{
		"id":                       {"action-invalid-retry"},
		"name":                     {"Action Invalid Retry"},
		"severity":                 {"NOTICE"},
		"enabled":                  {"true"},
		"target_selector_type":     {targetSelectorTypeAllEmergencyEnabled},
		"template_selector":        {"template-by-egm"},
		"step_template_action_key": {"regular_operation_notice"},
		"retry_count":              {"abc"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/actions", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "invalid retry count") {
		t.Fatalf("expected invalid retry count error, body=%s", res.Body.String())
	}
}

func TestPostActionsRejectsInvalidEscalationAttempts(t *testing.T) {
	mux := setupOperatorServer(t)
	body := url.Values{
		"id":                        {"action-invalid-escalation"},
		"name":                      {"Action Invalid Escalation"},
		"severity":                  {"NOTICE"},
		"enabled":                   {"true"},
		"target_selector_type":      {targetSelectorTypeAllEmergencyEnabled},
		"template_selector":         {"template-by-egm"},
		"step_template_action_key":  {"regular_operation_notice"},
		"escalation_action_id":      {"emergency-broadcast-trigger"},
		"escalation_after_attempts": {"0"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/actions", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "escalation after attempts must be greater than zero") {
		t.Fatalf("expected invalid escalation error, body=%s", res.Body.String())
	}
}

func TestEGMsPageRendersRegistryRowsAndTemplateID(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/egms", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"EGM Registry",
		"Cabinet 001",
		"template-generic-g2s-action",
		"Current Action State",
		"Available Template IDs",
		"EGM Groups",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/egms", expected)
		}
	}
}

func TestEGMsPageShowsTemplateAndEmergencyWarnings(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if err := st.UpsertEGMRecord(ctx, egms.EGMRecord{
		EGMID:              "EGM-004",
		DisplayName:        "Cabinet 004",
		Enabled:            false,
		EmergencyEnabled:   true,
		TemplateID:         "template-missing",
		CurrentActionState: egms.EGMActionStateNormal,
	}); err != nil {
		t.Fatalf("upsert egm: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/egms", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Template not found",
		"Emergency participation requires Enabled.",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/egms warnings", expected)
		}
	}
}

func TestPostEGMsCreatesOrUpsertsRecord(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	body := url.Values{
		"egm_id":            {"EGM-010"},
		"display_name":      {"Cabinet 010"},
		"ip_address":        {"10.10.0.10"},
		"endpoint_path":     {"/g2s"},
		"vendor":            {"Generic"},
		"cabinet_family":    {"Family A"},
		"game_title":        {"Example Game"},
		"software_version":  {"1.0.0"},
		"zone":              {"Zone-A"},
		"enabled":           {"true"},
		"emergency_enabled": {"true"},
		"template_id":       {"template-generic-g2s-action"},
		"notes":             {"Primary floor cabinet"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/egms", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	record, err := st.GetEGMRecord(context.Background(), "EGM-010")
	if err != nil {
		t.Fatalf("get egm: %v", err)
	}
	if record == nil {
		t.Fatal("expected EGM record to be created")
	}
	if record.DisplayName != "Cabinet 010" || record.TemplateID != "template-generic-g2s-action" {
		t.Fatalf("unexpected record: %+v", *record)
	}
	if record.CabinetFamily != "Family A" || record.GameTitle != "Example Game" || record.SoftwareVersion != "1.0.0" || record.Notes != "Primary floor cabinet" {
		t.Fatalf("expected cabinet/game/software/notes fields to persist, record=%+v", *record)
	}
}

func TestPostEGMByIDUpdatesAndPreservesSystemFields(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if err := st.UpsertEGMRecord(ctx, egms.EGMRecord{
		EGMID:                 "EGM-001",
		DisplayName:           "Cabinet 001",
		IPAddress:             "10.0.0.1",
		EndpointPath:          "/g2s",
		Vendor:                "Generic",
		CabinetFamily:         "Family 1",
		GameTitle:             "Title 1",
		SoftwareVersion:       "2.0.0",
		Zone:                  "Zone-1",
		Enabled:               true,
		EmergencyEnabled:      true,
		TemplateID:            "template-generic-g2s-action",
		CurrentActionState:    egms.EGMActionStatePending,
		LastSeenAt:            &now,
		HeartbeatOverrideJSON: `{"interval_ms":1000}`,
		Notes:                 "Existing notes",
	}); err != nil {
		t.Fatalf("upsert setup egm: %v", err)
	}

	body := url.Values{
		"display_name":      {"Cabinet 001 Updated"},
		"ip_address":        {"10.0.0.11"},
		"endpoint_path":     {"/g2s-updated"},
		"vendor":            {"Generic"},
		"cabinet_family":    {"Family 2"},
		"game_title":        {"Title 2"},
		"software_version":  {"2.1.0"},
		"zone":              {"Zone-2"},
		"enabled":           {"true"},
		"emergency_enabled": {"true"},
		"template_id":       {"template-generic-g2s-action"},
		"notes":             {"Updated notes"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/egms/EGM-001", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	updated, err := st.GetEGMRecord(ctx, "EGM-001")
	if err != nil {
		t.Fatalf("get egm: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated record")
	}
	if updated.DisplayName != "Cabinet 001 Updated" || updated.Zone != "Zone-2" || updated.CabinetFamily != "Family 2" {
		t.Fatalf("expected mutable fields updated, got %+v", *updated)
	}
	if updated.CurrentActionState != egms.EGMActionStatePending {
		t.Fatalf("expected current action state preserved, got %q", updated.CurrentActionState)
	}
	if updated.LastSeenAt == nil || !updated.LastSeenAt.Equal(now) {
		t.Fatalf("expected last seen preserved, got %v want %s", updated.LastSeenAt, now)
	}
	if updated.HeartbeatOverrideJSON != `{"interval_ms":1000}` {
		t.Fatalf("expected heartbeat override preserved, got %q", updated.HeartbeatOverrideJSON)
	}
}

func TestEGMsFormIncludesExpectedFields(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/egms", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		`name="enabled"`,
		`name="emergency_enabled"`,
		`name="zone"`,
		`name="vendor"`,
		`name="cabinet_family"`,
		`name="game_title"`,
		`name="software_version"`,
		`name="notes"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected field %q in /operator/egms form", expected)
		}
	}
}

func TestTemplatesPageRendersTemplateRowsActiveVersionAndActionKeys(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/templates", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Templates",
		"template-generic-g2s-action",
		"Generic G2S Action Template",
		"Active Version",
		"Action Keys",
		"Expected Response Matcher",
		"Failure Matcher",
		"emergency_broadcast_silence",
		"Render Preview",
		"Supported variables: ActionID",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/templates", expected)
		}
	}
}

func TestPostTemplatesCreatesOrUpsertsTemplate(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	body := url.Values{
		"id":                     {"template-ops"},
		"name":                   {"Operations Template"},
		"vendor":                 {"Generic"},
		"cabinet_family":         {"Family X"},
		"software_version_match": {"3.x"},
		"status":                 {"ACTIVE"},
		"notes":                  {"Operator maintained"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	row, err := st.GetG2STemplate(context.Background(), "template-ops")
	if err != nil {
		t.Fatalf("get template: %v", err)
	}
	if row == nil {
		t.Fatal("expected template row")
	}
	if row.Name != "Operations Template" || row.CabinetFamily != "Family X" || row.SoftwareVersionMatch != "3.x" || row.Notes != "Operator maintained" {
		t.Fatalf("unexpected template row: %+v", *row)
	}
}

func TestPostTemplateVersionCreatesVersionWithActionsAndMatchers(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	body := url.Values{
		"version_label":           {"2"},
		"version_id":              {"template-generic-g2s-action-v2"},
		"notes":                   {"Version 2"},
		"endpoint_quirks_json":    {`{"soap_action":"urn:test"}`},
		"heartbeat_profile_json":  {`{"interval_ms":1000}`},
		"confirmation_rules_json": {`{"expect":["ok"]}`},
		"failure_rules_json":      {`{"fail":["fault"]}`},
		"actions_json":            {`{"actions":{"general_broadcast_notice":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"}}}`},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates/template-generic-g2s-action/versions", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	versionRow, err := st.GetG2STemplateVersion(context.Background(), "template-generic-g2s-action", 2)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if versionRow == nil {
		t.Fatal("expected template version row")
	}
	if strings.TrimSpace(versionRow.ConfirmationRulesJSON) == "" || strings.TrimSpace(versionRow.FailureRulesJSON) == "" || strings.TrimSpace(versionRow.ActionsJSON) == "" {
		t.Fatalf("expected matcher/actions json saved: %+v", *versionRow)
	}
}

func TestPostTemplateActiveVersionSetsVersion(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if err := st.UpsertG2STemplateVersion(ctx, templates.G2STemplateVersion{
		ID:           "template-generic-g2s-action-v2",
		TemplateID:   "template-generic-g2s-action",
		VersionLabel: "2",
		ActionsJSON:  `{"actions":{"emergency_broadcast_restore":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"}}}`,
	}); err != nil {
		t.Fatalf("upsert version: %v", err)
	}

	body := url.Values{"active_version": {"2"}}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates/template-generic-g2s-action/active-version", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	row, err := st.GetG2STemplate(ctx, "template-generic-g2s-action")
	if err != nil {
		t.Fatalf("get template: %v", err)
	}
	if row == nil || row.CurrentVersionID != "2" {
		t.Fatalf("expected current_version_id=2, row=%+v", row)
	}
}

func TestPostTemplateVersionInvalidJSONShowsErrorAndDoesNotSave(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	body := url.Values{
		"version_label": {"3"},
		"actions_json":  {`{"actions":`},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates/template-generic-g2s-action/versions", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "invalid actions_json") {
		t.Fatalf("expected invalid actions_json error, body=%s", res.Body.String())
	}
	row, err := st.GetG2STemplateVersion(context.Background(), "template-generic-g2s-action", 3)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if row != nil {
		t.Fatalf("expected invalid version not saved, row=%+v", row)
	}
}

func TestTemplateRenderPreviewRendersSubstitutionAndNoJournalWrites(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	beforeRows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 200})
	if err != nil {
		t.Fatalf("list before messages: %v", err)
	}

	body := url.Values{
		"template_id":         {"template-generic-g2s-action"},
		"template_action_key": {"emergency_broadcast_silence"},
		"action_id":           {"emergency-broadcast-trigger"},
		"action_run_id":       {"run-preview"},
		"action_step_id":      {"step-1"},
		"egm_id":              {"EGM-001"},
		"host_id":             {"host-1"},
		"ip_address":          {"10.0.0.10"},
		"endpoint_path":       {"/g2s"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates/render-preview", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	responseBody := res.Body.String()
	for _, expected := range []string{
		"Preview Result",
		"Message Type",
		"emergency-broadcast-trigger",
		"EGM-001",
	} {
		if !strings.Contains(responseBody, expected) {
			t.Fatalf("expected %q in render preview output", expected)
		}
	}

	afterRows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 200})
	if err != nil {
		t.Fatalf("list after messages: %v", err)
	}
	if len(afterRows) != len(beforeRows) {
		t.Fatalf("render preview should not write message journal rows; before=%d after=%d", len(beforeRows), len(afterRows))
	}
}

func TestTemplateRenderPreviewMissingActionKeyShowsError(t *testing.T) {
	mux := setupOperatorServer(t)
	body := url.Values{
		"template_id":         {"template-generic-g2s-action"},
		"template_action_key": {"missing_action_key"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates/render-preview", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "template action key") || !strings.Contains(res.Body.String(), "not found") {
		t.Fatalf("expected missing action key error, body=%s", res.Body.String())
	}
}

func TestSettingsPageReturnsOKAndRendersIdentity(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/settings", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Appliance",
		"Controller",
		"controller-test",
		"Site",
		"Site Alpha",
		"Network Listener",
		"Operator Console URL",
		"Runtime",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/settings", expected)
		}
	}
}

func TestSettingsPageRendersBindAndDatabase(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/settings", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"127.0.0.1:8444",
		"C:\\\\data\\\\g2s.db",
		"https://127.0.0.1:8444/operator",
		"https://127.0.0.1:8444/g2s",
		"/g2s",
		"HOST-1",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in settings page", expected)
		}
	}
}

func TestSettingsPageRendersCertificateInventoryRows(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/settings", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Certificates",
		"web_server_cert",
		"/certs/web.crt",
		"OK",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in certificate section", expected)
		}
	}
}

func TestSettingsPageDoesNotExposePrivateKeyMaterial(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/settings", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, forbidden := range []string{
		"BEGIN PRIVATE KEY",
		"PRIVATE KEY",
		"/certs/web.key",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("settings page must not expose private key material: found %q", forbidden)
		}
	}
}

func TestSettingsPageHandlesEmptyCertificateInventory(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	if err := st.ReplaceCertificateInventory(context.Background(), nil); err != nil {
		t.Fatalf("clear cert inventory: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/settings", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "No certificate inventory recorded.") {
		t.Fatalf("expected empty certificate inventory message, body=%s", res.Body.String())
	}
}

func setupOperatorServer(t *testing.T) *http.ServeMux {
	mux, _ := setupOperatorServerWithStore(t)
	return mux
}

func setupOperatorServerWithStore(t *testing.T) (*http.ServeMux, *store.SQLiteStore) {
	t.Helper()
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	mux := http.NewServeMux()
	server := NewServer(st, defaultOperatorOptions(), allowMutation)
	server.RegisterRoutes(mux)
	return mux, st
}

func defaultOperatorOptions() Options {
	return Options{
		AppVersion:              "operator-console",
		ControllerID:            "controller-test",
		SiteName:                "Site Alpha",
		DatabasePath:            `C:\\data\\g2s.db`,
		ConfigPath:              `C:\\configs\\g2s.json`,
		BindAddress:             "127.0.0.1:8444",
		G2SHostURL:              "https://127.0.0.1:8444/g2s",
		G2SEndpointPath:         "/g2s",
		G2SHostID:               "HOST-1",
		TLSRequired:             true,
		ClientCertRequired:      true,
		WebLoginRequired:        true,
		AdminClientCertRequired: true,
		CAConfigured:            true,
		ClientCertConfigured:    true,
		ServerCertConfigured:    true,
		StartedAt:               time.Now().UTC().Add(-5 * time.Minute),
	}
}

func allowMutation(_ http.ResponseWriter, _ *http.Request) bool { return true }

func newTestStore(t *testing.T, ctx context.Context) *store.SQLiteStore {
	t.Helper()
	st, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return st
}

func seedOperatorData(t *testing.T, ctx context.Context, st *store.SQLiteStore) {
	t.Helper()
	now := time.Now().UTC()

	channels := []inputs.InputChannel{
		{ID: "regular-operation", Name: "Regular Operation", GPIOChannel: "GPIO16", Enabled: true, NormalState: inputs.InputStateHigh, CurrentState: inputs.InputStateHigh, DerivedState: inputs.DerivedStateNormal, DebounceMS: 100, Priority: 100, OnTriggerActionID: "regular-operation-trigger", LatchingMode: inputs.LatchingAutoClear},
		{ID: "general-broadcast", Name: "General Broadcast", GPIOChannel: "GPIO20", Enabled: true, NormalState: inputs.InputStateHigh, CurrentState: inputs.InputStateHigh, DerivedState: inputs.DerivedStateNormal, DebounceMS: 100, Priority: 200, OnTriggerActionID: "general-broadcast-trigger", OnNormalActionID: "general-broadcast-restore", LatchingMode: inputs.LatchingAutoClear},
		{ID: "emergency-broadcast", Name: "Emergency Broadcast", GPIOChannel: "GPIO21", Enabled: true, NormalState: inputs.InputStateHigh, CurrentState: inputs.InputStateLow, DerivedState: inputs.DerivedStateTriggered, DebounceMS: 120, Priority: 400, OnTriggerActionID: "emergency-broadcast-trigger", OnNormalActionID: "emergency-broadcast-restore", LatchingMode: inputs.LatchingManualClear},
		{ID: "local-notice", Name: "Local Notice", GPIOChannel: "GPIO26", Enabled: true, NormalState: inputs.InputStateHigh, CurrentState: inputs.InputStateHigh, DerivedState: inputs.DerivedStateNormal, DebounceMS: 80, Priority: 150, OnTriggerActionID: "local-notice-trigger", OnNormalActionID: "local-notice-restore", LatchingMode: inputs.LatchingAutoClear},
	}
	for _, row := range channels {
		if err := st.UpsertInputChannel(ctx, row); err != nil {
			t.Fatalf("upsert input: %v", err)
		}
	}
	if err := st.UpsertInputRuntimeState(ctx, inputruntime.InputRuntimeState{
		InputID:              "emergency-broadcast",
		StableRawState:       inputs.InputStateLow,
		DerivedState:         inputs.DerivedStateTriggered,
		LatchActive:          true,
		StableSince:          now,
		LastObservedRawState: inputs.InputStateLow,
		LastObservedAt:       now,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("upsert runtime: %v", err)
	}
	transitionID, err := st.RecordInputTransition(ctx, inputs.InputTransition{
		InputChannelID:  "emergency-broadcast",
		PreviousDerived: inputs.DerivedStateNormal,
		NewDerived:      inputs.DerivedStateTriggered,
		TransitionAt:    now,
	})
	if err != nil {
		t.Fatalf("record transition: %v", err)
	}

	def := func(id, name, key string, sev actions.ActionSeverity, restore string) actions.ActionDefinition {
		return actions.ActionDefinition{
			ID:               id,
			Name:             name,
			Severity:         sev,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary Step", Sequence: 0, TemplateActionKey: key}},
			ReturnActionID:   restore,
			Version:          1,
		}
	}
	actionRows := []actions.ActionDefinition{
		def("regular-operation-trigger", "Regular Operation Trigger", "regular_operation_notice", actions.SeverityNotice, ""),
		def("general-broadcast-trigger", "General Broadcast Trigger", "general_broadcast_notice", actions.SeverityBroadcast, "general-broadcast-restore"),
		def("general-broadcast-restore", "General Broadcast Restore", "general_broadcast_restore", actions.SeverityRestore, ""),
		def("emergency-broadcast-trigger", "Emergency Broadcast Trigger", "emergency_broadcast_silence", actions.SeverityEmergency, ""),
		def("emergency-broadcast-restore", "Emergency Broadcast Restore", "emergency_broadcast_restore", actions.SeverityRestore, ""),
		def("local-notice-trigger", "Local Notice Trigger", "local_notice", actions.SeverityNotice, "local-notice-restore"),
		def("local-notice-restore", "Local Notice Restore", "local_notice_restore", actions.SeverityRestore, ""),
		{
			ID:               "action-zero-targets",
			Name:             "Action Zero Targets",
			Severity:         actions.SeverityNotice,
			Enabled:          true,
			TargetSelector:   "EGM_IDS:EGM-404",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary Step", Sequence: 0, TemplateActionKey: "regular_operation_notice"}},
			Version:          1,
		},
		{
			ID:               "action-missing-template",
			Name:             "Action Missing Template",
			Severity:         actions.SeverityNotice,
			Enabled:          true,
			TargetSelector:   "EGM_IDS:EGM-003",
			TemplateSelector: "template-by-egm",
			Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary Step", Sequence: 0, TemplateActionKey: "regular_operation_notice"}},
			Version:          1,
		},
	}
	for _, row := range actionRows {
		if err := st.UpsertActionDefinition(ctx, row); err != nil {
			t.Fatalf("upsert action: %v", err)
		}
	}
	if _, err := st.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-1",
		ActionDefinitionID: "emergency-broadcast-trigger",
		InputTransitionID:  transitionID,
		StartedAt:          now,
		Status:             actions.RunStatusDispatchPrepared,
		TargetCount:        2,
	}); err != nil {
		t.Fatalf("create run: %v", err)
	}

	if err := st.UpsertG2STemplate(ctx, templates.G2STemplate{
		ID:     "template-generic-g2s-action",
		Name:   "Generic G2S Action Template",
		Vendor: "Generic",
		Status: templates.TemplateStatusActive,
	}); err != nil {
		t.Fatalf("upsert template: %v", err)
	}
	if err := st.UpsertG2STemplateVersion(ctx, templates.G2STemplateVersion{
		ID:           "template-generic-g2s-action-v1",
		TemplateID:   "template-generic-g2s-action",
		VersionLabel: "1",
		ActionsJSON:  `{"actions":{"emergency_broadcast_silence":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"emergency_broadcast_restore":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"general_broadcast_notice":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"general_broadcast_restore":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"local_notice":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"local_notice_restore":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"regular_operation_notice":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"}}}`,
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}
	if err := st.SetActiveG2STemplateVersion(ctx, "template-generic-g2s-action", 1); err != nil {
		t.Fatalf("set active template: %v", err)
	}

	egmRows := []egms.EGMRecord{
		{EGMID: "EGM-001", DisplayName: "Cabinet 001", Enabled: true, EmergencyEnabled: true, TemplateID: "template-generic-g2s-action", CurrentActionState: egms.EGMActionStatePending},
		{EGMID: "EGM-002", DisplayName: "Cabinet 002", Enabled: true, EmergencyEnabled: true, TemplateID: "template-generic-g2s-action", CurrentActionState: egms.EGMActionStatePending},
		{EGMID: "EGM-003", DisplayName: "Cabinet 003", Enabled: true, EmergencyEnabled: false, CurrentActionState: egms.EGMActionStatePending},
	}
	for _, row := range egmRows {
		if err := st.UpsertEGMRecord(ctx, row); err != nil {
			t.Fatalf("upsert egm: %v", err)
		}
	}

	messageID, err := st.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:         now,
		Direction:         g2sengine.DirectionOutbound,
		FromEndpoint:      "from://controller",
		ToEndpoint:        "to://cabinet/EGM-001",
		EGMID:             "EGM-001",
		ActionRunID:       "run-1",
		InputTransitionID: transitionID,
		TemplateID:        "template-generic-g2s-action",
		TemplateVersion:   "1",
		MessageType:       "emergency_broadcast_silence",
		RawPayload:        `<message action="emergency-broadcast-trigger" egm="EGM-001"/>`,
		ParsedSummaryJSON: `{"summary":"dry-run rendered","egm_id":"EGM-001"}`,
		Result:            g2sengine.MessageResultSendBlocked,
		TransportMode:     "HTTP",
		Error:             "send_disabled",
		HTTPStatusCode:    403,
		LatencyMS:         17,
		ResponseExcerpt:   "blocked",
	})
	if err != nil {
		t.Fatalf("record message: %v", err)
	}

	if _, err := st.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:        now,
		Severity:          audit.AuditSeverityEmergency,
		EventType:         audit.EventTypeInputTransition,
		Summary:           "Input transition recorded",
		DetailJSON:        `{"egm_id":"EGM-001","note":"transition detail"}`,
		InputTransitionID: transitionID,
		ActionRunID:       "run-1",
		MessageJournalID:  messageID,
		Operator:          "operator",
	}); err != nil {
		t.Fatalf("record audit: %v", err)
	}

	if err := st.ReplaceCertificateInventory(ctx, []model.CertificateInventory{
		{Role: "web_server_cert", Path: "/certs/web.crt", Status: "OK", LastCheckedAt: now},
	}); err != nil {
		t.Fatalf("replace cert inventory: %v", err)
	}
}
