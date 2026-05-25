package certs

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
)

type Source struct {
	Role string
	Path string
}

func SourcesFromConfig(cfg config.Crypto) []Source {
	return []Source{
		{Role: "g2s_client_cert", Path: cfg.G2SClientCertPath},
		{Role: "g2s_client_key", Path: cfg.G2SClientKeyPath},
		{Role: "g2s_ca_cert", Path: cfg.G2SCAPath},
		{Role: "web_server_cert", Path: cfg.WebServerCertPath},
	}
}

func InspectAll(sources []Source, now time.Time) []model.CertificateInventory {
	results := make([]model.CertificateInventory, 0, len(sources))
	for _, source := range sources {
		results = append(results, Inspect(source, now))
	}
	return results
}

func Inspect(source Source, now time.Time) model.CertificateInventory {
	if isPrivateKeyRole(source.Role) {
		return inspectPrivateKey(source, now)
	}

	record := model.CertificateInventory{
		Role:          source.Role,
		Path:          source.Path,
		LastCheckedAt: now,
	}
	if source.Path == "" {
		record.Status = "NOT_CONFIGURED"
		return record
	}

	raw, err := os.ReadFile(source.Path)
	if err != nil {
		record.Status = "MISSING"
		record.Error = err.Error()
		return record
	}

	block, _ := pem.Decode(raw)
	if block == nil {
		record.Status = "INVALID"
		record.Error = "no PEM certificate block found"
		return record
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		record.Status = "INVALID"
		record.Error = err.Error()
		return record
	}

	fingerprint := sha256.Sum256(cert.Raw)
	notBefore := cert.NotBefore
	notAfter := cert.NotAfter
	record.Subject = cert.Subject.String()
	record.Issuer = cert.Issuer.String()
	record.NotBefore = &notBefore
	record.NotAfter = &notAfter
	record.SHA256Fingerprint = formatFingerprint(fingerprint[:])
	record.Status = statusFor(cert, now)
	return record
}

func inspectPrivateKey(source Source, now time.Time) model.CertificateInventory {
	record := model.CertificateInventory{
		Role:          source.Role,
		Path:          source.Path,
		LastCheckedAt: now,
	}
	if source.Path == "" {
		record.Status = "NOT_CONFIGURED"
		return record
	}

	raw, err := os.ReadFile(source.Path)
	if err != nil {
		record.Status = "MISSING"
		record.Error = err.Error()
		return record
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		record.Status = "INVALID"
		record.Error = "no PEM private key block found"
		return record
	}
	if _, err := parsePrivateKeyBlock(block.Bytes); err != nil {
		record.Status = "INVALID"
		record.Error = err.Error()
		return record
	}

	fingerprint := sha256.Sum256(block.Bytes)
	record.SHA256Fingerprint = formatFingerprint(fingerprint[:])
	record.Status = "VALID"
	return record
}

func parsePrivateKeyBlock(raw []byte) (any, error) {
	if key, err := x509.ParsePKCS1PrivateKey(raw); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(raw); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(raw); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("unsupported private key format")
}

func isPrivateKeyRole(role string) bool {
	normalized := strings.ToLower(strings.TrimSpace(role))
	return strings.HasSuffix(normalized, "_key")
}

func statusFor(cert *x509.Certificate, now time.Time) string {
	if now.Before(cert.NotBefore) {
		return "NOT_YET_VALID"
	}
	if now.After(cert.NotAfter) {
		return "EXPIRED"
	}
	if cert.NotAfter.Sub(now) <= 30*24*time.Hour {
		return "EXPIRING_SOON"
	}
	return "VALID"
}

func formatFingerprint(bytes []byte) string {
	encoded := stringsToUpper(hex.EncodeToString(bytes))
	out := make([]byte, 0, len(encoded)+len(encoded)/2)
	for i := 0; i < len(encoded); i += 2 {
		if i > 0 {
			out = append(out, ':')
		}
		out = append(out, encoded[i:i+2]...)
	}
	return string(out)
}

func stringsToUpper(value string) string {
	out := []byte(value)
	for i, b := range out {
		if b >= 'a' && b <= 'f' {
			out[i] = b - 32
		}
	}
	return string(out)
}

func ValidateNoInvalid(records []model.CertificateInventory) error {
	for _, record := range records {
		if record.Status == "INVALID" {
			return fmt.Errorf("%s certificate is invalid: %s", record.Role, record.Error)
		}
	}
	return nil
}
