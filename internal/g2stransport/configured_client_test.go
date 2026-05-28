package g2stransport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/certs"
)

func TestConfiguredHTTPSenderSendsWithTrustedCA(t *testing.T) {
	paths := generateTransportCerts(t)
	server := newTLSServer(t, paths, false)
	defer server.Close()

	sender := NewConfiguredHTTPSender(HTTPClientConfig{
		TLSRequired:      true,
		RootCAPath:       paths.CACert,
		DefaultTimeoutMS: 3000,
	})

	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:     11,
		EGMID:         "EGM-001",
		EndpointURL:   server.URL,
		RawPayload:    "<cmd/>",
		TransportMode: ModeHTTP,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !result.Sent || result.HTTPStatusCode != http.StatusAccepted {
		t.Fatalf("unexpected send result: %+v", result)
	}
}

func TestConfiguredHTTPSenderFailsWhenTLSRequiredWithoutCA(t *testing.T) {
	paths := generateTransportCerts(t)
	server := newTLSServer(t, paths, false)
	defer server.Close()

	sender := NewConfiguredHTTPSender(HTTPClientConfig{
		TLSRequired:      true,
		DefaultTimeoutMS: 3000,
	})
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:     12,
		EGMID:         "EGM-001",
		EndpointURL:   server.URL,
		RawPayload:    "<cmd/>",
		TransportMode: ModeHTTP,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if result.Sent {
		t.Fatalf("expected send failure: %+v", result)
	}
	if !strings.Contains(strings.ToLower(result.Error), "ca trust path") {
		t.Fatalf("unexpected error: %q", result.Error)
	}
}

func TestConfiguredHTTPSenderSupportsMutualTLS(t *testing.T) {
	paths := generateTransportCerts(t)
	server := newTLSServer(t, paths, true)
	defer server.Close()

	sender := NewConfiguredHTTPSender(HTTPClientConfig{
		TLSRequired:      true,
		RootCAPath:       paths.CACert,
		ClientCertPath:   paths.ClientCert,
		ClientKeyPath:    paths.ClientKey,
		DefaultTimeoutMS: 3000,
	})
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:     13,
		EGMID:         "EGM-001",
		EndpointURL:   server.URL,
		RawPayload:    "<cmd/>",
		TransportMode: ModeHTTP,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !result.Sent {
		t.Fatalf("expected mutual TLS success: %+v", result)
	}
}

func TestConfiguredHTTPSenderInvalidClientKeyPathFailsClearly(t *testing.T) {
	paths := generateTransportCerts(t)
	server := newTLSServer(t, paths, false)
	defer server.Close()

	sender := NewConfiguredHTTPSender(HTTPClientConfig{
		TLSRequired:      true,
		RootCAPath:       paths.CACert,
		ClientCertPath:   paths.ClientCert,
		ClientKeyPath:    filepath.Join(t.TempDir(), "missing-client.key"),
		DefaultTimeoutMS: 3000,
	})
	result, err := sender.Send(context.Background(), SendRequest{
		MessageID:     14,
		EGMID:         "EGM-001",
		EndpointURL:   server.URL,
		RawPayload:    "<cmd/>",
		TransportMode: ModeHTTP,
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if result.Sent {
		t.Fatalf("expected send failure: %+v", result)
	}
	if strings.Contains(result.Error, "BEGIN PRIVATE KEY") {
		t.Fatalf("error must not include private key material: %q", result.Error)
	}
}

func generateTransportCerts(t *testing.T) certs.DevCertPaths {
	t.Helper()
	paths, err := certs.GenerateDevCerts(certs.DevCertOptions{
		OutputDir: t.TempDir(),
		HostDNS:   []string{"localhost"},
		Now:       time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("generate certs: %v", err)
	}
	return paths
}

func newTLSServer(t *testing.T, paths certs.DevCertPaths, requireClientCert bool) *httptest.Server {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("accepted"))
	})
	server := httptest.NewUnstartedServer(handler)

	certPair, err := tls.LoadX509KeyPair(paths.HostCert, paths.HostKey)
	if err != nil {
		t.Fatalf("load host cert: %v", err)
	}
	server.TLS = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certPair},
	}
	if requireClientCert {
		caBytes, readErr := os.ReadFile(paths.CACert)
		if readErr != nil {
			t.Fatalf("read ca cert: %v", readErr)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caBytes) {
			t.Fatalf("append ca cert")
		}
		server.TLS.ClientAuth = tls.RequireAndVerifyClientCert
		server.TLS.ClientCAs = caPool
	}
	server.StartTLS()
	return server
}

