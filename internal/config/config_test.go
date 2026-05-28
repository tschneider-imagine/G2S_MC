package config

import (
	"os"
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

func TestLoadFileRuntimeSectionParses(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	content := `{
  "controller_id": "controller",
  "site_name": "lab",
  "cabinet_profile": {
    "wire_host_url": "https://host.example/g2s",
    "listener_dns_name": "host.example",
    "required_san_dns": ["host.example"],
    "host_id": "HOST-1",
    "first_test_egm_ids": ["EGM-1"]
  },
  "hardware_io": {"voltage_drop_threshold_ms": 250},
  "database": {"path": "controller.db"},
  "web_ui": {"bind_address": "127.0.0.1:8444"},
  "g2s": {"host_id": "HOST-1", "host_url": "http://127.0.0.1:8444/g2s", "endpoint_path": "/g2s"},
  "runtime": {
    "input_runtime_enabled": true,
    "input_runtime_seed_defaults": true,
    "input_runtime_execute_actions": true,
    "input_runtime_interval_ms": 175,
    "pending_delivery_sweep_enabled": true,
    "pending_delivery_sweep_interval_ms": 2500,
    "delivery_topology": "HOST_LISTENER"
  }
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Runtime.InputRuntimeEnabled {
		t.Fatalf("input_runtime_enabled=false, want true")
	}
	if !cfg.Runtime.InputRuntimeExecuteActions {
		t.Fatalf("input_runtime_execute_actions=false, want true")
	}
	if cfg.Runtime.InputRuntimeIntervalMS != 175 {
		t.Fatalf("input_runtime_interval_ms=%d, want 175", cfg.Runtime.InputRuntimeIntervalMS)
	}
	if cfg.Runtime.DeliveryTopology != "HOST_LISTENER" {
		t.Fatalf("delivery_topology=%q, want HOST_LISTENER", cfg.Runtime.DeliveryTopology)
	}
	if !cfg.Runtime.PendingDeliverySweepEnabled {
		t.Fatalf("pending_delivery_sweep_enabled=false, want true")
	}
	if cfg.Runtime.PendingDeliverySweepIntervalMS != 2500 {
		t.Fatalf("pending_delivery_sweep_interval_ms=%d, want 2500", cfg.Runtime.PendingDeliverySweepIntervalMS)
	}
}

func TestLoadFileRuntimeDefaultsWhenSectionMissing(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	content := `{
  "controller_id": "controller",
  "site_name": "lab",
  "cabinet_profile": {
    "wire_host_url": "https://host.example/g2s",
    "listener_dns_name": "host.example",
    "required_san_dns": ["host.example"],
    "host_id": "HOST-1",
    "first_test_egm_ids": ["EGM-1"]
  },
  "hardware_io": {"voltage_drop_threshold_ms": 250},
  "database": {"path": "controller.db"},
  "web_ui": {"bind_address": "127.0.0.1:8444"},
  "g2s": {"host_id": "HOST-1", "host_url": "http://127.0.0.1:8444/g2s", "endpoint_path": "/g2s"}
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Runtime.InputRuntimeEnabled {
		t.Fatalf("input_runtime_enabled=true, want false")
	}
	if cfg.Runtime.InputRuntimeExecuteActions {
		t.Fatalf("input_runtime_execute_actions=true, want false")
	}
	if cfg.Runtime.InputRuntimeIntervalMS != DefaultInputRuntimeIntervalMS {
		t.Fatalf("input_runtime_interval_ms=%d, want %d", cfg.Runtime.InputRuntimeIntervalMS, DefaultInputRuntimeIntervalMS)
	}
	if cfg.Runtime.DeliveryTopology != DefaultDeliveryTopology {
		t.Fatalf("delivery_topology=%q, want %q", cfg.Runtime.DeliveryTopology, DefaultDeliveryTopology)
	}
	if cfg.Runtime.PendingDeliverySweepEnabled {
		t.Fatalf("pending_delivery_sweep_enabled=true, want false")
	}
	if cfg.Runtime.PendingDeliverySweepIntervalMS != DefaultPendingDeliverySweepIntervalMS {
		t.Fatalf("pending_delivery_sweep_interval_ms=%d, want %d", cfg.Runtime.PendingDeliverySweepIntervalMS, DefaultPendingDeliverySweepIntervalMS)
	}
}

func TestPiFieldTestConfigEnablesRuntimeExecution(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "config.pi-field-test.example.json")
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load %s: %v", path, err)
	}
	if !cfg.Runtime.InputRuntimeEnabled {
		t.Fatalf("input_runtime_enabled=false, want true")
	}
	if !cfg.Runtime.InputRuntimeSeedDefaults {
		t.Fatalf("input_runtime_seed_defaults=false, want true")
	}
	if !cfg.Runtime.InputRuntimeExecuteActions {
		t.Fatalf("input_runtime_execute_actions=false, want true")
	}
	if cfg.Runtime.InputRuntimeIntervalMS != 100 {
		t.Fatalf("input_runtime_interval_ms=%d, want 100", cfg.Runtime.InputRuntimeIntervalMS)
	}
	if cfg.Runtime.DeliveryTopology != "OUTBOUND_ENDPOINT" {
		t.Fatalf("delivery_topology=%q, want OUTBOUND_ENDPOINT", cfg.Runtime.DeliveryTopology)
	}
	if cfg.Runtime.PendingDeliverySweepEnabled {
		t.Fatalf("pending_delivery_sweep_enabled=true, want false")
	}
	if cfg.Runtime.PendingDeliverySweepIntervalMS != 5000 {
		t.Fatalf("pending_delivery_sweep_interval_ms=%d, want 5000", cfg.Runtime.PendingDeliverySweepIntervalMS)
	}
}
