package actionplanner

import (
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
)

type PlanningWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ActionPlanStep struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Sequence          int    `json:"sequence"`
	TemplateActionKey string `json:"template_action_key"`
}

type ActionPlanTarget struct {
	EGMID           string `json:"egm_id"`
	DisplayName     string `json:"display_name,omitempty"`
	TemplateID      string `json:"template_id,omitempty"`
	IPAddress       string `json:"ip_address,omitempty"`
	EndpointPath    string `json:"endpoint_path,omitempty"`
	MissingTemplate bool   `json:"missing_template"`
}

type ActionPlan struct {
	ActionID    string             `json:"action_id"`
	ActionName  string             `json:"action_name"`
	Version     int                `json:"version"`
	TargetCount int                `json:"target_count"`
	Targets     []ActionPlanTarget `json:"targets"`
	Steps       []ActionPlanStep   `json:"steps"`
	Warnings    []PlanningWarning  `json:"warnings"`
}

const (
	SelectorAllEmergencyEnabled = "ALL_EMERGENCY_ENABLED"
	SelectorEGMIDsPrefix        = "EGM_IDS:"
	SelectorGroupPrefix         = "GROUP:"
	SelectorTemplatePrefix      = "TEMPLATE:"
	SelectorZonePrefix          = "ZONE:"
)

func planStepsFromDefinition(definition actions.ActionDefinition) []ActionPlanStep {
	steps := make([]ActionPlanStep, 0, len(definition.Steps))
	for _, step := range definition.Steps {
		steps = append(steps, ActionPlanStep{
			ID:                step.ID,
			Name:              step.Name,
			Sequence:          step.Sequence,
			TemplateActionKey: step.TemplateActionKey,
		})
	}
	return steps
}
