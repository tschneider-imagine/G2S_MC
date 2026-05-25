//go:build devtools

package fakeegm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/certs"
	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2s"
)

func TestClientStartsSessionAgainstG2SServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-01", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	g2s.NewServer("HOST-001", eng).RegisterRoutes(mux, "/g2s")
	server := httptest.NewServer(mux)
	defer server.Close()

	client := New(server.URL+"/g2s", "EGM-01")
	response, err := client.CommsOnLine(ctx)
	if err != nil {
		t.Fatalf("comms online: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", response.StatusCode)
	}
	if !strings.Contains(response.Body, "commsOnLineAck") {
		t.Fatalf("expected commsOnLineAck, got %s", response.Body)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := eng.Snapshot()
		if len(snapshot.EGMs) == 1 && !snapshot.EGMs[0].LastSeen.IsZero() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected fake EGM to update engine session state")
}

func TestClientSendsKeepAlive(t *testing.T) {
	var sawKeepAlive bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		sawKeepAlive = strings.Contains(string(body), "keepAlive")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<keepAliveAck/>"))
	}))
	defer server.Close()

	client := New(server.URL, "EGM-01")
	if _, err := client.KeepAlive(context.Background()); err != nil {
		t.Fatalf("keepalive: %v", err)
	}
	if !sawKeepAlive {
		t.Fatal("expected keepAlive in request body")
	}
}

func TestClientUsesMutualTLSFiles(t *testing.T) {
	paths, err := certs.GenerateDevCerts(certs.DevCertOptions{OutputDir: t.TempDir()})
	if err != nil {
		t.Fatalf("generate dev certs: %v", err)
	}

	hostCert, err := tls.LoadX509KeyPair(paths.HostCert, paths.HostKey)
	if err != nil {
		t.Fatalf("load host key pair: %v", err)
	}
	caPEM, err := os.ReadFile(paths.CACert)
	if err != nil {
		t.Fatalf("read CA: %v", err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(caPEM) {
		t.Fatal("expected CA pool to load")
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			t.Error("expected verified client certificate")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<keepAliveAck/>"))
	}))
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{hostCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAs,
		MinVersion:   tls.VersionTLS12,
	}
	server.StartTLS()
	defer server.Close()

	client, err := NewWithTLSFiles(server.URL, "EGM-01", paths.CACert, paths.ClientCert, paths.ClientKey)
	if err != nil {
		t.Fatalf("create mTLS client: %v", err)
	}
	if _, err := client.KeepAlive(context.Background()); err != nil {
		t.Fatalf("keepAlive with mTLS: %v", err)
	}
}
