package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

type DevCertOptions struct {
	OutputDir string
	HostDNS   []string
	HostIPs   []net.IP
	Now       time.Time
}

type DevCertPaths struct {
	CACert     string
	HostCert   string
	HostKey    string
	ClientCert string
	ClientKey  string
}

func GenerateDevCerts(options DevCertOptions) (DevCertPaths, error) {
	if options.OutputDir == "" {
		return DevCertPaths{}, fmt.Errorf("output directory is required")
	}
	if options.Now.IsZero() {
		options.Now = time.Now()
	}
	if len(options.HostDNS) == 0 {
		options.HostDNS = []string{"localhost"}
	}
	if len(options.HostIPs) == 0 {
		options.HostIPs = []net.IP{net.ParseIP("127.0.0.1")}
	}
	if err := os.MkdirAll(options.OutputDir, 0o700); err != nil {
		return DevCertPaths{}, err
	}

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return DevCertPaths{}, err
	}
	caTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "G2S MC Dev CA", OrganizationalUnit: []string{"G2S_dev_ca"}},
		NotBefore:             options.Now.Add(-time.Hour),
		NotAfter:              options.Now.AddDate(2, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return DevCertPaths{}, err
	}

	hostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return DevCertPaths{}, err
	}
	hostTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost", OrganizationalUnit: []string{"G2S_host"}},
		DNSNames:     options.HostDNS,
		IPAddresses:  options.HostIPs,
		NotBefore:    options.Now.Add(-time.Hour),
		NotAfter:     options.Now.AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	hostDER, err := x509.CreateCertificate(rand.Reader, &hostTemplate, &caTemplate, &hostKey.PublicKey, caKey)
	if err != nil {
		return DevCertPaths{}, err
	}

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return DevCertPaths{}, err
	}
	clientTemplate := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "fake-egm", OrganizationalUnit: []string{"G2S_egm"}},
		NotBefore:    options.Now.Add(-time.Hour),
		NotAfter:     options.Now.AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clientDER, err := x509.CreateCertificate(rand.Reader, &clientTemplate, &caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		return DevCertPaths{}, err
	}

	paths := DevCertPaths{
		CACert:     filepath.Join(options.OutputDir, "ca.crt"),
		HostCert:   filepath.Join(options.OutputDir, "host.crt"),
		HostKey:    filepath.Join(options.OutputDir, "host.key"),
		ClientCert: filepath.Join(options.OutputDir, "client.crt"),
		ClientKey:  filepath.Join(options.OutputDir, "client.key"),
	}
	if err := writeCert(paths.CACert, caDER); err != nil {
		return DevCertPaths{}, err
	}
	if err := writeCert(paths.HostCert, hostDER); err != nil {
		return DevCertPaths{}, err
	}
	if err := writeKey(paths.HostKey, hostKey); err != nil {
		return DevCertPaths{}, err
	}
	if err := writeCert(paths.ClientCert, clientDER); err != nil {
		return DevCertPaths{}, err
	}
	if err := writeKey(paths.ClientKey, clientKey); err != nil {
		return DevCertPaths{}, err
	}
	return paths, nil
}

func writeCert(path string, der []byte) error {
	return writePEM(path, &pem.Block{Type: "CERTIFICATE", Bytes: der}, 0o644)
}

func writeKey(path string, key *rsa.PrivateKey) error {
	return writePEM(path, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}, 0o600)
}

func writePEM(path string, block *pem.Block, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer file.Close()
	return pem.Encode(file, block)
}
