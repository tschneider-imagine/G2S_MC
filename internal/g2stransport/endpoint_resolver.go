package g2stransport

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type EndpointDefaults struct {
	Scheme string
	Port   int
}

func EndpointDefaultsFromHostURL(raw string) EndpointDefaults {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || strings.TrimSpace(parsed.Scheme) == "" {
		return EndpointDefaults{}
	}
	defaults := EndpointDefaults{
		Scheme: strings.ToLower(strings.TrimSpace(parsed.Scheme)),
	}
	if parsed.Port() != "" {
		if port, convErr := strconv.Atoi(parsed.Port()); convErr == nil && port > 0 {
			defaults.Port = port
		}
	} else {
		switch defaults.Scheme {
		case "https":
			defaults.Port = 443
		case "http":
			defaults.Port = 80
		}
	}
	return defaults
}

type DeliveryTarget struct {
	EndpointURL string
	Method      string
	ContentType string
	Headers     map[string]string
	TimeoutMS   int
}

type DeliveryTargetResolveRequest struct {
	EGMRecord           *egms.EGMRecord
	TemplateVersion     *templates.G2STemplateVersion
	FallbackMethod      string
	FallbackContentType string
	FallbackTimeoutMS   int
	Defaults            EndpointDefaults
}

type endpointQuirks struct {
	Method       string            `json:"method"`
	ContentType  string            `json:"content_type"`
	Headers      map[string]string `json:"headers"`
	TimeoutMS    int               `json:"timeout_ms"`
	EndpointPath string            `json:"endpoint_path"`
}

func ResolveDeliveryTarget(request DeliveryTargetResolveRequest) (DeliveryTarget, error) {
	if request.EGMRecord == nil {
		return DeliveryTarget{}, fmt.Errorf("missing EGM record")
	}
	quirks, err := parseEndpointQuirks(request.TemplateVersion)
	if err != nil {
		return DeliveryTarget{}, err
	}
	if quirks.TimeoutMS < 0 {
		return DeliveryTarget{}, fmt.Errorf("template endpoint quirks timeout_ms must be >= 0")
	}

	rawEndpoint := strings.TrimSpace(request.EGMRecord.EndpointPath)
	if strings.TrimSpace(quirks.EndpointPath) != "" {
		rawEndpoint = strings.TrimSpace(quirks.EndpointPath)
	}
	resolvedURL, err := resolveEndpointURL(request.EGMRecord, rawEndpoint, request.Defaults)
	if err != nil {
		return DeliveryTarget{}, err
	}

	method := strings.ToUpper(strings.TrimSpace(quirks.Method))
	if method == "" {
		method = strings.ToUpper(strings.TrimSpace(request.FallbackMethod))
	}
	if method == "" {
		method = "POST"
	}
	contentType := strings.TrimSpace(quirks.ContentType)
	if contentType == "" {
		contentType = strings.TrimSpace(request.FallbackContentType)
	}

	timeoutMS := quirks.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = request.FallbackTimeoutMS
	}
	headers := map[string]string{}
	for key, value := range quirks.Headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		headers[trimmedKey] = value
	}

	return DeliveryTarget{
		EndpointURL: resolvedURL,
		Method:      method,
		ContentType: contentType,
		Headers:     headers,
		TimeoutMS:   timeoutMS,
	}, nil
}

func parseEndpointQuirks(version *templates.G2STemplateVersion) (endpointQuirks, error) {
	if version == nil || strings.TrimSpace(version.EndpointQuirksJSON) == "" {
		return endpointQuirks{}, nil
	}
	var quirks endpointQuirks
	if err := json.Unmarshal([]byte(version.EndpointQuirksJSON), &quirks); err != nil {
		return endpointQuirks{}, fmt.Errorf("template endpoint quirks JSON invalid: %w", err)
	}
	return quirks, nil
}

func resolveEndpointURL(record *egms.EGMRecord, endpointValue string, defaults EndpointDefaults) (string, error) {
	trimmedEndpoint := strings.TrimSpace(endpointValue)
	if trimmedEndpoint == "" {
		return "", fmt.Errorf("missing endpoint URL for EGM %s", strings.TrimSpace(record.EGMID))
	}

	if strings.HasPrefix(strings.ToLower(trimmedEndpoint), "http://") || strings.HasPrefix(strings.ToLower(trimmedEndpoint), "https://") {
		parsed, err := url.Parse(trimmedEndpoint)
		if err != nil || strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
			return "", fmt.Errorf("invalid endpoint URL %q", trimmedEndpoint)
		}
		return parsed.String(), nil
	}

	host := strings.TrimSpace(record.IPAddress)
	if host == "" {
		return "", fmt.Errorf("missing endpoint host for EGM %s", strings.TrimSpace(record.EGMID))
	}
	scheme := strings.ToLower(strings.TrimSpace(defaults.Scheme))
	if scheme == "" {
		return "", fmt.Errorf("missing configured delivery scheme for EGM %s endpoint path %q", strings.TrimSpace(record.EGMID), trimmedEndpoint)
	}
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("unsupported configured delivery scheme %q", scheme)
	}

	path := trimmedEndpoint
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	resolvedHost := host
	if defaults.Port > 0 {
		resolvedHost = net.JoinHostPort(host, strconv.Itoa(defaults.Port))
	}
	resolved := url.URL{
		Scheme: scheme,
		Host:   resolvedHost,
		Path:   path,
	}
	return resolved.String(), nil
}

func MergeHeaders(base map[string]string, overlay map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range base {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		out[trimmedKey] = value
	}
	for key, value := range overlay {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		out[trimmedKey] = value
	}
	return out
}
