package g2stransport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// These tests verify the current capture-proof guardrails. They do not
// establish permanent production endpoint policy.

func TestHTTPSenderBlocksWhenAllowFalse(t *testing.T) {
	var hitCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := &HTTPSender{}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       1,
		EGMID:           "EGM-1",
		EndpointURL:     server.URL,
		RawPayload:      "<x/>",
		TransportMode:   ModeHTTP,
		AllowRealSend:   false,
		CaptureOnlySend: true,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !result.Blocked || result.Sent {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := atomic.LoadInt32(&hitCount); got != 0 {
		t.Fatalf("expected no network call, hitCount=%d", got)
	}
}

func TestHTTPSenderBlocksWhenModeNotHTTP(t *testing.T) {
	var hitCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := &HTTPSender{}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       2,
		EGMID:           "EGM-2",
		EndpointURL:     server.URL,
		RawPayload:      "<x/>",
		TransportMode:   ModeDisabled,
		AllowRealSend:   true,
		CaptureOnlySend: true,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !result.Blocked || result.Sent {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := atomic.LoadInt32(&hitCount); got != 0 {
		t.Fatalf("expected no network call, hitCount=%d", got)
	}
}

func TestHTTPSenderSendsWhenAllowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s want POST", r.Method)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("acknowledged"))
	}))
	defer server.Close()

	sender := &HTTPSender{}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       3,
		EGMID:           "EGM-3",
		EndpointURL:     server.URL,
		RawPayload:      "<send/>",
		TransportMode:   ModeHTTP,
		AllowRealSend:   true,
		CaptureOnlySend: true,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if result.Blocked {
		t.Fatalf("expected not blocked: %+v", result)
	}
	if result.HTTPStatusCode != http.StatusAccepted {
		t.Fatalf("status=%d want %d", result.HTTPStatusCode, http.StatusAccepted)
	}
	if result.ResponseExcerpt != "acknowledged" {
		t.Fatalf("excerpt=%q", result.ResponseExcerpt)
	}
}

func TestHTTPSenderAllowsConfiguredHTTPWhenCaptureOnlyNotSet(t *testing.T) {
	var hitCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sender := &HTTPSender{}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       4,
		EGMID:           "EGM-4",
		EndpointURL:     server.URL,
		RawPayload:      "<send/>",
		TransportMode:   ModeHTTP,
		AllowRealSend:   true,
		CaptureOnlySend: false,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if result.Blocked || !result.Sent {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := atomic.LoadInt32(&hitCount); got != 1 {
		t.Fatalf("expected one network call, hitCount=%d", got)
	}
}

func TestHTTPSenderCaptureProofBlocksNonLocalEndpoint(t *testing.T) {
	sender := &HTTPSender{}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       5,
		EGMID:           "EGM-5",
		EndpointURL:     "http://10.20.30.40:18080/capture",
		RawPayload:      "<send/>",
		TransportMode:   ModeHTTP,
		AllowRealSend:   true,
		CaptureOnlySend: true,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !result.Blocked || result.Sent {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Error == "" || result.Error == "send blocked: capture_only_send_required" {
		t.Fatalf("unexpected blocked reason: %q", result.Error)
	}
}

func TestHTTPSenderMissingEndpointFailsClearly(t *testing.T) {
	sender := &HTTPSender{}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       6,
		EGMID:           "EGM-6",
		EndpointURL:     "",
		RawPayload:      "<send/>",
		TransportMode:   ModeHTTP,
		AllowRealSend:   true,
		CaptureOnlySend: false,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if result.Blocked {
		t.Fatalf("missing endpoint should fail clearly, not block: %+v", result)
	}
	if result.Sent {
		t.Fatalf("unexpected sent result: %+v", result)
	}
	if result.Error == "" || !strings.Contains(strings.ToLower(result.Error), "missing endpoint") {
		t.Fatalf("unexpected error: %q", result.Error)
	}
}
