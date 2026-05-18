package certs

import (
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateDevCerts(t *testing.T) {
	dir := t.TempDir()
	paths, err := GenerateDevCerts(DevCertOptions{
		OutputDir: dir,
		HostDNS:   []string{"localhost"},
		HostIPs:   []net.IP{net.ParseIP("127.0.0.1")},
		Now:       time.Now(),
	})
	if err != nil {
		t.Fatalf("generate dev certs: %v", err)
	}

	for _, path := range []string{paths.CACert, paths.HostCert, paths.HostKey, paths.ClientCert, paths.ClientKey} {
		if filepath.Dir(path) != dir {
			t.Fatalf("path %s not under %s", path, dir)
		}
		record := Inspect(Source{Role: "test", Path: path}, time.Now())
		if filepath.Ext(path) == ".crt" && record.Status != "VALID" {
			t.Fatalf("cert %s status = %s", path, record.Status)
		}
	}
}
