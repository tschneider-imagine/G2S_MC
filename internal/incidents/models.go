package incidents

import "time"

type Status string

const (
	StatusOpen   Status = "OPEN"
	StatusClosed Status = "CLOSED"
)

type IncidentRecord struct {
	ID                   int64      `json:"id"`
	OpenedAt             time.Time  `json:"opened_at"`
	ClosedAt             *time.Time `json:"closed_at,omitempty"`
	Status               Status     `json:"status"`
	Severity             string     `json:"severity"`
	PrimaryInputID       string     `json:"primary_input_id"`
	PrimaryActionRunID   string     `json:"primary_action_run_id,omitempty"`
	OpenedByTransitionID int64      `json:"opened_by_transition_id"`
	ClosedByTransitionID int64      `json:"closed_by_transition_id,omitempty"`
	CloseReason          string     `json:"close_reason,omitempty"`
	Summary              string     `json:"summary"`
	DetailJSON           string     `json:"detail_json,omitempty"`
}

type TransitionResult struct {
	Incident *IncidentRecord
	Opened   bool
	Closed   bool
	Reason   string
}
