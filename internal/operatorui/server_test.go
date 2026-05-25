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

var forbiddenRuntimeTerms = []string{
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
	for _, label := range []string{"Live", "Inputs", "Actions", "Comms", "EGMs", "Templates", "Audit", "Settings"} {
		if !strings.Contains(body, ">"+label+"<") {
			t.Fatalf("missing nav label %q", label)
		}
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
		"/operator/readiness",
		"/operator/readiness.json",
		"/operator/settings/system-check",
		"/operator/settings/system-check.json",
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
	var rows []g2sengine.MessageJournalEntry
	if err := json.Unmarshal(res.Body.Bytes(), &rows); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected comms rows")
	}
}

func TestAuditAndEvidenceExportsStillWork(t *testing.T) {
	mux := setupOperatorServer(t)

	auditRes := httptest.NewRecorder()
	auditReq := httptest.NewRequest(http.MethodGet, "/operator/audit/export", nil)
	mux.ServeHTTP(auditRes, auditReq)
	if auditRes.Code != http.StatusOK {
		t.Fatalf("audit export status=%d body=%s", auditRes.Code, auditRes.Body.String())
	}
	var auditRows []audit.AuditTimelineEntry
	if err := json.Unmarshal(auditRes.Body.Bytes(), &auditRows); err != nil {
		t.Fatalf("audit unmarshal: %v", err)
	}
	if len(auditRows) == 0 {
		t.Fatal("expected audit rows")
	}

	evidenceRes := httptest.NewRecorder()
	evidenceReq := httptest.NewRequest(http.MethodGet, "/operator/export", nil)
	mux.ServeHTTP(evidenceRes, evidenceReq)
	if evidenceRes.Code != http.StatusOK {
		t.Fatalf("evidence export status=%d body=%s", evidenceRes.Code, evidenceRes.Body.String())
	}
	var payload OperatorEvidencePackage
	if err := json.Unmarshal(evidenceRes.Body.Bytes(), &payload); err != nil {
		t.Fatalf("evidence unmarshal: %v", err)
	}
	if payload.GeneratedAt.IsZero() {
		t.Fatal("expected generated timestamp")
	}
}

func setupOperatorServer(t *testing.T) *http.ServeMux {
	t.Helper()
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	mux := http.NewServeMux()
	server := NewServer(st, Options{RealSendDefaultDisabled: true}, allowMutation)
	server.RegisterRoutes(mux)
	return mux
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
		def("emergency-broadcast-trigger", "Emergency Broadcast Trigger", "emergency_broadcast_silence", actions.SeverityEmergency, "emergency-broadcast-restore"),
		def("emergency-broadcast-restore", "Emergency Broadcast Restore", "emergency_broadcast_restore", actions.SeverityRestore, ""),
		def("local-notice-trigger", "Local Notice Trigger", "local_notice", actions.SeverityNotice, "local-notice-restore"),
		def("local-notice-restore", "Local Notice Restore", "local_notice_restore", actions.SeverityRestore, ""),
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
	}
	for _, row := range egmRows {
		if err := st.UpsertEGMRecord(ctx, row); err != nil {
			t.Fatalf("upsert egm: %v", err)
		}
	}

	if _, err := st.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:       now,
		Direction:       g2sengine.DirectionOutbound,
		EGMID:           "EGM-001",
		ActionRunID:     "run-1",
		TemplateID:      "template-generic-g2s-action",
		TemplateVersion: "1",
		MessageType:     "emergency_broadcast_silence",
		RawPayload:      `<message action="emergency-broadcast-trigger" egm="EGM-001"/>`,
		Result:          g2sengine.MessageResultSendBlocked,
		TransportMode:   "HTTP",
		Error:           "send_disabled",
	}); err != nil {
		t.Fatalf("record message: %v", err)
	}

	if _, err := st.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:        now,
		Severity:          audit.AuditSeverityEmergency,
		EventType:         audit.EventTypeInputTransition,
		Summary:           "Input transition recorded",
		InputTransitionID: transitionID,
		ActionRunID:       "run-1",
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
