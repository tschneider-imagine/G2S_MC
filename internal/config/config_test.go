package config

import (
	"path/filepath"
	"testing"
)

func TestValidateAllowsEmptyRoster(t *testing.T) {
	cfg := Config{
		ControllerID: "controller",
		Database:     Database{Path: "controller.db"},
		WebUI:        WebUI{BindAddress: "127.0.0.1:8444"},
		G2S:          G2S{HostID: "HOST-1", HostURL: "http://127.0.0.1:8444/g2s", EndpointPath: "/g2s"},
		CabinetProfile: CabinetProfile{
			WireHostURL:     "https://host.example/g2s",
			ListenerDNSName: "host.example",
			RequiredSANDNS:  []string{"host.example"},
			HostID:          "HOST-1",
			FirstTestEGMIDs: []string{"EGM-1"},
		},
		HardwareIO: HardwareIO{VoltageDropThresholdMS: 250},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected empty roster to validate: %v", err)
	}
}

func TestValidateAcceptsMinimalConfig(t *testing.T) {
	cfg := Config{
		ControllerID: "controller",
		Database:     Database{Path: "controller.db"},
		WebUI:        WebUI{BindAddress: "127.0.0.1:8444"},
		G2S:          G2S{HostID: "HOST-1", HostURL: "http://127.0.0.1:8444/g2s", EndpointPath: "/g2s"},
		CabinetProfile: CabinetProfile{
			WireHostURL:     "https://host.example/g2s",
			ListenerDNSName: "host.example",
			RequiredSANDNS:  []string{"host.example"},
			HostID:          "HOST-1",
			FirstTestEGMIDs: []string{"EGM-1"},
		},
		HardwareIO: HardwareIO{VoltageDropThresholdMS: 250},
		EGMRoster:  []EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}},
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
		CabinetProfile: CabinetProfile{
			WireHostURL:     "https://host.example/g2s",
			ListenerDNSName: "host.example",
			RequiredSANDNS:  []string{"host.example"},
			HostID:          "HOST-1",
			FirstTestEGMIDs: []string{"EGM-1"},
		},
		HardwareIO: HardwareIO{VoltageDropThresholdMS: 250},
		EGMRoster:  []EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}},
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
		CabinetProfile: CabinetProfile{
			WireHostURL:     "https://host.example/g2s",
			ListenerDNSName: "host.example",
			RequiredSANDNS:  []string{"host.example"},
			HostID:          "HOST-1",
			FirstTestEGMIDs: []string{"EGM-1"},
		},
		HardwareIO: HardwareIO{VoltageDropThresholdMS: 250},
		EGMRoster:  []EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected mutual TLS config without TLS and CA to fail validation")
	}
}

func TestExampleConfigsLoad(t *testing.T) {
	for _, name := range []string{
		"config.example.json",
		"config.tls.example.json",
		"config.pi.example.json",
		"config.pi-field-test.example.json",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("..", "..", "configs", name)
			if _, err := LoadFile(path); err != nil {
				t.Fatalf("load %s: %v", name, err)
			}
		})
	}
}

func TestValidateRejectsTrustedPrivateNetworkMutationsWhenLoginRequired(t *testing.T) {
	cfg := Config{
		ControllerID: "controller",
		Database:     Database{Path: "controller.db"},
		WebUI: WebUI{
			BindAddress:                         "127.0.0.1:8444",
			RequireLogin:                        true,
			AllowTrustedPrivateNetworkMutations: true,
		},
		G2S: G2S{HostID: "HOST-1", HostURL: "http://127.0.0.1:8444/g2s", EndpointPath: "/g2s"},
		CabinetProfile: CabinetProfile{
			WireHostURL:     "https://host.example/g2s",
			ListenerDNSName: "host.example",
			RequiredSANDNS:  []string{"host.example"},
			HostID:          "HOST-1",
			FirstTestEGMIDs: []string{"EGM-1"},
		},
		HardwareIO: HardwareIO{VoltageDropThresholdMS: 250},
		EGMRoster:  []EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected trusted private network mutations with login required to fail validation")
	}
}

func TestValidateCabinetProfileRules(t *testing.T) {
	valid := CabinetProfile{
		WireHostURL:     "https://cabinet-host.example/g2s",
		ListenerDNSName: "cabinet-host.example",
		RequiredSANDNS:  []string{"cabinet-host.example"},
		HostID:          "HOST-1",
		FirstTestEGMIDs: []string{"EGM-1"},
	}
	if err := ValidateCabinetProfile(valid); err != nil {
		t.Fatalf("expected valid profile: %v", err)
	}

	tests := []struct {
		name    string
		profile CabinetProfile
	}{
		{
			name: "missing wire host url",
			profile: CabinetProfile{
				ListenerDNSName: "cabinet-host.example",
				RequiredSANDNS:  []string{"cabinet-host.example"},
				HostID:          "HOST-1",
				FirstTestEGMIDs: []string{"EGM-1"},
			},
		},
		{
			name: "invalid url scheme",
			profile: CabinetProfile{
				WireHostURL:     "ftp://cabinet-host.example/g2s",
				ListenerDNSName: "cabinet-host.example",
				RequiredSANDNS:  []string{"cabinet-host.example"},
				HostID:          "HOST-1",
				FirstTestEGMIDs: []string{"EGM-1"},
			},
		},
		{
			name: "missing listener identity",
			profile: CabinetProfile{
				WireHostURL:     "https://cabinet-host.example/g2s",
				RequiredSANDNS:  []string{"cabinet-host.example"},
				HostID:          "HOST-1",
				FirstTestEGMIDs: []string{"EGM-1"},
			},
		},
		{
			name: "missing sans",
			profile: CabinetProfile{
				WireHostURL:     "https://cabinet-host.example/g2s",
				ListenerDNSName: "cabinet-host.example",
				HostID:          "HOST-1",
				FirstTestEGMIDs: []string{"EGM-1"},
			},
		},
		{
			name: "missing host id",
			profile: CabinetProfile{
				WireHostURL:     "https://cabinet-host.example/g2s",
				ListenerDNSName: "cabinet-host.example",
				RequiredSANDNS:  []string{"cabinet-host.example"},
				FirstTestEGMIDs: []string{"EGM-1"},
			},
		},
		{
			name: "missing first test egm ids",
			profile: CabinetProfile{
				WireHostURL:     "https://cabinet-host.example/g2s",
				ListenerDNSName: "cabinet-host.example",
				RequiredSANDNS:  []string{"cabinet-host.example"},
				HostID:          "HOST-1",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateCabinetProfile(tc.profile); err == nil {
				t.Fatalf("expected profile validation failure")
			}
		})
	}
}
