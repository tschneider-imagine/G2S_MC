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

type SendRequest struct {
	MessageID       int64
	ActionRunID     string
	EGMID           string
	EndpointURL     string
	Method          string
	ContentType     string
	Headers         map[string]string
	RawPayload      string
	TimeoutMS       int
	AllowRealSend   bool
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
