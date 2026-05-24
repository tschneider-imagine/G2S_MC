package g2stransport

import (
	"context"
	"time"
)

type DisabledSender struct {
	Clock func() time.Time
}

func (s *DisabledSender) Send(_ context.Context, request SendRequest) (SendResult, error) {
	clock := s.Clock
	if clock == nil {
		clock = time.Now
	}
	return SendResult{
		MessageID:     request.MessageID,
		EGMID:         request.EGMID,
		TransportMode: ModeDisabled,
		Sent:          false,
		Blocked:       true,
		Error:         "send disabled: transport mode is DISABLED",
		CompletedAt:   clock().UTC(),
	}, nil
}
