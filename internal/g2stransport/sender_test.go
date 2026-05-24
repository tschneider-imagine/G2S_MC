package g2stransport

import (
	"context"
	"testing"
	"time"
)

func TestDisabledSenderNeverSends(t *testing.T) {
	now := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
	sender := &DisabledSender{Clock: func() time.Time { return now }}

	result, err := sender.Send(context.Background(), SendRequest{
		MessageID: 123,
		EGMID:     "EGM-1",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if result.Sent {
		t.Fatal("expected sent=false")
	}
	if !result.Blocked {
		t.Fatal("expected blocked=true")
	}
	if result.Error == "" {
		t.Fatal("expected blocked reason")
	}
}
