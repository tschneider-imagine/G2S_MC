package fakeegm

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	HostURL    string
	EGMID      string
	HTTPClient *http.Client
}

type Response struct {
	StatusCode int
	Body       string
	Duration   time.Duration
}

func New(hostURL string, egmID string) *Client {
	return &Client{
		HostURL: hostURL,
		EGMID:   egmID,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func NewWithCAFile(hostURL string, egmID string, caPath string) (*Client, error) {
	return NewWithTLSFiles(hostURL, egmID, caPath, "", "")
}

func NewWithTLSFiles(hostURL string, egmID string, caPath string, certPath string, keyPath string) (*Client, error) {
	client := New(hostURL, egmID)
	if caPath == "" && certPath == "" && keyPath == "" {
		return client, nil
	}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if caPath != "" {
		raw, err := os.ReadFile(caPath)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(raw) {
			return nil, fmt.Errorf("no CA certificate found in %s", caPath)
		}
		tlsConfig.RootCAs = pool
	}
	if certPath != "" || keyPath != "" {
		if certPath == "" || keyPath == "" {
			return nil, fmt.Errorf("both client certificate and key are required when using mutual TLS")
		}
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	client.HTTPClient.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	return client, nil
}

func (c *Client) CommsOnLine(ctx context.Context) (Response, error) {
	return c.send(ctx, "commsOnLine")
}

func (c *Client) KeepAlive(ctx context.Context) (Response, error) {
	return c.send(ctx, "keepAlive")
}

func (c *Client) send(ctx context.Context, message string) (Response, error) {
	if c.HostURL == "" {
		return Response{}, fmt.Errorf("host URL is required")
	}
	if c.EGMID == "" {
		return Response{}, fmt.Errorf("EGM ID is required")
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	payload := buildEnvelope(c.EGMID, message)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.HostURL, bytes.NewReader([]byte(payload)))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Response{}, err
	}

	result := Response{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Duration:   time.Since(start),
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("host returned HTTP %d", resp.StatusCode)
	}
	return result, nil
}

func buildEnvelope(egmID string, message string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">` +
		`<soap:Body>` +
		`<g2sBody egmId="` + escapeAttribute(egmID) + `">` +
		`<` + message + `/>` +
		`</g2sBody>` +
		`</soap:Body>` +
		`</soap:Envelope>`
}

func escapeAttribute(value string) string {
	escaped := make([]rune, 0, len(value))
	for _, r := range value {
		switch r {
		case '&':
			escaped = append(escaped, []rune("&amp;")...)
		case '"':
			escaped = append(escaped, []rune("&quot;")...)
		case '<':
			escaped = append(escaped, []rune("&lt;")...)
		case '>':
			escaped = append(escaped, []rune("&gt;")...)
		default:
			escaped = append(escaped, r)
		}
	}
	return string(escaped)
}
