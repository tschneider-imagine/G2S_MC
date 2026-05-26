package operatorui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/incidents"
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

func TestOperatorLivePageRendersOperationalPanels(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Current Operation",
		"Active Inputs",
		"Active Actions",
		"EGM Attention",
		"Pending Delivery",
		"Recent Messages",
		"Recent Audit Events",
		"Last Updated",
		"/operator/live.json",
		"/operator/inputs",
		"/operator/actions",
		"/operator/egms",
		"/operator/comms",
		"/operator/audit",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator", expected)
		}
	}
}

func TestOperatorLiveJSONReturnsNoStoreAndSummaries(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/live.json", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control=%q", got)
	}

	var payload LiveView
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.GeneratedAt.IsZero() {
		t.Fatal("expected generated_at")
	}
	if payload.CurrentOperation != "Emergency Broadcast" {
		t.Fatalf("current_operation=%q", payload.CurrentOperation)
	}
	if payload.ActiveInputID != "emergency-broadcast" {
		t.Fatalf("active_input_id=%q", payload.ActiveInputID)
	}
	if payload.ActiveLatchCount < 1 {
		t.Fatalf("active_latch_count=%d", payload.ActiveLatchCount)
	}
	if len(payload.RecentMessages) == 0 {
		t.Fatal("expected recent message rows")
	}
	if payload.PendingDeliveryCount < 0 {
		t.Fatalf("pending_delivery_count=%d", payload.PendingDeliveryCount)
	}
	if len(payload.RecentAuditEvents) == 0 {
		t.Fatal("expected recent audit rows")
	}
}

func TestOperatorLiveJSONShowsPendingDeliveryCount(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if _, err := st.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:   time.Now().UTC(),
		Direction:   g2sengine.DirectionOutbound,
		EGMID:       "EGM-001",
		ActionRunID: "run-1",
		RawPayload:  "<prepared/>",
		Result:      g2sengine.MessageResultPrepared,
	}); err != nil {
		t.Fatalf("seed pending message: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/live.json", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var payload LiveView
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.PendingDeliveryCount < 1 {
		t.Fatalf("pending_delivery_count=%d", payload.PendingDeliveryCount)
	}
}

func TestOperatorLiveJSONMultipleActiveInputsDeterministicPrimary(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := st.UpsertInputRuntimeState(ctx, inputruntime.InputRuntimeState{
		InputID:              "general-broadcast",
		StableRawState:       inputs.InputStateLow,
		DerivedState:         inputs.DerivedStateTriggered,
		LatchActive:          false,
		StableSince:          now,
		LastObservedRawState: inputs.InputStateLow,
		LastObservedAt:       now,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("upsert runtime: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/live.json", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	var payload LiveView
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.ActiveInputCount < 2 {
		t.Fatalf("active_input_count=%d", payload.ActiveInputCount)
	}
	if payload.CurrentOperation != "Multiple Active" {
		t.Fatalf("current_operation=%q", payload.CurrentOperation)
	}
	if payload.ActiveInputID != "emergency-broadcast" {
		t.Fatalf("active_input_id=%q want emergency-broadcast", payload.ActiveInputID)
	}
}

func TestOperatorLiveShowsFailedAndEscalatedActionRuns(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := st.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-failed",
		ActionDefinitionID: "general-broadcast-trigger",
		StartedAt:          now,
		Status:             actions.RunStatusFailed,
		TargetCount:        2,
		FailedCount:        1,
	}); err != nil {
		t.Fatalf("create failed run: %v", err)
	}
	if _, err := st.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-escalated",
		ActionDefinitionID: "local-notice-trigger",
		StartedAt:          now,
		Status:             actions.RunStatusEscalated,
		TargetCount:        2,
		EscalatedCount:     1,
	}); err != nil {
		t.Fatalf("create escalated run: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"run-failed", "FAILED", "run-escalated", "ESCALATED"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in live page", expected)
		}
	}
}

func TestOperatorLiveDoesNotTreatSupersededRunAsActive(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := st.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-superseded",
		ActionDefinitionID: "general-broadcast-trigger",
		StartedAt:          now,
		Status:             actions.RunStatusSuperseded,
		TargetCount:        1,
	}); err != nil {
		t.Fatalf("create superseded run: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/live.json", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var payload LiveView
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, row := range payload.ActiveActions {
		if row.RunID == "run-superseded" {
			t.Fatalf("superseded run should not appear in active actions: %+v", payload.ActiveActions)
		}
	}
}

func TestOperatorLiveShowsEGMAttentionForMissingTemplate(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, "EGM-003") {
		t.Fatalf("expected EGM-003 attention row, body=%s", body)
	}
	if !strings.Contains(body, "Template not assigned") {
		t.Fatalf("expected missing template attention reason, body=%s", body)
	}
}

func TestOperatorLiveShowsActiveIncident(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"Active Incident", "/operator/audit?incident_id=", "emergency-broadcast"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in live page", expected)
		}
	}
}

func TestAuditPageSupportsIncidentFilter(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	runs, err := st.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list action runs: %v", err)
	}
	if len(runs) == 0 || strings.TrimSpace(runs[0].IncidentID) == "" {
		t.Fatalf("expected seeded incident-linked run: %+v", runs)
	}
	incidentID := strings.TrimSpace(runs[0].IncidentID)

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit?incident_id="+incidentID, nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"Incident ID", "Active Incident", incidentID, "run-1"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in incident audit filter page", expected)
		}
	}
}

func TestAuditEvidenceExportByIncidentIncludesIncidentPackage(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	runs, err := st.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list action runs: %v", err)
	}
	if len(runs) == 0 || strings.TrimSpace(runs[0].IncidentID) == "" {
		t.Fatalf("expected seeded incident-linked run: %+v", runs)
	}
	incidentID := strings.TrimSpace(runs[0].IncidentID)

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit/evidence-export?incident_id="+incidentID, nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var payload auditEvidencePackage
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Incident == nil {
		t.Fatal("expected incident in evidence package")
	}
	if strconv.FormatInt(payload.Incident.ID, 10) != incidentID {
		t.Fatalf("incident id=%d want=%s", payload.Incident.ID, incidentID)
	}
	if len(payload.ActionRuns) == 0 || len(payload.Messages) == 0 || len(payload.AuditTimeline) == 0 {
		t.Fatalf("expected linked evidence in package: %+v", payload)
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

func TestAuditPageRendersFilterAndEvidenceExportControls(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		`name="action_run_id"`,
		`name="input_transition_id"`,
		`name="egm_id"`,
		`name="limit"`,
		"/operator/audit/export",
		"/operator/audit/evidence-export",
		"Operator Note",
		"Add Operator Note",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/audit", expected)
		}
	}
}

func TestAuditPageFilterActionRunShowsRelatedEvidence(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := st.CreateActionTargetResult(ctx, actions.ActionTargetResult{
		ActionRunID:  "run-1",
		TargetEGMID:  "EGM-001",
		Status:       actions.TargetStatusConfirmed,
		AttemptCount: 1,
		LastResultAt: &now,
	}); err != nil {
		t.Fatalf("seed target result: %v", err)
	}
	rows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 20, ActionRunID: "run-1"})
	if err != nil {
		t.Fatalf("list run messages: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected seeded run message")
	}
	if err := st.UpdateMessageJournalResult(ctx, rows[0].ID, g2sengine.MessageResultSuperseded, "superseded for export coverage", "", 0, 0, rows[0].TransportMode, rows[0].SentAt, &now); err != nil {
		t.Fatalf("mark superseded message: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit?action_run_id=run-1", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Related Action Runs",
		"Related Targets",
		"Related Messages",
		"run-1",
		"emergency-broadcast-trigger",
		"EGM-001",
		"CONFIRMED",
		"emergency_broadcast_silence",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/audit?action_run_id=run-1", expected)
		}
	}
}

func TestAuditPageFilterTransitionShowsRelatedTransition(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	transitions, err := st.ListInputTransitions(ctx, 10)
	if err != nil {
		t.Fatalf("list transitions: %v", err)
	}
	if len(transitions) == 0 {
		t.Fatal("expected seeded input transition")
	}
	transitionID := transitions[0].ID

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit?input_transition_id="+strconv.FormatInt(transitionID, 10), nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Related Input Transition",
		"emergency-broadcast",
		"TRIGGERED",
		strconv.FormatInt(transitionID, 10),
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/audit?input_transition_id=...", expected)
		}
	}
}

func TestAuditNotesRequiresMutationAuth(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	mux := http.NewServeMux()
	server := NewServer(st, defaultOperatorOptions(), func(w http.ResponseWriter, _ *http.Request) bool {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	})
	server.RegisterRoutes(mux)

	body := url.Values{
		"note": {"operator note"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/audit/notes", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=401", res.Code)
	}
}

func TestAuditNotesRecordsAuditEntry(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	transitions, err := st.ListInputTransitions(ctx, 10)
	if err != nil {
		t.Fatalf("list transitions: %v", err)
	}
	if len(transitions) == 0 {
		t.Fatal("expected seeded transition")
	}
	transitionID := transitions[0].ID

	body := url.Values{
		"action_run_id":       {"run-1"},
		"input_transition_id": {strconv.FormatInt(transitionID, 10)},
		"message_id":          {"1"},
		"egm_id":              {"EGM-001"},
		"actor":               {"operator"},
		"note":                {"Confirmed operator note path"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/audit/notes", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "Operator note recorded.") {
		t.Fatalf("expected operator note confirmation, body=%s", res.Body.String())
	}

	rows, err := st.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{
		Limit:     20,
		EventType: audit.EventTypeOperatorAction,
	})
	if err != nil {
		t.Fatalf("list audit timeline: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected operator note audit row")
	}
	if rows[0].Summary != "Operator Note" {
		t.Fatalf("summary=%q want Operator Note", rows[0].Summary)
	}
	if !strings.Contains(rows[0].DetailJSON, "Confirmed operator note path") {
		t.Fatalf("expected note detail text, row=%+v", rows[0])
	}
}

func TestAuditEvidenceExportIncludesRelatedDataAndNoPrivateKeys(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := st.CreateActionTargetResult(ctx, actions.ActionTargetResult{
		ActionRunID:  "run-1",
		TargetEGMID:  "EGM-001",
		Status:       actions.TargetStatusConfirmed,
		AttemptCount: 1,
		LastResultAt: &now,
	}); err != nil {
		t.Fatalf("seed target result: %v", err)
	}
	messageRows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{
		Limit:       20,
		ActionRunID: "run-1",
	})
	if err != nil {
		t.Fatalf("list message rows: %v", err)
	}
	if len(messageRows) == 0 {
		t.Fatal("expected seeded message rows")
	}
	if err := st.UpdateMessageJournalResult(
		ctx,
		messageRows[0].ID,
		g2sengine.MessageResultSuperseded,
		"superseded for export coverage",
		"",
		0,
		0,
		messageRows[0].TransportMode,
		messageRows[0].SentAt,
		&now,
	); err != nil {
		t.Fatalf("update message row to superseded: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit/evidence-export?action_run_id=run-1&limit=200", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	contentDisposition := res.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "emergency-evidence-") {
		t.Fatalf("content-disposition=%q", contentDisposition)
	}

	var payload auditEvidencePackage
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.GeneratedAt.IsZero() {
		t.Fatal("expected generated_at")
	}
	if payload.Filters.ActionRunID != "run-1" {
		t.Fatalf("filters.action_run_id=%q", payload.Filters.ActionRunID)
	}
	if len(payload.AuditTimeline) == 0 {
		t.Fatal("expected audit timeline rows")
	}
	if len(payload.Messages) == 0 {
		t.Fatal("expected message rows")
	}
	foundSuperseded := false
	for _, row := range payload.Messages {
		if row.Result == g2sengine.MessageResultSuperseded {
			foundSuperseded = true
			break
		}
	}
	if !foundSuperseded {
		t.Fatalf("expected superseded message evidence row, rows=%+v", payload.Messages)
	}
	if len(payload.ActionRuns) == 0 {
		t.Fatal("expected action run rows")
	}
	if len(payload.TargetResults) == 0 {
		t.Fatal("expected target results")
	}
	if len(payload.InputTransitions) == 0 {
		t.Fatal("expected input transitions")
	}
	if strings.Contains(strings.ToUpper(res.Body.String()), "PRIVATE KEY") {
		t.Fatalf("evidence export must not contain private key material")
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
		"/operator/comms/handler-rules",
		"Create Handler Rule",
		"from://controller",
		"to://cabinet/EGM-001",
		"run-1",
		"template-generic-g2s-action@1",
		"emergency_broadcast_silence",
		"SEND_BLOCKED",
		"Match",
		"EXPECTED:message_ok",
		"dry-run rendered",
		"emergency-broadcast-trigger",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in comms page", expected)
		}
	}
}

func TestCommsHandlerRulesListRenders(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if err := st.UpsertHandlerRule(ctx, g2sengine.HandlerRule{
		ID:        "rule-ack",
		Name:      "ACK Confirmation",
		Enabled:   true,
		Direction: g2sengine.HandlerRuleDirectionInbound,
		MatchJSON: `{"contains":["accepted"]}`,
		Outcome:   g2sengine.HandlerRuleOutcomeConfirmation,
	}); err != nil {
		t.Fatalf("upsert handler rule: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/comms/handler-rules", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"Handler Rules", "rule-ack", "ACK Confirmation"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in handler rules page", expected)
		}
	}
}

func TestCommsHandlerRuleNewPrefillsFromMessage(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	rows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 1})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected seeded message")
	}
	messageID := rows[0].ID

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/comms/handler-rules/new?message_id="+strconv.FormatInt(messageID, 10), nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"Selected Message", "Direction", "Message Payload", "Match Preview"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q on new handler rule page", expected)
		}
	}
}

func TestCommsHandlerRulePreviewShowsMatch(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	rows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 1})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected seeded message")
	}
	messageID := rows[0].ID
	body := url.Values{
		"id":           {"preview-rule"},
		"name":         {"Preview Rule"},
		"enabled":      {"true"},
		"direction":    {"OUTBOUND"},
		"outcome":      {"NOTE"},
		"message_id":   {strconv.FormatInt(messageID, 10)},
		"match_json":   {`{"contains":["emergency-broadcast-trigger"]}`},
		"template_id":  {"template-generic-g2s-action"},
		"message_type": {"emergency_broadcast_silence"},
		"egm_id":       {"EGM-001"},
		"mode":         {"preview"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/comms/handler-rules", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "Match: <span class=\"mono\">yes</span>") {
		t.Fatalf("expected preview match yes, body=%s", res.Body.String())
	}
}

func TestCommsHandlerRulePostRequiresMutationAuth(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	mux := http.NewServeMux()
	server := NewServer(st, defaultOperatorOptions(), func(http.ResponseWriter, *http.Request) bool { return false })
	server.RegisterRoutes(mux)

	body := url.Values{
		"id":         {"rule-auth"},
		"name":       {"Auth Rule"},
		"enabled":    {"true"},
		"direction":  {"INBOUND"},
		"outcome":    {"NOTE"},
		"match_json": {`{"contains":["accepted"]}`},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/comms/handler-rules", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=401 body=%s", res.Code, res.Body.String())
	}
}

func TestCommsHandlerRulePostCreatesRuleAndAuditEntry(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	body := url.Values{
		"id":           {"rule-create"},
		"name":         {"Created Rule"},
		"enabled":      {"true"},
		"direction":    {"INBOUND"},
		"outcome":      {"FAILURE"},
		"match_json":   {`{"contains":["rejected"]}`},
		"template_id":  {"template-generic-g2s-action"},
		"message_type": {"ACK"},
		"egm_id":       {"EGM-001"},
		"action_id":    {"emergency-broadcast-trigger"},
		"mode":         {"save"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/comms/handler-rules", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	row, err := st.GetHandlerRule(ctx, "rule-create")
	if err != nil {
		t.Fatalf("get handler rule: %v", err)
	}
	if row == nil || row.Name != "Created Rule" || row.Outcome != g2sengine.HandlerRuleOutcomeFailure {
		t.Fatalf("unexpected handler rule: %+v", row)
	}
	auditRows, err := st.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{Limit: 200})
	if err != nil {
		t.Fatalf("list audit rows: %v", err)
	}
	found := false
	for _, auditRow := range auditRows {
		if auditRow.EventType == audit.EventTypeHandlerRule && strings.Contains(auditRow.Summary, "Handler Rule") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected handler rule audit event in rows: %+v", auditRows)
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

func TestCommsPageRendersInboundRow(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if _, err := st.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:         time.Now().UTC(),
		Direction:         g2sengine.DirectionInbound,
		FromEndpoint:      "10.0.0.10:9443",
		ToEndpoint:        "/g2s",
		EGMID:             "EGM-001",
		ActionRunID:       "run-1",
		MessageType:       "ACK",
		RawPayload:        `<ack status="accepted"/>`,
		ParsedSummaryJSON: `{"match_outcome":"EXPECTED"}`,
		Result:            g2sengine.MessageResultReceived,
	}); err != nil {
		t.Fatalf("seed inbound row: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/comms", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"INBOUND", "accepted"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/comms", expected)
		}
	}
}

func TestCommsPageRendersPreparedMessageResult(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if _, err := st.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:         time.Now().UTC(),
		Direction:         g2sengine.DirectionOutbound,
		EGMID:             "EGM-001",
		ActionRunID:       "run-1",
		ActionStepID:      "step-1",
		TemplateID:        "template-generic-g2s-action",
		TemplateVersion:   "1",
		MessageType:       "emergency_broadcast_silence",
		RawPayload:        `<prepared/>`,
		ParsedSummaryJSON: `{"reason":"Awaiting inbound confirmation from EGM"}`,
		Result:            g2sengine.MessageResultPrepared,
	}); err != nil {
		t.Fatalf("seed prepared row: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/comms", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "PREPARED") {
		t.Fatalf("expected PREPARED in /operator/comms, body=%s", res.Body.String())
	}
}

func TestCommsPageRendersPendingLifecycleResults(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	rows := []g2sengine.MessageResult{
		g2sengine.MessageResultOffered,
		g2sengine.MessageResultConfirmed,
		g2sengine.MessageResultFailed,
		g2sengine.MessageResultExpired,
		g2sengine.MessageResultSuperseded,
	}
	for i, result := range rows {
		if _, err := st.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
			Timestamp:       time.Now().UTC().Add(time.Duration(i) * time.Second),
			Direction:       g2sengine.DirectionOutbound,
			EGMID:           "EGM-001",
			ActionRunID:     "run-1",
			ActionStepID:    "step-1",
			TemplateID:      "template-generic-g2s-action",
			TemplateVersion: "1",
			MessageType:     "emergency_broadcast_silence",
			RawPayload:      "<message/>",
			Result:          result,
		}); err != nil {
			t.Fatalf("seed row %d: %v", i, err)
		}
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/comms", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"OFFERED", "CONFIRMED", "FAILED", "EXPIRED", "SUPERSEDED"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/comms", expected)
		}
	}
}

func TestAuditPageRendersInboundAuditRows(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if _, err := st.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:       time.Now().UTC(),
		Severity:         audit.AuditSeverityInfo,
		EventType:        audit.EventTypeMessageReceived,
		Summary:          "Inbound message received",
		ActionRunID:      "run-1",
		MessageJournalID: 1,
		Operator:         "listener",
	}); err != nil {
		t.Fatalf("seed inbound audit row: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"MESSAGE_RECEIVED", "Inbound message received"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/audit", expected)
		}
	}
}

func TestAuditPageRendersSupersededByReturnEvent(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if _, err := st.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:  time.Now().UTC(),
		Severity:    audit.AuditSeverityInfo,
		EventType:   audit.EventTypeReturnToNormal,
		Summary:     "Action run superseded by return-to-normal success",
		ActionRunID: "run-1",
		Operator:    "operator",
	}); err != nil {
		t.Fatalf("seed supersede audit row: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"RETURN_TO_NORMAL", "superseded by return-to-normal success"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/audit", expected)
		}
	}
}

func TestAuditPageRendersPreparedDeliveryAuditRows(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if _, err := st.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:  time.Now().UTC(),
		Severity:    audit.AuditSeverityInfo,
		EventType:   audit.EventTypeMessagePrepared,
		Summary:     "Message prepared for host listener delivery",
		ActionRunID: "run-1",
		Operator:    "operator",
	}); err != nil {
		t.Fatalf("seed prepared audit row: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"MESSAGE_PREPARED", "Message prepared for host listener delivery"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/audit", expected)
		}
	}
}

func TestAuditPageRendersOfferedAndExpiredAuditRows(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	for _, row := range []audit.AuditTimelineEntry{
		{
			OccurredAt:  time.Now().UTC(),
			Severity:    audit.AuditSeverityInfo,
			EventType:   audit.EventTypeMessageOffered,
			Summary:     "Prepared message offered to EGM-001",
			ActionRunID: "run-1",
			Operator:    "listener",
		},
		{
			OccurredAt:  time.Now().UTC(),
			Severity:    audit.AuditSeverityWarning,
			EventType:   audit.EventTypeMessageExpired,
			Summary:     "Pending delivery expired while waiting confirmation",
			ActionRunID: "run-1",
			Operator:    "listener",
		},
	} {
		if _, err := st.RecordAuditTimelineEntry(ctx, row); err != nil {
			t.Fatalf("seed audit row: %v", err)
		}
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/audit", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"MESSAGE_OFFERED", "MESSAGE_EXPIRED"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator/audit", expected)
		}
	}
}

func TestOperatorLiveRendersWaitingConfirmationAction(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := st.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-waiting-confirm",
		ActionDefinitionID: "emergency-broadcast-trigger",
		StartedAt:          now,
		Status:             actions.RunStatusWaitingConfirmation,
		TargetCount:        1,
	}); err != nil {
		t.Fatalf("create waiting run: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{"run-waiting-confirm", "WAITING_CONFIRMATION"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in /operator live page", expected)
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
		"Groups",
		"Group Members",
		"Export Registry",
		"Import Registry",
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

func TestPostEGMGroupsCreatesOrUpdatesGroup(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	body := url.Values{
		"group_id":    {"group-main-floor"},
		"name":        {"Main Floor"},
		"description": {"Primary cabinets"},
		"egm_ids":     {"EGM-001, EGM-002"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/egms/groups", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	group, err := st.GetEGMGroup(context.Background(), "group-main-floor")
	if err != nil {
		t.Fatalf("get group: %v", err)
	}
	if group == nil {
		t.Fatal("expected group saved")
	}
	if group.Name != "Main Floor" || group.Description != "Primary cabinets" {
		t.Fatalf("unexpected group fields: %+v", *group)
	}
	if len(group.EGMIDs) != 2 || group.EGMIDs[0] != "EGM-001" || group.EGMIDs[1] != "EGM-002" {
		t.Fatalf("unexpected group members: %+v", group.EGMIDs)
	}
}

func TestEGMExportReturnsJSONWithEGMsAndGroups(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if err := st.UpsertEGMGroup(ctx, egms.EGMGroup{ID: "group-east-bank", Name: "East Bank", EGMIDs: []string{"EGM-001"}}); err != nil {
		t.Fatalf("upsert group: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/egms/export", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(strings.ToLower(res.Header().Get("Content-Type")), "application/json") {
		t.Fatalf("content-type=%q", res.Header().Get("Content-Type"))
	}

	var payload struct {
		GeneratedAt         string           `json:"generated_at"`
		EGMs                []egms.EGMRecord `json:"egms"`
		Groups              []egms.EGMGroup  `json:"groups"`
		TemplatesReferenced []string         `json:"templates_referenced"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal export: %v", err)
	}
	if payload.GeneratedAt == "" || len(payload.EGMs) == 0 || len(payload.Groups) == 0 {
		t.Fatalf("unexpected export payload: %+v", payload)
	}
}

func TestEGMImportRequiresMutationAuth(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	mux := http.NewServeMux()
	server := NewServer(st, defaultOperatorOptions(), func(http.ResponseWriter, *http.Request) bool { return false })
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/operator/egms/import", strings.NewReader(`{"egms":[],"groups":[]}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=401 body=%s", res.Code, res.Body.String())
	}
}

func TestEGMImportRejectsMalformedJSON(t *testing.T) {
	mux := setupOperatorServer(t)
	req := httptest.NewRequest(http.MethodPost, "/operator/egms/import", strings.NewReader(`{"egms":`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "invalid import JSON") {
		t.Fatalf("expected malformed JSON error, body=%s", res.Body.String())
	}
}

func TestEGMImportUpsertsRecords(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	importJSON := `{
  "egms": [
    {
      "egm_id": "EGM-020",
      "display_name": "Cabinet 020",
      "ip_address": "10.20.0.20",
      "endpoint_path": "/g2s",
      "vendor": "Generic",
      "cabinet_family": "Family B",
      "game_title": "Title B",
      "software_version": "2.0.0",
      "zone": "Zone-B",
      "enabled": true,
      "emergency_enabled": true,
      "template_id": "template-generic-g2s-action",
      "current_action_state": "NORMAL",
      "notes": "Imported cabinet"
    }
  ],
  "groups": [
    {
      "id": "group-east-bank",
      "name": "East Bank",
      "description": "East side",
      "egm_ids": ["EGM-020"]
    }
  ]
}`
	req := httptest.NewRequest(http.MethodPost, "/operator/egms/import", strings.NewReader(importJSON))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	record, err := st.GetEGMRecord(context.Background(), "EGM-020")
	if err != nil {
		t.Fatalf("get imported egm: %v", err)
	}
	if record == nil || record.DisplayName != "Cabinet 020" {
		t.Fatalf("unexpected imported egm: %+v", record)
	}
	group, err := st.GetEGMGroup(context.Background(), "group-east-bank")
	if err != nil {
		t.Fatalf("get imported group: %v", err)
	}
	if group == nil || len(group.EGMIDs) != 1 || group.EGMIDs[0] != "EGM-020" {
		t.Fatalf("unexpected imported group: %+v", group)
	}
}

func TestEGMGroupsShowUnknownMemberWarning(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if err := st.UpsertEGMGroup(ctx, egms.EGMGroup{
		ID:          "group-unknown",
		Name:        "Unknown Members",
		Description: "Contains unknown member",
		EGMIDs:      []string{"EGM-001", "EGM-999"},
	}); err != nil {
		t.Fatalf("upsert group: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/egms", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "Unknown members: EGM-999") {
		t.Fatalf("expected unknown member warning, body=%s", res.Body.String())
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

func TestOperatorPagesRenderConfigurationValidationSummary(t *testing.T) {
	mux := setupOperatorServer(t)
	for _, path := range []string{"/operator/actions", "/operator/templates", "/operator/egms"} {
		res := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, res.Code, res.Body.String())
		}
		if !strings.Contains(res.Body.String(), "Configuration Validation") {
			t.Fatalf("expected configuration validation summary on %s", path)
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

func TestPostTemplateVersionBlocksActiveVersionOverwriteWhenInUse(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	before, err := st.GetG2STemplateVersion(ctx, "template-generic-g2s-action", 1)
	if err != nil {
		t.Fatalf("get before version: %v", err)
	}
	if before == nil {
		t.Fatal("expected active template version")
	}

	body := url.Values{
		"version_label": {"1"},
		"actions_json":  {`{"actions":{"emergency_broadcast_silence":{"message_type":"NOTICE","payload_template":"<changed/>"}}}`},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates/template-generic-g2s-action/versions", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "active template version is in use") {
		t.Fatalf("expected overwrite block error, body=%s", res.Body.String())
	}

	after, err := st.GetG2STemplateVersion(ctx, "template-generic-g2s-action", 1)
	if err != nil {
		t.Fatalf("get after version: %v", err)
	}
	if after == nil {
		t.Fatal("expected version row after blocked update")
	}
	if after.ActionsJSON != before.ActionsJSON {
		t.Fatalf("expected active version unchanged")
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

func TestTemplateMatcherPreviewExpectedOutcome(t *testing.T) {
	mux := setupOperatorServer(t)
	body := url.Values{
		"template_id":         {"template-generic-g2s-action"},
		"message_type":        {"NOTICE"},
		"raw_payload":         {"<ack>ok</ack>"},
		"parsed_summary_json": {`{"summary":"dry-run rendered"}`},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates/match-preview", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "Match Result") || !strings.Contains(res.Body.String(), "EXPECTED") {
		t.Fatalf("expected matcher expected outcome, body=%s", res.Body.String())
	}
}

func TestTemplateMatcherPreviewFailureOutcome(t *testing.T) {
	mux := setupOperatorServer(t)
	body := url.Values{
		"template_id":  {"template-generic-g2s-action"},
		"message_type": {"NOTICE"},
		"raw_payload":  {"<ack>error</ack>"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates/match-preview", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "Match Result") || !strings.Contains(res.Body.String(), "FAILURE") {
		t.Fatalf("expected matcher failure outcome, body=%s", res.Body.String())
	}
}

func TestTemplateMatcherPreviewInvalidJSONShowsError(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	if err := st.UpsertG2STemplateVersion(ctx, templates.G2STemplateVersion{
		ID:                    "template-generic-g2s-action-v4",
		TemplateID:            "template-generic-g2s-action",
		VersionLabel:          "4",
		ActionsJSON:           `{"actions":{"emergency_broadcast_silence":{"message_type":"NOTICE","payload_template":"<x/>"}}}`,
		ConfirmationRulesJSON: `{"rules":`,
	}); err != nil {
		t.Fatalf("upsert version: %v", err)
	}
	if err := st.SetActiveG2STemplateVersion(ctx, "template-generic-g2s-action", 4); err != nil {
		t.Fatalf("set active version: %v", err)
	}

	body := url.Values{
		"template_id": {"template-generic-g2s-action"},
		"raw_payload": {"<ack>ok</ack>"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates/match-preview", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "invalid matcher JSON") {
		t.Fatalf("expected invalid matcher JSON error, body=%s", res.Body.String())
	}
}

func TestTemplateMatcherPreviewDoesNotCreateMessageJournalEntries(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	beforeRows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 200})
	if err != nil {
		t.Fatalf("list before messages: %v", err)
	}
	body := url.Values{
		"template_id":         {"template-generic-g2s-action"},
		"message_type":        {"NOTICE"},
		"raw_payload":         {"<ack>ok</ack>"},
		"parsed_summary_json": {`{"summary":"dry-run rendered"}`},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/templates/match-preview", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	afterRows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 200})
	if err != nil {
		t.Fatalf("list after messages: %v", err)
	}
	if len(afterRows) != len(beforeRows) {
		t.Fatalf("match preview should not write message journal rows; before=%d after=%d", len(beforeRows), len(afterRows))
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
		"Runtime Version",
		"Build Revision",
		"Build Time",
		"Go Version",
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

func TestSettingsPageShowsDeliverySettings(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/settings", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Delivery Settings",
		"Input Runtime Enabled",
		"Seed Defaults",
		"Execute Actions",
		"Runtime Interval (ms)",
		"Delivery Topology",
		"Delivery Mode",
		"Pending Delivery Sweep",
		"Sweep Interval (ms)",
		"DISABLED",
		"Delivery Default",
		"Approved Delivery",
		"Delivery Timeout (ms)",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in settings page", expected)
		}
	}
}

func TestSettingsPageRendersMessageDeliveryCheckPanel(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/settings", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Message Delivery Check",
		"Run Message Delivery Check",
		"EGM ID",
		"Include Network Check",
		"Include TLS Check",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in settings page", expected)
		}
	}
}

func TestPostSettingsMessageDeliveryCheckReadOnly(t *testing.T) {
	mux := setupOperatorServer(t)
	form := url.Values{
		"egm_id": {"EGM-001"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/settings/message-delivery-check", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Message Delivery Check completed.",
		"Result",
		"Delivery Topology",
		"Endpoint Required",
		"Listener",
		"Host ID",
		"Endpoint",
		"Certificate Status",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in response body", expected)
		}
	}
}

func TestPostSettingsMessageDeliveryCheckNetworkRequiresAuth(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	mux := http.NewServeMux()
	server := NewServer(st, defaultOperatorOptions(), func(http.ResponseWriter, *http.Request) bool { return false })
	server.RegisterRoutes(mux)

	form := url.Values{
		"egm_id":                {"EGM-001"},
		"include_network_check": {"true"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/settings/message-delivery-check", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d body=%s", res.Code, http.StatusUnauthorized, res.Body.String())
	}
}

func TestPostSettingsMessageDeliveryCheckTLSRequiresAuth(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	mux := http.NewServeMux()
	server := NewServer(st, defaultOperatorOptions(), func(http.ResponseWriter, *http.Request) bool { return false })
	server.RegisterRoutes(mux)

	form := url.Values{
		"egm_id":            {"EGM-001"},
		"include_tls_check": {"true"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/settings/message-delivery-check", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d body=%s", res.Code, http.StatusUnauthorized, res.Body.String())
	}
}

func TestPostSettingsMessageDeliveryCheckUnauthorizedNetworkIsNonMutating(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	beforeMessages, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list message journal before: %v", err)
	}
	beforeRuns, err := st.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list runs before: %v", err)
	}
	beforeAudit, err := st.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list audit before: %v", err)
	}

	mux := http.NewServeMux()
	server := NewServer(st, defaultOperatorOptions(), func(http.ResponseWriter, *http.Request) bool { return false })
	server.RegisterRoutes(mux)

	form := url.Values{
		"egm_id":                {"EGM-001"},
		"include_network_check": {"true"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/settings/message-delivery-check", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d body=%s", res.Code, http.StatusUnauthorized, res.Body.String())
	}

	afterMessages, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list message journal after: %v", err)
	}
	afterRuns, err := st.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list runs after: %v", err)
	}
	afterAudit, err := st.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list audit after: %v", err)
	}

	if len(afterMessages) != len(beforeMessages) {
		t.Fatalf("message journal mutated on unauthorized check: before=%d after=%d", len(beforeMessages), len(afterMessages))
	}
	if len(afterRuns) != len(beforeRuns) {
		t.Fatalf("action runs mutated on unauthorized check: before=%d after=%d", len(beforeRuns), len(afterRuns))
	}
	if len(afterAudit) != len(beforeAudit) {
		t.Fatalf("audit timeline mutated on unauthorized check: before=%d after=%d", len(beforeAudit), len(afterAudit))
	}
}

func TestPostSettingsMessageDeliveryCheckReadOnlyIsNonMutating(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	beforeMessages, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list message journal before: %v", err)
	}
	beforeRuns, err := st.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list runs before: %v", err)
	}
	beforeAudit, err := st.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list audit before: %v", err)
	}

	mux := http.NewServeMux()
	server := NewServer(st, defaultOperatorOptions(), allowMutation)
	server.RegisterRoutes(mux)

	form := url.Values{
		"egm_id": {"EGM-001"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/settings/message-delivery-check", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	afterMessages, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list message journal after: %v", err)
	}
	afterRuns, err := st.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list runs after: %v", err)
	}
	afterAudit, err := st.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{Limit: 500})
	if err != nil {
		t.Fatalf("list audit after: %v", err)
	}

	if len(afterMessages) != len(beforeMessages) {
		t.Fatalf("message journal mutated by read-only check: before=%d after=%d", len(beforeMessages), len(afterMessages))
	}
	if len(afterRuns) != len(beforeRuns) {
		t.Fatalf("action runs mutated by read-only check: before=%d after=%d", len(beforeRuns), len(afterRuns))
	}
	if len(afterAudit) != len(beforeAudit) {
		t.Fatalf("audit timeline mutated by read-only check: before=%d after=%d", len(beforeAudit), len(afterAudit))
	}
}

func TestPostSettingsMessageDeliveryCheckRendersSelectedFieldsAndTLSResult(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	row, err := st.GetEGMRecord(ctx, "EGM-001")
	if err != nil {
		t.Fatalf("get egm: %v", err)
	}
	if row == nil {
		t.Fatal("missing EGM-001")
	}
	row.EndpointPath = "http://127.0.0.1:18080/g2s"
	if err := st.UpsertEGMRecord(ctx, *row); err != nil {
		t.Fatalf("upsert egm: %v", err)
	}

	form := url.Values{
		"egm_id":              {"EGM-001"},
		"template_id":         {"template-generic-g2s-action"},
		"template_action_key": {"emergency_broadcast_silence"},
		"include_tls_check":   {"true"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/settings/message-delivery-check", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Template Action Key",
		"emergency_broadcast_silence",
		"Template",
		"template-generic-g2s-action",
		"Delivery Topology",
		"HOST_LISTENER",
		"Endpoint Required",
		"no",
		"Endpoint",
		"TLS Check",
		"TLS check skipped for non-HTTPS endpoint",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in response body", expected)
		}
	}
}

func TestPostSettingsMessageDeliveryCheckHostListenerWithoutEndpointShowsWarningNotError(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	row, err := st.GetEGMRecord(ctx, "EGM-001")
	if err != nil {
		t.Fatalf("get egm: %v", err)
	}
	if row == nil {
		t.Fatal("missing EGM-001")
	}
	row.EndpointPath = ""
	if err := st.UpsertEGMRecord(ctx, *row); err != nil {
		t.Fatalf("upsert egm: %v", err)
	}

	form := url.Values{
		"egm_id":              {"EGM-001"},
		"template_id":         {"template-generic-g2s-action"},
		"template_action_key": {"emergency_broadcast_silence"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/settings/message-delivery-check", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"HOST_LISTENER",
		"Endpoint Required",
		"no",
		"Outbound endpoint is not configured; not required for host listener delivery.",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in response body", expected)
		}
	}
	if strings.Contains(strings.ToLower(body), "missing endpoint url") {
		t.Fatalf("expected no missing-endpoint error in host listener mode, body=%s", body)
	}
}

func TestPostSettingsMessageDeliveryCheckDoesNotExposePrivateKeyMaterial(t *testing.T) {
	mux := setupOperatorServer(t)
	form := url.Values{
		"egm_id": {"EGM-001"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/settings/message-delivery-check", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, forbidden := range []string{
		"BEGIN PRIVATE KEY",
		"PRIVATE KEY-----",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("delivery check output must not expose private key material: found %q", forbidden)
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
		"Certificate Status",
		"Configured",
		"File Exists",
		"Parse Status",
		"web_server_cert",
		"/certs/web.crt",
		"OK",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in certificate section", expected)
		}
	}
}

func TestSettingsPageDoesNotExposeCertificateSubjectOrIssuer(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	now := time.Now().UTC()
	if err := st.ReplaceCertificateInventory(context.Background(), []model.CertificateInventory{
		{
			Role:          "g2s_client_cert",
			Path:          "/certs/client.crt",
			Status:        "VALID",
			Subject:       "CN=fake-egm,OU=lab",
			Issuer:        "CN=Lab CA",
			LastCheckedAt: now,
		},
	}); err != nil {
		t.Fatalf("replace cert inventory: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/settings", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, forbidden := range []string{
		">Subject<",
		">Issuer<",
		"CN=fake-egm",
		"CN=Lab CA",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("settings page must not render certificate subject/issuer values: found %q", forbidden)
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

func TestSettingsPageSanitizesCertificateRuntimeNotePrivateKeyMaterial(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	now := time.Now().UTC()
	if err := st.ReplaceCertificateInventory(context.Background(), []model.CertificateInventory{
		{
			Role:          "g2s_client_key",
			Path:          "/certs/client.key",
			Status:        "INVALID",
			Error:         "load failed: -----BEGIN PRIVATE KEY-----\nabc123\n-----END PRIVATE KEY-----",
			LastCheckedAt: now,
		},
	}); err != nil {
		t.Fatalf("replace cert inventory: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/settings", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, forbidden := range []string{
		"BEGIN PRIVATE KEY",
		"END PRIVATE KEY",
		"abc123",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("settings page leaked key material %q body=%s", forbidden, body)
		}
	}
	if !strings.Contains(body, "private key material redacted") {
		t.Fatalf("expected redaction text in settings page, body=%s", body)
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

func TestInputsPageRendersExpectedInputsAndStates(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/inputs", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"regular-operation",
		"general-broadcast",
		"emergency-broadcast",
		"local-notice",
		"HIGH",
		"LOW",
		"NORMAL",
		"TRIGGERED",
		"Latch Active",
		"yes",
		"Recent Input Transitions",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in inputs page", expected)
		}
	}
}

func TestInputsPageIncludesLiveStateMarkers(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/inputs", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	for _, expected := range []string{
		"Live State",
		"Current State",
		`id="inputs-live-status"`,
		`id="inputs-live-updated"`,
		`id="inputs-live-latches"`,
		`data-input-id="emergency-broadcast"`,
		`data-field="raw_state"`,
		`data-field="derived_state"`,
		`data-field="latch_active"`,
		`data-field="last_observed_at"`,
		`data-field="last_transition"`,
		`/operator/inputs/live.json`,
		`/operator/inputs/fragments/transitions`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in inputs page", expected)
		}
	}
}

func TestInputsLiveJSONReturnsNoStoreAndFields(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/inputs/live.json", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control=%q", got)
	}
	var payload struct {
		GeneratedAt string `json:"generated_at"`
		Inputs      []struct {
			ID             string `json:"id"`
			RawState       string `json:"raw_state"`
			DerivedState   string `json:"derived_state"`
			LatchActive    bool   `json:"latch_active"`
			LastObservedAt string `json:"last_observed_at"`
			LastTransition string `json:"last_transition"`
		} `json:"inputs"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.GeneratedAt == "" {
		t.Fatal("expected generated_at")
	}
	if len(payload.Inputs) == 0 {
		t.Fatal("expected input rows")
	}
	var emergency *struct {
		ID             string `json:"id"`
		RawState       string `json:"raw_state"`
		DerivedState   string `json:"derived_state"`
		LatchActive    bool   `json:"latch_active"`
		LastObservedAt string `json:"last_observed_at"`
		LastTransition string `json:"last_transition"`
	}
	for i := range payload.Inputs {
		if payload.Inputs[i].ID == "emergency-broadcast" {
			emergency = &payload.Inputs[i]
			break
		}
	}
	if emergency == nil {
		t.Fatal("missing emergency-broadcast in live payload")
	}
	if emergency.RawState == "" || emergency.DerivedState == "" {
		t.Fatalf("expected raw/derived fields, row=%+v", *emergency)
	}
	if emergency.LastObservedAt == "" {
		t.Fatalf("expected last_observed_at, row=%+v", *emergency)
	}
	if emergency.LastTransition == "" {
		t.Fatalf("expected last_transition, row=%+v", *emergency)
	}
}

func TestInputsTransitionsFragmentRendersRows(t *testing.T) {
	mux := setupOperatorServer(t)
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/inputs/fragments/transitions", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control=%q", got)
	}
	body := res.Body.String()
	for _, expected := range []string{
		`data-transition-input-id="emergency-broadcast"`,
		`data-transition-summary="`,
		"TRIGGERED",
		"Input transition recorded",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in transition fragment", expected)
		}
	}
}

func TestInputsPageRendersMissingExpectedInputWarning(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	now := time.Now().UTC()
	rows := []inputs.InputChannel{
		{ID: "regular-operation", Name: "Regular Operation", GPIOChannel: "GPIO16", Enabled: true, NormalState: inputs.InputStateHigh, CurrentState: inputs.InputStateHigh, DerivedState: inputs.DerivedStateNormal, DebounceMS: 100, Priority: 100, OnTriggerActionID: "regular-operation-trigger", LatchingMode: inputs.LatchingAutoClear},
		{ID: "general-broadcast", Name: "General Broadcast", GPIOChannel: "GPIO20", Enabled: true, NormalState: inputs.InputStateHigh, CurrentState: inputs.InputStateHigh, DerivedState: inputs.DerivedStateNormal, DebounceMS: 100, Priority: 200, OnTriggerActionID: "general-broadcast-trigger", OnNormalActionID: "general-broadcast-restore", LatchingMode: inputs.LatchingAutoClear},
		{ID: "emergency-broadcast", Name: "Emergency Broadcast", GPIOChannel: "GPIO21", Enabled: true, NormalState: inputs.InputStateHigh, CurrentState: inputs.InputStateLow, DerivedState: inputs.DerivedStateTriggered, DebounceMS: 120, Priority: 400, OnTriggerActionID: "emergency-broadcast-trigger", OnNormalActionID: "emergency-broadcast-restore", LatchingMode: inputs.LatchingManualClear},
	}
	for _, row := range rows {
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

	mux := http.NewServeMux()
	server := NewServer(st, defaultOperatorOptions(), allowMutation)
	server.RegisterRoutes(mux)

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/operator/inputs", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "Missing configured input channels") || !strings.Contains(res.Body.String(), "local-notice") {
		t.Fatalf("expected missing input warning, body=%s", res.Body.String())
	}
}

func TestPostInputByIDUpdatesFields(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	body := url.Values{
		"normal_state":         {"LOW"},
		"debounce_ms":          {"321"},
		"latching_mode":        {"MANUAL_CLEAR"},
		"priority":             {"77"},
		"on_trigger_action_id": {"custom-trigger"},
		"on_normal_action_id":  {"custom-normal"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/inputs/local-notice", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	updated, err := st.GetInputChannel(context.Background(), "local-notice")
	if err != nil {
		t.Fatalf("get input: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated input")
	}
	if updated.NormalState != inputs.InputStateLow || updated.DebounceMS != 321 || updated.Priority != 77 {
		t.Fatalf("expected normal/debounce/priority updated, got %+v", *updated)
	}
	if updated.LatchingMode != inputs.LatchingManualClear || updated.OnTriggerActionID != "custom-trigger" || updated.OnNormalActionID != "custom-normal" {
		t.Fatalf("expected latch/actions updated, got %+v", *updated)
	}
}

func TestPostInputByIDInvalidDebounceDoesNotSave(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	before, err := st.GetInputChannel(context.Background(), "local-notice")
	if err != nil || before == nil {
		t.Fatalf("setup get input: %v", err)
	}
	body := url.Values{
		"normal_state":         {"HIGH"},
		"debounce_ms":          {"abc"},
		"latching_mode":        {"AUTO_CLEAR"},
		"priority":             {"150"},
		"on_trigger_action_id": {"local-notice-trigger"},
		"on_normal_action_id":  {"local-notice-restore"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/inputs/local-notice", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "invalid debounce") {
		t.Fatalf("expected invalid debounce error, body=%s", res.Body.String())
	}
	after, err := st.GetInputChannel(context.Background(), "local-notice")
	if err != nil || after == nil {
		t.Fatalf("get input after: %v", err)
	}
	if after.DebounceMS != before.DebounceMS {
		t.Fatalf("debounce must not change on invalid input; before=%d after=%d", before.DebounceMS, after.DebounceMS)
	}
}

func TestPostInputByIDInvalidPriorityDoesNotSave(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	before, err := st.GetInputChannel(context.Background(), "local-notice")
	if err != nil || before == nil {
		t.Fatalf("setup get input: %v", err)
	}
	body := url.Values{
		"normal_state":         {"HIGH"},
		"debounce_ms":          {"80"},
		"latching_mode":        {"AUTO_CLEAR"},
		"priority":             {"-3"},
		"on_trigger_action_id": {"local-notice-trigger"},
		"on_normal_action_id":  {"local-notice-restore"},
	}
	req := httptest.NewRequest(http.MethodPost, "/operator/inputs/local-notice", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "invalid priority") {
		t.Fatalf("expected invalid priority error, body=%s", res.Body.String())
	}
	after, err := st.GetInputChannel(context.Background(), "local-notice")
	if err != nil || after == nil {
		t.Fatalf("get input after: %v", err)
	}
	if after.Priority != before.Priority {
		t.Fatalf("priority must not change on invalid input; before=%d after=%d", before.Priority, after.Priority)
	}
}

func TestClearLatchRouteRequiresMutationAuth(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	mux := http.NewServeMux()
	server := NewServer(st, defaultOperatorOptions(), func(w http.ResponseWriter, _ *http.Request) bool {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	})
	server.RegisterRoutes(mux)

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operator/inputs/emergency-broadcast/clear-latch", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=401", res.Code)
	}
}

func TestClearLatchRouteWorksWhenPhysicalStateNormal(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := st.UpsertInputRuntimeState(ctx, inputruntime.InputRuntimeState{
		InputID:              "emergency-broadcast",
		StableRawState:       inputs.InputStateHigh,
		DerivedState:         inputs.DerivedStateTriggered,
		LatchActive:          true,
		StableSince:          now,
		LastObservedRawState: inputs.InputStateHigh,
		LastObservedAt:       now,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("upsert runtime: %v", err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/operator/inputs/emergency-broadcast/clear-latch", nil)
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, "Latch clear: emergency-broadcast") || !strings.Contains(body, "action queued: emergency-broadcast-restore") {
		t.Fatalf("expected clear-latch success message with queued action, body=%s", body)
	}
	state, err := st.GetInputRuntimeState(ctx, "emergency-broadcast")
	if err != nil || state == nil {
		t.Fatalf("get runtime state: %v", err)
	}
	if state.LatchActive {
		t.Fatal("expected latch to be inactive after clear")
	}
}

func TestClearLatchRouteQueuesReturnActionRun(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	channel, err := st.GetInputChannel(ctx, "emergency-broadcast")
	if err != nil || channel == nil {
		t.Fatalf("get input channel: %v", err)
	}
	channel.OnNormalActionID = "emergency-broadcast-normal"
	if err := st.UpsertInputChannel(ctx, *channel); err != nil {
		t.Fatalf("upsert channel: %v", err)
	}
	if err := st.UpsertActionDefinition(ctx, actions.ActionDefinition{
		ID:               "emergency-broadcast-normal",
		Name:             "Emergency Broadcast Normal",
		Severity:         actions.SeverityRestore,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary Step", Sequence: 0, TemplateActionKey: "emergency_broadcast_restore"}},
		Version:          1,
	}); err != nil {
		t.Fatalf("upsert action definition: %v", err)
	}
	if err := st.UpsertInputRuntimeState(ctx, inputruntime.InputRuntimeState{
		InputID:              "emergency-broadcast",
		StableRawState:       inputs.InputStateHigh,
		DerivedState:         inputs.DerivedStateTriggered,
		LatchActive:          true,
		StableSince:          now,
		LastObservedRawState: inputs.InputStateHigh,
		LastObservedAt:       now,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("upsert runtime: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/operator/inputs/emergency-broadcast/clear-latch", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	transition := latestManualClearTransition(t, st, "emergency-broadcast")
	runs, err := st.ListActionRuns(ctx, store.ActionRunListQuery{
		Limit:             20,
		InputTransitionID: transition.ID,
	})
	if err != nil {
		t.Fatalf("list action runs: %v", err)
	}
	var matched *actions.ActionRun
	for i := range runs {
		if runs[i].ActionDefinitionID == "emergency-broadcast-normal" {
			row := runs[i]
			matched = &row
			break
		}
	}
	if matched == nil {
		t.Fatalf("expected queued return action run for transition_id=%d", transition.ID)
	}
	if matched.InputTransitionID != transition.ID {
		t.Fatalf("run transition=%d want %d", matched.InputTransitionID, transition.ID)
	}
	if matched.Status != actions.RunStatusPending {
		t.Fatalf("run status=%q want PENDING", matched.Status)
	}
	if !strings.Contains(res.Body.String(), "run queued: "+matched.ID) {
		t.Fatalf("expected queued run id in response body, body=%s", res.Body.String())
	}
}

func TestClearLatchRouteExecutesQueuedRunWhenInputRuntimeExecutionEnabled(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t, ctx)
	t.Cleanup(func() { st.Close() })
	seedOperatorData(t, ctx, st)

	options := defaultOperatorOptions()
	options.InputRuntimeExecuteActions = true

	mux := http.NewServeMux()
	server := NewServer(st, options, allowMutation)
	server.RegisterRoutes(mux)

	now := time.Now().UTC()
	channel, err := st.GetInputChannel(ctx, "emergency-broadcast")
	if err != nil || channel == nil {
		t.Fatalf("get input channel: %v", err)
	}
	channel.OnNormalActionID = "emergency-broadcast-normal"
	if err := st.UpsertInputChannel(ctx, *channel); err != nil {
		t.Fatalf("upsert channel: %v", err)
	}
	if err := st.UpsertActionDefinition(ctx, actions.ActionDefinition{
		ID:               "emergency-broadcast-normal",
		Name:             "Emergency Broadcast Normal",
		Severity:         actions.SeverityRestore,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary Step", Sequence: 0, TemplateActionKey: "emergency_broadcast_restore"}},
		Version:          1,
	}); err != nil {
		t.Fatalf("upsert action definition: %v", err)
	}
	if err := st.UpsertInputRuntimeState(ctx, inputruntime.InputRuntimeState{
		InputID:              "emergency-broadcast",
		StableRawState:       inputs.InputStateHigh,
		DerivedState:         inputs.DerivedStateTriggered,
		LatchActive:          true,
		StableSince:          now,
		LastObservedRawState: inputs.InputStateHigh,
		LastObservedAt:       now,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("upsert runtime: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/operator/inputs/emergency-broadcast/clear-latch", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}

	transition := latestManualClearTransition(t, st, "emergency-broadcast")
	runs, err := st.ListActionRuns(ctx, store.ActionRunListQuery{
		Limit:             20,
		InputTransitionID: transition.ID,
	})
	if err != nil {
		t.Fatalf("list action runs: %v", err)
	}
	var matched *actions.ActionRun
	for i := range runs {
		if runs[i].ActionDefinitionID == "emergency-broadcast-normal" {
			row := runs[i]
			matched = &row
			break
		}
	}
	if matched == nil {
		t.Fatalf("expected queued return action run for transition_id=%d", transition.ID)
	}
	if matched.Status != actions.RunStatusWaitingConfirmation {
		t.Fatalf("run status=%q want WAITING_CONFIRMATION", matched.Status)
	}
	if !strings.Contains(res.Body.String(), "run status: WAITING_CONFIRMATION") {
		t.Fatalf("expected executed run status in response, body=%s", res.Body.String())
	}

	rows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{
		Limit:       50,
		ActionRunID: matched.ID,
	})
	if err != nil {
		t.Fatalf("list message journal: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected prepared message rows for action run %s", matched.ID)
	}
	foundPrepared := false
	for _, row := range rows {
		if row.Result == g2sengine.MessageResultPrepared {
			foundPrepared = true
		}
		if row.Result == g2sengine.MessageResultSendFailed {
			t.Fatalf("unexpected SEND_FAILED row in host listener execution: %+v", row)
		}
	}
	if !foundPrepared {
		t.Fatalf("expected PREPARED message result for action run %s", matched.ID)
	}
}

func TestClearLatchRouteSecondClearDoesNotCreateDuplicateRun(t *testing.T) {
	mux, st := setupOperatorServerWithStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	channel, err := st.GetInputChannel(ctx, "emergency-broadcast")
	if err != nil || channel == nil {
		t.Fatalf("get input channel: %v", err)
	}
	channel.OnNormalActionID = "emergency-broadcast-normal"
	if err := st.UpsertInputChannel(ctx, *channel); err != nil {
		t.Fatalf("upsert channel: %v", err)
	}
	if err := st.UpsertActionDefinition(ctx, actions.ActionDefinition{
		ID:               "emergency-broadcast-normal",
		Name:             "Emergency Broadcast Normal",
		Severity:         actions.SeverityRestore,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps:            []actions.ActionStep{{ID: "step-1", Name: "Primary Step", Sequence: 0, TemplateActionKey: "emergency_broadcast_restore"}},
		Version:          1,
	}); err != nil {
		t.Fatalf("upsert action definition: %v", err)
	}
	if err := st.UpsertInputRuntimeState(ctx, inputruntime.InputRuntimeState{
		InputID:              "emergency-broadcast",
		StableRawState:       inputs.InputStateHigh,
		DerivedState:         inputs.DerivedStateTriggered,
		LatchActive:          true,
		StableSince:          now,
		LastObservedRawState: inputs.InputStateHigh,
		LastObservedAt:       now,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("upsert runtime: %v", err)
	}

	firstReq := httptest.NewRequest(http.MethodPost, "/operator/inputs/emergency-broadcast/clear-latch", nil)
	firstRes := httptest.NewRecorder()
	mux.ServeHTTP(firstRes, firstReq)
	if firstRes.Code != http.StatusOK {
		t.Fatalf("first clear status=%d body=%s", firstRes.Code, firstRes.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/operator/inputs/emergency-broadcast/clear-latch", nil)
	secondRes := httptest.NewRecorder()
	mux.ServeHTTP(secondRes, secondReq)
	if secondRes.Code != http.StatusOK {
		t.Fatalf("second clear status=%d body=%s", secondRes.Code, secondRes.Body.String())
	}

	transitions, err := st.ListInputTransitions(ctx, 100)
	if err != nil {
		t.Fatalf("list transitions: %v", err)
	}
	clearTransitionCount := 0
	clearTransitionID := int64(0)
	for _, row := range transitions {
		if row.InputChannelID == "emergency-broadcast" && row.NewDerived == inputs.DerivedStateNormal && strings.Contains(strings.ToLower(row.Reason), "manual clear") {
			clearTransitionCount++
			clearTransitionID = row.ID
		}
	}
	if clearTransitionCount != 1 {
		t.Fatalf("manual clear transitions=%d want 1", clearTransitionCount)
	}

	runs, err := st.ListActionRuns(ctx, store.ActionRunListQuery{
		Limit:             50,
		InputTransitionID: clearTransitionID,
	})
	if err != nil {
		t.Fatalf("list action runs: %v", err)
	}
	matchCount := 0
	for _, row := range runs {
		if row.ActionDefinitionID == "emergency-broadcast-normal" {
			matchCount++
		}
	}
	if matchCount != 1 {
		t.Fatalf("queued return action runs=%d want 1", matchCount)
	}
}

func setupOperatorServer(t *testing.T) *http.ServeMux {
	mux, _ := setupOperatorServerWithStore(t)
	return mux
}

func latestManualClearTransition(t *testing.T, st *store.SQLiteStore, inputID string) inputs.InputTransition {
	t.Helper()
	rows, err := st.ListInputTransitions(context.Background(), 100)
	if err != nil {
		t.Fatalf("list transitions: %v", err)
	}
	for _, row := range rows {
		if row.InputChannelID == inputID && row.NewDerived == inputs.DerivedStateNormal && strings.Contains(strings.ToLower(row.Reason), "manual clear") {
			return row
		}
	}
	t.Fatalf("manual clear transition not found for input %s", inputID)
	return inputs.InputTransition{}
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
		AppVersion:               "operator-console",
		RuntimeVersion:           "dev",
		BuildRevision:            "abcdef123456",
		BuildTime:                "2026-05-25T00:00:00Z",
		GoVersion:                "go1.test",
		ControllerID:             "controller-test",
		SiteName:                 "Site Alpha",
		DatabasePath:             `C:\\data\\g2s.db`,
		ConfigPath:               `C:\\configs\\g2s.json`,
		BindAddress:              "127.0.0.1:8444",
		G2SHostURL:               "https://127.0.0.1:8444/g2s",
		G2SEndpointPath:          "/g2s",
		G2SHostID:                "HOST-1",
		TLSRequired:              true,
		ClientCertRequired:       true,
		WebLoginRequired:         true,
		AdminClientCertRequired:  true,
		CAConfigured:             true,
		ClientCertConfigured:     true,
		ServerCertConfigured:     true,
		DeliveryMode:             "DISABLED",
		DeliveryTopology:         string(g2stransport.DeliveryTopologyHostListener),
		AllowDeliveryDefault:     false,
		CaptureOnlyDefault:       false,
		DeliveryTimeoutMS:        5000,
		DeliveryEndpointDefaults: g2stransport.EndpointDefaults{Scheme: "https", Port: 8444},
		DeliveryClientConfig: g2stransport.HTTPClientConfig{
			TLSRequired:      true,
			RootCAPath:       "/certs/ca.crt",
			ClientCertPath:   "/certs/client.crt",
			ClientKeyPath:    "/certs/client.key",
			DefaultTimeoutMS: 5000,
		},
		InputRuntimeEnabled:            true,
		InputRuntimeSeedDefaults:       true,
		InputRuntimeExecuteActions:     false,
		InputRuntimeIntervalMS:         100,
		PendingDeliverySweepEnabled:    false,
		PendingDeliverySweepIntervalMS: 5000,
		StartedAt:                      time.Now().UTC().Add(-5 * time.Minute),
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
	incidentRecord, err := st.CreateIncidentRecord(ctx, incidents.IncidentRecord{
		OpenedAt:             now,
		Status:               incidents.StatusOpen,
		Severity:             "EMERGENCY",
		PrimaryInputID:       "emergency-broadcast",
		OpenedByTransitionID: transitionID,
		Summary:              "Emergency Broadcast triggered",
	})
	if err != nil {
		t.Fatalf("create incident: %v", err)
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
		IncidentID:         strconv.FormatInt(incidentRecord.ID, 10),
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
		ID:                    "template-generic-g2s-action-v1",
		TemplateID:            "template-generic-g2s-action",
		VersionLabel:          "1",
		ActionsJSON:           `{"actions":{"emergency_broadcast_silence":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"emergency_broadcast_restore":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"general_broadcast_notice":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"general_broadcast_restore":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"local_notice":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"local_notice_restore":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"},"regular_operation_notice":{"message_type":"NOTICE","payload_template":"<message action=\"{{.ActionID}}\" egm=\"{{.EGMID}}\"/>"}}}`,
		ConfirmationRulesJSON: `{"rules":[{"id":"message_ok","contains":["dry-run rendered"]}]}`,
		FailureRulesJSON:      `{"rules":[{"id":"message_error","contains":["error"]}]}`,
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
	groupRows := []egms.EGMGroup{
		{ID: "group-main-floor", Name: "Main Floor", Description: "Primary floor cabinets", EGMIDs: []string{"EGM-001", "EGM-002"}},
		{ID: "group-east-bank", Name: "East Bank", Description: "East-side cabinets", EGMIDs: []string{"EGM-002"}},
	}
	for _, row := range groupRows {
		if err := st.UpsertEGMGroup(ctx, row); err != nil {
			t.Fatalf("upsert egm group: %v", err)
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
