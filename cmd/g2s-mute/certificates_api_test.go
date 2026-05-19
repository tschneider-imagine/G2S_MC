package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestResolveCertificateRole(t *testing.T) {
	cryptoCfg := config.Crypto{
		G2SCAPath:         "/etc/g2s-mute/certs/ca.crt",
		G2SClientCertPath: "/etc/g2s-mute/certs/client.crt",
		G2SClientKeyPath:  "/etc/g2s-mute/certs/client.key",
		WebServerCertPath: "/etc/g2s-mute/certs/host.crt",
		WebServerKeyPath:  "/etc/g2s-mute/certs/host.key",
	}

	caRole, err := resolveCertificateRole("g2s_ca_cert", cryptoCfg)
	if err != nil {
		t.Fatalf("resolve g2s_ca_cert: %v", err)
	}
	if caRole.CertificatePath != cryptoCfg.G2SCAPath || caRole.RequiresPrivateKey {
		t.Fatalf("unexpected ca role resolution: %+v", caRole)
	}

	clientRole, err := resolveCertificateRole("g2s_client_cert", cryptoCfg)
	if err != nil {
		t.Fatalf("resolve g2s_client_cert: %v", err)
	}
	if clientRole.CertificatePath != cryptoCfg.G2SClientCertPath || clientRole.PrivateKeyPath != cryptoCfg.G2SClientKeyPath || !clientRole.RequiresPrivateKey {
		t.Fatalf("unexpected client role resolution: %+v", clientRole)
	}

	webRole, err := resolveCertificateRole("web_server_cert", cryptoCfg)
	if err != nil {
		t.Fatalf("resolve web_server_cert: %v", err)
	}
	if webRole.CertificatePath != cryptoCfg.WebServerCertPath || webRole.PrivateKeyPath != cryptoCfg.WebServerKeyPath || !webRole.RequiresPrivateKey {
		t.Fatalf("unexpected web role resolution: %+v", webRole)
	}

	if _, err := resolveCertificateRole("unknown_role", cryptoCfg); err == nil {
		t.Fatal("expected invalid role to fail")
	}
}

func TestValidateCertificateImportPayload(t *testing.T) {
	certificatePEM, privateKeyPEM := generateTestCertificateAndKey(t, "import-test.local", 90*24*time.Hour)
	_, mismatchedPrivateKey := generateTestCertificateAndKey(t, "mismatch.local", 90*24*time.Hour)

	keyRole := certificateRolePaths{
		Role:               "g2s_client_cert",
		RequiresPrivateKey: true,
	}
	certOnlyRole := certificateRolePaths{
		Role:               "g2s_ca_cert",
		RequiresPrivateKey: false,
	}

	if _, err := validateCertificateImportPayload(keyRole, certificateImportRequest{
		Role:           "g2s_client_cert",
		CertificatePEM: certificatePEM,
		PrivateKeyPEM:  privateKeyPEM,
	}); err != nil {
		t.Fatalf("expected valid certificate/key pair: %v", err)
	}

	if _, err := validateCertificateImportPayload(keyRole, certificateImportRequest{
		Role:           "g2s_client_cert",
		CertificatePEM: certificatePEM,
	}); err == nil {
		t.Fatal("expected missing private key to fail")
	}

	if _, err := validateCertificateImportPayload(keyRole, certificateImportRequest{
		Role:           "g2s_client_cert",
		CertificatePEM: certificatePEM,
		PrivateKeyPEM:  mismatchedPrivateKey,
	}); err == nil {
		t.Fatal("expected key mismatch to fail")
	}

	if _, err := validateCertificateImportPayload(certOnlyRole, certificateImportRequest{
		Role:           "g2s_ca_cert",
		CertificatePEM: certificatePEM,
		PrivateKeyPEM:  privateKeyPEM,
	}); err == nil {
		t.Fatal("expected private key on cert-only role to fail")
	}
}

func TestCertificateImportHandlerPersistsMaterialsAndRefreshesInventory(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	tempDir := t.TempDir()
	clientCertPath := filepath.Join(tempDir, "client.crt")
	clientKeyPath := filepath.Join(tempDir, "client.key")
	caPath := filepath.Join(tempDir, "ca.crt")
	if err := os.WriteFile(clientCertPath, []byte("old cert\n"), 0o644); err != nil {
		t.Fatalf("seed old cert: %v", err)
	}
	if err := os.WriteFile(clientKeyPath, []byte("old key\n"), 0o600); err != nil {
		t.Fatalf("seed old key: %v", err)
	}

	cfg := config.Config{
		Crypto: config.Crypto{
			G2SCAPath:         caPath,
			G2SClientCertPath: clientCertPath,
			G2SClientKeyPath:  clientKeyPath,
			WebServerCertPath: "",
			WebServerKeyPath:  "",
		},
	}
	handler := certificateImportHandler(auditStore, cfg)
	certificatePEM, privateKeyPEM := generateTestCertificateAndKey(t, "client-import.local", 90*24*time.Hour)

	payload, err := json.Marshal(certificateImportRequest{
		Role:           "g2s_client_cert",
		CertificatePEM: certificatePEM,
		PrivateKeyPEM:  privateKeyPEM,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/certificates/import", bytes.NewReader(payload))
	request.RemoteAddr = "127.0.0.1:8444"
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var body certificateImportResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Role != "g2s_client_cert" {
		t.Fatalf("role = %q", body.Role)
	}
	if body.CertificateStatus == "" {
		t.Fatal("expected certificate status in response")
	}

	certificateBytes, err := os.ReadFile(clientCertPath)
	if err != nil {
		t.Fatalf("read client cert: %v", err)
	}
	if !strings.Contains(string(certificateBytes), "BEGIN CERTIFICATE") {
		t.Fatalf("unexpected cert content: %q", string(certificateBytes))
	}

	keyBytes, err := os.ReadFile(clientKeyPath)
	if err != nil {
		t.Fatalf("read client key: %v", err)
	}
	if !strings.Contains(string(keyBytes), "BEGIN RSA PRIVATE KEY") {
		t.Fatalf("unexpected key content: %q", string(keyBytes))
	}

	certInfo, err := os.Stat(clientCertPath)
	if err != nil {
		t.Fatalf("stat cert: %v", err)
	}
	if certInfo.Mode().Perm() != 0o644 {
		t.Fatalf("cert mode = %o, want 644", certInfo.Mode().Perm())
	}

	keyInfo, err := os.Stat(clientKeyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("key mode = %o, want 600", keyInfo.Mode().Perm())
	}

	certBackups, err := filepath.Glob(clientCertPath + ".bak-*")
	if err != nil {
		t.Fatalf("glob cert backups: %v", err)
	}
	if len(certBackups) == 0 {
		t.Fatal("expected cert backup file")
	}

	keyBackups, err := filepath.Glob(clientKeyPath + ".bak-*")
	if err != nil {
		t.Fatalf("glob key backups: %v", err)
	}
	if len(keyBackups) == 0 {
		t.Fatal("expected key backup file")
	}

	inventory, err := auditStore.ListCertificateInventory(ctx)
	if err != nil {
		t.Fatalf("list certificate inventory: %v", err)
	}
	status := certificateStatusByRole(inventory, "g2s_client_cert")
	if !strings.HasPrefix(status, "VALID") && !strings.HasPrefix(status, "EXPIRING_SOON") {
		t.Fatalf("unexpected g2s_client_cert status: %q", status)
	}
}

func TestCertificateImportHandlerLoopbackRestriction(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		Crypto: config.Crypto{
			G2SClientCertPath: filepath.Join(t.TempDir(), "client.crt"),
			G2SClientKeyPath:  filepath.Join(t.TempDir(), "client.key"),
		},
	}
	handler := certificateImportHandler(auditStore, cfg)
	request := httptest.NewRequest(http.MethodPost, "/api/certificates/import", bytes.NewBufferString(`{}`))
	request.RemoteAddr = "10.1.2.3:4444"
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestCertificateExportHandlerIncludeKeyGuardAndSuccess(t *testing.T) {
	tempDir := t.TempDir()
	certificatePEM, privateKeyPEM := generateTestCertificateAndKey(t, "export-test.local", 90*24*time.Hour)
	clientCertPath := filepath.Join(tempDir, "client.crt")
	clientKeyPath := filepath.Join(tempDir, "client.key")
	caPath := filepath.Join(tempDir, "ca.crt")
	if err := os.WriteFile(clientCertPath, []byte(certificatePEM), 0o644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(clientKeyPath, []byte(privateKeyPEM), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	if err := os.WriteFile(caPath, []byte(certificatePEM), 0o644); err != nil {
		t.Fatalf("write ca cert: %v", err)
	}

	cfg := config.Config{
		Crypto: config.Crypto{
			G2SCAPath:         caPath,
			G2SClientCertPath: clientCertPath,
			G2SClientKeyPath:  clientKeyPath,
		},
		WebUI: config.WebUI{
			AllowPrivateKeyExport: false,
		},
	}

	guardedHandler := certificateExportHandler(cfg)
	guardedRequest := httptest.NewRequest(http.MethodGet, "/api/certificates/export?role=g2s_client_cert&include_key=true", nil)
	guardedRequest.RemoteAddr = "127.0.0.1:6553"
	guardedResponse := httptest.NewRecorder()
	guardedHandler(guardedResponse, guardedRequest)
	if guardedResponse.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", guardedResponse.Code, http.StatusForbidden)
	}

	cfg.WebUI.AllowPrivateKeyExport = true
	exportHandler := certificateExportHandler(cfg)

	requestNoKey := httptest.NewRequest(http.MethodGet, "/api/certificates/export?role=g2s_client_cert", nil)
	requestNoKey.RemoteAddr = "127.0.0.1:9000"
	responseNoKey := httptest.NewRecorder()
	exportHandler(responseNoKey, requestNoKey)
	if responseNoKey.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", responseNoKey.Code, http.StatusOK, responseNoKey.Body.String())
	}
	var noKeyBody certificateExportResponse
	if err := json.Unmarshal(responseNoKey.Body.Bytes(), &noKeyBody); err != nil {
		t.Fatalf("decode no-key response: %v", err)
	}
	if noKeyBody.PrivateKeyPEM != "" {
		t.Fatal("expected no private key in default export response")
	}

	requestWithKey := httptest.NewRequest(http.MethodGet, "/api/certificates/export?role=g2s_client_cert&include_key=true", nil)
	requestWithKey.RemoteAddr = "127.0.0.1:9001"
	responseWithKey := httptest.NewRecorder()
	exportHandler(responseWithKey, requestWithKey)
	if responseWithKey.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", responseWithKey.Code, http.StatusOK, responseWithKey.Body.String())
	}
	var withKeyBody certificateExportResponse
	if err := json.Unmarshal(responseWithKey.Body.Bytes(), &withKeyBody); err != nil {
		t.Fatalf("decode with-key response: %v", err)
	}
	if withKeyBody.PrivateKeyPEM == "" {
		t.Fatal("expected private key in include_key export response")
	}

	invalidRoleRequest := httptest.NewRequest(http.MethodGet, "/api/certificates/export?role=g2s_ca_cert&include_key=true", nil)
	invalidRoleRequest.RemoteAddr = "127.0.0.1:9002"
	invalidRoleResponse := httptest.NewRecorder()
	exportHandler(invalidRoleResponse, invalidRoleRequest)
	if invalidRoleResponse.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", invalidRoleResponse.Code, http.StatusBadRequest)
	}
}

func TestCertificateExportHandlerLoopbackRestriction(t *testing.T) {
	cfg := config.Config{
		Crypto: config.Crypto{
			G2SCAPath: filepath.Join(t.TempDir(), "ca.crt"),
		},
	}
	handler := certificateExportHandler(cfg)
	request := httptest.NewRequest(http.MethodGet, "/api/certificates/export?role=g2s_ca_cert", nil)
	request.RemoteAddr = "10.5.6.7:4040"
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func generateTestCertificateAndKey(t *testing.T, commonName string, validFor time.Duration) (string, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	now := time.Now().UTC()
	template := x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano()),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(validFor),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
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

func TestCertificateImportHandlerAuthTokenGuard(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		Crypto: config.Crypto{
			G2SClientCertPath: filepath.Join(t.TempDir(), "client.crt"),
			G2SClientKeyPath:  filepath.Join(t.TempDir(), "client.key"),
		},
	}
	handler := certificateImportHandler(auditStore, cfg)

	requestNoToken := httptest.NewRequest(http.MethodPost, "/api/certificates/import", bytes.NewBufferString(`{}`))
	requestNoToken.RemoteAddr = "127.0.0.1:4444"
	responseNoToken := httptest.NewRecorder()
	handler(responseNoToken, requestNoToken)
	if responseNoToken.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", responseNoToken.Code, http.StatusUnauthorized)
	}

	requestInvalidToken := httptest.NewRequest(http.MethodPost, "/api/certificates/import", bytes.NewBufferString(`{}`))
	requestInvalidToken.RemoteAddr = "127.0.0.1:4445"
	requestInvalidToken.Header.Set("Authorization", "Bearer wrong-token")
	responseInvalidToken := httptest.NewRecorder()
	handler(responseInvalidToken, requestInvalidToken)
	if responseInvalidToken.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", responseInvalidToken.Code, http.StatusUnauthorized)
	}

	requestValidToken := httptest.NewRequest(http.MethodPost, "/api/certificates/import", bytes.NewBufferString(`{}`))
	requestValidToken.RemoteAddr = "127.0.0.1:4446"
	requestValidToken.Header.Set("Authorization", "Bearer lab-secret")
	responseValidToken := httptest.NewRecorder()
	handler(responseValidToken, requestValidToken)
	if responseValidToken.Code == http.StatusUnauthorized {
		t.Fatalf("expected auth to pass, got status %d: %s", responseValidToken.Code, responseValidToken.Body.String())
	}
}

func TestCertificateExportHandlerPrivateKeyAuthTokenGuard(t *testing.T) {
	tempDir := t.TempDir()
	certificatePEM, privateKeyPEM := generateTestCertificateAndKey(t, "export-auth-test.local", 90*24*time.Hour)
	clientCertPath := filepath.Join(tempDir, "client.crt")
	clientKeyPath := filepath.Join(tempDir, "client.key")
	if err := os.WriteFile(clientCertPath, []byte(certificatePEM), 0o644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(clientKeyPath, []byte(privateKeyPEM), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		Crypto: config.Crypto{
			G2SClientCertPath: clientCertPath,
			G2SClientKeyPath:  clientKeyPath,
		},
		WebUI: config.WebUI{AllowPrivateKeyExport: true},
	}
	handler := certificateExportHandler(cfg)

	certificateOnlyReq := httptest.NewRequest(http.MethodGet, "/api/certificates/export?role=g2s_client_cert", nil)
	certificateOnlyReq.RemoteAddr = "127.0.0.1:9010"
	certificateOnlyRec := httptest.NewRecorder()
	handler(certificateOnlyRec, certificateOnlyReq)
	if certificateOnlyRec.Code != http.StatusOK {
		t.Fatalf("certificate-only status = %d: %s", certificateOnlyRec.Code, certificateOnlyRec.Body.String())
	}

	withKeyNoTokenReq := httptest.NewRequest(http.MethodGet, "/api/certificates/export?role=g2s_client_cert&include_key=true", nil)
	withKeyNoTokenReq.RemoteAddr = "127.0.0.1:9011"
	withKeyNoTokenRec := httptest.NewRecorder()
	handler(withKeyNoTokenRec, withKeyNoTokenReq)
	if withKeyNoTokenRec.Code != http.StatusUnauthorized {
		t.Fatalf("include_key without token status = %d, want %d", withKeyNoTokenRec.Code, http.StatusUnauthorized)
	}

	withKeyBadTokenReq := httptest.NewRequest(http.MethodGet, "/api/certificates/export?role=g2s_client_cert&include_key=true", nil)
	withKeyBadTokenReq.RemoteAddr = "127.0.0.1:9012"
	withKeyBadTokenReq.Header.Set("Authorization", "Bearer wrong-token")
	withKeyBadTokenRec := httptest.NewRecorder()
	handler(withKeyBadTokenRec, withKeyBadTokenReq)
	if withKeyBadTokenRec.Code != http.StatusUnauthorized {
		t.Fatalf("include_key with invalid token status = %d, want %d", withKeyBadTokenRec.Code, http.StatusUnauthorized)
	}

	withKeyReq := httptest.NewRequest(http.MethodGet, "/api/certificates/export?role=g2s_client_cert&include_key=true", nil)
	withKeyReq.RemoteAddr = "127.0.0.1:9013"
	withKeyReq.Header.Set("Authorization", "Bearer lab-secret")
	withKeyRec := httptest.NewRecorder()
	handler(withKeyRec, withKeyReq)
	if withKeyRec.Code != http.StatusOK {
		t.Fatalf("include_key with token status = %d: %s", withKeyRec.Code, withKeyRec.Body.String())
	}
	var withKeyBody certificateExportResponse
	if err := json.Unmarshal(withKeyRec.Body.Bytes(), &withKeyBody); err != nil {
		t.Fatalf("decode include_key response: %v", err)
	}
	if strings.TrimSpace(withKeyBody.PrivateKeyPEM) == "" {
		t.Fatal("expected private key in include_key export response")
	}
}
