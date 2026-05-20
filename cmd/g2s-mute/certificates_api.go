package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/certs"
	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

type certificateRolePaths struct {
	Role               string
	CertificatePath    string
	PrivateKeyPath     string
	RequiresPrivateKey bool
}

type certificateImportRequest struct {
	Role           string `json:"role"`
	CertificatePEM string `json:"certificate_pem"`
	PrivateKeyPEM  string `json:"private_key_pem"`
}

type certificateImportResponse struct {
	Role               string    `json:"role"`
	CertificatePath    string    `json:"certificate_path"`
	PrivateKeyPath     string    `json:"private_key_path,omitempty"`
	BackupPaths        []string  `json:"backup_paths,omitempty"`
	CertificateSubject string    `json:"certificate_subject"`
	CertificateStatus  string    `json:"certificate_status"`
	ImportedAt         time.Time `json:"imported_at"`
}

type certificateExportResponse struct {
	Role           string `json:"role"`
	CertificatePEM string `json:"certificate_pem"`
	PrivateKeyPEM  string `json:"private_key_pem,omitempty"`
}

func certificateImportHandler(store *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !requireMutationAuth(w, r, cfg) {
			return
		}
		if !certificateMaterialRequestAllowed(r, cfg) {
			http.Error(w, "forbidden: loopback or trusted private network requests only", http.StatusForbidden)
			return
		}

		var request certificateImportRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		role, err := resolveCertificateRole(request.Role, cfg.Crypto)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		parsedCert, err := validateCertificateImportPayload(role, request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		importedAt := time.Now().UTC()
		backups, err := persistImportedCertificate(role, request.CertificatePEM, request.PrivateKeyPEM, importedAt)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}

		inventory, err := refreshCertificateInventory(r.Context(), store, cfg, importedAt)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}

		response := certificateImportResponse{
			Role:               role.Role,
			CertificatePath:    role.CertificatePath,
			PrivateKeyPath:     role.PrivateKeyPath,
			BackupPaths:        backups,
			CertificateSubject: parsedCert.Subject.String(),
			CertificateStatus:  certificateStatusByRole(inventory, role.Role),
			ImportedAt:         importedAt,
		}
		writeJSON(w, response, nil)
	}
}

func certificateExportHandler(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !certificateMaterialRequestAllowed(r, cfg) {
			http.Error(w, "forbidden: loopback or trusted private network requests only", http.StatusForbidden)
			return
		}

		role, err := resolveCertificateRole(r.URL.Query().Get("role"), cfg.Crypto)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		includeKey, err := parseIncludeKeyParam(r.URL.Query().Get("include_key"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if includeKey {
			if !requireMutationAuth(w, r, cfg) {
				return
			}
			if !cfg.WebUI.AllowPrivateKeyExport {
				http.Error(w, "private key export is disabled by web_ui.allow_private_key_export", http.StatusForbidden)
				return
			}
			if !role.RequiresPrivateKey {
				http.Error(w, "role does not support private key export", http.StatusBadRequest)
				return
			}
		}

		certificatePEM, err := readPEMFile(role.CertificatePath, "certificate")
		if err != nil {
			http.Error(w, err.Error(), httpStatusForReadError(err))
			return
		}
		if _, err := parseCertificatePEM(certificatePEM); err != nil {
			http.Error(w, "configured certificate PEM is invalid: "+err.Error(), http.StatusUnprocessableEntity)
			return
		}

		response := certificateExportResponse{
			Role:           role.Role,
			CertificatePEM: certificatePEM,
		}

		if includeKey {
			privateKeyPEM, err := readPEMFile(role.PrivateKeyPath, "private key")
			if err != nil {
				http.Error(w, err.Error(), httpStatusForReadError(err))
				return
			}
			if _, err := parsePrivateKeyPEM(privateKeyPEM); err != nil {
				http.Error(w, "configured private key PEM is invalid: "+err.Error(), http.StatusUnprocessableEntity)
				return
			}
			response.PrivateKeyPEM = privateKeyPEM
		}

		writeJSON(w, response, nil)
	}
}

func resolveCertificateRole(role string, cryptoCfg config.Crypto) (certificateRolePaths, error) {
	normalizedRole := strings.TrimSpace(role)
	switch normalizedRole {
	case "g2s_ca_cert":
		if strings.TrimSpace(cryptoCfg.G2SCAPath) == "" {
			return certificateRolePaths{}, fmt.Errorf("role g2s_ca_cert is not configured: crypto.g2s_ca_cert_path is required")
		}
		return certificateRolePaths{
			Role:               normalizedRole,
			CertificatePath:    cryptoCfg.G2SCAPath,
			RequiresPrivateKey: false,
		}, nil
	case "g2s_client_cert":
		return roleWithKeyPaths(
			normalizedRole,
			cryptoCfg.G2SClientCertPath,
			cryptoCfg.G2SClientKeyPath,
			"crypto.g2s_client_cert_path",
			"crypto.g2s_client_key_path",
		)
	case "web_server_cert":
		return roleWithKeyPaths(
			normalizedRole,
			cryptoCfg.WebServerCertPath,
			cryptoCfg.WebServerKeyPath,
			"crypto.web_server_cert_path",
			"crypto.web_server_key_path",
		)
	default:
		return certificateRolePaths{}, fmt.Errorf("invalid role %q", normalizedRole)
	}
}

func roleWithKeyPaths(role string, certificatePath string, keyPath string, certificateField string, keyField string) (certificateRolePaths, error) {
	missing := []string{}
	if strings.TrimSpace(certificatePath) == "" {
		missing = append(missing, certificateField)
	}
	if strings.TrimSpace(keyPath) == "" {
		missing = append(missing, keyField)
	}
	if len(missing) > 0 {
		return certificateRolePaths{}, fmt.Errorf("role %s is not configured: %s required", role, strings.Join(missing, " and "))
	}
	return certificateRolePaths{
		Role:               role,
		CertificatePath:    certificatePath,
		PrivateKeyPath:     keyPath,
		RequiresPrivateKey: true,
	}, nil
}

func validateCertificateImportPayload(role certificateRolePaths, request certificateImportRequest) (*x509.Certificate, error) {
	certificatePEM := strings.TrimSpace(request.CertificatePEM)
	privateKeyPEM := strings.TrimSpace(request.PrivateKeyPEM)
	if certificatePEM == "" {
		return nil, errors.New("certificate_pem is required")
	}

	certificate, err := parseCertificatePEM(certificatePEM)
	if err != nil {
		return nil, fmt.Errorf("invalid certificate_pem: %w", err)
	}

	if role.RequiresPrivateKey {
		if privateKeyPEM == "" {
			return nil, errors.New("private_key_pem is required for this role")
		}
		if _, err := parsePrivateKeyPEM(privateKeyPEM); err != nil {
			return nil, fmt.Errorf("invalid private_key_pem: %w", err)
		}
		if _, err := tls.X509KeyPair([]byte(certificatePEM), []byte(privateKeyPEM)); err != nil {
			return nil, fmt.Errorf("certificate and private key do not match: %w", err)
		}
	} else if privateKeyPEM != "" {
		return nil, fmt.Errorf("private_key_pem is not allowed for role %s", role.Role)
	}

	return certificate, nil
}

func parseCertificatePEM(raw string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("unexpected PEM type %q", block.Type)
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return certificate, nil
}

func parsePrivateKeyPEM(raw string) (any, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		return x509.ParsePKCS8PrivateKey(block.Bytes)
	default:
		if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			return key, nil
		}
		return nil, fmt.Errorf("unexpected PEM type %q", block.Type)
	}
}

func persistImportedCertificate(role certificateRolePaths, certificatePEM string, privateKeyPEM string, now time.Time) ([]string, error) {
	backups := []string{}

	certificateBackup, err := safeWritePEM(role.CertificatePath, certificatePEM, 0o644, now)
	if err != nil {
		return nil, fmt.Errorf("persist certificate for %s: %w", role.Role, err)
	}
	if certificateBackup != "" {
		backups = append(backups, certificateBackup)
	}

	if role.RequiresPrivateKey {
		keyBackup, err := safeWritePEM(role.PrivateKeyPath, privateKeyPEM, 0o600, now)
		if err != nil {
			return nil, fmt.Errorf("persist private key for %s: %w", role.Role, err)
		}
		if keyBackup != "" {
			backups = append(backups, keyBackup)
		}
	}

	return backups, nil
}

func safeWritePEM(path string, pemContent string, mode os.FileMode, now time.Time) (string, error) {
	targetPath := strings.TrimSpace(path)
	if targetPath == "" {
		return "", errors.New("target path is required")
	}

	normalizedContent := strings.TrimSpace(pemContent)
	if normalizedContent == "" {
		return "", errors.New("PEM content is required")
	}
	data := []byte(normalizedContent + "\n")

	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	backupPath := ""
	if info, err := os.Stat(targetPath); err == nil {
		existing, readErr := os.ReadFile(targetPath)
		if readErr != nil {
			return "", readErr
		}
		backupPath = fmt.Sprintf("%s.bak-%s", targetPath, now.UTC().Format("20060102T150405Z"))
		if err := os.WriteFile(backupPath, existing, info.Mode().Perm()); err != nil {
			return "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	tmpFile, err := os.CreateTemp(dir, filepath.Base(targetPath)+".tmp-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Chmod(targetPath, mode); err != nil {
		return "", err
	}

	return backupPath, nil
}

func refreshCertificateInventory(ctx context.Context, store *store.SQLiteStore, cfg config.Config, now time.Time) ([]model.CertificateInventory, error) {
	records := certs.InspectAll(certs.SourcesFromConfig(cfg.Crypto), now)
	if err := store.ReplaceCertificateInventory(ctx, records); err != nil {
		return nil, err
	}
	return records, nil
}

func certificateStatusByRole(records []model.CertificateInventory, role string) string {
	for _, record := range records {
		if record.Role == role {
			return record.Status
		}
	}
	return "UNKNOWN"
}

func parseIncludeKeyParam(raw string) (bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false, nil
	}
	value, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, errors.New("include_key must be true or false")
	}
	return value, nil
}

func readPEMFile(path string, materialType string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%s file is missing at %s", materialType, path)
		}
		return "", err
	}
	return string(raw), nil
}

func httpStatusForReadError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if strings.Contains(err.Error(), "is missing at") {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}
