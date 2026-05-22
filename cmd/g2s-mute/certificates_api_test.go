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
	"runtime"
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

func TestCertificatePreviewHandlerValidationScenarios(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.Config{
		Crypto: config.Crypto{
			G2SCAPath:         filepath.Join(tempDir, "ca.crt"),
			G2SClientCertPath: filepath.Join(tempDir, "client.crt"),
			G2SClientKeyPath:  filepath.Join(tempDir, "client.key"),
			WebServerCertPath: filepath.Join(tempDir, "server.crt"),
			WebServerKeyPath:  filepath.Join(tempDir, "server.key"),
		},
	}
	handler := certificatePreviewHandler(cfg)

	validCertPEM, validKeyPEM := generateTestCertificateAndKey(t, "preview-valid.local", 90*24*time.Hour)
	_, mismatchedKeyPEM := generateTestCertificateAndKey(t, "preview-mismatch.local", 90*24*time.Hour)

	tests := []struct {
		name            string
		request         certificateImportRequest
		wantParseOK     bool
		wantKeyRequired bool
		wantKeyPresent  bool
		wantKeyMatch    bool
		errorContains   []string
	}{
		{
			name: "invalid pem",
			request: certificateImportRequest{
				Role:           "g2s_ca_cert",
				CertificatePEM: "not-a-pem",
			},
			wantParseOK:     false,
			wantKeyRequired: false,
			wantKeyPresent:  false,
			wantKeyMatch:    true,
			errorContains:   []string{"invalid certificate_pem"},
		},
		{
			name: "missing key when required",
			request: certificateImportRequest{
				Role:           "g2s_client_cert",
				CertificatePEM: validCertPEM,
			},
			wantParseOK:     false,
			wantKeyRequired: true,
			wantKeyPresent:  false,
			wantKeyMatch:    false,
			errorContains:   []string{"private_key_pem is required"},
		},
		{
			name: "key mismatch",
			request: certificateImportRequest{
				Role:           "g2s_client_cert",
				CertificatePEM: validCertPEM,
				PrivateKeyPEM:  mismatchedKeyPEM,
			},
			wantParseOK:     false,
			wantKeyRequired: true,
			wantKeyPresent:  true,
			wantKeyMatch:    false,
			errorContains:   []string{"do not match"},
		},
		{
			name: "valid cert only",
			request: certificateImportRequest{
				Role:           "g2s_ca_cert",
				CertificatePEM: validCertPEM,
			},
			wantParseOK:     true,
			wantKeyRequired: false,
			wantKeyPresent:  false,
			wantKeyMatch:    true,
		},
		{
			name: "valid cert and key",
			request: certificateImportRequest{
				Role:           "g2s_client_cert",
				CertificatePEM: validCertPEM,
				PrivateKeyPEM:  validKeyPEM,
			},
			wantParseOK:     true,
			wantKeyRequired: true,
			wantKeyPresent:  true,
			wantKeyMatch:    true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			payload, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/certificates/preview", bytes.NewReader(payload))
			req.RemoteAddr = "127.0.0.1:8444"
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
			}

			var body certificatePreviewResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode preview response: %v", err)
			}
			if body.ParseOK != tc.wantParseOK {
				t.Fatalf("parse_ok = %v, want %v", body.ParseOK, tc.wantParseOK)
			}
			if body.KeyRequired != tc.wantKeyRequired {
				t.Fatalf("key_required = %v, want %v", body.KeyRequired, tc.wantKeyRequired)
			}
			if body.KeyPresent != tc.wantKeyPresent {
				t.Fatalf("key_present = %v, want %v", body.KeyPresent, tc.wantKeyPresent)
			}
			if body.KeyMatchesCert != tc.wantKeyMatch {
				t.Fatalf("key_matches_cert = %v, want %v", body.KeyMatchesCert, tc.wantKeyMatch)
			}
			for _, needle := range tc.errorContains {
				found := false
				for _, item := range body.Errors {
					if strings.Contains(item, needle) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected error containing %q, got %#v", needle, body.Errors)
				}
			}
			if tc.wantParseOK {
				if strings.TrimSpace(body.CertSubject) == "" {
					t.Fatal("expected cert_subject on parse success")
				}
				if strings.TrimSpace(body.CertIssuer) == "" {
					t.Fatal("expected cert_issuer on parse success")
				}
				if body.NotBefore == nil || body.NotAfter == nil {
					t.Fatal("expected validity window on parse success")
				}
			}
		})
	}
}

func TestCertificatePreviewHandlerReadOnlyAuthModel(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		Crypto: config.Crypto{
			G2SCAPath: filepath.Join(tempDir, "ca.crt"),
		},
	}
	handler := certificatePreviewHandler(cfg)
	certificatePEM, _ := generateTestCertificateAndKey(t, "preview-auth.local", 90*24*time.Hour)
	payload, err := json.Marshal(certificateImportRequest{
		Role:           "g2s_ca_cert",
		CertificatePEM: certificatePEM,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/certificates/preview", bytes.NewReader(payload))
	req.RemoteAddr = "127.0.0.1:9018"
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestCertificateBackupsHandlerListsRoleBackups(t *testing.T) {
	tempDir := t.TempDir()
	clientCertPath := filepath.Join(tempDir, "client.crt")
	clientKeyPath := filepath.Join(tempDir, "client.key")
	if err := os.WriteFile(clientCertPath, []byte("seed-cert"), 0o644); err != nil {
		t.Fatalf("seed cert: %v", err)
	}
	if err := os.WriteFile(clientKeyPath, []byte("seed-key"), 0o600); err != nil {
		t.Fatalf("seed key: %v", err)
	}

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		Crypto: config.Crypto{
			G2SClientCertPath: clientCertPath,
			G2SClientKeyPath:  clientKeyPath,
		},
	}
	role, err := resolveCertificateRole("g2s_client_cert", cfg.Crypto)
	if err != nil {
		t.Fatalf("resolve role: %v", err)
	}
	certPEM, keyPEM := generateTestCertificateAndKey(t, "backup-list.local", 90*24*time.Hour)
	if _, err := persistImportedCertificate(role, certPEM, keyPEM, time.Date(2026, 5, 22, 1, 2, 3, 0, time.UTC)); err != nil {
		t.Fatalf("persist imported certificate: %v", err)
	}

	handler := certificateBackupsHandler(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/certificates/backups?role=g2s_client_cert", nil)
	req.RemoteAddr = "127.0.0.1:9440"
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body certificateBackupsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode backups response: %v", err)
	}
	if body.Role != "g2s_client_cert" {
		t.Fatalf("role = %q", body.Role)
	}
	if len(body.Backups) == 0 {
		t.Fatal("expected at least one backup record")
	}
	first := body.Backups[0]
	if first.ID == "" {
		t.Fatal("expected backup id")
	}
	if first.Certificate == nil || first.Certificate.SizeBytes == 0 || strings.TrimSpace(first.Certificate.SHA256) == "" {
		t.Fatalf("expected certificate backup metadata, got %#v", first.Certificate)
	}
	if first.PrivateKey == nil || first.PrivateKey.SizeBytes == 0 || strings.TrimSpace(first.PrivateKey.SHA256) == "" {
		t.Fatalf("expected private key backup metadata, got %#v", first.PrivateKey)
	}
}

func TestCertificateRestoreHandlerSuccessAndMissingBackup(t *testing.T) {
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
	certA, keyA := generateTestCertificateAndKey(t, "restore-a.local", 90*24*time.Hour)
	certB, keyB := generateTestCertificateAndKey(t, "restore-b.local", 90*24*time.Hour)
	caCert, _ := generateTestCertificateAndKey(t, "restore-ca.local", 90*24*time.Hour)
	if err := os.WriteFile(clientCertPath, []byte(certA), 0o644); err != nil {
		t.Fatalf("seed cert: %v", err)
	}
	if err := os.WriteFile(clientKeyPath, []byte(keyA), 0o600); err != nil {
		t.Fatalf("seed key: %v", err)
	}
	if err := os.WriteFile(caPath, []byte(caCert), 0o644); err != nil {
		t.Fatalf("seed ca: %v", err)
	}

	cfg := config.Config{
		Crypto: config.Crypto{
			G2SCAPath:         caPath,
			G2SClientCertPath: clientCertPath,
			G2SClientKeyPath:  clientKeyPath,
		},
	}
	role, err := resolveCertificateRole("g2s_client_cert", cfg.Crypto)
	if err != nil {
		t.Fatalf("resolve role: %v", err)
	}
	backupTime := time.Date(2026, 5, 22, 2, 3, 4, 0, time.UTC)
	if _, err := persistImportedCertificate(role, certB, keyB, backupTime); err != nil {
		t.Fatalf("persist imported certificate: %v", err)
	}
	backupID := backupTime.Format("20060102T150405Z")
	handler := certificateRestoreHandler(auditStore, cfg)

	restorePayload, err := json.Marshal(certificateRestoreRequest{
		Role:     "g2s_client_cert",
		BackupID: backupID,
	})
	if err != nil {
		t.Fatalf("marshal restore payload: %v", err)
	}
	restoreReq := httptest.NewRequest(http.MethodPost, "/api/certificates/restore", bytes.NewReader(restorePayload))
	restoreReq.RemoteAddr = "127.0.0.1:9441"
	restoreRec := httptest.NewRecorder()
	handler(restoreRec, restoreReq)
	if restoreRec.Code != http.StatusOK {
		t.Fatalf("restore status = %d, want %d: %s", restoreRec.Code, http.StatusOK, restoreRec.Body.String())
	}

	var restoreBody certificateRestoreResponse
	if err := json.Unmarshal(restoreRec.Body.Bytes(), &restoreBody); err != nil {
		t.Fatalf("decode restore response: %v", err)
	}
	if restoreBody.BackupID != backupID {
		t.Fatalf("backup id = %q, want %q", restoreBody.BackupID, backupID)
	}
	if restoreBody.CertificateStatus == "" {
		t.Fatal("expected certificate status after restore")
	}
	if len(restoreBody.CertificateInventory) == 0 {
		t.Fatal("expected certificate inventory after restore")
	}

	certBytes, err := os.ReadFile(clientCertPath)
	if err != nil {
		t.Fatalf("read restored cert: %v", err)
	}
	keyBytes, err := os.ReadFile(clientKeyPath)
	if err != nil {
		t.Fatalf("read restored key: %v", err)
	}
	if strings.TrimSpace(string(certBytes)) != strings.TrimSpace(certA) {
		t.Fatal("expected certificate restore to previous backup value")
	}
	if strings.TrimSpace(string(keyBytes)) != strings.TrimSpace(keyA) {
		t.Fatal("expected private key restore to previous backup value")
	}

	missingPayload, err := json.Marshal(certificateRestoreRequest{
		Role:     "g2s_client_cert",
		BackupID: "missing-backup-id",
	})
	if err != nil {
		t.Fatalf("marshal missing payload: %v", err)
	}
	missingReq := httptest.NewRequest(http.MethodPost, "/api/certificates/restore", bytes.NewReader(missingPayload))
	missingReq.RemoteAddr = "127.0.0.1:9442"
	missingRec := httptest.NewRecorder()
	handler(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing backup status = %d, want %d", missingRec.Code, http.StatusNotFound)
	}
}

func TestCertificateBackupsAndRestoreAuthAndRoleValidation(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	tempDir := t.TempDir()
	caPath := filepath.Join(tempDir, "ca.crt")
	certificatePEM, _ := generateTestCertificateAndKey(t, "restore-auth.local", 90*24*time.Hour)
	if err := os.WriteFile(caPath, []byte(certificatePEM), 0o644); err != nil {
		t.Fatalf("seed ca: %v", err)
	}

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		Crypto: config.Crypto{
			G2SCAPath: caPath,
		},
	}
	backupsHandler := certificateBackupsHandler(cfg)
	backupsReq := httptest.NewRequest(http.MethodGet, "/api/certificates/backups?role=bad-role", nil)
	backupsReq.RemoteAddr = "127.0.0.1:9443"
	backupsRec := httptest.NewRecorder()
	backupsHandler(backupsRec, backupsReq)
	if backupsRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid role backups status = %d, want %d", backupsRec.Code, http.StatusBadRequest)
	}

	restoreHandler := certificateRestoreHandler(auditStore, cfg)
	restorePayload, err := json.Marshal(certificateRestoreRequest{
		Role:     "g2s_ca_cert",
		BackupID: "20260522T010203Z",
	})
	if err != nil {
		t.Fatalf("marshal restore payload: %v", err)
	}

	noTokenReq := httptest.NewRequest(http.MethodPost, "/api/certificates/restore", bytes.NewReader(restorePayload))
	noTokenReq.RemoteAddr = "127.0.0.1:9444"
	noTokenRec := httptest.NewRecorder()
	restoreHandler(noTokenRec, noTokenReq)
	if noTokenRec.Code != http.StatusUnauthorized {
		t.Fatalf("restore without token status = %d, want %d", noTokenRec.Code, http.StatusUnauthorized)
	}

	invalidRolePayload, err := json.Marshal(certificateRestoreRequest{
		Role:     "bad-role",
		BackupID: "20260522T010203Z",
	})
	if err != nil {
		t.Fatalf("marshal invalid role restore payload: %v", err)
	}
	invalidRoleReq := httptest.NewRequest(http.MethodPost, "/api/certificates/restore", bytes.NewReader(invalidRolePayload))
	invalidRoleReq.RemoteAddr = "127.0.0.1:9447"
	invalidRoleReq.Header.Set("Authorization", "Bearer lab-secret")
	invalidRoleRec := httptest.NewRecorder()
	restoreHandler(invalidRoleRec, invalidRoleReq)
	if invalidRoleRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid role restore status = %d, want %d", invalidRoleRec.Code, http.StatusBadRequest)
	}

	badTokenReq := httptest.NewRequest(http.MethodPost, "/api/certificates/restore", bytes.NewReader(restorePayload))
	badTokenReq.RemoteAddr = "127.0.0.1:9445"
	badTokenReq.Header.Set("Authorization", "Bearer wrong-token")
	badTokenRec := httptest.NewRecorder()
	restoreHandler(badTokenRec, badTokenReq)
	if badTokenRec.Code != http.StatusUnauthorized {
		t.Fatalf("restore with invalid token status = %d, want %d", badTokenRec.Code, http.StatusUnauthorized)
	}

	validTokenReq := httptest.NewRequest(http.MethodPost, "/api/certificates/restore", bytes.NewReader(restorePayload))
	validTokenReq.RemoteAddr = "127.0.0.1:9446"
	validTokenReq.Header.Set("Authorization", "Bearer lab-secret")
	validTokenRec := httptest.NewRecorder()
	restoreHandler(validTokenRec, validTokenReq)
	if validTokenRec.Code == http.StatusUnauthorized {
		t.Fatalf("expected authorized restore request, got %d", validTokenRec.Code)
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

	if runtime.GOOS != "windows" {
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

func TestCertificateImportHandlerAllowsTrustedPrivateNetworkWithoutToken(t *testing.T) {
	ctx := context.Background()
	auditStore, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = auditStore.Close() })

	cfg := config.Config{
		API: config.API{AuthToken: "lab-secret"},
		WebUI: config.WebUI{
			RequireLogin:                        false,
			AllowTrustedPrivateNetworkMutations: true,
		},
		Crypto: config.Crypto{
			G2SClientCertPath: filepath.Join(t.TempDir(), "client.crt"),
			G2SClientKeyPath:  filepath.Join(t.TempDir(), "client.key"),
		},
	}
	handler := certificateImportHandler(auditStore, cfg)
	request := httptest.NewRequest(http.MethodPost, "/api/certificates/import", bytes.NewBufferString(`{}`))
	request.RemoteAddr = "192.168.10.55:4444"
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code == http.StatusUnauthorized {
		t.Fatalf("expected trusted private network request to bypass token auth, got %d", response.Code)
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

func TestCertificateExportHandlerAllowsTrustedPrivateNetwork(t *testing.T) {
	tempDir := t.TempDir()
	certificatePEM, _ := generateTestCertificateAndKey(t, "export-private.local", 90*24*time.Hour)
	caPath := filepath.Join(tempDir, "ca.crt")
	if err := os.WriteFile(caPath, []byte(certificatePEM), 0o644); err != nil {
		t.Fatalf("write ca cert: %v", err)
	}

	cfg := config.Config{
		WebUI: config.WebUI{
			RequireLogin:                        false,
			AllowTrustedPrivateNetworkMutations: true,
		},
		Crypto: config.Crypto{
			G2SCAPath: caPath,
		},
	}
	handler := certificateExportHandler(cfg)
	request := httptest.NewRequest(http.MethodGet, "/api/certificates/export?role=g2s_ca_cert", nil)
	request.RemoteAddr = "192.168.10.56:4040"
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
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
