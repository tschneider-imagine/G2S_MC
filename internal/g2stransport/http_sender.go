package g2stransport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	ClientFactory        func(timeoutMS int) (*http.Client, error)
	Clock                func() time.Time
	ResponseExcerptBytes int
}

func (s *HTTPSender) Send(ctx context.Context, request SendRequest) (SendResult, error) {
	clock := s.Clock
	if clock == nil {
		clock = time.Now
	}
	result := SendResult{
		MessageID:     request.MessageID,
		EGMID:         request.EGMID,
		TransportMode: ModeHTTP,
		CompletedAt:   clock().UTC(),
	}

	endpointURL := strings.TrimSpace(request.EndpointURL)
	if endpointURL == "" {
		result.Error = "missing endpoint URL"
		return result, nil
	}
	parsedURL, parseErr := url.Parse(endpointURL)
	if parseErr != nil || strings.TrimSpace(parsedURL.Scheme) == "" || strings.TrimSpace(parsedURL.Host) == "" {
		result.Error = "invalid endpoint URL"
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
	var client *http.Client
	if s.ClientFactory != nil {
		factoryClient, factoryErr := s.ClientFactory(timeoutMS)
		if factoryErr != nil {
			result.Error = fmt.Sprintf("http client configuration failed: %v", factoryErr)
			return result, nil
		}
		client = factoryClient
	} else if s.Client != nil {
		clone := *s.Client
		clone.Timeout = time.Duration(timeoutMS) * time.Millisecond
		client = &clone
	} else {
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
