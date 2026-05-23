package ui

import (
	"net/http"
	"net/http/httptest"
	"regexp"
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
	if !regexp.MustCompile(`/static/dashboard\.css\?v=[0-9a-f]+`).MatchString(rr.Body.String()) {
		t.Fatalf("expected cache-busted dashboard css url")
	}
	if !regexp.MustCompile(`/static/dashboard\.js\?v=[0-9a-f]+`).MatchString(rr.Body.String()) {
		t.Fatalf("expected cache-busted dashboard js url")
	}
	if !strings.Contains(rr.Body.String(), "Service Status Details") {
		t.Fatalf("expected service status details panel")
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
	if !strings.Contains(rr.Body.String(), "id=\"cert-preview-button\"") {
		t.Fatalf("expected certificate preview action button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"cert-preview-summary\"") {
		t.Fatalf("expected certificate preview summary marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"cert-preview-list\"") {
		t.Fatalf("expected certificate preview list marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"cert-backup-list\"") {
		t.Fatalf("expected certificate backup history list marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"cert-backup-refresh-button\"") {
		t.Fatalf("expected certificate backup history refresh control")
	}
	if !strings.Contains(rr.Body.String(), "Backup History") {
		t.Fatalf("expected backup history section label")
	}
	if !strings.Contains(rr.Body.String(), "Operator Audit Timeline") {
		t.Fatalf("expected operator audit timeline panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-audit-action-filter\"") {
		t.Fatalf("expected operator audit action filter control")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-audit-result-filter\"") {
		t.Fatalf("expected operator audit result filter control")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-audit-search-filter\"") {
		t.Fatalf("expected operator audit search filter control")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-audit-list\"") {
		t.Fatalf("expected operator audit timeline list")
	}
	if !strings.Contains(rr.Body.String(), "data-export-role=\"g2s_client_cert\"") {
		t.Fatalf("expected quick certificate export actions")
	}
	if !strings.Contains(rr.Body.String(), "Cabinet Signal Monitor") {
		t.Fatalf("expected cabinet signal monitor panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"first-cabinet-overall\"") {
		t.Fatalf("expected first cabinet session state marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"first-cabinet-endpoint-alerts\"") {
		t.Fatalf("expected first cabinet endpoint integrity marker")
	}
	if !strings.Contains(rr.Body.String(), "Mute Path and System Check") {
		t.Fatalf("expected mute path/system check separation panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"mute-path-state\"") {
		t.Fatalf("expected mute path state marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"system-check-state\"") {
		t.Fatalf("expected system check state marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"mute-path-prep-status\"") {
		t.Fatalf("expected cabinet prep status marker")
	}
	if !strings.Contains(rr.Body.String(), "Evidence Capture") {
		t.Fatalf("expected evidence capture panel")
	}
	for _, marker := range []string{
		"Overview",
		"EGMs",
		"EGM Traffic",
		"Cabinet Signal",
		"Cabinet Signal Monitor",
		"Certificates",
		"Evidence",
		"Evidence Capture",
		"Settings",
		"Diagnostics",
		"id=\"dashboard-tab-menu\"",
		"data-dashboard-tab-button=\"overview\"",
		"data-dashboard-tab-button=\"diagnostics\"",
		"System Status",
		"Next Action",
	} {
		if !strings.Contains(rr.Body.String(), marker) {
			t.Fatalf("expected dashboard tab/status marker %q", marker)
		}
	}
	for _, legacy := range []string{
		"System Check Rules",
		"Blocker Governance",
		"First Cabinet Session",
		"Operator Readiness Model",
		"Runbook Readiness",
		"Session Evidence Capture",
		"Session Complete",
		"Current workflow step",
		"Cabinet prep can continue",
		"Escalation History",
		"Escalation Rationale",
		"Approve Selected Finding",
		"Revoke Selected Finding",
		"approve stop condition",
		"revoke stop condition",
		"escalation history",
		"rationale",
		"Ready Now",
		"Needs Operator Action",
		"Lab Warning",
		"Informational",
	} {
		if strings.Contains(rr.Body.String(), legacy) {
			t.Fatalf("expected legacy label to be removed from main html: %q", legacy)
		}
	}
	if strings.Contains(strings.ToLower(rr.Body.String()), "runbook") {
		t.Fatalf("expected runbook term removed from dashboard html")
	}
	for _, legacy := range []string{"validation detail", "evidence follow-up", "cabinet prep"} {
		if strings.Contains(strings.ToLower(rr.Body.String()), legacy) {
			t.Fatalf("expected legacy phrase removed from dashboard html: %q", legacy)
		}
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
	if !strings.Contains(rr.Body.String(), "Export All Captures") {
		t.Fatalf("expected export all captures action label")
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
	if !strings.Contains(rr.Body.String(), "id=\"api-connection-indicator\"") {
		t.Fatalf("expected api connection indicator")
	}
	if !strings.Contains(rr.Body.String(), "id=\"api-connection-state\"") {
		t.Fatalf("expected api connection state marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"api-connection-last-success\"") {
		t.Fatalf("expected api connection last success marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"api-connection-poll-attempts\"") {
		t.Fatalf("expected api connection poll attempts marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"api-connection-startup\"") {
		t.Fatalf("expected api connection startup marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"api-connection-last-error\"") {
		t.Fatalf("expected api connection last error marker")
	}
	if !strings.Contains(rr.Body.String(), "Operator Actions") {
		t.Fatalf("expected operator actions bar")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-actions-refresh-button\"") {
		t.Fatalf("expected operator actions refresh button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-actions-capture-evidence-button\"") {
		t.Fatalf("expected operator actions capture evidence button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-actions-export-package-button\"") {
		t.Fatalf("expected operator actions export package button")
	}
	if !strings.Contains(rr.Body.String(), "Global View Controls") {
		t.Fatalf("expected global view controls panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"global-severity-filter\"") {
		t.Fatalf("expected global severity filter control")
	}
	if !strings.Contains(rr.Body.String(), "id=\"global-text-filter\"") {
		t.Fatalf("expected global text filter control")
	}
	if !strings.Contains(rr.Body.String(), "id=\"global-compact-mode-toggle\"") {
		t.Fatalf("expected global compact mode toggle")
	}
	if !strings.Contains(rr.Body.String(), "id=\"global-view-summary\"") {
		t.Fatalf("expected global view summary marker")
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
	if !strings.Contains(rr.Body.String(), "data-filter=\"endpoint_integrity\"") {
		t.Fatalf("expected endpoint integrity filter tab")
	}
	if !strings.Contains(rr.Body.String(), "Source</th>") {
		t.Fatalf("expected EGM source column")
	}
	if !strings.Contains(rr.Body.String(), "Last Endpoint") {
		t.Fatalf("expected egm last endpoint column")
	}
	if !strings.Contains(rr.Body.String(), "Endpoint Drift") {
		t.Fatalf("expected egm endpoint drift column")
	}
	if !strings.Contains(rr.Body.String(), "Endpoint Integrity") {
		t.Fatalf("expected endpoint integrity panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"endpoint-integrity-filter-button\"") {
		t.Fatalf("expected endpoint integrity quick filter button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"endpoint-integrity-refresh-button\"") {
		t.Fatalf("expected endpoint integrity refresh button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"endpoint-integrity-custom-minutes\"") {
		t.Fatalf("expected endpoint integrity custom snooze minutes input")
	}
	if !strings.Contains(rr.Body.String(), "id=\"endpoint-integrity-counts\"") {
		t.Fatalf("expected endpoint integrity active/suppressed counts marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"endpoint-integrity-list\"") {
		t.Fatalf("expected endpoint integrity active warning list")
	}
	if !strings.Contains(rr.Body.String(), "id=\"endpoint-integrity-acked-list\"") {
		t.Fatalf("expected endpoint integrity acknowledged list")
	}
	if !strings.Contains(rr.Body.String(), "id=\"endpoint-integrity-snoozed-list\"") {
		t.Fatalf("expected endpoint integrity snoozed list")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-registry-drawer\"") {
		t.Fatalf("expected egm registry drawer panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-registry-form\"") {
		t.Fatalf("expected egm registry form")
	}
	if !strings.Contains(rr.Body.String(), "Promote Discovered EGM") {
		t.Fatalf("expected egm promote action label")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-registry-save-button\"") {
		t.Fatalf("expected egm registry save action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-registry-delete-button\"") {
		t.Fatalf("expected egm registry delete action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-registry-promote-button\"") {
		t.Fatalf("expected egm registry promote action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-registry-notes\"") {
		t.Fatalf("expected egm registry notes textarea")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-select-all-visible-button\"") {
		t.Fatalf("expected egm bulk select-all-visible action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-clear-selection-button\"") {
		t.Fatalf("expected egm bulk clear-selection action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-bulk-promote-button\"") {
		t.Fatalf("expected egm bulk promote action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-bulk-apply-profile-button\"") {
		t.Fatalf("expected egm bulk apply-to-profile action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"egm-select-all-checkbox\"") {
		t.Fatalf("expected egm table select-all checkbox")
	}
	if !strings.Contains(rr.Body.String(), "Add Selected to First-Test EGM IDs") {
		t.Fatalf("expected egm bulk apply label")
	}
	if !strings.Contains(rr.Body.String(), "table-wrap-scroll-safe") {
		t.Fatalf("expected scroll-safe roster table wrapper")
	}
	if !strings.Contains(rr.Body.String(), "panel-scroll-safe-audit") {
		t.Fatalf("expected scroll-safe audit list wrapper")
	}
	if !strings.Contains(rr.Body.String(), "panel-scroll-safe-history") {
		t.Fatalf("expected scroll-safe history wrapper")
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
	if !strings.Contains(rr.Body.String(), "id=\"timeline-rollup-heartbeat-toggle\"") {
		t.Fatalf("expected timeline heartbeat rollup toggle")
	}
	if !strings.Contains(rr.Body.String(), "id=\"timeline-show-raw-heartbeat-toggle\"") {
		t.Fatalf("expected timeline raw heartbeat toggle")
	}
	if !strings.Contains(rr.Body.String(), "id=\"timeline-heartbeat-mode-label\"") {
		t.Fatalf("expected timeline heartbeat mode label marker")
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
	if !strings.Contains(rr.Body.String(), "id=\"heartbeat-policy-interval\"") {
		t.Fatalf("expected heartbeat policy interval input")
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
	if !strings.Contains(rr.Body.String(), "id=\"session-package-export-button\"") {
		t.Fatalf("expected session package export action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-package-export-message\"") {
		t.Fatalf("expected session package export message marker")
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
	if !strings.Contains(rr.Body.String(), "Cabinet Profile Settings") {
		t.Fatalf("expected cabinet profile settings panel")
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
	if !strings.Contains(rr.Body.String(), "id=\"setup-use-observed-egms-button\"") {
		t.Fatalf("expected use observed egms button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"setup-observed-egms-preview\"") {
		t.Fatalf("expected observed egms preview marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"setup-copy-token-button\"") {
		t.Fatalf("expected cabinet setup copy token button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runtime-overrides-save-snapshot-button\"") {
		t.Fatalf("expected runtime override snapshot save action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runtime-overrides-restore-button\"") {
		t.Fatalf("expected runtime override snapshot restore action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runtime-overrides-restore-json\"") {
		t.Fatalf("expected runtime override restore JSON textarea")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runtime-overrides-restore-file\"") {
		t.Fatalf("expected runtime override restore file input")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runtime-overrides-message\"") {
		t.Fatalf("expected runtime override message marker")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runtime-preset-name\"") {
		t.Fatalf("expected runtime preset name input")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runtime-preset-note\"") {
		t.Fatalf("expected runtime preset note textarea")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runtime-preset-save-button\"") {
		t.Fatalf("expected runtime preset save action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runtime-preset-refresh-button\"") {
		t.Fatalf("expected runtime preset refresh action")
	}
	if !strings.Contains(rr.Body.String(), "id=\"runtime-presets-list\"") {
		t.Fatalf("expected runtime presets list marker")
	}
	if strings.Contains(rr.Body.String(), "~/.g2s_api_token") || strings.Contains(rr.Body.String(), "cat ~/.g2s_api_token") {
		t.Fatalf("dashboard should not expose local token file instructions")
	}
}

func TestDashboardRouteNoCacheHeaders(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	cacheControl := rr.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-store") || !strings.Contains(cacheControl, "no-cache") {
		t.Fatalf("expected no-cache dashboard headers, got %q", cacheControl)
	}
	if rr.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("expected Pragma no-cache, got %q", rr.Header().Get("Pragma"))
	}
	if rr.Header().Get("Expires") != "0" {
		t.Fatalf("expected Expires 0, got %q", rr.Header().Get("Expires"))
	}
}

func TestDashboardScriptRevalidationHeaders(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	firstReq := httptest.NewRequest(http.MethodGet, "/static/dashboard.js", nil)
	firstRec := httptest.NewRecorder()
	mux.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", firstRec.Code, http.StatusOK)
	}
	cacheControl := firstRec.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-cache") || !strings.Contains(cacheControl, "must-revalidate") {
		t.Fatalf("expected revalidation cache headers, got %q", cacheControl)
	}
	etag := firstRec.Header().Get("ETag")
	if strings.TrimSpace(etag) == "" {
		t.Fatalf("expected script etag")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/static/dashboard.js", nil)
	secondReq.Header.Set("If-None-Match", etag)
	secondRec := httptest.NewRecorder()
	mux.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusNotModified {
		t.Fatalf("revalidation status = %d, want %d", secondRec.Code, http.StatusNotModified)
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
			for _, marker := range []string{"lastGoodStatus", "lastGoodAt", "inFlight", "pollIntervalMs", "nextBackoffMs", "compactModeStorageKey", "globalSeverityFilter", "globalTextFilter", "compactMode", "normalizeGlobalSeverityFilter", "setGlobalSeverityFilter", "setGlobalTextFilter", "setCompactMode", "renderGlobalViewControls", "refreshGlobalFilteredViews", "operator-actions-refresh-button", "operator-actions-capture-evidence-button", "operator-actions-export-package-button", "global-severity-filter", "global-text-filter", "global-compact-mode-toggle", "g2s.dashboard.compact_mode", "renderAlerts", "fetchReadyz", "readyz unavailable", "showAPIFailureBanner", "(asc)", "renderCabinetProfile", "profile_source", "saveCabinetProfileOverride", "clearCabinetProfileOverride", "copySetupTokenToClipboard", "useObservedEGMSuggestions", "normalizeCabinetProfileSuggestions", "renderCabinetProfileSuggestions", "observed_egm_ids", "recommended_first_test_egm_ids", "cabinetProfileSuggestions", "sessionWorkflow", "normalizeSessionWorkflowProgress", "renderSessionWorkflowProgress", "saveSessionWorkflowProgress", "clearSessionWorkflowProgress", "workflow-progress-save-button", "workflow-progress-unsaved", "Authorization", "cabinetProfile", "heartbeatPolicy", "saveHeartbeatPolicyOverride", "clearHeartbeatPolicyOverride", "reloadHeartbeatPolicyForm", "currentHeartbeatPolicy", "runWindowIsActive", "operatorAudit", "operatorAuditEndpointURL", "normalizeOperatorAuditEvents", "renderOperatorAuditTimeline", "operatorAuditActionFilter", "operatorAuditResultFilter", "operatorAuditSearchFilter", "operator-audit-action-filter", "operator-audit-result-filter", "operator-audit-search-filter", "operator-audit-list", "operator-audit-summary", "operator-audit-entry", "sessionPackageExport", "exportSessionPackage", "session-package-export-button", "session-package-export-message", "/api/session-package/export", "runtimeOverridesSnapshot", "runtimeOverridesRestore", "runtimeOverridePresets", "runtimeOverridePresetSave", "runtimeOverridePresetLoad", "normalizeRuntimeOverridePresetListResponse", "runtimeOverridePresetsFromSnapshot", "renderRuntimePresetLibrary", "saveCurrentRuntimePreset", "loadRuntimePresetByName", "deleteRuntimePresetByName", "reloadRuntimePresetList", "setRuntimePresetMessage", "saveRuntimeOverridesSnapshot", "restoreRuntimeOverridesSnapshot", "loadRuntimeOverridesRestoreFile", "runtimeOverridesRestoreHeaders", "runtime-overrides-save-snapshot-button", "runtime-overrides-restore-button", "runtime-overrides-restore-json", "runtime-overrides-restore-file", "runtime-overrides-message", "runtime-preset-name", "runtime-preset-note", "runtime-preset-save-button", "runtime-preset-refresh-button", "runtime-presets-list", "runtime-preset-message", "runtime-preset-action-button", "data-runtime-preset-action", "/api/runtime-overrides/snapshot", "/api/runtime-overrides/restore", "/api/runtime-overrides/presets", "/api/runtime-overrides/presets/save", "/api/runtime-overrides/presets/load", "egmRegistry", "egmRegistryPromoteBulk", "egmRegistryApplyToCabinetProfile", "normalizeEGMRegistryResponse", "egmRegistryOverridesFromSnapshot", "selectedEGMBulkIDSet", "selectedEGMBulkIDs", "setAllVisibleEGMBulkSelection", "promoteSelectedEGMBulk", "applySelectedToCabinetProfileFirstTest", "renderEGMBulkSelectionSummary", "egm-select-all-visible-button", "egm-clear-selection-button", "egm-bulk-promote-button", "egm-bulk-apply-profile-button", "egm-select-all-checkbox", "egm-row-select-checkbox", "egm-bulk-selection-summary", "egm-bulk-action-message", "renderEGMRegistryDrawer", "saveSelectedEGMRegistry", "promoteSelectedEGMRegistry", "deleteSelectedEGMRegistryOverride", "egm-registry-drawer", "egm-registry-form", "egm-row-action-button", "data-egm-action", "/api/egm-registry", "/api/egm-registry/promote", "/api/egm-registry/promote-bulk", "/api/egm-registry/apply-to-cabinet-profile", "endpointIntegrityAlerts", "normalizeEndpointIntegrityAlertsResponse", "endpointIntegrityAlertSections", "postEndpointIntegrityAlertAction", "handleEndpointIntegrityAlertActionFromUI", "reloadEndpointIntegrityAlerts", "endpoint_collision_summary", "endpoint_collisions", "endpoint_collision_warning", "endpoint_collision_types", "renderEndpointIntegrity", "endpointCollisionTypeLabel", "normalizeEndpointCollisionRows", "endpoint-integrity-filter-button", "endpoint-integrity-refresh-button", "endpoint-integrity-custom-minutes", "endpoint-integrity-acked-list", "endpoint-integrity-snoozed-list", "data-endpoint-alert-action", "acked_at", "snoozed_until", "endpoint_integrity", "X-EGM-Focus", "certificateBackups", "certificateRestore", "certificatePreview", "certificateImport", "certificateExport", "loadCertificateBackups", "normalizeCertificateBackups", "renderCertificateBackupHistory", "restoreCertificateBackup", "certBackupsByRole", "cert-backup-list", "cert-backup-refresh-button", "cert-restore-backup-button", "previewCertificateMaterial", "normalizeCertificatePreview", "renderCertificatePreview", "certPreviewFingerprint", "certPreviewResult", "cert-preview-button", "cert-preview-summary", "cert-preview-list", "Run Preview before importing certificate material.", "importCertificateMaterial", "exportCertificateMaterial", "cert-manager-state", "validateCertificateManagerForm", "certificateRoleRulesText", "cert-role-export-policy", "allow_private_key_export", "cabinetPreflight", "renderFirstCabinetSession", "buildOperatorReadinessModel", "renderOperatorReadinessModel", "buildMutePathStatus", "renderMutePathStatus", "preflightCheckByID", "lab_warning_code=FIRST_TEST_EGM_IDS_PLACEHOLDER", "Replace placeholder first-test EGM IDs before real cabinet deployment", "selectedEGMDetailForSnapshot", "renderSelectedEGMDetail", "live_signal", "live_signal_detail", "last_endpoint_ip", "last_endpoint_port", "last_endpoint_seen_at", "endpoint_drift_warning", "endpoint_drift_ips", "recent_endpoints", "endpoint_warning_text", "Recent Endpoints (newest first)", "first-cabinet-session-state", "mute_path", "runbook_readiness", "mute_path_note", "trusted_mutation_bypass_active", "mutationTokenRequired", "setup-token-help-text", "cert-token-help-text", "buildSessionEvidence", "buildSessionEvidenceMarkdown", "heartbeat_summary", "heartbeatEventTypes", "isHeartbeatEventType", "heartbeatSummary", "egmHistoryEndpointURL", "rollup_heartbeat", "timelineRollupHeartbeat", "timelineShowRawHeartbeat", "historyRowIsKeepAliveRollup", "heartbeat_rollup_count", "heartbeat_rollup_first_seen_at", "heartbeat_rollup_last_seen_at", "timeline-rollup-heartbeat-toggle", "timeline-show-raw-heartbeat-toggle", "timeline-heartbeat-mode-label", "operatorDrillEvidence", "operator_drill", "renderHeartbeatSummary", "run_markers", "session-evidence-state", "action_model", "next_operator_actions", "selected_egm_detail", "exportSessionEvidenceJSON", "saveSessionEvidenceToHistory", "viewSavedSessionEvidence", "exportSavedSessionEvidenceJSON", "exportSavedSessionEvidenceMarkdown", "exportAllSavedSessionEvidence", "sessionEvidenceExportAll", "deleteSavedSessionEvidence", "window.confirm", "/api/session-evidence/", "session-evidence-export-all-button", "session-evidence-selected", "buildCabinetRunTimeline", "renderCabinetRunTimeline", "timeline-filter-tab", "timeline-count", "timeline-filter-label", "timelineEntryHTML", "renderGroupedTimelineAll", "runMarkers", "run-marker-start-button", "submitRunMarker", "renderRunMarkerControls", "operatorDrill", "renderOperatorDrill", "submitOperatorDrillAction", "operator-drill-comms-online-button", "operator-drill-pause-button", "boundedRunReport", "buildRunReportMarkdown", "run-report-start-marker", "run-report-end-marker", "renderRunReportControls", "exportRunReportJSON", "exportRunReportMarkdown", "sessionEvidence", "egmSourcePill", "data-egm-source=\\\"discovered\\\"", "renderEGMFocusControl", "renderEGMGroupedSummary", "buildEGMGroupedSummaryRows", "renderEGMHistory", "buildCabinetSessionWorkflow", "setEGMFocus", "egmFocusID", "egm_focus", "grouped_summary_scope", "egm_grouped_summary"} {
				if !strings.Contains(body, marker) {
					t.Fatalf("%s missing marker %q", path, marker)
				}
			}
			for _, marker := range []string{"heartbeat-policy-interval", "interval_ms", "Interval (ms) must be a whole number greater than zero.", "Heartbeat policy override saved", "Heartbeat policy override cleared"} {
				if !strings.Contains(body, marker) {
					t.Fatalf("%s missing marker %q", path, marker)
				}
			}
			if strings.Contains(body, "# Session Evidence Capture") {
				t.Fatalf("%s should not include legacy evidence heading", path)
			}
			if !strings.Contains(body, "# Evidence Capture") {
				t.Fatalf("%s missing evidence capture heading marker", path)
			}
		}
		if path == "/static/dashboard.css" {
			for _, marker := range []string{"alert-strip", "stale-warning", "stale-critical", "filter-tabs", "summary-blocking", "cert-item", "api-banner", "source-pill", "cabinet-warning", "setup-form", "token-help", "trusted-bypass-hidden", "validation-list", "secondary-button", "operator-toolbar-grid", "operator-actions-bar", "operator-actions-buttons", "global-view-controls-panel", "global-view-controls", "global-compact-toggle", "table-wrap-scroll-safe", "panel-scroll-safe", "panel-scroll-safe-audit", "panel-scroll-safe-history", "panel-scroll-safe-integrity", "anchor-target", "compact-mode", "cert-manager-form", "cert-role-summary", "cert-preview-wrap", "cert-preview-detail", "cert-preview-list", "cert-backup-history", "cert-backup-list", "cert-backup-item", "cert-backup-item-head", "cert-backup-meta", "cert-backup-actions", "operator-audit-filters", "operator-audit-summary", "operator-audit-list", "operator-audit-entry", "operator-audit-head", "operator-audit-meta", "operator-audit-detail", "operator-audit-pill-success", "operator-audit-pill-fail", "endpoint-integrity-panel", "egm-registry-drawer", "egm-registry-drawer-hidden", "egm-row-actions", "egm-bulk-actions", "egm-row-select-cell", "egm-row-select-checkbox", "runtime-overrides-upload-label", "endpoint-integrity-sections", "endpoint-integrity-section", "endpoint-integrity-section-head", "endpoint-integrity-custom-snooze-label", "endpoint-integrity-warning", "endpoint-integrity-warning-head", "endpoint-integrity-warning-meta", "endpoint-integrity-warning-actions", "cert-manager-details", "cert-manager-detail", "cert-impact", "first-cabinet-session-panel", "first-cabinet-session-blockers", "first-cabinet-session-workflow", "first-cabinet-session-workflow-step", "first-cabinet-session-workflow-progress-wrap", "workflow-progress-meta", "workflow-progress-unsaved", "workflow-progress-unsaved-dirty", "workflow-progress-steps", "workflow-progress-step", "first-cabinet-session-actions-wrap", "mute-path-status-wrap", "mute-path-summary-grid", "mute-path-status-card", "system-check-status-card", "operator-action-summary-grid", "operator-readiness-model", "operator-readiness-group", "focus-selected-egm-wrap", "selected-egm-detail", "evidence-capture-panel", "evidence-actions", "session-evidence-history-item", "session-evidence-history-actions", "session-evidence-selected-detail", "timeline-entry", "timeline-entry-head", "timeline-entry-tags", "timeline-group-heading", "timeline-egm-chip", "timeline-scope-global", "timeline-kind", "timeline-kind-marker", "timeline-kind-heartbeat", "cabinet-run-panel", "timeline-toolbar", "timeline-filter-tabs", "timeline-rollup-controls", "timeline-toggle", "run-marker-form", "run-marker-grid", "run-marker-notes-label", "operator-drill-form", "operator-drill-grid", "operator-drill-actions", "run-report-form", "run-report-grid", "run-report-details", "heartbeat-summary-wrap", "heartbeat-summary-grid", "heartbeat-summary-message", "blocker-governance-panel", "blocker-governance-summary", "blocker-governance-list", "blocker-governance-item", "blocker-governance-actions", "egm-source", "egm-source-discovered", "focus-controls", "focus-egm-summary-wrap", "egm-grouped-summary", "egm-focus-panel"} {
				if !strings.Contains(body, marker) {
					t.Fatalf("%s missing marker %q", path, marker)
				}
			}
		}
	}
}

func TestDashboardTelemetryReadinessFocusMarkers(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/static/dashboard.js", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		"telemetryRecentThresholdMS",
		"rowHasRecentTelemetry",
		"telemetryReadinessForSnapshot",
		"System Check or readyz is healthy; profile/certificate/auth checks are ready.",
		"No recent commsOnLine/keepAlive observed (threshold ",
		"Selected EGM not active; global telemetry is healthy.",
		"Selected EGM not active: ",
		"Recent EGM telemetry observed across ",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("dashboard.js missing marker %q", marker)
		}
	}
}

func TestDashboardJSUsesRelativeAPIPaths(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/static/dashboard.js", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if strings.Contains(body, "127.0.0.1") || strings.Contains(body, "localhost") {
		t.Fatalf("dashboard.js should not contain hardcoded loopback hosts")
	}
	for _, marker := range []string{
		`status: "/api/status"`,
		`readyz: "/readyz"`,
		`cabinetPreflight: "/api/cabinet-preflight"`,
		"fetchJSON(endpoints.status)",
		"fetchJSON(endpoints.cabinetPreflight)",
		"fetchReadyz()",
		"api-connection-indicator",
		"api-connection-poll-attempts",
		"api-connection-startup",
		"updateAPIConnectionIndicator",
		"updateAPIConnectionIndicator(\"attempting\")",
		"updateAPIConnectionIndicator(clientState.lastError ? \"disconnected\" : \"connected\")",
		"pollAttempts",
		"pollStarted",
		"jsLoadedAt",
		"safePollRender",
		"runStartupStep",
		"bootstrapDashboard",
		"poll startup failed",
		"schedulePoll(0)",
		"settledFailureSummary",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("dashboard.js missing marker %q", marker)
		}
	}
}

func TestDashboardJSSyntaxQuoteRegression(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/static/dashboard.js", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()

	for _, broken := range []string{
		`escapeHTML(item.message || \"-\")`,
		`escapeHTML((item.action || \"-\") + \" \" + (item.finding_id || \"\"))`,
		`escapeHTML(\"at \" + fmtTime(item.created_at)`,
		`<span class=\\\"muted-text\\\">`,
	} {
		if strings.Contains(body, broken) {
			t.Fatalf("dashboard.js contains broken escaped quote fragment %q", broken)
		}
	}
	for _, expected := range []string{
		`escapeHTML(item.message || "")`,
		`escapeHTML(fmtTime(item.created_at))`,
		`<span class=\"muted-text\">`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("dashboard.js missing expected quote fragment %q", expected)
		}
	}
}

func TestDashboardTabLayoutMarkers(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	htmlReq := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	htmlRec := httptest.NewRecorder()
	mux.ServeHTTP(htmlRec, htmlReq)
	if htmlRec.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want %d", htmlRec.Code, http.StatusOK)
	}
	htmlBody := htmlRec.Body.String()
	for _, marker := range []string{
		"id=\"dashboard-tab-menu\"",
		"data-dashboard-tab-button=\"overview\"",
		"data-dashboard-tab-button=\"egms\"",
		"data-dashboard-tab-button=\"cabinet-signal\"",
		"data-dashboard-tab-button=\"certificates\"",
		"data-dashboard-tab-button=\"evidence\"",
		"data-dashboard-tab-button=\"settings\"",
		"data-dashboard-tab-button=\"diagnostics\"",
		"data-dashboard-tab=\"overview,cabinet-signal\"",
		"data-dashboard-tab=\"certificates\"",
		"data-dashboard-tab=\"diagnostics\"",
		"Cabinet Signal Monitor",
		"Evidence Capture",
		"System Status",
		"EGM Traffic",
		"Certificates",
		"Next Action",
	} {
		if !strings.Contains(htmlBody, marker) {
			t.Fatalf("dashboard html missing marker %q", marker)
		}
	}
	for _, legacy := range []string{
		"System Check Rules",
		"Blocker Governance",
		"Runbook Readiness",
		"Operator Readiness Model",
		"First Cabinet Session",
		"Session Evidence Capture",
		"Session Complete",
		"Current workflow step",
		"Cabinet prep can continue",
		"Escalation Rationale",
		"Escalation History",
		"Approve Selected Finding",
		"Revoke Selected Finding",
		"approve stop condition",
		"revoke stop condition",
		"escalation history",
		"rationale",
		"Ready Now",
		"Needs Operator Action",
		"Lab Warning",
		"Informational",
	} {
		if strings.Contains(htmlBody, legacy) {
			t.Fatalf("expected legacy label removed from dashboard html: %q", legacy)
		}
	}

	jsReq := httptest.NewRequest(http.MethodGet, "/static/dashboard.js", nil)
	jsRec := httptest.NewRecorder()
	mux.ServeHTTP(jsRec, jsReq)
	if jsRec.Code != http.StatusOK {
		t.Fatalf("dashboard.js status = %d, want %d", jsRec.Code, http.StatusOK)
	}
	jsBody := jsRec.Body.String()
	for _, marker := range []string{
		"dashboardTabStorageKey",
		"dashboardTabs",
		"normalizeDashboardTab",
		"setDashboardTab",
		"applyDashboardTabVisibility",
		"dashboard-tab-button",
		"dashboard-tab-hidden",
		"dashboard-tab-section-hidden",
	} {
		if !strings.Contains(jsBody, marker) {
			t.Fatalf("dashboard.js missing tab marker %q", marker)
		}
	}
	for _, legacy := range []string{
		"System Check Rules",
		"Blocker Governance",
		"Runbook Readiness",
		"Operator Readiness Model",
		"First Cabinet Session",
		"Session Evidence Capture",
		"Session Complete",
		"Current workflow step",
		"Cabinet prep can continue",
		"Escalation Rationale",
		"Escalation History",
		"Approve Selected Finding",
		"Revoke Selected Finding",
		"approve stop condition",
		"revoke stop condition",
		"escalation history",
		"rationale",
		"Ready Now",
		"Needs Operator Action",
		"Lab Warning",
		"Informational",
	} {
		if strings.Contains(jsBody, legacy) {
			t.Fatalf("expected legacy label removed from dashboard.js: %q", legacy)
		}
	}

	cssReq := httptest.NewRequest(http.MethodGet, "/static/dashboard.css", nil)
	cssRec := httptest.NewRecorder()
	mux.ServeHTTP(cssRec, cssReq)
	if cssRec.Code != http.StatusOK {
		t.Fatalf("dashboard.css status = %d, want %d", cssRec.Code, http.StatusOK)
	}
	cssBody := cssRec.Body.String()
	for _, marker := range []string{
		"dashboard-tab-menu",
		"dashboard-tab-button",
		"dashboard-tab-hidden",
		"dashboard-tab-section-hidden",
		"system-status-grid",
	} {
		if !strings.Contains(cssBody, marker) {
			t.Fatalf("dashboard.css missing tab marker %q", marker)
		}
	}
}
