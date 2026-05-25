package deliverycheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type CheckRequest struct {
	EGMID               string
	ActionID            string
	TemplateID          string
	TemplateActionKey   string
	IncludeNetworkCheck bool
	IncludeTLSCheck     bool
	TimeoutMS           int
}

type CertificateCheck struct {
	Role               string `json:"role"`
	Configured         bool   `json:"configured"`
	FileExists         bool   `json:"file_exists"`
	ParseStatus        string `json:"parse_status"`
	Status             string `json:"status"`
	Detail             string `json:"detail,omitempty"`
	Fingerprint        string `json:"fingerprint,omitempty"`
	DaysUntilExpiry    string `json:"days_until_expiry,omitempty"`
	LastCheckedRFC3339 string `json:"last_checked_rfc3339,omitempty"`
}

type ResultItem struct {
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type CheckResult struct {
	CheckedAt          time.Time          `json:"checked_at"`
	EGMID              string             `json:"egm_id"`
	ActionID           string             `json:"action_id,omitempty"`
	TemplateID         string             `json:"template_id,omitempty"`
	TemplateVersion    string             `json:"template_version,omitempty"`
	TemplateActionKey  string             `json:"template_action_key,omitempty"`
	DeliveryTopology   string             `json:"delivery_topology"`
	EndpointRequired   bool               `json:"endpoint_required"`
	ListenerURL        string             `json:"listener_url,omitempty"`
	HostID             string             `json:"host_id,omitempty"`
	EndpointURL        string             `json:"endpoint_url,omitempty"`
	Method             string             `json:"method,omitempty"`
	ContentType        string             `json:"content_type,omitempty"`
	Headers            map[string]string  `json:"headers,omitempty"`
	EndpointConfigured bool               `json:"endpoint_configured"`
	TemplateConfigured bool               `json:"template_configured"`
	ActionConfigured   bool               `json:"action_configured"`
	CertificateSummary []CertificateCheck `json:"certificate_summary"`
	DeliveryMode       string             `json:"delivery_mode"`
	NetworkCheck       ResultItem         `json:"network_check"`
	TLSCheck           ResultItem         `json:"tls_check"`
	OverallStatus      string             `json:"overall_status"`
	Warnings           []string           `json:"warnings,omitempty"`
	Errors             []string           `json:"errors,omitempty"`
}

type Options struct {
	EndpointDefaults g2stransport.EndpointDefaults
	ClientConfig     g2stransport.HTTPClientConfig
	DeliveryMode     string
	DeliveryTopology string
	CaptureEndpoint  string
	ListenerURL      string
	HostID           string
	DefaultTimeoutMS int
}

type Store interface {
	GetEGMRecord(ctx context.Context, egmID string) (*egms.EGMRecord, error)
	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error)
	GetActiveG2STemplateVersion(ctx context.Context, templateID string) (*templates.G2STemplateVersion, error)
	ListCertificateInventory(ctx context.Context) ([]model.CertificateInventory, error)
}

type Service struct {
	Store   Store
	Options Options
	Now     func() time.Time
}

func (s *Service) Check(ctx context.Context, request CheckRequest) (CheckResult, error) {
	if s.Store == nil {
		return CheckResult{}, fmt.Errorf("store is required")
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	timeoutMS := request.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = s.Options.DefaultTimeoutMS
	}
	if timeoutMS <= 0 {
		timeoutMS = 5000
	}

	topology, topologyValid := g2stransport.NormalizeDeliveryTopology(s.Options.DeliveryTopology)
	result := CheckResult{
		CheckedAt:         now,
		EGMID:             strings.TrimSpace(request.EGMID),
		ActionID:          strings.TrimSpace(request.ActionID),
		TemplateID:        strings.TrimSpace(request.TemplateID),
		TemplateActionKey: strings.TrimSpace(request.TemplateActionKey),
		DeliveryMode:      defaultText(strings.TrimSpace(s.Options.DeliveryMode), "DISABLED"),
		DeliveryTopology:  string(topology),
		EndpointRequired:  topology != g2stransport.DeliveryTopologyHostListener,
		ListenerURL:       strings.TrimSpace(s.Options.ListenerURL),
		HostID:            strings.TrimSpace(s.Options.HostID),
		Headers:           map[string]string{},
		NetworkCheck:      ResultItem{Status: "SKIPPED", Detail: "Network check not requested"},
		TLSCheck:          ResultItem{Status: "SKIPPED", Detail: "TLS check not requested"},
		OverallStatus:     "OK",
	}
	if !topologyValid {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Unknown delivery topology %q; using host listener.", strings.TrimSpace(s.Options.DeliveryTopology)))
	}

	certRows, certErr := s.Store.ListCertificateInventory(ctx)
	if certErr != nil {
		return CheckResult{}, certErr
	}
	result.CertificateSummary = buildCertificateSummary(certRows)

	if result.EGMID == "" {
		result.Errors = append(result.Errors, "EGM ID is required")
		result.OverallStatus = "ERROR"
		return result, nil
	}

	egmRow, err := s.Store.GetEGMRecord(ctx, result.EGMID)
	if err != nil {
		return CheckResult{}, err
	}
	if egmRow == nil {
		result.Errors = append(result.Errors, fmt.Sprintf("EGM %s not found", result.EGMID))
		result.OverallStatus = "ERROR"
		return result, nil
	}

	if result.ActionID != "" {
		definition, getErr := s.Store.GetActionDefinition(ctx, result.ActionID)
		if getErr != nil {
			return CheckResult{}, getErr
		}
		if definition == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Action %s not found", result.ActionID))
		} else {
			result.ActionConfigured = true
			if result.TemplateActionKey == "" && len(definition.Steps) > 0 {
				result.TemplateActionKey = strings.TrimSpace(definition.Steps[0].TemplateActionKey)
			}
		}
	}

	if result.TemplateID == "" {
		result.TemplateID = strings.TrimSpace(egmRow.TemplateID)
	}
	if result.TemplateID == "" {
		result.Errors = append(result.Errors, "Template is not configured for this EGM")
		result.OverallStatus = "ERROR"
		return result, nil
	}
	tpl, getTplErr := s.Store.GetG2STemplate(ctx, result.TemplateID)
	if getTplErr != nil {
		return CheckResult{}, getTplErr
	}
	if tpl == nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Template %s not found", result.TemplateID))
		result.OverallStatus = "ERROR"
		return result, nil
	}
	result.TemplateConfigured = true

	activeVersion, versionErr := s.Store.GetActiveG2STemplateVersion(ctx, result.TemplateID)
	if versionErr != nil {
		return CheckResult{}, versionErr
	}
	if activeVersion == nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Template %s has no active version", result.TemplateID))
		result.OverallStatus = "ERROR"
		return result, nil
	}
	result.TemplateVersion = strings.TrimSpace(activeVersion.VersionLabel)
	if strings.TrimSpace(result.TemplateActionKey) != "" {
		templateDoc, parseErr := g2sengine.ParseActionTemplateDocument(strings.TrimSpace(activeVersion.ActionsJSON))
		if parseErr != nil {
			result.Errors = append(result.Errors, "Template ActionsJSON is invalid")
			result.OverallStatus = "ERROR"
			return result, nil
		}
		if _, ok := templateDoc.Actions[strings.TrimSpace(result.TemplateActionKey)]; !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("Template action key %s not found in template %s", strings.TrimSpace(result.TemplateActionKey), result.TemplateID))
			result.OverallStatus = "ERROR"
			return result, nil
		}
	}

	switch topology {
	case g2stransport.DeliveryTopologyHostListener:
		deliveryTarget, targetErr := g2stransport.ResolveDeliveryTarget(g2stransport.DeliveryTargetResolveRequest{
			EGMRecord:           egmRow,
			TemplateVersion:     activeVersion,
			FallbackMethod:      "POST",
			FallbackContentType: "",
			FallbackTimeoutMS:   timeoutMS,
			Defaults:            s.Options.EndpointDefaults,
		})
		if targetErr == nil {
			result.EndpointConfigured = true
			result.EndpointURL = strings.TrimSpace(deliveryTarget.EndpointURL)
			result.Method = strings.TrimSpace(deliveryTarget.Method)
			result.ContentType = strings.TrimSpace(deliveryTarget.ContentType)
			result.Headers = g2stransport.MergeHeaders(map[string]string{}, deliveryTarget.Headers)
		} else {
			targetErrText := sanitizeDetail(targetErr.Error())
			if strings.Contains(strings.ToLower(targetErrText), "missing endpoint") {
				result.Warnings = append(result.Warnings, "Outbound endpoint is not configured; not required for host listener delivery.")
			} else {
				result.Warnings = append(result.Warnings, "Outbound endpoint could not be resolved for host listener delivery: "+targetErrText)
			}
		}

		if request.IncludeNetworkCheck {
			if result.EndpointConfigured {
				result.NetworkCheck = checkTCPConnectivity(result.EndpointURL, timeoutMS)
			} else {
				result.NetworkCheck = ResultItem{
					Status: "SKIPPED",
					Detail: "Network check skipped; outbound endpoint is not required for host listener delivery.",
				}
			}
		}
		if request.IncludeTLSCheck {
			if result.EndpointConfigured {
				result.TLSCheck = checkTLSConnectivity(result.EndpointURL, timeoutMS, s.Options.ClientConfig)
			} else {
				result.TLSCheck = ResultItem{
					Status: "SKIPPED",
					Detail: "TLS check skipped; outbound endpoint is not required for host listener delivery.",
				}
			}
		}
	case g2stransport.DeliveryTopologyOutboundEndpoint:
		deliveryTarget, targetErr := g2stransport.ResolveDeliveryTarget(g2stransport.DeliveryTargetResolveRequest{
			EGMRecord:           egmRow,
			TemplateVersion:     activeVersion,
			FallbackMethod:      "POST",
			FallbackContentType: "",
			FallbackTimeoutMS:   timeoutMS,
			Defaults:            s.Options.EndpointDefaults,
		})
		if targetErr != nil {
			result.Errors = append(result.Errors, sanitizeDetail(targetErr.Error()))
			result.OverallStatus = "ERROR"
			return result, nil
		}
		result.EndpointConfigured = true
		result.EndpointURL = strings.TrimSpace(deliveryTarget.EndpointURL)
		result.Method = strings.TrimSpace(deliveryTarget.Method)
		result.ContentType = strings.TrimSpace(deliveryTarget.ContentType)
		result.Headers = g2stransport.MergeHeaders(map[string]string{}, deliveryTarget.Headers)

		if request.IncludeNetworkCheck {
			result.NetworkCheck = checkTCPConnectivity(result.EndpointURL, timeoutMS)
		}
		if request.IncludeTLSCheck {
			result.TLSCheck = checkTLSConnectivity(result.EndpointURL, timeoutMS, s.Options.ClientConfig)
		}
	case g2stransport.DeliveryTopologyCaptureEndpoint:
		captureEndpoint := strings.TrimSpace(s.Options.CaptureEndpoint)
		if captureEndpoint == "" {
			result.Errors = append(result.Errors, "Missing configured capture endpoint URL.")
			result.OverallStatus = "ERROR"
			return result, nil
		}
		if _, err := endpointHostPort(captureEndpoint); err != nil {
			result.Errors = append(result.Errors, sanitizeDetail(err.Error()))
			result.OverallStatus = "ERROR"
			return result, nil
		}
		result.EndpointConfigured = true
		result.EndpointURL = captureEndpoint
		result.Method = "POST"
		if request.IncludeNetworkCheck {
			result.NetworkCheck = checkTCPConnectivity(result.EndpointURL, timeoutMS)
		}
		if request.IncludeTLSCheck {
			result.TLSCheck = checkTLSConnectivity(result.EndpointURL, timeoutMS, s.Options.ClientConfig)
		}
	default:
		result.Warnings = append(result.Warnings, "Unknown delivery topology; host listener behavior applied.")
		result.EndpointRequired = false
	}

	result.OverallStatus = deriveOverallStatus(result)
	return result, nil
}

func checkTCPConnectivity(endpointURL string, timeoutMS int) ResultItem {
	hostPort, err := endpointHostPort(endpointURL)
	if err != nil {
		return ResultItem{Status: "ERROR", Detail: sanitizeDetail(err.Error())}
	}
	conn, dialErr := net.DialTimeout("tcp", hostPort, time.Duration(timeoutMS)*time.Millisecond)
	if dialErr != nil {
		return ResultItem{Status: "ERROR", Detail: sanitizeDetail(dialErr.Error())}
	}
	_ = conn.Close()
	return ResultItem{Status: "OK", Detail: "TCP connection successful"}
}

func checkTLSConnectivity(endpointURL string, timeoutMS int, clientConfig g2stransport.HTTPClientConfig) ResultItem {
	parsed, err := url.Parse(strings.TrimSpace(endpointURL))
	if err != nil || strings.TrimSpace(parsed.Scheme) == "" {
		return ResultItem{Status: "ERROR", Detail: "invalid endpoint URL"}
	}
	if strings.ToLower(parsed.Scheme) != "https" {
		return ResultItem{Status: "SKIPPED", Detail: "TLS check skipped for non-HTTPS endpoint"}
	}
	hostPort, err := endpointHostPort(endpointURL)
	if err != nil {
		return ResultItem{Status: "ERROR", Detail: sanitizeDetail(err.Error())}
	}

	factory := g2stransport.NewHTTPClientFactory(clientConfig)
	client, clientErr := factory.NewClient(timeoutMS)
	if clientErr != nil {
		return ResultItem{Status: "ERROR", Detail: sanitizeDetail(clientErr.Error())}
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.TLSClientConfig == nil {
		return ResultItem{Status: "ERROR", Detail: "TLS client is not configured"}
	}
	tlsConfig := transport.TLSClientConfig.Clone()
	if strings.TrimSpace(tlsConfig.ServerName) == "" {
		tlsConfig.ServerName = parsed.Hostname()
	}

	dialer := &net.Dialer{Timeout: time.Duration(timeoutMS) * time.Millisecond}
	conn, dialErr := tls.DialWithDialer(dialer, "tcp", hostPort, tlsConfig)
	if dialErr != nil {
		return ResultItem{Status: "ERROR", Detail: sanitizeDetail(dialErr.Error())}
	}
	_ = conn.Close()
	return ResultItem{Status: "OK", Detail: "TLS handshake successful"}
}

func endpointHostPort(endpointURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpointURL))
	if err != nil {
		return "", fmt.Errorf("invalid endpoint URL")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("invalid endpoint host")
	}
	port := strings.TrimSpace(parsed.Port())
	if port == "" {
		switch strings.ToLower(strings.TrimSpace(parsed.Scheme)) {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			return "", fmt.Errorf("endpoint scheme is not supported")
		}
	}
	if _, convErr := strconv.Atoi(port); convErr != nil {
		return "", fmt.Errorf("invalid endpoint port")
	}
	return net.JoinHostPort(host, port), nil
}

func buildCertificateSummary(rows []model.CertificateInventory) []CertificateCheck {
	byRole := map[string]model.CertificateInventory{}
	for _, row := range rows {
		byRole[strings.TrimSpace(row.Role)] = row
	}
	roles := []string{"web_server_cert", "g2s_ca_cert", "g2s_client_cert", "g2s_client_key"}
	out := make([]CertificateCheck, 0, len(roles))
	for _, role := range roles {
		row, ok := byRole[role]
		if !ok {
			out = append(out, CertificateCheck{
				Role:        role,
				Configured:  false,
				FileExists:  false,
				ParseStatus: "not configured",
				Status:      "NOT_CONFIGURED",
				Detail:      "Not configured",
			})
			continue
		}
		status := statusKeyFromInventory(row.Status)
		check := CertificateCheck{
			Role:               role,
			Configured:         strings.TrimSpace(row.Path) != "",
			FileExists:         status != "MISSING" && status != "NOT_CONFIGURED",
			ParseStatus:        parseStatusFromInventory(status),
			Status:             status,
			Detail:             sanitizeDetail(strings.TrimSpace(row.Error)),
			Fingerprint:        strings.TrimSpace(row.SHA256Fingerprint),
			LastCheckedRFC3339: row.LastCheckedAt.UTC().Format(time.RFC3339),
			DaysUntilExpiry:    daysUntilExpiry(row.NotAfter),
		}
		out = append(out, check)
	}
	return out
}

func statusKeyFromInventory(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "UNKNOWN"
	}
	if idx := strings.Index(trimmed, ":"); idx >= 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	upper := strings.ToUpper(trimmed)
	if strings.Contains(upper, "PRIVATE KEY") {
		return "INVALID"
	}
	return upper
}

func parseStatusFromInventory(status string) string {
	switch status {
	case "VALID", "EXPIRING_SOON", "EXPIRED", "NOT_YET_VALID":
		return "parsed"
	case "INVALID":
		return "invalid"
	case "MISSING":
		return "missing"
	case "NOT_CONFIGURED":
		return "not configured"
	default:
		return "unknown"
	}
}

func daysUntilExpiry(notAfter *time.Time) string {
	if notAfter == nil {
		return "-"
	}
	now := time.Now().UTC()
	if notAfter.Before(now) {
		return "expired"
	}
	return strconv.Itoa(int(notAfter.Sub(now).Hours() / 24))
}

func deriveOverallStatus(result CheckResult) string {
	if len(result.Errors) > 0 {
		return "ERROR"
	}
	if strings.EqualFold(result.NetworkCheck.Status, "ERROR") || strings.EqualFold(result.TLSCheck.Status, "ERROR") {
		return "ERROR"
	}
	if len(result.Warnings) > 0 || strings.EqualFold(result.NetworkCheck.Status, "WARN") || strings.EqualFold(result.TLSCheck.Status, "WARN") {
		return "WARN"
	}
	return "OK"
}

func sanitizeDetail(detail string) string {
	text := strings.TrimSpace(detail)
	if text == "" {
		return ""
	}
	upper := strings.ToUpper(text)
	if strings.Contains(upper, "BEGIN PRIVATE KEY") || strings.Contains(upper, "END PRIVATE KEY") || strings.Contains(upper, "PRIVATE KEY-----") {
		return "private key material redacted"
	}
	sensitivePair := regexp.MustCompile(`(?i)\b(password|token|secret)\b\s*[:=]\s*([^\s,;]+)`)
	text = sensitivePair.ReplaceAllString(text, "$1=[REDACTED]")
	return text
}

func defaultText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
