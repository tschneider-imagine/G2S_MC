package g2stransport

import (
	"context"
	"strings"
	"time"
)

type Mode string

const (
	ModeHTTP Mode = "HTTP"
)

type DeliveryMode string

const (
	DeliveryModeHTTP DeliveryMode = "HTTP"
)

type DeliveryTopology string

const (
	DeliveryTopologyHostListener     DeliveryTopology = "HOST_LISTENER"
	DeliveryTopologyOutboundEndpoint DeliveryTopology = "OUTBOUND_ENDPOINT"
	DeliveryTopologyCaptureEndpoint  DeliveryTopology = "CAPTURE_ENDPOINT"
)

type DeliverySettings struct {
	Mode      DeliveryMode `json:"mode"`
	TimeoutMS int          `json:"timeout_ms"`
}

func (s DeliverySettings) Normalize() DeliverySettings {
	normalized := DeliverySettings{
		Mode:      DeliveryModeHTTP,
		TimeoutMS: s.TimeoutMS,
	}
	if normalized.TimeoutMS < 0 {
		normalized.TimeoutMS = 0
	}
	return normalized
}

func (s DeliverySettings) TransportMode() Mode {
	return ModeHTTP
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
