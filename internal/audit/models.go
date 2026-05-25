package audit

import (
	"fmt"
	"strings"
	"time"
)

type AuditSeverity string

const (
	AuditSeverityInfo      AuditSeverity = "INFO"
	AuditSeverityWarning   AuditSeverity = "WARNING"
	AuditSeverityEmergency AuditSeverity = "EMERGENCY"
)

const (
	EventTypeInputTransition          = "INPUT_TRANSITION"
	EventTypeInputLatchClearAttempted = "INPUT_LATCH_CLEAR_ATTEMPTED"
	EventTypeInputLatchClearSucceeded = "INPUT_LATCH_CLEAR_SUCCEEDED"
	EventTypeInputLatchClearFailed    = "INPUT_LATCH_CLEAR_FAILED"
	EventTypeActionQueued             = "ACTION_QUEUED"
	EventTypeActionDispatchPrepared   = "ACTION_DISPATCH_PREPARED"
	EventTypeMessageSendBlocked       = "MESSAGE_SEND_BLOCKED"
	EventTypeMessageSendAttempted     = "MESSAGE_SEND_ATTEMPTED"
	EventTypeMessageSendSucceeded     = "MESSAGE_SEND_SUCCEEDED"
	EventTypeMessageSendFailed        = "MESSAGE_SEND_FAILED"
	EventTypeActionStarted            = "ACTION_STARTED"
	EventTypeActionStep               = "ACTION_STEP"
	EventTypeMessageSent              = "MESSAGE_SENT"
	EventTypeMessageReceived          = "MESSAGE_RECEIVED"
	EventTypeHandlerRule              = "HANDLER_RULE"
	EventTypeConfirmation             = "CONFIRMATION"
	EventTypeRetry                    = "RETRY"
	EventTypeEscalation               = "ESCALATION"
	EventTypeReturnToNormal           = "RETURN_TO_NORMAL"
	EventTypeIncidentOpened           = "INCIDENT_OPENED"
	EventTypeIncidentClosed           = "INCIDENT_CLOSED"
	EventTypeIncidentLinked           = "INCIDENT_LINKED"
	EventTypeOperatorAction           = "OPERATOR_ACTION"
	EventTypeSystemWarning            = "SYSTEM_WARNING"
)

type AuditTimelineEntry struct {
	ID                int64         `json:"id"`
	OccurredAt        time.Time     `json:"occurred_at"`
	Severity          AuditSeverity `json:"severity"`
	EventType         string        `json:"event_type"`
	Summary           string        `json:"summary"`
	DetailJSON        string        `json:"detail_json,omitempty"`
	ActionRunID       string        `json:"action_run_id,omitempty"`
	InputTransitionID int64         `json:"input_transition_id,omitempty"`
	MessageJournalID  int64         `json:"message_journal_id,omitempty"`
	Operator          string        `json:"operator,omitempty"`
}

func (e AuditTimelineEntry) Validate() error {
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("occurred_at is required")
	}
	switch e.Severity {
	case AuditSeverityInfo, AuditSeverityWarning, AuditSeverityEmergency:
	default:
		return fmt.Errorf("severity is invalid")
	}
	if strings.TrimSpace(e.EventType) == "" {
		return fmt.Errorf("event_type is required")
	}
	if strings.TrimSpace(e.Summary) == "" {
		return fmt.Errorf("summary is required")
	}
	return nil
}
