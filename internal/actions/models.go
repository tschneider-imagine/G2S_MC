package actions

import (
	"fmt"
	"strings"
	"time"
)

type ActionSeverity string

const (
	SeverityNotice      ActionSeverity = "NOTICE"
	SeverityBroadcast   ActionSeverity = "BROADCAST"
	SeverityEmergency   ActionSeverity = "EMERGENCY"
	SeverityRestore     ActionSeverity = "RESTORE"
	SeverityMaintenance ActionSeverity = "MAINTENANCE"
)

type ActionRunStatus string

const (
	RunStatusPending   ActionRunStatus = "PENDING"
	RunStatusRunning   ActionRunStatus = "RUNNING"
	RunStatusSucceeded ActionRunStatus = "SUCCEEDED"
	RunStatusFailed    ActionRunStatus = "FAILED"
	RunStatusEscalated ActionRunStatus = "ESCALATED"
)

type ActionTargetStatus string

const (
	TargetStatusPending   ActionTargetStatus = "PENDING"
	TargetStatusConfirmed ActionTargetStatus = "CONFIRMED"
	TargetStatusFailed    ActionTargetStatus = "FAILED"
	TargetStatusEscalated ActionTargetStatus = "ESCALATED"
)

type ActionStep struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Sequence           int    `json:"sequence"`
	TemplateActionKey  string `json:"template_action_key"`
	ConfirmationRuleID string `json:"confirmation_rule_id,omitempty"`
}

func (s ActionStep) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("step id is required")
	}
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("step name is required")
	}
	if s.Sequence < 0 {
		return fmt.Errorf("step sequence must be >= 0")
	}
	if strings.TrimSpace(s.TemplateActionKey) == "" {
		return fmt.Errorf("template_action_key is required")
	}
	return nil
}

type ActionDefinition struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Severity         ActionSeverity `json:"severity"`
	Enabled          bool           `json:"enabled"`
	TargetSelector   string         `json:"target_selector"`
	TemplateSelector string         `json:"template_selector"`
	Steps            []ActionStep   `json:"steps"`
	RetryPolicyJSON  string         `json:"retry_policy_json,omitempty"`
	EscalationJSON   string         `json:"escalation_policy_json,omitempty"`
	ReturnActionID   string         `json:"return_action_id,omitempty"`
	AuditPolicyJSON  string         `json:"audit_policy_json,omitempty"`
	Version          int            `json:"version"`
	CreatedAt        time.Time      `json:"created_at,omitempty"`
	UpdatedAt        time.Time      `json:"updated_at,omitempty"`
}

func (d ActionDefinition) Validate() error {
	if strings.TrimSpace(d.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("name is required")
	}
	switch d.Severity {
	case SeverityNotice, SeverityBroadcast, SeverityEmergency, SeverityRestore, SeverityMaintenance:
	default:
		return fmt.Errorf("severity is invalid")
	}
	if strings.TrimSpace(d.TargetSelector) == "" {
		return fmt.Errorf("target_selector is required")
	}
	if strings.TrimSpace(d.TemplateSelector) == "" {
		return fmt.Errorf("template_selector is required")
	}
	if len(d.Steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}
	for i := range d.Steps {
		if err := d.Steps[i].Validate(); err != nil {
			return fmt.Errorf("steps[%d]: %w", i, err)
		}
	}
	if d.Version < 1 {
		return fmt.Errorf("version must be >= 1")
	}
	return nil
}

type ActionRun struct {
	ID                 string          `json:"id"`
	ActionDefinitionID string          `json:"action_definition_id"`
	IncidentID         string          `json:"incident_id,omitempty"`
	InputTransitionID  int64           `json:"input_transition_id,omitempty"`
	StartedAt          time.Time       `json:"started_at"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
	Status             ActionRunStatus `json:"status"`
	TriggerReason      string          `json:"trigger_reason,omitempty"`
	TargetCount        int             `json:"target_count"`
	ConfirmedCount     int             `json:"confirmed_count"`
	FailedCount        int             `json:"failed_count"`
	EscalatedCount     int             `json:"escalated_count"`
}

func (r ActionRun) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(r.ActionDefinitionID) == "" {
		return fmt.Errorf("action_definition_id is required")
	}
	if r.StartedAt.IsZero() {
		return fmt.Errorf("started_at is required")
	}
	switch r.Status {
	case RunStatusPending, RunStatusRunning, RunStatusSucceeded, RunStatusFailed, RunStatusEscalated:
	default:
		return fmt.Errorf("status is invalid")
	}
	if r.TargetCount < 0 || r.ConfirmedCount < 0 || r.FailedCount < 0 || r.EscalatedCount < 0 {
		return fmt.Errorf("run counters must be >= 0")
	}
	return nil
}

type ActionTargetResult struct {
	ID           int64              `json:"id"`
	ActionRunID  string             `json:"action_run_id"`
	TargetEGMID  string             `json:"target_egm_id"`
	Status       ActionTargetStatus `json:"status"`
	AttemptCount int                `json:"attempt_count"`
	LastError    string             `json:"last_error,omitempty"`
	LastResultAt *time.Time         `json:"last_result_at,omitempty"`
}

func (r ActionTargetResult) Validate() error {
	if strings.TrimSpace(r.ActionRunID) == "" {
		return fmt.Errorf("action_run_id is required")
	}
	if strings.TrimSpace(r.TargetEGMID) == "" {
		return fmt.Errorf("target_egm_id is required")
	}
	switch r.Status {
	case TargetStatusPending, TargetStatusConfirmed, TargetStatusFailed, TargetStatusEscalated:
	default:
		return fmt.Errorf("status is invalid")
	}
	if r.AttemptCount < 0 {
		return fmt.Errorf("attempt_count must be >= 0")
	}
	return nil
}
