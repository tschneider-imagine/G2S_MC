package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardRouteServesHTML(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "G2S Muting Controller") {
		t.Fatalf("expected dashboard title")
	}
	if !strings.Contains(rr.Body.String(), "Appliance Readiness") {
		t.Fatalf("expected appliance readiness panel")
	}
	if !strings.Contains(rr.Body.String(), "Certificate Summary") {
		t.Fatalf("expected certificate summary panel")
	}
	if !strings.Contains(rr.Body.String(), "Certificate Manager") {
		t.Fatalf("expected certificate manager panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"cert-manager-form\"") {
		t.Fatalf("expected certificate manager form")
	}
	if !strings.Contains(rr.Body.String(), "id=\"cert-role-rules\"") {
		t.Fatalf("expected certificate manager role rules block")
	}
	if !strings.Contains(rr.Body.String(), "id=\"cert-validation-summary\"") {
		t.Fatalf("expected certificate manager validation summary")
	}
	if !strings.Contains(rr.Body.String(), "id=\"cert-api-token\"") {
		t.Fatalf("expected certificate manager token field")
	}
	if !strings.Contains(rr.Body.String(), "data-export-role=\"g2s_client_cert\"") {
		t.Fatalf("expected quick certificate export actions")
	}
	if !strings.Contains(rr.Body.String(), "First Cabinet Session") {
		t.Fatalf("expected first cabinet session panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"first-cabinet-overall\"") {
		t.Fatalf("expected first cabinet session state marker")
	}
	if !strings.Contains(rr.Body.String(), "Mute Path vs Runbook Readiness") {
		t.Fatalf("expected mute path separation panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"mute-path-state\"") {
		t.Fatalf("expected mute path state marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runbook-readiness-state\"") {
		t.Fatalf("expected runbook readiness state marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"mute-path-prep-status\"") {
		t.Fatalf("expected cabinet prep status marker")
	}
	if !strings.Contains(rr.Body.String(), "Session Evidence Capture") {
		t.Fatalf("expected session evidence capture panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-evidence-json-button\"") {
		t.Fatalf("expected session evidence export button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-evidence-save-button\"") {
		t.Fatalf("expected session evidence save button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-evidence-export-all-button\"") {
		t.Fatalf("expected session evidence export-all button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-evidence-run-marker-count\"") {
		t.Fatalf("expected session evidence run marker count")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-evidence-heartbeat-health\"") {
		t.Fatalf("expected session evidence heartbeat health")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-evidence-heartbeat-source\"") {
		t.Fatalf("expected session evidence heartbeat source")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-evidence-selected\"") {
		t.Fatalf("expected selected saved evidence detail area")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-alert\"") {
		t.Fatalf("expected operator alert strip")
	}
	if !strings.Contains(rr.Body.String(), "EGM Focus") {
		t.Fatalf("expected egm focus panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-focus-select\"") {
		t.Fatalf("expected egm focus selector")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-grouped-summary\"") {
		t.Fatalf("expected grouped egm summary panel marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"selected-egm-detail\"") {
		t.Fatalf("expected selected egm detail marker")
	}
	if !strings.Contains(rr.Body.String(), "/readyz Primary") {
		t.Fatalf("expected explicit readyz indicator")
	}
	if !strings.Contains(rr.Body.String(), "data-filter=\"unhealthy\"") {
		t.Fatalf("expected unhealthy filter tab")
	}
	if !strings.Contains(rr.Body.String(), "Source</th>") {
		t.Fatalf("expected EGM source column")
	}
	if !strings.Contains(rr.Body.String(), "Cabinet Run Timeline") {
		t.Fatalf("expected cabinet run timeline panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"cabinet-run-timeline\"") {
		t.Fatalf("expected cabinet run timeline marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"timeline-grouping-label\"") {
		t.Fatalf("expected timeline grouping label marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"run-marker-start-button\"") {
		t.Fatalf("expected run marker start action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"run-marker-message\"") {
		t.Fatalf("expected run marker status message")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-drill-form\"") {
		t.Fatalf("expected operator drill form")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-drill-egm-id\"") {
		t.Fatalf("expected operator drill egm selector")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-drill-comms-online-button\"") {
		t.Fatalf("expected operator drill comms online action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-drill-resume-button\"") {
		t.Fatalf("expected operator drill resume action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"run-report-form\"") {
		t.Fatalf("expected run report form")
	}
	if !strings.Contains(rr.Body.String(), "id=\"run-report-start-marker\"") {
		t.Fatalf("expected run report start marker selector")
	}
	if !strings.Contains(rr.Body.String(), "id=\"run-report-end-marker\"") {
		t.Fatalf("expected run report end marker selector")
	}
	if !strings.Contains(rr.Body.String(), "id=\"run-report-json-button\"") {
		t.Fatalf("expected run report json export button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"run-report-markdown-button\"") {
		t.Fatalf("expected run report markdown export button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"heartbeat-policy-form\"") {
		t.Fatalf("expected heartbeat policy form")
	}
	if !strings.Contains(rr.Body.String(), "id=\"heartbeat-policy-warning-after-missed\"") {
		t.Fatalf("expected heartbeat policy warning threshold input")
	}
	if !strings.Contains(rr.Body.String(), "id=\"heartbeat-policy-block-after-missed\"") {
		t.Fatalf("expected heartbeat policy blocker threshold input")
	}
	if !strings.Contains(rr.Body.String(), "data-timeline-filter=\"heartbeat\"") {
		t.Fatalf("expected heartbeat timeline filter")
	}
	if !strings.Contains(rr.Body.String(), "id=\"heartbeat-health\"") {
		t.Fatalf("expected heartbeat summary health field")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-history-grouping\"") {
		t.Fatalf("expected egm history grouping marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"first-cabinet-session-workflow\"") {
		t.Fatalf("expected first cabinet session workflow marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-readiness-model\"") {
		t.Fatalf("expected operator readiness model marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"next-operator-actions\"") {
		t.Fatalf("expected next operator actions marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-evidence-egm-groups\"") {
		t.Fatalf("expected session evidence grouped egm count marker")
	}
	if !strings.Contains(rr.Body.String(), "data-sort-key=\"last_seen\"") {
		t.Fatalf("expected sortable last seen header")
	}
	if !strings.Contains(rr.Body.String(), "id=\"api-failure-banner\"") {
		t.Fatalf("expected api failure banner")
	}
	if !strings.Contains(rr.Body.String(), "Cabinet Identity Profile") {
		t.Fatalf("expected cabinet identity profile panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"cabinet-wire-host-url\"") {
		t.Fatalf("expected cabinet profile field markers")
	}
	if !strings.Contains(rr.Body.String(), "Cabinet Setup") {
		t.Fatalf("expected cabinet setup panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"setup-api-token\"") {
		t.Fatalf("expected cabinet setup auth token field")
	}
	if !strings.Contains(rr.Body.String(), "id=\"setup-token-controls\"") {
		t.Fatalf("expected cabinet setup token controls wrapper")
	}
	if !strings.Contains(rr.Body.String(), "id=\"setup-save-button\"") {
		t.Fatalf("expected cabinet setup save button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"setup-copy-token-button\"") {
		t.Fatalf("expected cabinet setup copy token button")
	}
	if strings.Contains(rr.Body.String(), "~/.g2s_api_token") || strings.Contains(rr.Body.String(), "cat ~/.g2s_api_token") {
		t.Fatalf("dashboard should not expose local token file instructions")
	}
}

func TestDashboardAssets(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	for _, path := range []string{"/static/dashboard.css", "/static/dashboard.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, rr.Code, http.StatusOK)
		}
		body := rr.Body.String()
		if path == "/static/dashboard.js" {
			for _, marker := range []string{"lastGoodStatus", "lastGoodAt", "inFlight", "pollIntervalMs", "nextBackoffMs", "renderAlerts", "fetchReadyz", "readyz unavailable", "showAPIFailureBanner", "(asc)", "renderCabinetProfile", "profile_source", "saveCabinetProfileOverride", "clearCabinetProfileOverride", "copySetupTokenToClipboard", "Authorization", "cabinetProfile", "heartbeatPolicy", "saveHeartbeatPolicyOverride", "clearHeartbeatPolicyOverride", "reloadHeartbeatPolicyForm", "currentHeartbeatPolicy", "runWindowIsActive", "certificateImport", "certificateExport", "importCertificateMaterial", "exportCertificateMaterial", "cert-manager-state", "validateCertificateManagerForm", "certificateRoleRulesText", "cert-role-export-policy", "allow_private_key_export", "cabinetPreflight", "renderFirstCabinetSession", "buildOperatorReadinessModel", "renderOperatorReadinessModel", "buildMutePathStatus", "renderMutePathStatus", "preflightCheckByID", "lab_warning_code=FIRST_TEST_EGM_IDS_PLACEHOLDER", "Replace placeholder first-test EGM IDs before real cabinet deployment", "selectedEGMDetailForSnapshot", "renderSelectedEGMDetail", "live_signal", "live_signal_detail", "first-cabinet-session-state", "mute_path", "runbook_readiness", "mute_path_note", "trusted_mutation_bypass_active", "mutationTokenRequired", "setup-token-help-text", "cert-token-help-text", "buildSessionEvidence", "buildSessionEvidenceMarkdown", "heartbeat_summary", "heartbeatEventTypes", "isHeartbeatEventType", "heartbeatSummary", "operatorDrillEvidence", "operator_drill", "renderHeartbeatSummary", "run_markers", "session-evidence-state", "action_model", "next_operator_actions", "selected_egm_detail", "exportSessionEvidenceJSON", "saveSessionEvidenceToHistory", "viewSavedSessionEvidence", "exportSavedSessionEvidenceJSON", "exportSavedSessionEvidenceMarkdown", "exportAllSavedSessionEvidence", "deleteSavedSessionEvidence", "session-evidence-export-all-button", "session-evidence-selected", "buildCabinetRunTimeline", "renderCabinetRunTimeline", "timeline-filter-tab", "timeline-count", "timeline-filter-label", "timelineEntryHTML", "renderGroupedTimelineAll", "runMarkers", "run-marker-start-button", "submitRunMarker", "renderRunMarkerControls", "operatorDrill", "renderOperatorDrill", "submitOperatorDrillAction", "operator-drill-comms-online-button", "operator-drill-pause-button", "boundedRunReport", "buildRunReportMarkdown", "run-report-start-marker", "run-report-end-marker", "renderRunReportControls", "exportRunReportJSON", "exportRunReportMarkdown", "sessionEvidence", "egmSourcePill", "data-egm-source=\\\"discovered\\\"", "renderEGMFocusControl", "renderEGMGroupedSummary", "buildEGMGroupedSummaryRows", "renderEGMHistory", "buildCabinetSessionWorkflow", "setEGMFocus", "egmFocusID", "egm_focus", "grouped_summary_scope", "egm_grouped_summary"} {
				if !strings.Contains(body, marker) {
					t.Fatalf("%s missing marker %q", path, marker)
				}
			}
		}
		if path == "/static/dashboard.css" {
			for _, marker := range []string{"alert-strip", "stale-warning", "stale-critical", "filter-tabs", "summary-blocking", "cert-item", "api-banner", "source-pill", "cabinet-warning", "setup-form", "token-help", "trusted-bypass-hidden", "validation-list", "secondary-button", "cert-manager-form", "cert-role-summary", "cert-manager-details", "cert-manager-detail", "cert-impact", "first-cabinet-session-panel", "first-cabinet-session-blockers", "first-cabinet-session-workflow", "first-cabinet-session-workflow-step", "first-cabinet-session-actions-wrap", "mute-path-status-wrap", "mute-path-summary-grid", "mute-path-status-card", "runbook-readiness-status-card", "operator-action-summary-grid", "operator-readiness-model", "operator-readiness-group", "focus-selected-egm-wrap", "selected-egm-detail", "evidence-capture-panel", "evidence-actions", "session-evidence-history-item", "session-evidence-history-actions", "session-evidence-selected-detail", "timeline-entry", "timeline-entry-head", "timeline-entry-tags", "timeline-group-heading", "timeline-egm-chip", "timeline-scope-global", "timeline-kind", "timeline-kind-marker", "timeline-kind-heartbeat", "cabinet-run-panel", "timeline-toolbar", "timeline-filter-tabs", "run-marker-form", "run-marker-grid", "run-marker-notes-label", "operator-drill-form", "operator-drill-grid", "operator-drill-actions", "run-report-form", "run-report-grid", "run-report-details", "heartbeat-summary-wrap", "heartbeat-summary-grid", "heartbeat-summary-message", "egm-source", "egm-source-discovered", "focus-controls", "focus-egm-summary-wrap", "egm-grouped-summary", "egm-focus-panel"} {
				if !strings.Contains(body, marker) {
					t.Fatalf("%s missing marker %q", path, marker)
				}
			}
		}
	}
}
