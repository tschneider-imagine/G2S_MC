package templates

import (
	"fmt"
	"strings"
	"time"
)

type TemplateStatus string

const (
	TemplateStatusDraft    TemplateStatus = "DRAFT"
	TemplateStatusActive   TemplateStatus = "ACTIVE"
	TemplateStatusArchived TemplateStatus = "ARCHIVED"
)

type G2STemplate struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Vendor               string         `json:"vendor"`
	CabinetFamily        string         `json:"cabinet_family,omitempty"`
	SoftwareVersionMatch string         `json:"software_version_match,omitempty"`
	Status               TemplateStatus `json:"status"`
	CurrentVersionID     string         `json:"current_version_id,omitempty"`
	Notes                string         `json:"notes,omitempty"`
	CreatedAt            time.Time      `json:"created_at,omitempty"`
	UpdatedAt            time.Time      `json:"updated_at,omitempty"`
}

func (t G2STemplate) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(t.Vendor) == "" {
		return fmt.Errorf("vendor is required")
	}
	switch t.Status {
	case TemplateStatusDraft, TemplateStatusActive, TemplateStatusArchived:
	default:
		return fmt.Errorf("status is invalid")
	}
	return nil
}

type G2STemplateVersion struct {
	ID                    string    `json:"id"`
	TemplateID            string    `json:"template_id"`
	VersionLabel          string    `json:"version_label"`
	EndpointQuirksJSON    string    `json:"endpoint_quirks_json,omitempty"`
	ActionsJSON           string    `json:"actions_json"`
	ConfirmationRulesJSON string    `json:"confirmation_rules_json,omitempty"`
	FailureRulesJSON      string    `json:"failure_rules_json,omitempty"`
	HeartbeatProfileJSON  string    `json:"heartbeat_profile_json,omitempty"`
	VariablesJSON         string    `json:"variables_json,omitempty"`
	Notes                 string    `json:"notes,omitempty"`
	CreatedAt             time.Time `json:"created_at,omitempty"`
}

func (v G2STemplateVersion) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(v.TemplateID) == "" {
		return fmt.Errorf("template_id is required")
	}
	if strings.TrimSpace(v.VersionLabel) == "" {
		return fmt.Errorf("version_label is required")
	}
	if strings.TrimSpace(v.ActionsJSON) == "" {
		return fmt.Errorf("actions_json is required")
	}
	return nil
}
