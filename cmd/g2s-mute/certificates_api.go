package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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

type certificatePreviewResponse struct {
	Role           string     `json:"role"`
	ParseOK        bool       `json:"parse_ok"`
	CertSubject    string     `json:"cert_subject,omitempty"`
	CertIssuer     string     `json:"cert_issuer,omitempty"`
	NotBefore      *time.Time `json:"not_before,omitempty"`
	NotAfter       *time.Time `json:"not_after,omitempty"`
	SANDNS         []string   `json:"san_dns"`
	SANIPs         []string   `json:"san_ips"`
	KeyRequired    bool       `json:"key_required"`
	KeyPresent     bool       `json:"key_present"`
	KeyMatchesCert bool       `json:"key_matches_cert"`
	Errors         []string   `json:"errors"`
}

type certificateBackupMaterialSummary struct {
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256,omitempty"`
}

type certificateBackupRecord struct {
	ID          string                            `json:"id"`
	CreatedAt   *time.Time                        `json:"created_at,omitempty"`
	Certificate *certificateBackupMaterialSummary `json:"certificate,omitempty"`
	PrivateKey  *certificateBackupMaterialSummary `json:"private_key,omitempty"`
	TotalSize   int64                             `json:"total_size_bytes"`
	Restorable  bool                              `json:"restorable"`
}

type certificateBackupsResponse struct {
	Role    string                    `json:"role"`
	Backups []certificateBackupRecord `json:"backups"`
}

type certificateRestoreRequest struct {
	Role     string `json:"role"`
	BackupID string `json:"backup_id"`
}

type certificateRestoreResponse struct {
	Role                 string                       `json:"role"`
	BackupID             string                       `json:"backup_id"`
	RestoredAt           time.Time                    `json:"restored_at"`
	CertificatePath      string                       `json:"certificate_path"`
	PrivateKeyPath       string                       `json:"private_key_path,omitempty"`
	CertificateStatus    string                       `json:"certificate_status"`
	CertificateInventory []model.CertificateInventory `json:"certificate_inventory"`
}

var (
	errCertificateBackupNotFound = errors.New("certificate backup not found")
	errCertificateBackupInvalid  = errors.New("certificate backup is invalid")
)

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

func certificatePreviewHandler(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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

		writeJSON(w, buildCertificatePreviewResponse(role, request), nil)
	}
}

func certificateBackupsHandler(cfg config.Config) http.HandlerFunc {
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
		backups, err := listCertificateBackups(role)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		writeJSON(w, certificateBackupsResponse{
			Role:    role.Role,
			Backups: backups,
		}, nil)
	}
}

func certificateRestoreHandler(store *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
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

		var request certificateRestoreRequest
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
		backupID := strings.TrimSpace(request.BackupID)
		if backupID == "" {
			http.Error(w, "backup_id is required", http.StatusBadRequest)
			return
		}

		if err := restoreCertificateBackup(role, backupID); err != nil {
			switch {
			case errors.Is(err, errCertificateBackupNotFound):
				http.Error(w, err.Error(), http.StatusNotFound)
			case errors.Is(err, errCertificateBackupInvalid):
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			default:
				writeJSON(w, nil, err)
			}
			return
		}

		restoredAt := time.Now().UTC()
		inventory, err := refreshCertificateInventory(r.Context(), store, cfg, restoredAt)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		writeJSON(w, certificateRestoreResponse{
			Role:                 role.Role,
			BackupID:             backupID,
			RestoredAt:           restoredAt,
			CertificatePath:      role.CertificatePath,
			PrivateKeyPath:       role.PrivateKeyPath,
			CertificateStatus:    certificateStatusByRole(inventory, role.Role),
			CertificateInventory: inventory,
		}, nil)
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
	preview := buildCertificatePreviewResponse(role, request)
	if !preview.ParseOK {
		return nil, errors.New(strings.Join(preview.Errors, "; "))
	}
	certificate, err := parseCertificatePEM(strings.TrimSpace(request.CertificatePEM))
	if err != nil {
		return nil, fmt.Errorf("invalid certificate_pem: %w", err)
	}
	return certificate, nil
}

func buildCertificatePreviewResponse(role certificateRolePaths, request certificateImportRequest) certificatePreviewResponse {
	certificatePEM := strings.TrimSpace(request.CertificatePEM)
	privateKeyPEM := strings.TrimSpace(request.PrivateKeyPEM)
	response := certificatePreviewResponse{
		Role:           role.Role,
		ParseOK:        false,
		SANDNS:         []string{},
		SANIPs:         []string{},
		KeyRequired:    role.RequiresPrivateKey,
		KeyPresent:     privateKeyPEM != "",
		KeyMatchesCert: !role.RequiresPrivateKey && privateKeyPEM == "",
		Errors:         []string{},
	}

	var parsedCertificate *x509.Certificate
	if certificatePEM == "" {
		response.Errors = append(response.Errors, "certificate_pem is required")
	} else {
		certificate, err := parseCertificatePEM(certificatePEM)
		if err != nil {
			response.Errors = append(response.Errors, "invalid certificate_pem: "+err.Error())
		} else {
			parsedCertificate = certificate
		}
	}

	if parsedCertificate != nil {
		response.CertSubject = parsedCertificate.Subject.String()
		response.CertIssuer = parsedCertificate.Issuer.String()
		notBefore := parsedCertificate.NotBefore.UTC()
		notAfter := parsedCertificate.NotAfter.UTC()
		response.NotBefore = &notBefore
		response.NotAfter = &notAfter
		response.SANDNS = append(response.SANDNS, parsedCertificate.DNSNames...)
		for _, ip := range parsedCertificate.IPAddresses {
			response.SANIPs = append(response.SANIPs, ip.String())
		}
	}

	if role.RequiresPrivateKey {
		if privateKeyPEM == "" {
			response.Errors = append(response.Errors, "private_key_pem is required for this role")
		} else {
			if _, err := parsePrivateKeyPEM(privateKeyPEM); err != nil {
				response.Errors = append(response.Errors, "invalid private_key_pem: "+err.Error())
			} else if parsedCertificate != nil {
				if _, err := tls.X509KeyPair([]byte(certificatePEM), []byte(privateKeyPEM)); err != nil {
					response.Errors = append(response.Errors, "certificate and private key do not match: "+err.Error())
				} else {
					response.KeyMatchesCert = true
				}
			}
		}
	} else if privateKeyPEM != "" {
		response.Errors = append(response.Errors, "private_key_pem is not allowed for role "+role.Role)
		response.KeyMatchesCert = false
	}

	response.ParseOK = len(response.Errors) == 0
	return response
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

func listCertificateBackups(role certificateRolePaths) ([]certificateBackupRecord, error) {
	certFilesByID, err := backupFilesByID(role.CertificatePath)
	if err != nil {
		return nil, err
	}
	keyFilesByID := map[string]string{}
	if role.RequiresPrivateKey {
		keyFilesByID, err = backupFilesByID(role.PrivateKeyPath)
		if err != nil {
			return nil, err
		}
	}

	idSet := map[string]struct{}{}
	for id := range certFilesByID {
		idSet[id] = struct{}{}
	}
	for id := range keyFilesByID {
		idSet[id] = struct{}{}
	}

	records := make([]certificateBackupRecord, 0, len(idSet))
	for id := range idSet {
		record := certificateBackupRecord{
			ID: id,
		}
		if parsedAt, ok := parseBackupIDTime(id); ok {
			record.CreatedAt = &parsedAt
		}
		if certPath, ok := certFilesByID[id]; ok {
			meta, err := backupMaterialSummary(certPath)
			if err != nil {
				return nil, err
			}
			record.Certificate = &meta
			record.TotalSize += meta.SizeBytes
		}
		if keyPath, ok := keyFilesByID[id]; ok {
			meta, err := backupMaterialSummary(keyPath)
			if err != nil {
				return nil, err
			}
			record.PrivateKey = &meta
			record.TotalSize += meta.SizeBytes
		}
		record.Restorable = record.Certificate != nil && (!role.RequiresPrivateKey || record.PrivateKey != nil)
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		left := records[i]
		right := records[j]
		if left.CreatedAt != nil && right.CreatedAt != nil {
			return left.CreatedAt.After(*right.CreatedAt)
		}
		if left.CreatedAt != nil && right.CreatedAt == nil {
			return true
		}
		if left.CreatedAt == nil && right.CreatedAt != nil {
			return false
		}
		return left.ID > right.ID
	})

	return records, nil
}

func backupFilesByID(targetPath string) (map[string]string, error) {
	results := map[string]string{}
	glob := strings.TrimSpace(targetPath) + ".bak-*"
	matches, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}
	prefix := strings.TrimSpace(targetPath) + ".bak-"
	for _, path := range matches {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		id := strings.TrimSpace(strings.TrimPrefix(path, prefix))
		if id == "" {
			continue
		}
		results[id] = path
	}
	return results, nil
}

func parseBackupIDTime(backupID string) (time.Time, bool) {
	parsed, err := time.Parse("20060102T150405Z", strings.TrimSpace(backupID))
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func backupMaterialSummary(path string) (certificateBackupMaterialSummary, error) {
	info, err := os.Stat(path)
	if err != nil {
		return certificateBackupMaterialSummary{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return certificateBackupMaterialSummary{}, err
	}
	sum := sha256.Sum256(data)
	return certificateBackupMaterialSummary{
		SizeBytes: info.Size(),
		SHA256:    hex.EncodeToString(sum[:]),
	}, nil
}

func restoreCertificateBackup(role certificateRolePaths, backupID string) error {
	trimmedID := strings.TrimSpace(backupID)
	if trimmedID == "" {
		return errors.New("backup_id is required")
	}
	certBackupPath := role.CertificatePath + ".bak-" + trimmedID
	certificateData, err := os.ReadFile(certBackupPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", errCertificateBackupNotFound, certBackupPath)
		}
		return err
	}
	if _, err := parseCertificatePEM(string(certificateData)); err != nil {
		return fmt.Errorf("%w: invalid certificate backup: %s", errCertificateBackupInvalid, err.Error())
	}

	keyData := []byte{}
	if role.RequiresPrivateKey {
		keyBackupPath := role.PrivateKeyPath + ".bak-" + trimmedID
		keyData, err = os.ReadFile(keyBackupPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("%w: %s", errCertificateBackupNotFound, keyBackupPath)
			}
			return err
		}
		if _, err := parsePrivateKeyPEM(string(keyData)); err != nil {
			return fmt.Errorf("%w: invalid private key backup: %s", errCertificateBackupInvalid, err.Error())
		}
		if _, err := tls.X509KeyPair(certificateData, keyData); err != nil {
			return fmt.Errorf("%w: certificate and private key do not match: %s", errCertificateBackupInvalid, err.Error())
		}
	}

	certRestore, err := captureCurrentMaterial(role.CertificatePath)
	if err != nil {
		return err
	}
	keyRestore := materialSnapshot{}
	if role.RequiresPrivateKey {
		keyRestore, err = captureCurrentMaterial(role.PrivateKeyPath)
		if err != nil {
			return err
		}
	}

	certMode := certRestore.Mode
	if certMode == 0 {
		certMode = 0o644
	}
	if err := writeFileAtomically(role.CertificatePath, certificateData, certMode); err != nil {
		return err
	}
	if !role.RequiresPrivateKey {
		return nil
	}

	keyMode := keyRestore.Mode
	if keyMode == 0 {
		keyMode = 0o600
	}
	if err := writeFileAtomically(role.PrivateKeyPath, keyData, keyMode); err != nil {
		_ = restoreMaterialSnapshot(role.CertificatePath, certRestore)
		return err
	}
	return nil
}

type materialSnapshot struct {
	Exists bool
	Mode   os.FileMode
	Data   []byte
}

func captureCurrentMaterial(path string) (materialSnapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return materialSnapshot{}, nil
		}
		return materialSnapshot{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return materialSnapshot{}, err
	}
	return materialSnapshot{
		Exists: true,
		Mode:   info.Mode().Perm(),
		Data:   data,
	}, nil
}

func restoreMaterialSnapshot(path string, snapshot materialSnapshot) error {
	if !snapshot.Exists {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	mode := snapshot.Mode
	if mode == 0 {
		mode = 0o644
	}
	return writeFileAtomically(path, snapshot.Data, mode)
}

func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	targetPath := strings.TrimSpace(path)
	if targetPath == "" {
		return errors.New("target path is required")
	}
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, filepath.Base(targetPath)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Chmod(targetPath, mode)
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
