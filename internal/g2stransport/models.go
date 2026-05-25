package g2stransport

import (
	"context"
	"net"
	"net/url"
	"strings"
	"time"
)

type Mode string

const (
	ModeDisabled Mode = "DISABLED"
	ModeDryRun   Mode = "DRY_RUN"
	ModeHTTP     Mode = "HTTP"
)

type DeliveryMode string

const (
	DeliveryModeDisabled DeliveryMode = "DISABLED"
	DeliveryModeHTTP     DeliveryMode = "HTTP"
)

type DeliveryTopology string

const (
	DeliveryTopologyHostListener     DeliveryTopology = "HOST_LISTENER"
	DeliveryTopologyOutboundEndpoint DeliveryTopology = "OUTBOUND_ENDPOINT"
	DeliveryTopologyCaptureEndpoint  DeliveryTopology = "CAPTURE_ENDPOINT"
)

type DeliverySettings struct {
	Mode          DeliveryMode `json:"mode"`
	AllowDelivery bool         `json:"allow_delivery"`
	CaptureOnly   bool         `json:"capture_only"`
	TimeoutMS     int          `json:"timeout_ms"`
}

func (s DeliverySettings) Normalize() DeliverySettings {
	normalized := DeliverySettings{
		Mode:          DeliveryMode(strings.ToUpper(strings.TrimSpace(string(s.Mode)))),
		AllowDelivery: s.AllowDelivery,
		CaptureOnly:   s.CaptureOnly,
		TimeoutMS:     s.TimeoutMS,
	}
	switch normalized.Mode {
	case DeliveryModeHTTP:
	default:
		normalized.Mode = DeliveryModeDisabled
	}
	if normalized.TimeoutMS < 0 {
		normalized.TimeoutMS = 0
	}
	return normalized
}

func (s DeliverySettings) TransportMode() Mode {
	switch s.Normalize().Mode {
	case DeliveryModeHTTP:
		return ModeHTTP
	default:
		return ModeDisabled
	}
}

func NormalizeDeliveryTopology(raw string) (DeliveryTopology, bool) {
	normalized := DeliveryTopology(strings.ToUpper(strings.TrimSpace(raw)))
	switch normalized {
	case "":
		return DeliveryTopologyHostListener, true
	case DeliveryTopologyHostListener, DeliveryTopologyOutboundEndpoint, DeliveryTopologyCaptureEndpoint:
		return normalized, true
	default:
		return normalized, false
	}
}

type SendRequest struct {
	MessageID     int64
	ActionRunID   string
	EGMID         string
	EndpointURL   string
	Method        string
	ContentType   string
	Headers       map[string]string
	RawPayload    string
	TimeoutMS     int
	AllowRealSend bool
	// CaptureOnlySend enforces the phase-2 capture-endpoint policy.
	// This is a temporary safety control for capture proofing and is not
	// the long-term production transport policy.
	CaptureOnlySend bool
	TransportMode   Mode
	RequestedAt     time.Time
}

type SendResult struct {
	MessageID       int64
	EGMID           string
	TransportMode   Mode
	Sent            bool
	Blocked         bool
	HTTPStatusCode  int
	LatencyMS       int
	ResponseExcerpt string
	Error           string
	CompletedAt     time.Time
}

type Sender interface {
	Send(ctx context.Context, request SendRequest) (SendResult, error)
}

func normalizeMode(mode Mode) Mode {
	switch Mode(strings.ToUpper(strings.TrimSpace(string(mode)))) {
	case ModeDisabled:
		return ModeDisabled
	case ModeDryRun:
		return ModeDryRun
	case ModeHTTP:
		return ModeHTTP
	default:
		return ModeDisabled
	}
}

// CaptureEndpointAllowed applies the phase-2 capture-only host restriction.
// This should remain scoped to capture-proof mode and must not become a hidden
// permanent blocker for explicitly configured product endpoints.
func CaptureEndpointAllowed(endpointURL string) (bool, string) {
	trimmed := strings.TrimSpace(endpointURL)
	if trimmed == "" {
		return false, "missing_endpoint"
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return false, "invalid_endpoint_url"
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return false, "invalid_endpoint_host"
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true, ""
	}
	parsedIP := net.ParseIP(host)
	if parsedIP != nil && parsedIP.IsLoopback() {
		return true, ""
	}
	return false, "endpoint_not_allowed_for_capture_phase"
}
