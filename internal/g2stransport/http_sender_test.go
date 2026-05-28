package g2stransport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// These tests verify the current capture-proof guardrails. They do not
// establish permanent production endpoint policy.

func TestHTTPSenderAllowFlagDoesNotBlockHTTPSend(t *testing.T) {
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

func TestHTTPSenderModeDoesNotBlockConfiguredHTTPSend(t *testing.T) {
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
		TransportMode:   Mode("DISABLED"),
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

func TestHTTPSenderCaptureOnlyFlagDoesNotBlockNonLocalEndpoint(t *testing.T) {
	calls := int32(0)
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&calls, 1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}
	sender := &HTTPSender{Client: client}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       5,
		EGMID:           "EGM-5",
		EndpointURL:     "http://10.20.30.40:18080/capture",
		RawPayload:      "<send/>",
		TransportMode:   ModeHTTP,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if result.Blocked || !result.Sent {
		t.Fatalf("unexpected result: %+v", result)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected one transport call, got %d", got)
	}
}

func TestHTTPSenderNonCaptureModeDoesNotApplyLoopbackRestriction(t *testing.T) {
	calls := int32(0)
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&calls, 1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	sender := &HTTPSender{Client: client}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       55,
		EGMID:           "EGM-55",
		EndpointURL:     "http://10.20.30.40:8080/g2s",
		RawPayload:      "<send/>",
		TransportMode:   ModeHTTP,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if result.Blocked {
		t.Fatalf("result should not be blocked in non-capture mode: %+v", result)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected one transport call, got %d", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHTTPSenderMissingEndpointFailsClearly(t *testing.T) {
	sender := &HTTPSender{}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       6,
		EGMID:           "EGM-6",
		EndpointURL:     "",
		RawPayload:      "<send/>",
		TransportMode:   ModeHTTP,
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

func TestHTTPSenderInvalidEndpointFailsClearly(t *testing.T) {
	sender := &HTTPSender{}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       7,
		EGMID:           "EGM-7",
		EndpointURL:     "://bad-url",
		RawPayload:      "<send/>",
		TransportMode:   ModeHTTP,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if result.Sent || result.Blocked {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !strings.Contains(strings.ToLower(result.Error), "invalid endpoint") {
		t.Fatalf("unexpected error: %q", result.Error)
	}
}

func TestHTTPSenderNon2xxReturnsFailedResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream failure"))
	}))
	defer server.Close()

	sender := &HTTPSender{}
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:       8,
		EGMID:           "EGM-8",
		EndpointURL:     server.URL,
		RawPayload:      "<send/>",
		TransportMode:   ModeHTTP,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if result.Sent {
		t.Fatalf("expected failed result: %+v", result)
	}
	if result.HTTPStatusCode != http.StatusBadGateway {
		t.Fatalf("status=%d", result.HTTPStatusCode)
	}
	if !strings.Contains(result.ResponseExcerpt, "upstream failure") {
		t.Fatalf("excerpt=%q", result.ResponseExcerpt)
	}
}

