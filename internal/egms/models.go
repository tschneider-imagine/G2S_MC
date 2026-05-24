package egms

import (
	"fmt"
	"strings"
	"time"
)

type EGMActionState string

const (
	EGMActionStateNormal     EGMActionState = "NORMAL"
	EGMActionStatePending    EGMActionState = "PENDING"
	EGMActionStateSilenced   EGMActionState = "SILENCED"
	EGMActionStateFailed     EGMActionState = "FAILED"
	EGMActionStateEscalating EGMActionState = "ESCALATING"
	EGMActionStateRestoring  EGMActionState = "RESTORING"
)

type EGMRecord struct {
	EGMID                 string         `json:"egm_id"`
	DisplayName           string         `json:"display_name,omitempty"`
	IPAddress             string         `json:"ip_address,omitempty"`
	EndpointPath          string         `json:"endpoint_path,omitempty"`
	Vendor                string         `json:"vendor,omitempty"`
	CabinetFamily         string         `json:"cabinet_family,omitempty"`
	GameTitle             string         `json:"game_title,omitempty"`
	SoftwareVersion       string         `json:"software_version,omitempty"`
	Zone                  string         `json:"zone,omitempty"`
	Enabled               bool           `json:"enabled"`
	EmergencyEnabled      bool           `json:"emergency_enabled"`
	TemplateID            string         `json:"template_id,omitempty"`
	HeartbeatOverrideJSON string         `json:"heartbeat_override_json,omitempty"`
	LastSeenAt            *time.Time     `json:"last_seen_at,omitempty"`
	CurrentActionState    EGMActionState `json:"current_action_state"`
	Notes                 string         `json:"notes,omitempty"`
}

func (r EGMRecord) Validate() error {
	if strings.TrimSpace(r.EGMID) == "" {
		return fmt.Errorf("egm_id is required")
	}
	switch r.CurrentActionState {
	case EGMActionStateNormal, EGMActionStatePending, EGMActionStateSilenced, EGMActionStateFailed, EGMActionStateEscalating, EGMActionStateRestoring:
	default:
		return fmt.Errorf("current_action_state is invalid")
	}
	return nil
}

type EGMGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

func (g EGMGroup) Validate() error {
	if strings.TrimSpace(g.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(g.Name) == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
