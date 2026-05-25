package g2sengine

import (
	"fmt"
	"strings"
	"time"
)

type MessageDirection string

const (
	DirectionInbound  MessageDirection = "INBOUND"
	DirectionOutbound MessageDirection = "OUTBOUND"
)

type MessageResult string

const (
	MessageResultSent          MessageResult = "SENT"
	MessageResultReceived      MessageResult = "RECEIVED"
	MessageResultAcked         MessageResult = "ACKED"
	MessageResultConfirmed     MessageResult = "CONFIRMED"
	MessageResultFailed        MessageResult = "FAILED"
	MessageResultIgnored       MessageResult = "IGNORED"
	MessageResultEscalated     MessageResult = "ESCALATED"
	MessageResultDryRun        MessageResult = "DRY_RUN"
	MessageResultSendBlocked   MessageResult = "SEND_BLOCKED"
	MessageResultSendAttempted MessageResult = "SEND_ATTEMPTED"
	MessageResultSendFailed    MessageResult = "SEND_FAILED"
	MessageResultSendSucceeded MessageResult = "SEND_SUCCEEDED"
)

type MessageJournalEntry struct {
	ID                int64            `json:"id"`
	Timestamp         time.Time        `json:"timestamp"`
	Direction         MessageDirection `json:"direction"`
	FromEndpoint      string           `json:"from_endpoint,omitempty"`
	ToEndpoint        string           `json:"to_endpoint,omitempty"`
	EGMID             string           `json:"egm_id,omitempty"`
	ActionRunID       string           `json:"action_run_id,omitempty"`
	ActionStepID      string           `json:"action_step_id,omitempty"`
	InputTransitionID int64            `json:"input_transition_id,omitempty"`
	TemplateID        string           `json:"template_id,omitempty"`
	TemplateVersion   string           `json:"template_version,omitempty"`
	HandlerRuleID     string           `json:"handler_rule_id,omitempty"`
	MessageType       string           `json:"message_type,omitempty"`
	RawPayload        string           `json:"raw_payload"`
	ParsedSummaryJSON string           `json:"parsed_summary_json,omitempty"`
	Result            MessageResult    `json:"result"`
	Error             string           `json:"error,omitempty"`
	HTTPStatusCode    int              `json:"http_status_code,omitempty"`
	LatencyMS         int              `json:"latency_ms,omitempty"`
	ResponseExcerpt   string           `json:"response_excerpt,omitempty"`
	SentAt            *time.Time       `json:"sent_at,omitempty"`
	CompletedAt       *time.Time       `json:"completed_at,omitempty"`
	TransportMode     string           `json:"transport_mode,omitempty"`
}

func (m MessageJournalEntry) Validate() error {
	if m.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	if m.Direction != DirectionInbound && m.Direction != DirectionOutbound {
		return fmt.Errorf("direction must be INBOUND or OUTBOUND")
	}
	if strings.TrimSpace(m.RawPayload) == "" {
		return fmt.Errorf("raw_payload is required")
	}
	switch m.Result {
	case MessageResultSent, MessageResultReceived, MessageResultAcked, MessageResultConfirmed, MessageResultFailed, MessageResultIgnored, MessageResultEscalated, MessageResultDryRun, MessageResultSendBlocked, MessageResultSendAttempted, MessageResultSendFailed, MessageResultSendSucceeded:
	default:
		return fmt.Errorf("result is invalid")
	}
	return nil
}

type HandlerRule struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Enabled      bool                 `json:"enabled"`
	Direction    HandlerRuleDirection `json:"direction"`
	TemplateID   string               `json:"template_id,omitempty"`
	MessageType  string               `json:"message_type,omitempty"`
	EGMID        string               `json:"egm_id,omitempty"`
	ActionID     string               `json:"action_id,omitempty"`
	ActionStepID string               `json:"action_step_id,omitempty"`
	MatchJSON    string               `json:"match_json"`
	Outcome      HandlerRuleOutcome   `json:"outcome"`
	HandleJSON   string               `json:"handle_json,omitempty"`
	Notes        string               `json:"notes,omitempty"`
	CreatedAt    time.Time            `json:"created_at,omitempty"`
	UpdatedAt    time.Time            `json:"updated_at,omitempty"`
}

func (r HandlerRule) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(r.MatchJSON) == "" {
		return fmt.Errorf("match_json is required")
	}
	switch normalizeHandlerRuleDirection(r.Direction) {
	case HandlerRuleDirectionAny, HandlerRuleDirectionInbound, HandlerRuleDirectionOutbound:
	default:
		return fmt.Errorf("direction is invalid")
	}
	switch normalizeHandlerRuleOutcome(r.Outcome) {
	case HandlerRuleOutcomeConfirmation, HandlerRuleOutcomeFailure, HandlerRuleOutcomeIgnore, HandlerRuleOutcomeNote:
	default:
		return fmt.Errorf("outcome is invalid")
	}
	return nil
}

type HandlerRuleDirection string

const (
	HandlerRuleDirectionAny      HandlerRuleDirection = "ANY"
	HandlerRuleDirectionInbound  HandlerRuleDirection = "INBOUND"
	HandlerRuleDirectionOutbound HandlerRuleDirection = "OUTBOUND"
)

type HandlerRuleOutcome string

const (
	HandlerRuleOutcomeConfirmation HandlerRuleOutcome = "CONFIRMATION"
	HandlerRuleOutcomeFailure      HandlerRuleOutcome = "FAILURE"
	HandlerRuleOutcomeIgnore       HandlerRuleOutcome = "IGNORE"
	HandlerRuleOutcomeNote         HandlerRuleOutcome = "NOTE"
)

func normalizeHandlerRuleDirection(value HandlerRuleDirection) HandlerRuleDirection {
	normalized := strings.ToUpper(strings.TrimSpace(string(value)))
	switch normalized {
	case string(HandlerRuleDirectionInbound):
		return HandlerRuleDirectionInbound
	case string(HandlerRuleDirectionOutbound):
		return HandlerRuleDirectionOutbound
	default:
		return HandlerRuleDirectionAny
	}
}

func normalizeHandlerRuleOutcome(value HandlerRuleOutcome) HandlerRuleOutcome {
	normalized := strings.ToUpper(strings.TrimSpace(string(value)))
	switch normalized {
	case string(HandlerRuleOutcomeConfirmation):
		return HandlerRuleOutcomeConfirmation
	case string(HandlerRuleOutcomeFailure):
		return HandlerRuleOutcomeFailure
	case string(HandlerRuleOutcomeIgnore):
		return HandlerRuleOutcomeIgnore
	case string(HandlerRuleOutcomeNote):
		return HandlerRuleOutcomeNote
	default:
		return ""
	}
}
