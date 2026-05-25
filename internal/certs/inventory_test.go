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

	"github.com/tschneider-imagine/G2S_MC/internal/config"
)

func TestInspectMissingCertificate(t *testing.T) {
	record := Inspect(Source{Role: "missing", Path: filepath.Join(t.TempDir(), "missing.crt")}, time.Now())
	if record.Status != "MISSING" {
		t.Fatalf("status = %s, want MISSING", record.Status)
	}
}

func TestInspectNotConfigured(t *testing.T) {
	record := Inspect(Source{Role: "g2s_client_cert", Path: ""}, time.Now())
	if record.Status != "NOT_CONFIGURED" {
		t.Fatalf("status = %s, want NOT_CONFIGURED", record.Status)
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

func TestInspectInvalidCertificate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.crt")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatalf("write invalid cert: %v", err)
	}
	record := Inspect(Source{Role: "g2s_client_cert", Path: path}, time.Now().UTC())
	if record.Status != "INVALID" {
		t.Fatalf("status = %s, want INVALID", record.Status)
	}
	if record.Error == "" {
		t.Fatal("expected parse error")
	}
}

func TestInspectValidPrivateKey(t *testing.T) {
	path := writeTestKey(t)
	record := Inspect(Source{Role: "g2s_client_key", Path: path}, time.Now().UTC())
	if record.Status != "VALID" {
		t.Fatalf("status = %s, want VALID", record.Status)
	}
	if record.SHA256Fingerprint == "" {
		t.Fatal("expected fingerprint")
	}
	if record.Subject != "" || record.Issuer != "" {
		t.Fatalf("private key metadata should not set subject/issuer: %+v", record)
	}
}

func TestInspectInvalidPrivateKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.key")
	if err := os.WriteFile(path, []byte("-----BEGIN PRIVATE KEY-----\ninvalid\n-----END PRIVATE KEY-----\n"), 0o600); err != nil {
		t.Fatalf("write invalid key: %v", err)
	}
	record := Inspect(Source{Role: "g2s_client_key", Path: path}, time.Now().UTC())
	if record.Status != "INVALID" {
		t.Fatalf("status = %s, want INVALID", record.Status)
	}
	if record.Error == "" {
		t.Fatal("expected parse error")
	}
}

func TestSourcesFromConfigIncludesClientKeyRole(t *testing.T) {
	sources := SourcesFromConfig(config.Crypto{
		G2SClientCertPath: "/certs/client.crt",
		G2SClientKeyPath:  "/certs/client.key",
		G2SCAPath:         "/certs/ca.crt",
		WebServerCertPath: "/certs/web.crt",
	})
	roleSet := map[string]bool{}
	for _, source := range sources {
		roleSet[source.Role] = true
	}
	for _, role := range []string{"g2s_client_cert", "g2s_client_key", "g2s_ca_cert", "web_server_cert"} {
		if !roleSet[role] {
			t.Fatalf("missing role %s in sources", role)
		}
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

func writeTestKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	path := filepath.Join(t.TempDir(), "test.key")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	defer file.Close()
	if err := pem.Encode(file, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		t.Fatalf("encode key: %v", err)
	}
	return path
}
