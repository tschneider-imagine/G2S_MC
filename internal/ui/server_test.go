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
	if !strings.Contains(rr.Body.String(), "Session Evidence Capture") {
		t.Fatalf("expected session evidence capture panel")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-evidence-json-button\"") {
		t.Fatalf("expected session evidence export button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"session-evidence-save-button\"") {
		t.Fatalf("expected session evidence save button")
	}
	if !strings.Contains(rr.Body.String(), "id=\"operator-alert\"") {
		t.Fatalf("expected operator alert strip")
	}
	if !strings.Contains(rr.Body.String(), "/readyz Primary") {
		t.Fatalf("expected explicit readyz indicator")
	}
	if !strings.Contains(rr.Body.String(), "data-filter=\"unhealthy\"") {
		t.Fatalf("expected unhealthy filter tab")
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
			for _, marker := range []string{"lastGoodStatus", "lastGoodAt", "inFlight", "pollIntervalMs", "nextBackoffMs", "renderAlerts", "fetchReadyz", "readyz unavailable", "showAPIFailureBanner", "(asc)", "renderCabinetProfile", "profile_source", "saveCabinetProfileOverride", "clearCabinetProfileOverride", "copySetupTokenToClipboard", "Authorization", "cabinetProfile", "certificateImport", "certificateExport", "importCertificateMaterial", "exportCertificateMaterial", "cert-manager-state", "validateCertificateManagerForm", "certificateRoleRulesText", "cert-role-export-policy", "allow_private_key_export", "cabinetPreflight", "renderFirstCabinetSession", "first-cabinet-session-state", "trusted_mutation_bypass_active", "mutationTokenRequired", "setup-token-help-text", "cert-token-help-text", "buildSessionEvidence", "buildSessionEvidenceMarkdown", "session-evidence-state", "exportSessionEvidenceJSON", "saveSessionEvidenceToHistory", "sessionEvidence"} {
				if !strings.Contains(body, marker) {
					t.Fatalf("%s missing marker %q", path, marker)
				}
			}
		}
		if path == "/static/dashboard.css" {
			for _, marker := range []string{"alert-strip", "stale-warning", "stale-critical", "filter-tabs", "summary-blocking", "cert-item", "api-banner", "source-pill", "cabinet-warning", "setup-form", "token-help", "trusted-bypass-hidden", "validation-list", "secondary-button", "cert-manager-form", "cert-role-summary", "cert-manager-details", "cert-manager-detail", "cert-impact", "first-cabinet-session-panel", "first-cabinet-session-blockers", "evidence-capture-panel", "evidence-actions"} {
				if !strings.Contains(body, marker) {
					t.Fatalf("%s missing marker %q", path, marker)
				}
			}
		}
	}
}
