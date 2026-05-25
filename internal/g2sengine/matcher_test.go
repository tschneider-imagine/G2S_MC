package g2sengine

import "testing"

func TestMatchMessageEmptyMatchersNoMatch(t *testing.T) {
	result, err := MatchMessage("<message/>", "", "NOTICE", "", "")
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeNoMatch) {
		t.Fatalf("outcome=%q", result.Outcome)
	}
}

func TestMatchMessageExpectedMatcherMatchesRawPayload(t *testing.T) {
	expected := `{"rules":[{"id":"accepted","contains":["accepted"]}]}`
	result, err := MatchMessage("<ack>accepted</ack>", "", "NOTICE", expected, "")
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeExpected) || result.RuleID != "accepted" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMatchMessageFailureMatcherMatchesRawPayload(t *testing.T) {
	failure := `{"rules":[{"id":"rejected","contains":["rejected"]}]}`
	result, err := MatchMessage("<ack>rejected</ack>", "", "NOTICE", "", failure)
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeFailure) || result.RuleID != "rejected" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMatchMessageFailureWinsOverExpected(t *testing.T) {
	expected := `{"rules":[{"id":"expected","contains":["accepted"]}]}`
	failure := `{"rules":[{"id":"failure","contains":["accepted"]}]}`
	result, err := MatchMessage("<ack>accepted</ack>", "", "NOTICE", expected, failure)
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeFailure) || result.RuleID != "failure" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMatchMessageAllContainsRequiresAllStrings(t *testing.T) {
	expected := `{"rules":[{"id":"expected","all_contains":["<ack","accepted"]}]}`
	result, err := MatchMessage("<ack>accepted</ack>", "", "NOTICE", expected, "")
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeExpected) {
		t.Fatalf("result=%+v", result)
	}

	result, err = MatchMessage("<ack>ok</ack>", "", "NOTICE", expected, "")
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeNoMatch) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMatchMessageAnyContainsRequiresOneString(t *testing.T) {
	expected := `{"rules":[{"id":"expected","any_contains":["ok","accepted"]}]}`
	result, err := MatchMessage("<ack>accepted</ack>", "", "NOTICE", expected, "")
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeExpected) {
		t.Fatalf("result=%+v", result)
	}
	result, err = MatchMessage("<ack>pending</ack>", "", "NOTICE", expected, "")
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeNoMatch) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMatchMessageNotContainsPreventsMatch(t *testing.T) {
	expected := `{"rules":[{"id":"expected","contains":["accepted"],"not_contains":["error"]}]}`
	result, err := MatchMessage("<ack>accepted error</ack>", "", "NOTICE", expected, "")
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeNoMatch) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMatchMessageIsCaseInsensitive(t *testing.T) {
	expected := `{"rules":[{"id":"expected","contains":["accepted"]}]}`
	result, err := MatchMessage("<ACK>ACCEPTED</ACK>", "", "NOTICE", expected, "")
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeExpected) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMatchMessageMatchesParsedSummaryJSON(t *testing.T) {
	expected := `{"rules":[{"id":"expected","contains":["response_ok"]}]}`
	result, err := MatchMessage("", `{"result":"response_ok"}`, "NOTICE", expected, "")
	if err != nil {
		t.Fatalf("match message: %v", err)
	}
	if result.Outcome != string(MatchOutcomeExpected) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMatchMessageInvalidMatcherJSONReturnsError(t *testing.T) {
	_, err := MatchMessage("<ack/>", "", "NOTICE", `{"rules":`, "")
	if err == nil {
		t.Fatal("expected invalid matcher JSON error")
	}
}
