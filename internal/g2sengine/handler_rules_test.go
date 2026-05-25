package g2sengine

import "testing"

func TestEvaluateHandlerRulesMatchesRawPayload(t *testing.T) {
	rules := []HandlerRule{
		{
			ID:        "r1",
			Name:      "Accepted",
			Enabled:   true,
			Direction: HandlerRuleDirectionInbound,
			MatchJSON: `{"contains":["accepted"]}`,
			Outcome:   HandlerRuleOutcomeConfirmation,
		},
	}
	match, err := EvaluateHandlerRules(rules, DirectionInbound, "template-1", "ACK", "EGM-001", "action-1", "step-1", "<ack>accepted</ack>", "")
	if err != nil {
		t.Fatalf("evaluate handler rules: %v", err)
	}
	if match == nil || match.Rule.ID != "r1" {
		t.Fatalf("unexpected match: %+v", match)
	}
}

func TestEvaluateHandlerRulesMatchesParsedSummary(t *testing.T) {
	rules := []HandlerRule{
		{
			ID:        "r1",
			Name:      "Summary OK",
			Enabled:   true,
			Direction: HandlerRuleDirectionAny,
			MatchJSON: `{"contains":["result_ok"]}`,
			Outcome:   HandlerRuleOutcomeConfirmation,
		},
	}
	match, err := EvaluateHandlerRules(rules, DirectionInbound, "", "ACK", "", "", "", "", `{"result":"result_ok"}`)
	if err != nil {
		t.Fatalf("evaluate handler rules: %v", err)
	}
	if match == nil {
		t.Fatal("expected rule match")
	}
}

func TestEvaluateHandlerRulesAllAnyAndNotContains(t *testing.T) {
	rules := []HandlerRule{
		{
			ID:        "r1",
			Name:      "Rule",
			Enabled:   true,
			Direction: HandlerRuleDirectionAny,
			MatchJSON: `{"all_contains":["<ack","accepted"],"any_contains":["accepted","success"],"not_contains":["error"]}`,
			Outcome:   HandlerRuleOutcomeConfirmation,
		},
	}
	match, err := EvaluateHandlerRules(rules, DirectionInbound, "", "ACK", "", "", "", "<ack>accepted</ack>", "")
	if err != nil {
		t.Fatalf("evaluate handler rules: %v", err)
	}
	if match == nil {
		t.Fatal("expected match")
	}

	match, err = EvaluateHandlerRules(rules, DirectionInbound, "", "ACK", "", "", "", "<ack>accepted error</ack>", "")
	if err != nil {
		t.Fatalf("evaluate handler rules: %v", err)
	}
	if match != nil {
		t.Fatalf("expected no match, got %+v", match)
	}
}

func TestEvaluateHandlerRulesCaseInsensitive(t *testing.T) {
	rules := []HandlerRule{
		{
			ID:        "r1",
			Name:      "Case",
			Enabled:   true,
			Direction: HandlerRuleDirectionAny,
			MatchJSON: `{"contains":["accepted"]}`,
			Outcome:   HandlerRuleOutcomeConfirmation,
		},
	}
	match, err := EvaluateHandlerRules(rules, DirectionInbound, "", "ACK", "", "", "", "<ACK>ACCEPTED</ACK>", "")
	if err != nil {
		t.Fatalf("evaluate handler rules: %v", err)
	}
	if match == nil {
		t.Fatal("expected case-insensitive match")
	}
}

func TestEvaluateHandlerRulesFailurePrecedence(t *testing.T) {
	rules := []HandlerRule{
		{
			ID:        "confirm",
			Name:      "Confirm",
			Enabled:   true,
			Direction: HandlerRuleDirectionInbound,
			MatchJSON: `{"contains":["accepted"]}`,
			Outcome:   HandlerRuleOutcomeConfirmation,
		},
		{
			ID:        "fail",
			Name:      "Fail",
			Enabled:   true,
			Direction: HandlerRuleDirectionInbound,
			MatchJSON: `{"contains":["accepted"]}`,
			Outcome:   HandlerRuleOutcomeFailure,
		},
	}
	match, err := EvaluateHandlerRules(rules, DirectionInbound, "", "ACK", "", "", "", "<ack>accepted</ack>", "")
	if err != nil {
		t.Fatalf("evaluate handler rules: %v", err)
	}
	if match == nil || match.Outcome != HandlerRuleOutcomeFailure {
		t.Fatalf("expected failure precedence, got %+v", match)
	}
}
