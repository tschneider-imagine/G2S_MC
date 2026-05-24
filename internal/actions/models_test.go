package actions

import (
	"testing"
	"time"
)

func validDefinition() ActionDefinition {
	return ActionDefinition{
		ID:               "action-emergency-silence",
		Name:             "Emergency Silence",
		Severity:         SeverityEmergency,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps: []ActionStep{{
			ID:                "step-1",
			Name:              "Send mute",
			Sequence:          0,
			TemplateActionKey: "mute_primary",
		}},
		Version: 1,
	}
}

func TestActionDefinitionValidate(t *testing.T) {
	definition := validDefinition()
	if err := definition.Validate(); err != nil {
		t.Fatalf("validate definition: %v", err)
	}

	definition.Steps = nil
	if err := definition.Validate(); err == nil {
		t.Fatal("expected validation error for missing steps")
	}
}

func TestActionRunValidate(t *testing.T) {
	run := ActionRun{
		ID:                 "run-1",
		ActionDefinitionID: "action-emergency-silence",
		StartedAt:          time.Now(),
		Status:             RunStatusRunning,
		TargetCount:        10,
	}
	if err := run.Validate(); err != nil {
		t.Fatalf("validate run: %v", err)
	}

	run.Status = "UNKNOWN"
	if err := run.Validate(); err == nil {
		t.Fatal("expected validation error for invalid status")
	}
}

func TestActionTargetResultValidate(t *testing.T) {
	result := ActionTargetResult{
		ActionRunID:  "run-1",
		TargetEGMID:  "EGM-1",
		Status:       TargetStatusConfirmed,
		AttemptCount: 1,
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("validate target result: %v", err)
	}

	result.AttemptCount = -1
	if err := result.Validate(); err == nil {
		t.Fatal("expected validation error for negative attempt_count")
	}
}
