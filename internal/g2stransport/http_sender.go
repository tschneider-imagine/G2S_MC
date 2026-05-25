package g2stransport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeoutMS       = 5000
	defaultResponseExcerpt = 4096
	defaultContentType     = "application/soap+xml"
)

type HTTPSender struct {
	Client               *http.Client
	Clock                func() time.Time
	ResponseExcerptBytes int
}

func (s *HTTPSender) Send(ctx context.Context, request SendRequest) (SendResult, error) {
	clock := s.Clock
	if clock == nil {
		clock = time.Now
	}
	mode := normalizeMode(request.TransportMode)
	result := SendResult{
		MessageID:     request.MessageID,
		EGMID:         request.EGMID,
		TransportMode: mode,
		CompletedAt:   clock().UTC(),
	}

	if mode != ModeHTTP {
		result.Blocked = true
		result.Error = fmt.Sprintf("send blocked: transport mode %q is not HTTP", mode)
		return result, nil
	}
	if !request.AllowRealSend {
		result.Blocked = true
		result.Error = "send blocked: allow_real_send is false"
		return result, nil
	}
	if !request.CaptureOnlySend {
		result.Blocked = true
		result.Error = "send blocked: capture_only_send_required"
		return result, nil
	}

	endpointURL := strings.TrimSpace(request.EndpointURL)
	if endpointURL == "" {
		result.Error = "missing endpoint URL"
		return result, nil
	}
	allowed, reason := CaptureEndpointAllowed(endpointURL)
	if !allowed {
		result.Blocked = true
		result.Error = "send blocked: " + reason
		return result, nil
	}

	method := strings.ToUpper(strings.TrimSpace(request.Method))
	if method == "" {
		method = http.MethodPost
	}
	timeoutMS := request.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = defaultTimeoutMS
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: time.Duration(timeoutMS) * time.Millisecond}
	}

	req, err := http.NewRequestWithContext(ctx, method, endpointURL, strings.NewReader(request.RawPayload))
	if err != nil {
		result.Error = fmt.Sprintf("build request: %v", err)
		return result, nil
	}

	contentType := strings.TrimSpace(request.ContentType)
	if contentType == "" {
		contentType = defaultContentType
	}
	req.Header.Set("Content-Type", contentType)
	for key, value := range request.Headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		req.Header.Set(trimmedKey, value)
	}

	start := clock()
	resp, err := client.Do(req)
	latency := time.Since(start)
	result.LatencyMS = int(latency.Milliseconds())
	if err != nil {
		result.Error = fmt.Sprintf("http send failed: %v", err)
		result.CompletedAt = clock().UTC()
		return result, nil
	}
	defer resp.Body.Close()

	excerptLimit := s.ResponseExcerptBytes
	if excerptLimit <= 0 {
		excerptLimit = defaultResponseExcerpt
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(excerptLimit)))
	result.HTTPStatusCode = resp.StatusCode
	result.ResponseExcerpt = string(body)
	result.Sent = resp.StatusCode >= 200 && resp.StatusCode <= 299
	if !result.Sent {
		result.Error = fmt.Sprintf("http send non-success status: %d", resp.StatusCode)
	}
	result.CompletedAt = clock().UTC()
	return result, nil
}
