package inputruntime

import (
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

type InputSample struct {
	InputID    string
	RawState   inputs.DigitalState
	ObservedAt time.Time
}

type InputRuntimeState struct {
	InputID              string                   `json:"input_id"`
	StableRawState       inputs.DigitalState      `json:"stable_raw_state"`
	DerivedState         inputs.DerivedInputState `json:"derived_state"`
	LatchActive          bool                     `json:"latch_active"`
	LatchClearedAt       *time.Time               `json:"latch_cleared_at,omitempty"`
	StableSince          time.Time                `json:"stable_since"`
	LastObservedRawState inputs.DigitalState      `json:"last_observed_raw_state"`
	LastObservedAt       time.Time                `json:"last_observed_at"`
	PendingRawState      inputs.DigitalState      `json:"pending_raw_state,omitempty"`
	PendingSince         *time.Time               `json:"pending_since,omitempty"`
	LastTransitionID     int64                    `json:"last_transition_id,omitempty"`
	LastTransitionAt     *time.Time               `json:"last_transition_at,omitempty"`
	UpdatedAt            time.Time                `json:"updated_at"`
}

type EvaluationResult struct {
	State          InputRuntimeState
	Transition     *inputs.InputTransition
	AuditEntry     *audit.AuditTimelineEntry
	ActionQueuedID string
	Changed        bool
	Suppressed     bool
}

type ActiveInput struct {
	InputID      string
	Name         string
	Priority     int
	DerivedState inputs.DerivedInputState
	ActionID     string
}

type ClearLatchResult struct {
	InputID        string                     `json:"input_id"`
	Cleared        bool                       `json:"cleared"`
	Transition     *inputs.InputTransition    `json:"transition,omitempty"`
	AuditEntries   []audit.AuditTimelineEntry `json:"audit_entries,omitempty"`
	ActionQueuedID string                     `json:"action_queued_id,omitempty"`
	Reason         string                     `json:"reason,omitempty"`
}
