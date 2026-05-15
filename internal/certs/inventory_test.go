package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInspectMissingCertificate(t *testing.T) {
	record := Inspect(Source{Role: "missing", Path: filepath.Join(t.TempDir(), "missing.crt")}, time.Now())
	if record.Status != "MISSING" {
		t.Fatalf("status = %s, want MISSING", record.Status)
	}
}

func TestInspectValidCertificate(t *testing.T) {
	now := time.Now().UTC()
	certPath := writeTestCert(t, now.Add(-time.Hour), now.Add(48*time.Hour))

	record := Inspect(Source{Role: "test", Path: certPath}, now)
	if record.Status != "EXPIRING_SOON" {
		t.Fatalf("status = %s, want EXPIRING_SOON", record.Status)
	}
	if record.Subject == "" || record.Issuer == "" || record.SHA256Fingerprint == "" {
		t.Fatalf("expected certificate metadata: %+v", record)
	}
}

func writeTestCert(t *testing.T, notBefore time.Time, notAfter time.Time) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test.local"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	path := filepath.Join(t.TempDir(), "test.crt")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create cert file: %v", err)
	}
	defer file.Close()
	if err := pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("encode cert: %v", err)
	}
	return path
}
