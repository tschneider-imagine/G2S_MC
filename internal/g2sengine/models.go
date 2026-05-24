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
	MessageResultSent      MessageResult = "SENT"
	MessageResultReceived  MessageResult = "RECEIVED"
	MessageResultAcked     MessageResult = "ACKED"
	MessageResultConfirmed MessageResult = "CONFIRMED"
	MessageResultFailed    MessageResult = "FAILED"
	MessageResultIgnored   MessageResult = "IGNORED"
	MessageResultEscalated MessageResult = "ESCALATED"
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
	case MessageResultSent, MessageResultReceived, MessageResultAcked, MessageResultConfirmed, MessageResultFailed, MessageResultIgnored, MessageResultEscalated:
	default:
		return fmt.Errorf("result is invalid")
	}
	return nil
}

type HandlerRule struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Enabled    bool      `json:"enabled"`
	MatchJSON  string    `json:"match_json"`
	HandleJSON string    `json:"handle_json"`
	Notes      string    `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
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
	if strings.TrimSpace(r.HandleJSON) == "" {
		return fmt.Errorf("handle_json is required")
	}
	return nil
}
