package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

const (
	preflightPass = "PASS"
	preflightFail = "FAIL"
)

type cabinetPreflightResponse struct {
	Overall   string                  `json:"overall"`
	Checks    []cabinetPreflightCheck `json:"checks"`
	Issues    []string                `json:"issues,omitempty"`
	Warnings  []string                `json:"warnings,omitempty"`
	Timestamp time.Time               `json:"timestamp"`
}

type cabinetPreflightCheck struct {
	ID      string `json:"id"`
	Result  string `json:"result"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func cabinetPreflightHandler(eng *engine.Engine, store *store.SQLiteStore, cfg config.Config, runtime runtimeInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		result := evaluateCabinetPreflight(r.Context(), eng, store, cfg, runtime)
		writeJSON(w, result, nil)
	}
}

func evaluateCabinetPreflight(ctx context.Context, eng *engine.Engine, store *store.SQLiteStore, cfg config.Config, runtime runtimeInfo) cabinetPreflightResponse {
	response := cabinetPreflightResponse{
		Overall:   preflightPass,
		Checks:    []cabinetPreflightCheck{},
		Issues:    []string{},
		Warnings:  []string{},
		Timestamp: time.Now().UTC(),
	}

	addCheck := func(check cabinetPreflightCheck) {
		response.Checks = append(response.Checks, check)
	}

	status, statusErr := computeApplianceStatus(ctx, eng, store, cfg, runtime, nil)
	profile, profileErr := resolveCabinetProfile(ctx, store, cfg.CabinetProfile)
	certificates, certificatesErr := store.ListCertificateInventory(ctx)

	addCheck(evaluatePreflightReadiness(status, statusErr))
	addCheck(evaluatePreflightProfileCompleteness(status, statusErr, cfg, profile, profileErr))
	addCheck(evaluatePreflightProfileSource(profile, profileErr))
	addCheck(evaluatePreflightModeCertificates(cfg, certificates, certificatesErr))
	addCheck(evaluatePreflightWireIdentitySAN(cfg, profile, profileErr, certificates, certificatesErr))

	for _, check := range response.Checks {
		if check.Result != preflightFail {
			continue
		}
		response.Overall = preflightFail
		response.Issues = append(response.Issues, check.Message)
	}

	return response
}

func evaluatePreflightReadiness(status applianceStatus, statusErr error) cabinetPreflightCheck {
	if statusErr != nil {
		return cabinetPreflightCheck{
			ID:      "service_readiness",
			Result:  preflightFail,
			Message: "Service readiness could not be evaluated",
			Detail:  statusErr.Error(),
		}
	}

	if status.Readiness.Overall == "READY" || status.Readiness.Overall == "READY_LAB" {
		return cabinetPreflightCheck{
			ID:      "service_readiness",
			Result:  preflightPass,
			Message: "Readiness check is healthy for /readyz policy",
			Detail:  "overall=" + status.Readiness.Overall,
		}
	}

	detail := "overall=" + status.Readiness.Overall
	if len(status.Readiness.Issues) > 0 {
		detail = detail + "; issues=" + strings.Join(status.Readiness.Issues, " | ")
	}
	return cabinetPreflightCheck{
		ID:      "service_readiness",
		Result:  preflightFail,
		Message: "Readiness is degraded",
		Detail:  detail,
	}
}

func evaluatePreflightProfileCompleteness(status applianceStatus, statusErr error, cfg config.Config, profile resolvedCabinetProfile, profileErr error) cabinetPreflightCheck {
	if profileErr != nil {
		return cabinetPreflightCheck{
			ID:      "cabinet_profile",
			Result:  preflightFail,
			Message: "Cabinet profile could not be resolved",
			Detail:  profileErr.Error(),
		}
	}

	problems := []string{}
	if err := config.ValidateCabinetProfile(profile.Effective); err != nil {
		problems = append(problems, err.Error())
	}
	placeholderProblems := cabinetProfilePlaceholderProblems(profile.Effective)
	firstTestPlaceholderProblems, otherPlaceholderProblems := splitFirstTestPlaceholderProblems(placeholderProblems)
	problems = append(problems, otherPlaceholderProblems...)
	if profile.Warning != "" {
		problems = append(problems, profile.Warning)
	}
	if len(firstTestPlaceholderProblems) > 0 {
		allowAsLabWarning := len(problems) == 0 && preflightLabModeWithObservedEGMs(status, statusErr, cfg)
		if allowAsLabWarning {
			return cabinetPreflightCheck{
				ID:      "cabinet_profile",
				Result:  preflightPass,
				Message: "Cabinet profile is usable for active lab session; first-test EGM placeholders are warning-only",
				Detail:  "lab_warning_code=FIRST_TEST_EGM_IDS_PLACEHOLDER; action=replace placeholder first_test_egm_ids before real cabinet deployment; observed_egms=" + fmt.Sprintf("%d", observedEGMCount(status)) + "; issues=" + strings.Join(firstTestPlaceholderProblems, " | "),
			}
		}
		problems = append(problems, firstTestPlaceholderProblems...)
	}
	if len(problems) > 0 {
		remediation := "set non-placeholder cabinet profile fields via config.cabinet_profile or PUT /api/cabinet-profile: wire_host_url, listener_dns_name/listener_ip, required_san_dns/required_san_ips, host_id, first_test_egm_ids"
		return cabinetPreflightCheck{
			ID:      "cabinet_profile",
			Result:  preflightFail,
			Message: "Cabinet profile has missing or placeholder values",
			Detail:  remediation + " | issues=" + strings.Join(problems, " | "),
		}
	}
	return cabinetPreflightCheck{
		ID:      "cabinet_profile",
		Result:  preflightPass,
		Message: "Cabinet profile is complete",
		Detail:  "wire_host_url=" + profile.Effective.WireHostURL + "; host_id=" + profile.Effective.HostID,
	}
}

func splitFirstTestPlaceholderProblems(problems []string) ([]string, []string) {
	firstTest := []string{}
	other := []string{}
	for _, problem := range problems {
		if strings.Contains(problem, "first_test_egm_ids[") {
			firstTest = append(firstTest, problem)
			continue
		}
		other = append(other, problem)
	}
	return firstTest, other
}

func preflightLabModeWithObservedEGMs(status applianceStatus, statusErr error, cfg config.Config) bool {
	if statusErr != nil {
		return false
	}
	labMode := status.Readiness.Overall == "READY_LAB" || (!cfg.G2S.RequireTLS && !cfg.G2S.RequireClientCert)
	if !labMode {
		return false
	}
	return observedEGMCount(status) > 0
}

func observedEGMCount(status applianceStatus) int {
	count := 0
	for _, egm := range status.EGMs {
		if egm.LastSeen.IsZero() {
			continue
		}
		count++
	}
	return count
}

func evaluatePreflightProfileSource(profile resolvedCabinetProfile, profileErr error) cabinetPreflightCheck {
	if profileErr != nil {
		return cabinetPreflightCheck{
			ID:      "profile_source",
			Result:  preflightFail,
			Message: "Profile source marker is unavailable",
			Detail:  profileErr.Error(),
		}
	}

	switch profile.ProfileSource {
	case "file", "override":
		return cabinetPreflightCheck{
			ID:      "profile_source",
			Result:  preflightPass,
			Message: "Profile source is explicit",
			Detail:  "profile_source=" + profile.ProfileSource,
		}
	case "mixed":
		return cabinetPreflightCheck{
			ID:      "profile_source",
			Result:  preflightFail,
			Message: "Profile source is mixed and should be consolidated",
			Detail:  "profile_source=mixed indicates partial override fallback",
		}
	default:
		return cabinetPreflightCheck{
			ID:      "profile_source",
			Result:  preflightFail,
			Message: "Profile source marker is invalid",
			Detail:  "profile_source=" + profile.ProfileSource,
		}
	}
}

func evaluatePreflightModeCertificates(cfg config.Config, certificates []model.CertificateInventory, certificatesErr error) cabinetPreflightCheck {
	if certificatesErr != nil {
		return cabinetPreflightCheck{
			ID:      "certificate_mode_requirements",
			Result:  preflightFail,
			Message: "Certificate inventory is unavailable",
			Detail:  certificatesErr.Error(),
		}
	}

	requiredRoles := []string{}
	if cfg.G2S.RequireTLS {
		requiredRoles = append(requiredRoles, "web_server_cert")
	}
	if cfg.G2S.RequireClientCert {
		requiredRoles = append(requiredRoles, "g2s_ca_cert", "g2s_client_cert")
	}
	sort.Strings(requiredRoles)
	if len(requiredRoles) == 0 {
		return cabinetPreflightCheck{
			ID:      "certificate_mode_requirements",
			Result:  preflightPass,
			Message: "No certificates are required by the current runtime mode",
			Detail:  "g2s.require_tls=false; g2s.require_client_cert=false",
		}
	}

	byRole := map[string]model.CertificateInventory{}
	for _, record := range certificates {
		byRole[record.Role] = record
	}

	failures := []string{}
	for _, role := range requiredRoles {
		record, ok := byRole[role]
		if !ok {
			failures = append(failures, role+" inventory record is missing")
			continue
		}
		statusKey := certificateStatusKey(record.Status)
		if statusKey != "VALID" && statusKey != "EXPIRING_SOON" {
			failures = append(failures, role+" status="+statusKey)
		}
	}

	if len(failures) > 0 {
		return cabinetPreflightCheck{
			ID:      "certificate_mode_requirements",
			Result:  preflightFail,
			Message: "Required certificates are not ready for configured runtime mode",
			Detail:  strings.Join(failures, " | "),
		}
	}

	return cabinetPreflightCheck{
		ID:      "certificate_mode_requirements",
		Result:  preflightPass,
		Message: "Required certificates for configured runtime mode are valid",
		Detail:  "roles=" + strings.Join(requiredRoles, ","),
	}
}

func evaluatePreflightWireIdentitySAN(cfg config.Config, profile resolvedCabinetProfile, profileErr error, certificates []model.CertificateInventory, certificatesErr error) cabinetPreflightCheck {
	if profileErr != nil {
		return cabinetPreflightCheck{
			ID:      "certificate_san_wire_identity",
			Result:  preflightFail,
			Message: "Wire identity SAN check could not resolve profile",
			Detail:  profileErr.Error(),
		}
	}
	if certificatesErr != nil {
		return cabinetPreflightCheck{
			ID:      "certificate_san_wire_identity",
			Result:  preflightFail,
			Message: "Wire identity SAN check could not load certificate inventory",
			Detail:  certificatesErr.Error(),
		}
	}

	wireHost, wireHostErr := wireIdentityFromURL(profile.Effective.WireHostURL)
	if wireHostErr != nil {
		return cabinetPreflightCheck{
			ID:      "certificate_san_wire_identity",
			Result:  preflightFail,
			Message: "Wire host URL is invalid for SAN compatibility checks",
			Detail:  wireHostErr.Error(),
		}
	}

	record, ok := certificateByRole(certificates, "web_server_cert")
	if !ok {
		if !runtimeModeRequiresServerCertificates(cfg) {
			return cabinetPreflightCheck{
				ID:      "certificate_san_wire_identity",
				Result:  preflightPass,
				Message: "Wire identity SAN check is skipped for certificate-optional runtime mode",
				Detail:  "g2s.require_tls=false; g2s.require_client_cert=false; reason=web_server_cert inventory record is missing",
			}
		}
		return cabinetPreflightCheck{
			ID:      "certificate_san_wire_identity",
			Result:  preflightFail,
			Message: "Web server certificate inventory record is missing",
			Detail:  "role=web_server_cert",
		}
	}
	if !runtimeModeRequiresServerCertificates(cfg) && strings.TrimSpace(record.Path) == "" {
		return cabinetPreflightCheck{
			ID:      "certificate_san_wire_identity",
			Result:  preflightPass,
			Message: "Wire identity SAN check is skipped for certificate-optional runtime mode",
			Detail:  "g2s.require_tls=false; g2s.require_client_cert=false; reason=web_server_cert path is empty",
		}
	}

	serverCert, err := loadCertificateFromPath(record.Path)
	if err != nil {
		return cabinetPreflightCheck{
			ID:      "certificate_san_wire_identity",
			Result:  preflightFail,
			Message: "Web server certificate could not be parsed for SAN checks",
			Detail:  formatCertificateParseFailureDetail("web_server_cert", record.Path, err),
		}
	}

	if err := serverCert.VerifyHostname(wireHost); err != nil {
		return cabinetPreflightCheck{
			ID:      "certificate_san_wire_identity",
			Result:  preflightFail,
			Message: "Web server certificate SAN does not match configured wire identity",
			Detail:  "wire_identity=" + wireHost + "; verify_hostname_error=" + err.Error(),
		}
	}

	return cabinetPreflightCheck{
		ID:      "certificate_san_wire_identity",
		Result:  preflightPass,
		Message: "Web server certificate SAN matches configured wire identity",
		Detail:  "wire_identity=" + wireHost,
	}
}

func runtimeModeRequiresServerCertificates(cfg config.Config) bool {
	return cfg.G2S.RequireTLS || cfg.G2S.RequireClientCert
}

func cabinetProfilePlaceholderProblems(profile config.CabinetProfile) []string {
	problems := []string{}

	if looksPlaceholderText(profile.WireHostURL) {
		problems = append(problems, "wire_host_url appears to be a placeholder")
	}
	if profile.ListenerDNSName != "" && looksPlaceholderText(profile.ListenerDNSName) {
		problems = append(problems, "listener_dns_name appears to be a placeholder")
	}
	if strings.TrimSpace(profile.HostID) != "" && looksPlaceholderText(profile.HostID) {
		problems = append(problems, "host_id appears to be a placeholder")
	}
	for i, value := range profile.RequiredSANDNS {
		if looksPlaceholderText(value) {
			problems = append(problems, fmt.Sprintf("required_san_dns[%d] appears to be a placeholder", i))
		}
	}
	for i, value := range profile.FirstTestEGMIDs {
		if looksPlaceholderText(value) {
			problems = append(problems, fmt.Sprintf("first_test_egm_ids[%d] appears to be a placeholder", i))
		}
	}

	return problems
}

func looksPlaceholderText(raw string) bool {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return true
	}
	needles := []string{
		"example",
		"placeholder",
		"changeme",
		"replace",
		"<",
		">",
	}
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}

	switch value {
	case "host-001", "host-pi-001", "host-cab-001", "egm-01", "egm-02":
		return true
	}
	if strings.Contains(value, "test") {
		return true
	}
	return false
}

func wireIdentityFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	host := parsed.Hostname()
	if host == "" {
		return "", fmt.Errorf("wire_host_url=%q does not include a hostname", rawURL)
	}
	return host, nil
}

func certificateByRole(records []model.CertificateInventory, role string) (model.CertificateInventory, bool) {
	for _, record := range records {
		if record.Role == role {
			return record, true
		}
	}
	return model.CertificateInventory{}, false
}

func loadCertificateFromPath(path string) (*x509.Certificate, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return nil, fmt.Errorf("web_server_cert path is empty")
	}

	raw, err := os.ReadFile(trimmedPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("no PEM certificate block found in %s", trimmedPath)
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("unexpected PEM type %q in %s", block.Type, trimmedPath)
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return certificate, nil
}

func formatCertificateParseFailureDetail(role string, path string, parseErr error) string {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return "role=" + role + "; path=(empty); action=set crypto.web_server_cert_path and crypto.web_server_key_path in active config and restart g2s-mute; parse_error=" + parseErr.Error()
	}

	detail := "role=" + role + "; path=" + trimmedPath + "; parse_error=" + parseErr.Error()
	if os.IsPermission(parseErr) {
		detail += "; action=grant read permission to service user for certificate path"
	}
	if os.IsNotExist(parseErr) {
		detail += "; action=write/import certificate PEM to configured path and restart g2s-mute"
	}
	return detail
}
