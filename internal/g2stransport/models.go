package g2stransport

import (
	"context"
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
	TransportMode Mode
	RequestedAt   time.Time
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
