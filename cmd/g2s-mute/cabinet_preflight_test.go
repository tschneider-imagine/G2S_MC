package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestEvaluateCabinetPreflightPass(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	tempDir := t.TempDir()
	webCertPath := filepath.Join(tempDir, "web.crt")
	webKeyPath := filepath.Join(tempDir, "web.key")
	webCertPEM, webKeyPEM := generateSANCertificateAndKey(t, "cabinet-prod.local", "")
	if err := os.WriteFile(webCertPath, []byte(webCertPEM), 0o644); err != nil {
		t.Fatalf("write web cert: %v", err)
	}
	if err := os.WriteFile(webKeyPath, []byte(webKeyPEM), 0o600); err != nil {
		t.Fatalf("write web key: %v", err)
	}

	cfg := config.Config{
		ControllerID: "G2S-MC-TEST",
		Database:     config.Database{Path: ":memory:"},
		WebUI:        config.WebUI{BindAddress: "127.0.0.1:8444"},
		Crypto: config.Crypto{
			WebServerCertPath: webCertPath,
			WebServerKeyPath:  webKeyPath,
		},
		G2S: config.G2S{
			HostURL:      "https://cabinet-prod.local:8444/g2s",
			EndpointPath: "/g2s",
			RequireTLS:   true,
		},
		CabinetProfile: config.CabinetProfile{
			WireHostURL:     "https://cabinet-prod.local:8444/g2s",
			ListenerDNSName: "cabinet-prod.local",
			RequiredSANDNS:  []string{"cabinet-prod.local"},
			HostID:          "HOST-PROD-1001",
			FirstTestEGMIDs: []string{"EGM-A100"},
		},
		EGMRoster: []config.EGM{{EGMID: "EGM-A100", IPAddress: "10.10.50.11", Port: 9443}},
	}

	if _, err := refreshCertificateInventory(ctx, auditStore, cfg, time.Now().UTC()); err != nil {
		t.Fatalf("refresh certificate inventory: %v", err)
	}

	eng := engine.New(cfg.ControllerID, cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	eng.Start(runCtx)
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now()})
	waitForLastEvent(t, eng, string(engine.EventBootComplete))

	result := evaluateCabinetPreflight(ctx, eng, auditStore, cfg, runtimeInfo{
		ConfigPath: "/etc/g2s-mute/config.json",
		StartedAt:  time.Now().Add(-10 * time.Second),
	})
	if result.Overall != preflightPass {
		t.Fatalf("overall = %q, want %q; blockers=%v", result.Overall, preflightPass, result.Blockers)
	}
	if len(result.Blockers) != 0 {
		t.Fatalf("unexpected blockers: %v", result.Blockers)
	}
	for _, check := range result.Checks {
		if check.Result != preflightPass {
			t.Fatalf("check %s = %s, detail=%s", check.ID, check.Result, check.Detail)
		}
	}
}

func TestEvaluateCabinetPreflightFailCases(t *testing.T) {
	t.Run("profile placeholder and SAN mismatch", func(t *testing.T) {
		ctx := context.Background()
		auditStore, err := store.Open(ctx, ":memory:")
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { _ = auditStore.Close() })

		tempDir := t.TempDir()
		webCertPath := filepath.Join(tempDir, "web.crt")
		webKeyPath := filepath.Join(tempDir, "web.key")
		webCertPEM, webKeyPEM := generateSANCertificateAndKey(t, "different-host.local", "")
		if err := os.WriteFile(webCertPath, []byte(webCertPEM), 0o644); err != nil {
			t.Fatalf("write web cert: %v", err)
		}
		if err := os.WriteFile(webKeyPath, []byte(webKeyPEM), 0o600); err != nil {
			t.Fatalf("write web key: %v", err)
		}

		cfg := config.Config{
			ControllerID: "G2S-MC-TEST",
			Database:     config.Database{Path: ":memory:"},
			WebUI:        config.WebUI{BindAddress: "127.0.0.1:8444"},
			Crypto: config.Crypto{
				WebServerCertPath: webCertPath,
				WebServerKeyPath:  webKeyPath,
			},
			G2S: config.G2S{
				HostURL:      "https://cabinet-host.example:8444/g2s",
				EndpointPath: "/g2s",
				RequireTLS:   true,
			},
			CabinetProfile: config.CabinetProfile{
				WireHostURL:     "https://cabinet-host.example:8444/g2s",
				ListenerDNSName: "cabinet-host.example",
				RequiredSANDNS:  []string{"cabinet-host.example"},
				HostID:          "HOST-PI-001",
				FirstTestEGMIDs: []string{"EGM-01"},
			},
			EGMRoster: []config.EGM{{EGMID: "EGM-A100", IPAddress: "10.10.50.11", Port: 9443}},
		}

		if _, err := refreshCertificateInventory(ctx, auditStore, cfg, time.Now().UTC()); err != nil {
			t.Fatalf("refresh certificate inventory: %v", err)
		}
		eng := engine.New(cfg.ControllerID, cfg.EGMRoster)
		runCtx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)
		eng.Start(runCtx)
		eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now()})
		waitForLastEvent(t, eng, string(engine.EventBootComplete))

		result := evaluateCabinetPreflight(ctx, eng, auditStore, cfg, runtimeInfo{
			ConfigPath: "/etc/g2s-mute/config.json",
			StartedAt:  time.Now().Add(-5 * time.Second),
		})
		if result.Overall != preflightFail {
			t.Fatalf("overall = %q, want FAIL", result.Overall)
		}
		assertCheckFailed(t, result.Checks, "cabinet_profile")
		assertCheckFailed(t, result.Checks, "certificate_san_wire_identity")
		if len(result.Blockers) == 0 {
			t.Fatal("expected blocker list for fail case")
		}
	})

	t.Run("required certificate missing", func(t *testing.T) {
		ctx := context.Background()
		auditStore, err := store.Open(ctx, ":memory:")
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { _ = auditStore.Close() })

		tempDir := t.TempDir()
		cfg := config.Config{
			ControllerID: "G2S-MC-TEST",
			Database:     config.Database{Path: ":memory:"},
			WebUI:        config.WebUI{BindAddress: "127.0.0.1:8444"},
			Crypto: config.Crypto{
				WebServerCertPath: filepath.Join(tempDir, "missing-web.crt"),
				WebServerKeyPath:  filepath.Join(tempDir, "missing-web.key"),
			},
			G2S: config.G2S{
				HostURL:      "https://cabinet-prod.local:8444/g2s",
				EndpointPath: "/g2s",
				RequireTLS:   true,
			},
			CabinetProfile: config.CabinetProfile{
				WireHostURL:     "https://cabinet-prod.local:8444/g2s",
				ListenerDNSName: "cabinet-prod.local",
				RequiredSANDNS:  []string{"cabinet-prod.local"},
				HostID:          "HOST-PROD-1001",
				FirstTestEGMIDs: []string{"EGM-A100"},
			},
			EGMRoster: []config.EGM{{EGMID: "EGM-A100", IPAddress: "10.10.50.11", Port: 9443}},
		}

		if _, err := refreshCertificateInventory(ctx, auditStore, cfg, time.Now().UTC()); err != nil {
			t.Fatalf("refresh certificate inventory: %v", err)
		}
		eng := engine.New(cfg.ControllerID, cfg.EGMRoster)
		runCtx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)
		eng.Start(runCtx)
		eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now()})
		waitForLastEvent(t, eng, string(engine.EventBootComplete))

		result := evaluateCabinetPreflight(ctx, eng, auditStore, cfg, runtimeInfo{
			ConfigPath: "/etc/g2s-mute/config.json",
			StartedAt:  time.Now().Add(-5 * time.Second),
		})
		if result.Overall != preflightFail {
			t.Fatalf("overall = %q, want FAIL", result.Overall)
		}
		assertCheckFailed(t, result.Checks, "certificate_mode_requirements")
	})
}

func TestCabinetPreflightHandler(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	tempDir := t.TempDir()
	webCertPath := filepath.Join(tempDir, "web.crt")
	webKeyPath := filepath.Join(tempDir, "web.key")
	webCertPEM, webKeyPEM := generateSANCertificateAndKey(t, "cabinet-prod.local", "")
	if err := os.WriteFile(webCertPath, []byte(webCertPEM), 0o644); err != nil {
		t.Fatalf("write web cert: %v", err)
	}
	if err := os.WriteFile(webKeyPath, []byte(webKeyPEM), 0o600); err != nil {
		t.Fatalf("write web key: %v", err)
	}

	cfg := config.Config{
		ControllerID: "G2S-MC-TEST",
		Database:     config.Database{Path: ":memory:"},
		WebUI:        config.WebUI{BindAddress: "127.0.0.1:8444"},
		Crypto: config.Crypto{
			WebServerCertPath: webCertPath,
			WebServerKeyPath:  webKeyPath,
		},
		G2S: config.G2S{
			HostURL:      "https://cabinet-prod.local:8444/g2s",
			EndpointPath: "/g2s",
			RequireTLS:   true,
		},
		CabinetProfile: config.CabinetProfile{
			WireHostURL:     "https://cabinet-prod.local:8444/g2s",
			ListenerDNSName: "cabinet-prod.local",
			RequiredSANDNS:  []string{"cabinet-prod.local"},
			HostID:          "HOST-PROD-1001",
			FirstTestEGMIDs: []string{"EGM-A100"},
		},
		EGMRoster: []config.EGM{{EGMID: "EGM-A100", IPAddress: "10.10.50.11", Port: 9443}},
	}
	if _, err := refreshCertificateInventory(ctx, auditStore, cfg, time.Now().UTC()); err != nil {
		t.Fatalf("refresh certificate inventory: %v", err)
	}

	eng := engine.New(cfg.ControllerID, cfg.EGMRoster)
	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	eng.Start(runCtx)
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now()})
	waitForLastEvent(t, eng, string(engine.EventBootComplete))
	handler := cabinetPreflightHandler(eng, auditStore, cfg, runtimeInfo{
		ConfigPath: "/etc/g2s-mute/config.json",
		StartedAt:  time.Now().Add(-10 * time.Second),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/cabinet-preflight", nil)
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var payload cabinetPreflightResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Timestamp.IsZero() {
		t.Fatal("expected timestamp to be set")
	}
	if len(payload.Checks) == 0 {
		t.Fatal("expected checks in preflight response")
	}

	methodRequest := httptest.NewRequest(http.MethodPost, "/api/cabinet-preflight", nil)
	methodResponse := httptest.NewRecorder()
	handler(methodResponse, methodRequest)
	if methodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", methodResponse.Code)
	}
}

func generateSANCertificateAndKey(t *testing.T, dnsName string, ipAddress string) (string, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	now := time.Now().UTC()
	template := x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano()),
		Subject:      pkix.Name{CommonName: "preflight-cert"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if strings.TrimSpace(dnsName) != "" {
		template.DNSNames = []string{dnsName}
	}
	if strings.TrimSpace(ipAddress) != "" {
		parsedIP := net.ParseIP(strings.TrimSpace(ipAddress))
		if parsedIP == nil {
			t.Fatalf("invalid ipAddress for test helper: %q", ipAddress)
		}
		template.IPAddresses = []net.IP{parsedIP}
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	var certificateBuilder strings.Builder
	if err := pem.Encode(&certificateBuilder, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("encode certificate: %v", err)
	}

	var keyBuilder strings.Builder
	if err := pem.Encode(&keyBuilder, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}); err != nil {
		t.Fatalf("encode private key: %v", err)
	}

	return certificateBuilder.String(), keyBuilder.String()
}

func assertCheckFailed(t *testing.T, checks []cabinetPreflightCheck, id string) {
	t.Helper()
	for _, check := range checks {
		if check.ID == id {
			if check.Result != preflightFail {
				t.Fatalf("check %s result = %s, want FAIL", id, check.Result)
			}
			return
		}
	}
	t.Fatalf("check %s not found", id)
}
