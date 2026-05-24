package inputs

import (
	"fmt"
	"strings"
	"time"
)

type InputElectricalState string

const (
	InputStateHigh InputElectricalState = "HIGH"
	InputStateLow  InputElectricalState = "LOW"
)

type DigitalState = InputElectricalState

type InputDerivedState string

const (
	DerivedStateNormal    InputDerivedState = "NORMAL"
	DerivedStateTriggered InputDerivedState = "TRIGGERED"
)

type DerivedInputState = InputDerivedState

type InputLatchingMode string

const (
	LatchingAutoClear   InputLatchingMode = "AUTO_CLEAR"
	LatchingManualClear InputLatchingMode = "MANUAL_CLEAR"
)

type InputChannel struct {
	ID                string               `json:"id"`
	Name              string               `json:"name"`
	GPIOChannel       string               `json:"gpio_channel"`
	Enabled           bool                 `json:"enabled"`
	NormalState       InputElectricalState `json:"normal_state"`
	CurrentState      InputElectricalState `json:"current_state"`
	DerivedState      InputDerivedState    `json:"derived_state"`
	DebounceMS        int                  `json:"debounce_ms"`
	Priority          int                  `json:"priority"`
	OnTriggerActionID string               `json:"on_trigger_action_id,omitempty"`
	OnNormalActionID  string               `json:"on_normal_action_id,omitempty"`
	LatchingMode      InputLatchingMode    `json:"latching_mode"`
	LastTransitionAt  *time.Time           `json:"last_transition_at,omitempty"`
	CreatedAt         time.Time            `json:"created_at,omitempty"`
	UpdatedAt         time.Time            `json:"updated_at,omitempty"`
}

func (c InputChannel) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(c.GPIOChannel) == "" {
		return fmt.Errorf("gpio_channel is required")
	}
	if c.NormalState != InputStateHigh && c.NormalState != InputStateLow {
		return fmt.Errorf("normal_state must be HIGH or LOW")
	}
	if c.CurrentState != InputStateHigh && c.CurrentState != InputStateLow {
		return fmt.Errorf("current_state must be HIGH or LOW")
	}
	if c.DerivedState != DerivedStateNormal && c.DerivedState != DerivedStateTriggered {
		return fmt.Errorf("derived_state must be NORMAL or TRIGGERED")
	}
	if c.DebounceMS < 0 {
		return fmt.Errorf("debounce_ms must be >= 0")
	}
	if c.Priority < 0 {
		return fmt.Errorf("priority must be >= 0")
	}
	if c.LatchingMode != LatchingAutoClear && c.LatchingMode != LatchingManualClear {
		return fmt.Errorf("latching_mode must be AUTO_CLEAR or MANUAL_CLEAR")
	}
	return nil
}

type InputTransition struct {
	ID              int64             `json:"id"`
	InputChannelID  string            `json:"input_channel_id"`
	PreviousDerived InputDerivedState `json:"previous_derived_state"`
	NewDerived      InputDerivedState `json:"new_derived_state"`
	TransitionAt    time.Time         `json:"transition_at"`
	Reason          string            `json:"reason,omitempty"`
	ActionRunID     string            `json:"action_run_id,omitempty"`
}

func (t InputTransition) Validate() error {
	if strings.TrimSpace(t.InputChannelID) == "" {
		return fmt.Errorf("input_channel_id is required")
	}
	if t.PreviousDerived != DerivedStateNormal && t.PreviousDerived != DerivedStateTriggered {
		return fmt.Errorf("previous_derived_state must be NORMAL or TRIGGERED")
	}
	if t.NewDerived != DerivedStateNormal && t.NewDerived != DerivedStateTriggered {
		return fmt.Errorf("new_derived_state must be NORMAL or TRIGGERED")
	}
	if t.TransitionAt.IsZero() {
		return fmt.Errorf("transition_at is required")
	}
	return nil
}

func DeriveState(raw DigitalState, normal DigitalState) (DerivedInputState, error) {
	if raw != InputStateHigh && raw != InputStateLow {
		return "", fmt.Errorf("raw state must be HIGH or LOW")
	}
	if normal != InputStateHigh && normal != InputStateLow {
		return "", fmt.Errorf("normal state must be HIGH or LOW")
	}
	if raw == normal {
		return DerivedStateNormal, nil
	}
	return DerivedStateTriggered, nil
}
