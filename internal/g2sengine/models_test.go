package g2sengine

import (
	"testing"
	"time"
)

func TestMessageJournalEntryValidate(t *testing.T) {
	entry := MessageJournalEntry{
		Timestamp:  time.Now(),
		Direction:  DirectionOutbound,
		RawPayload: "<xml/>",
		Result:     MessageResultSent,
	}
	if err := entry.Validate(); err != nil {
		t.Fatalf("validate message journal entry: %v", err)
	}
	entry.RawPayload = ""
	if err := entry.Validate(); err == nil {
		t.Fatal("expected validation error for missing raw payload")
	}
}

func TestHandlerRuleValidate(t *testing.T) {
	rule := HandlerRule{
		ID:        "rule-1",
		Name:      "Ack rule",
		Direction: HandlerRuleDirectionInbound,
		MatchJSON: "{}",
		Outcome:   HandlerRuleOutcomeConfirmation,
	}
	if err := rule.Validate(); err != nil {
		t.Fatalf("validate handler rule: %v", err)
	}
	rule.MatchJSON = ""
	if err := rule.Validate(); err == nil {
		t.Fatal("expected validation error for missing match_json")
	}
	rule.MatchJSON = "{}"
	rule.Outcome = ""
	if err := rule.Validate(); err == nil {
		t.Fatal("expected validation error for missing outcome")
	}
}
