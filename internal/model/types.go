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

type EGMSource string

const (
	EGMSourceConfigured EGMSource = "CONFIGURED"
	EGMSourceDiscovered EGMSource = "DISCOVERED"
)

type EGMEndpointObservation struct {
	IPAddress   string    `json:"ip"`
	Port        int       `json:"port"`
	FirstSeenAt time.Time `json:"first_seen_at,omitempty"`
	LastSeenAt  time.Time `json:"last_seen_at,omitempty"`
	SeenCount   int       `json:"seen_count"`
}

type EGM struct {
	ID                 string                   `json:"id"`
	IPAddress          string                   `json:"ip_address"`
	Port               int                      `json:"port"`
	Vendor             string                   `json:"vendor,omitempty"`
	CabinetFamily      string                   `json:"cabinet_family,omitempty"`
	GameTitle          string                   `json:"game_title,omitempty"`
	SoftwareVersion    string                   `json:"software_version,omitempty"`
	Source             EGMSource                `json:"source"`
	Status             EGMHealth                `json:"status"`
	LastError          string                   `json:"last_error,omitempty"`
	LastSeen           time.Time                `json:"last_seen,omitempty"`
	LastEndpointIP     string                   `json:"last_endpoint_ip,omitempty"`
	LastEndpointPort   int                      `json:"last_endpoint_port,omitempty"`
	LastEndpointSeenAt time.Time                `json:"last_endpoint_seen_at,omitempty"`
	EndpointDrift      bool                     `json:"endpoint_drift_warning"`
	EndpointDriftIPs   []string                 `json:"endpoint_drift_ips,omitempty"`
	RecentEndpoints    []EGMEndpointObservation `json:"recent_endpoints,omitempty"`
}

type Incident struct {
	ID            int64           `json:"id"`
	TriggerType   string          `json:"trigger_type"`
	TriggerSource string          `json:"trigger_source,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	ResolvedAt    *time.Time      `json:"resolved_at,omitempty"`
	FinalState    ControllerState `json:"final_state"`
}

type EGMStatusSnapshot struct {
	EGMID     string    `json:"egm_id"`
	Status    EGMHealth `json:"status"`
	EventType string    `json:"event_type"`
	Detail    string    `json:"detail,omitempty"`
	LastError string    `json:"last_error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type EGMComplianceLog struct {
	IncidentID      int64     `json:"incident_id"`
	EGMID           string    `json:"egm_id"`
	IPAddress       string    `json:"ip_address"`
	ActionSent      string    `json:"action_sent"`
	StatusResult    string    `json:"status_result"`
	HTTPStatusCode  int       `json:"http_status_code,omitempty"`
	LatencyMS       int       `json:"latency_ms,omitempty"`
	ResponseExcerpt string    `json:"response_excerpt,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type StateChange struct {
	OldState  ControllerState `json:"old_state"`
	NewState  ControllerState `json:"new_state"`
	Reason    string          `json:"reason"`
	CreatedAt time.Time       `json:"created_at"`
}

type CertificateInventory struct {
	Role              string     `json:"role"`
	Path              string     `json:"path"`
	Subject           string     `json:"subject,omitempty"`
	Issuer            string     `json:"issuer,omitempty"`
	NotBefore         *time.Time `json:"not_before,omitempty"`
	NotAfter          *time.Time `json:"not_after,omitempty"`
	SHA256Fingerprint string     `json:"sha256_fingerprint,omitempty"`
	Status            string     `json:"status"`
	LastCheckedAt     time.Time  `json:"last_checked_at"`
	Error             string     `json:"error,omitempty"`
}

type HistoryLimits struct {
	Limit int
	EGMID string
}

type SessionEvidenceRecord struct {
	ID             int64     `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	OverallState   string    `json:"overall_state"`
	ReadyzState    string    `json:"readyz_state"`
	PreflightState string    `json:"preflight_state"`
	HostID         string    `json:"host_id"`
	WireHostURL    string    `json:"wire_host_url"`
	OperatorNotes  string    `json:"operator_notes,omitempty"`
	PayloadJSON    string    `json:"payload_json"`
}

type RunMarker struct {
	ID          int64     `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	MarkerType  string    `json:"marker_type"`
	Title       string    `json:"title"`
	Notes       string    `json:"notes,omitempty"`
	HostID      string    `json:"host_id,omitempty"`
	WireHostURL string    `json:"wire_host_url,omitempty"`
	Operator    string    `json:"operator,omitempty"`
}

type SessionWorkflowProgress struct {
	CurrentPhase   string    `json:"current_phase"`
	CompletedSteps []string  `json:"completed_steps"`
	OperatorNotes  string    `json:"operator_notes,omitempty"`
	LastUpdatedAt  time.Time `json:"last_updated_at"`
}
