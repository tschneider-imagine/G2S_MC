package config

import (
	"path/filepath"
	"testing"
)

func TestValidateRejectsMissingRoster(t *testing.T) {
	cfg := Config{
		ControllerID: "controller",
		Database:     Database{Path: "controller.db"},
		WebUI:        WebUI{BindAddress: "127.0.0.1:8444"},
		G2S:          G2S{HostID: "HOST-1", HostURL: "http://127.0.0.1:8444/g2s", EndpointPath: "/g2s"},
		HardwareIO:   HardwareIO{VoltageDropThresholdMS: 250},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing roster to fail validation")
	}
}

func TestValidateAcceptsMinimalConfig(t *testing.T) {
	cfg := Config{
		ControllerID: "controller",
		Database:     Database{Path: "controller.db"},
		WebUI:        WebUI{BindAddress: "127.0.0.1:8444"},
		G2S:          G2S{HostID: "HOST-1", HostURL: "http://127.0.0.1:8444/g2s", EndpointPath: "/g2s"},
		HardwareIO:   HardwareIO{VoltageDropThresholdMS: 250},
		EGMRoster:    []EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected config to validate: %v", err)
	}
}

func TestValidateTLSRequiresServerCertificatePaths(t *testing.T) {
	cfg := Config{
		ControllerID: "controller",
		Database:     Database{Path: "controller.db"},
		WebUI:        WebUI{BindAddress: "127.0.0.1:8444"},
		G2S:          G2S{HostID: "HOST-1", HostURL: "https://localhost:8444/g2s", EndpointPath: "/g2s", RequireTLS: true},
		HardwareIO:   HardwareIO{VoltageDropThresholdMS: 250},
		EGMRoster:    []EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected TLS config without server certificate paths to fail validation")
	}
}

func TestValidateMutualTLSRequiresTLSAndCA(t *testing.T) {
	cfg := Config{
		ControllerID: "controller",
		Database:     Database{Path: "controller.db"},
		WebUI:        WebUI{BindAddress: "127.0.0.1:8444"},
		G2S:          G2S{HostID: "HOST-1", HostURL: "https://localhost:8444/g2s", EndpointPath: "/g2s", RequireClientCert: true},
		HardwareIO:   HardwareIO{VoltageDropThresholdMS: 250},
		EGMRoster:    []EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected mutual TLS config without TLS and CA to fail validation")
	}
}

func TestExampleConfigsLoad(t *testing.T) {
	for _, name := range []string{"config.example.json", "config.tls.example.json", "config.pi.example.json"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "configs", name)
			if _, err := LoadFile(path); err != nil {
				t.Fatalf("load %s: %v", name, err)
			}
		})
	}
}
