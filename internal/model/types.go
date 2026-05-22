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

type EndpointCollisionType string

const (
	EndpointCollisionSharedEndpoint  EndpointCollisionType = "SHARED_ENDPOINT"
	EndpointCollisionIDEndpointDrift EndpointCollisionType = "ID_ENDPOINT_DRIFT"
)

type EndpointCollision struct {
	CollisionType  EndpointCollisionType `json:"collision_type"`
	InvolvedEGMIDs []string              `json:"involved_egm_ids"`
	Endpoint       string                `json:"endpoint"`
	FirstSeenAt    time.Time             `json:"first_seen_at,omitempty"`
	LastSeenAt     time.Time             `json:"last_seen_at,omitempty"`
}

type EndpointCollisionSummary struct {
	Total                int      `json:"total"`
	SharedEndpointCount  int      `json:"shared_endpoint_count"`
	IDEndpointDriftCount int      `json:"id_endpoint_drift_count"`
	AffectedEGMIDs       []string `json:"affected_egm_ids,omitempty"`
}

type EGM struct {
	ID                       string                   `json:"id"`
	IPAddress                string                   `json:"ip_address"`
	Port                     int                      `json:"port"`
	Vendor                   string                   `json:"vendor,omitempty"`
	CabinetFamily            string                   `json:"cabinet_family,omitempty"`
	GameTitle                string                   `json:"game_title,omitempty"`
	SoftwareVersion          string                   `json:"software_version,omitempty"`
	Source                   EGMSource                `json:"source"`
	Status                   EGMHealth                `json:"status"`
	LastError                string                   `json:"last_error,omitempty"`
	LastSeen                 time.Time                `json:"last_seen,omitempty"`
	LastEndpointIP           string                   `json:"last_endpoint_ip,omitempty"`
	LastEndpointPort         int                      `json:"last_endpoint_port,omitempty"`
	LastEndpointSeenAt       time.Time                `json:"last_endpoint_seen_at,omitempty"`
	EndpointDrift            bool                     `json:"endpoint_drift_warning"`
	EndpointDriftIPs         []string                 `json:"endpoint_drift_ips,omitempty"`
	EndpointCollisionWarning bool                     `json:"endpoint_collision_warning"`
	EndpointCollisionTypes   []EndpointCollisionType  `json:"endpoint_collision_types,omitempty"`
	RecentEndpoints          []EGMEndpointObservation `json:"recent_endpoints,omitempty"`
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
	EGMID                      string     `json:"egm_id"`
	Status                     EGMHealth  `json:"status"`
	EventType                  string     `json:"event_type"`
	Detail                     string     `json:"detail,omitempty"`
	LastError                  string     `json:"last_error,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
	HeartbeatRollup            bool       `json:"heartbeat_rollup,omitempty"`
	HeartbeatRollupCount       int        `json:"heartbeat_rollup_count,omitempty"`
	HeartbeatRollupFirstSeenAt *time.Time `json:"heartbeat_rollup_first_seen_at,omitempty"`
	HeartbeatRollupLastSeenAt  *time.Time `json:"heartbeat_rollup_last_seen_at,omitempty"`
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

type OperatorAuditEvent struct {
	ID         int64     `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Action     string    `json:"action"`
	Result     string    `json:"result"`
	ActorScope string    `json:"actor_scope"`
	EGMFocus   string    `json:"egm_focus,omitempty"`
	Summary    string    `json:"summary"`
	Detail     string    `json:"detail,omitempty"`
}

type OperatorAuditQuery struct {
	Limit  int
	Action string
	Result string
	Search string
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
