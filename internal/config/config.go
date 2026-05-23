package config

type Config struct {
	ControllerID    string          `json:"controller_id"`
	SiteName        string          `json:"site_name"`
	CabinetProfile  CabinetProfile  `json:"cabinet_profile"`
	HardwareIO      HardwareIO      `json:"hardware_io"`
	PowerManagement PowerManagement `json:"power_management"`
	Crypto          Crypto          `json:"crypto"`
	API             API             `json:"api"`
	Alerts          Alerts          `json:"alerts"`
	Timeouts        Timeouts        `json:"timeouts"`
	Database        Database        `json:"database"`
	WebUI           WebUI           `json:"web_ui"`
	G2S             G2S             `json:"g2s"`
	EGMRoster       []EGM           `json:"egm_roster"`
}

type HardwareIO struct {
	SecurityLineGPIOPin    int  `json:"security_line_gpio_pin"`
	VoltageDropThresholdMS int  `json:"voltage_drop_threshold_ms"`
	PSU1GPIOPin            int  `json:"psu_1_gpio_pin"`
	PSU2GPIOPin            int  `json:"psu_2_gpio_pin"`
	BuzzerGPIOPin          int  `json:"buzzer_gpio_pin"`
	ActiveLowInputs        bool `json:"active_low_inputs"`
}

type PowerManagement struct {
	CriticalBatteryThresholdPercent int  `json:"critical_battery_threshold_percent"`
	BuzzerOnSinglePSUFailure        bool `json:"buzzer_on_single_psu_failure"`
}

type Crypto struct {
	G2SClientCertPath string `json:"g2s_client_cert_path"`
	G2SClientKeyPath  string `json:"g2s_client_key_path"`
	G2SCAPath         string `json:"g2s_ca_cert_path"`
	WebServerCertPath string `json:"web_server_cert_path"`
	WebServerKeyPath  string `json:"web_server_key_path"`
}

type API struct {
	AuthToken string `json:"auth_token"`
}

type Alerts struct {
	GreyThresholdCountToBuzz   int     `json:"grey_threshold_count_to_buzz"`
	GreyThresholdPercentToBuzz float64 `json:"grey_threshold_percent_to_buzz"`
}

type Timeouts struct {
	G2SRequestTimeoutMS            int `json:"g2s_request_timeout_ms"`
	EGMHeartbeatIntervalMS         int `json:"egm_heartbeat_interval_ms"`
	EGMHeartbeatWarningAfterMissed int `json:"egm_heartbeat_warning_after_missed"`
	EGMHeartbeatBlockAfterMissed   int `json:"egm_heartbeat_block_after_missed"`
	UISessionTimeoutMinutes        int `json:"ui_session_timeout_minutes"`
}

type Database struct {
	Path string `json:"path"`
}

type WebUI struct {
	BindAddress                         string `json:"bind_address"`
	RequireLogin                        bool   `json:"require_login"`
	RequireClientCertForAdmin           bool   `json:"require_client_cert_for_admin"`
	AllowPrivateKeyExport               bool   `json:"allow_private_key_export"`
	AllowTrustedPrivateNetworkMutations bool   `json:"allow_trusted_private_network_mutations"`
}

type G2S struct {
	HostID            string `json:"host_id"`
	HostURL           string `json:"host_url"`
	EndpointPath      string `json:"endpoint_path"`
	RequireTLS        bool   `json:"require_tls"`
	RequireClientCert bool   `json:"require_client_cert"`
}

type EGM struct {
	EGMID           string `json:"egm_id"`
	IPAddress       string `json:"ip_address"`
	Port            int    `json:"port"`
	Vendor          string `json:"vendor"`
	CabinetFamily   string `json:"cabinet_family"`
	GameTitle       string `json:"game_title"`
	SoftwareVersion string `json:"software_version"`
}

type CabinetProfile struct {
	WireHostURL     string   `json:"wire_host_url"`
	ListenerDNSName string   `json:"listener_dns_name"`
	ListenerIP      string   `json:"listener_ip"`
	RequiredSANDNS  []string `json:"required_san_dns"`
	RequiredSANIPs  []string `json:"required_san_ips"`
	HostID          string   `json:"host_id"`
	FirstTestEGMIDs []string `json:"first_test_egm_ids"`
}
