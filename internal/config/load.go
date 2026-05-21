package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

const (
	DefaultHeartbeatWarningAfterMissed = 3
	DefaultHeartbeatBlockAfterMissed   = 6
)

func LoadFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func ChecksumFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (c Config) Validate() error {
	var problems []string

	requireText(&problems, "controller_id", c.ControllerID)
	requireText(&problems, "database.path", c.Database.Path)
	requireText(&problems, "web_ui.bind_address", c.WebUI.BindAddress)
	requireText(&problems, "g2s.host_id", c.G2S.HostID)
	requireText(&problems, "g2s.host_url", c.G2S.HostURL)
	requireText(&problems, "g2s.endpoint_path", c.G2S.EndpointPath)
	problems = append(problems, validateCabinetProfile(c.CabinetProfile)...)
	if c.G2S.RequireTLS {
		requireText(&problems, "crypto.web_server_cert_path", c.Crypto.WebServerCertPath)
		requireText(&problems, "crypto.web_server_key_path", c.Crypto.WebServerKeyPath)
	}
	if c.G2S.RequireClientCert {
		if !c.G2S.RequireTLS {
			problems = append(problems, "g2s.require_client_cert requires g2s.require_tls")
		}
		requireText(&problems, "crypto.g2s_ca_cert_path", c.Crypto.G2SCAPath)
	}

	if c.HardwareIO.VoltageDropThresholdMS <= 0 {
		problems = append(problems, "hardware_io.voltage_drop_threshold_ms must be greater than zero")
	}
	if c.Timeouts.EGMHeartbeatWarningAfterMissed < 0 {
		problems = append(problems, "timeouts.egm_heartbeat_warning_after_missed must not be negative")
	}
	if c.Timeouts.EGMHeartbeatBlockAfterMissed < 0 {
		problems = append(problems, "timeouts.egm_heartbeat_block_after_missed must not be negative")
	}
	if c.Timeouts.EGMHeartbeatWarningAfterMissed > 0 && c.Timeouts.EGMHeartbeatBlockAfterMissed > 0 &&
		c.Timeouts.EGMHeartbeatBlockAfterMissed < c.Timeouts.EGMHeartbeatWarningAfterMissed {
		problems = append(problems, "timeouts.egm_heartbeat_block_after_missed must be greater than or equal to timeouts.egm_heartbeat_warning_after_missed")
	}
	if c.Alerts.GreyThresholdCountToBuzz < 0 {
		problems = append(problems, "alerts.grey_threshold_count_to_buzz must not be negative")
	}
	if c.Alerts.GreyThresholdPercentToBuzz < 0 || c.Alerts.GreyThresholdPercentToBuzz > 100 {
		problems = append(problems, "alerts.grey_threshold_percent_to_buzz must be between 0 and 100")
	}
	if c.WebUI.AllowTrustedPrivateNetworkMutations && c.WebUI.RequireLogin {
		problems = append(problems, "web_ui.allow_trusted_private_network_mutations requires web_ui.require_login=false")
	}
	for i, egm := range c.EGMRoster {
		prefix := fmt.Sprintf("egm_roster[%d]", i)
		requireText(&problems, prefix+".egm_id", egm.EGMID)
		requireText(&problems, prefix+".ip_address", egm.IPAddress)
		if egm.Port <= 0 || egm.Port > 65535 {
			problems = append(problems, prefix+".port must be between 1 and 65535")
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid config: %s", strings.Join(problems, "; "))
	}
	return nil
}

func EffectiveHeartbeatWarningAfterMissed(timeouts Timeouts) int {
	if timeouts.EGMHeartbeatWarningAfterMissed > 0 {
		return timeouts.EGMHeartbeatWarningAfterMissed
	}
	return DefaultHeartbeatWarningAfterMissed
}

func EffectiveHeartbeatBlockAfterMissed(timeouts Timeouts) int {
	if timeouts.EGMHeartbeatBlockAfterMissed > 0 {
		if timeouts.EGMHeartbeatWarningAfterMissed > 0 && timeouts.EGMHeartbeatBlockAfterMissed < timeouts.EGMHeartbeatWarningAfterMissed {
			return timeouts.EGMHeartbeatWarningAfterMissed
		}
		return timeouts.EGMHeartbeatBlockAfterMissed
	}
	warningAfter := EffectiveHeartbeatWarningAfterMissed(timeouts)
	if DefaultHeartbeatBlockAfterMissed < warningAfter {
		return warningAfter
	}
	return DefaultHeartbeatBlockAfterMissed
}

func requireText(problems *[]string, field string, value string) {
	if strings.TrimSpace(value) == "" {
		*problems = append(*problems, field+" is required")
	}
}

func ValidateCabinetProfile(profile CabinetProfile) error {
	problems := validateCabinetProfile(profile)
	if len(problems) > 0 {
		return fmt.Errorf("invalid cabinet_profile: %s", strings.Join(problems, "; "))
	}
	return nil
}

func validateCabinetProfile(profile CabinetProfile) []string {
	problems := []string{}

	requireText(&problems, "cabinet_profile.wire_host_url", profile.WireHostURL)
	if strings.TrimSpace(profile.WireHostURL) != "" {
		u, err := url.ParseRequestURI(profile.WireHostURL)
		if err != nil {
			problems = append(problems, "cabinet_profile.wire_host_url must be a valid URL")
		} else if u.Scheme != "http" && u.Scheme != "https" {
			problems = append(problems, "cabinet_profile.wire_host_url must use http or https")
		}
	}

	if strings.TrimSpace(profile.ListenerDNSName) == "" && strings.TrimSpace(profile.ListenerIP) == "" {
		problems = append(problems, "cabinet_profile.listener_dns_name or cabinet_profile.listener_ip is required")
	}
	if strings.TrimSpace(profile.ListenerIP) != "" && net.ParseIP(strings.TrimSpace(profile.ListenerIP)) == nil {
		problems = append(problems, "cabinet_profile.listener_ip must be a valid IP address")
	}

	if len(profile.RequiredSANDNS) == 0 && len(profile.RequiredSANIPs) == 0 {
		problems = append(problems, "cabinet_profile.required_san_dns or cabinet_profile.required_san_ips must contain at least one entry")
	}
	for i, value := range profile.RequiredSANDNS {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Sprintf("cabinet_profile.required_san_dns[%d] is required", i))
		}
	}
	for i, value := range profile.RequiredSANIPs {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Sprintf("cabinet_profile.required_san_ips[%d] is required", i))
			continue
		}
		if net.ParseIP(strings.TrimSpace(value)) == nil {
			problems = append(problems, fmt.Sprintf("cabinet_profile.required_san_ips[%d] must be a valid IP address", i))
		}
	}

	requireText(&problems, "cabinet_profile.host_id", profile.HostID)
	if len(profile.FirstTestEGMIDs) == 0 {
		problems = append(problems, "cabinet_profile.first_test_egm_ids must contain at least one EGM ID")
	}
	for i, value := range profile.FirstTestEGMIDs {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Sprintf("cabinet_profile.first_test_egm_ids[%d] is required", i))
		}
	}

	return problems
}
