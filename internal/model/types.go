package model

import "time"

type ControllerState string

const (
	StateBooting          ControllerState = "BOOTING"
	StateHealthy          ControllerState = "HEALTHY"
	StateWarning          ControllerState = "WARNING"
	StateEmergencyPending ControllerState = "EMERGENCY_PENDING"
	StateEmergencyActive  ControllerState = "EMERGENCY_ACTIVE"
	StateRecoveryPending  ControllerState = "RECOVERY_PENDING"
	StateDegraded         ControllerState = "DEGRADED"
)

type EGMHealth string

const (
	EGMGreen  EGMHealth = "GREEN"
	EGMYellow EGMHealth = "YELLOW"
	EGMRed    EGMHealth = "RED"
	EGMGrey   EGMHealth = "GREY"
)

type EGM struct {
	ID              string    `json:"id"`
	IPAddress       string    `json:"ip_address"`
	Port            int       `json:"port"`
	Vendor          string    `json:"vendor,omitempty"`
	CabinetFamily   string    `json:"cabinet_family,omitempty"`
	GameTitle       string    `json:"game_title,omitempty"`
	SoftwareVersion string    `json:"software_version,omitempty"`
	Status          EGMHealth `json:"status"`
	LastError       string    `json:"last_error,omitempty"`
	LastSeen        time.Time `json:"last_seen,omitempty"`
}

type Incident struct {
	ID            int64           `json:"id"`
	TriggerType   string          `json:"trigger_type"`
	TriggerSource string          `json:"trigger_source,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	ResolvedAt    *time.Time      `json:"resolved_at,omitempty"`
	FinalState    ControllerState `json:"final_state"`
}
