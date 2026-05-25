package g2sengine

import (
	"encoding/json"
	"fmt"
	"strings"
)

type MatchOutcome string

const (
	MatchOutcomeNoMatch  MatchOutcome = "NO_MATCH"
	MatchOutcomeExpected MatchOutcome = "EXPECTED"
	MatchOutcomeFailure  MatchOutcome = "FAILURE"
)

type MatchRule struct {
	ID          string   `json:"id,omitempty"`
	Label       string   `json:"label,omitempty"`
	Contains    []string `json:"contains,omitempty"`
	AllContains []string `json:"all_contains,omitempty"`
	AnyContains []string `json:"any_contains,omitempty"`
	NotContains []string `json:"not_contains,omitempty"`
	MessageType string   `json:"message_type,omitempty"`
}

type MatcherDocument struct {
	Rules []MatchRule `json:"rules"`
}

type MatchResult struct {
	Outcome   string   `json:"outcome"`
	RuleID    string   `json:"rule_id,omitempty"`
	RuleLabel string   `json:"rule_label,omitempty"`
	Reason    string   `json:"reason"`
	Warnings  []string `json:"warnings,omitempty"`
}

func ParseMatcherDocument(raw string) (MatcherDocument, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return MatcherDocument{Rules: []MatchRule{}}, nil
	}
	var doc MatcherDocument
	if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
		return MatcherDocument{}, fmt.Errorf("invalid matcher JSON")
	}
	if doc.Rules == nil {
		doc.Rules = []MatchRule{}
	}
	return doc, nil
}

func MatchMessage(rawPayload string, parsedSummaryJSON string, messageType string, expectedRules string, failureRules string) (MatchResult, error) {
	expectedDoc, err := ParseMatcherDocument(expectedRules)
	if err != nil {
		return MatchResult{}, err
	}
	failureDoc, err := ParseMatcherDocument(failureRules)
	if err != nil {
		return MatchResult{}, err
	}

	haystack := strings.ToLower(strings.TrimSpace(rawPayload))
	summary := strings.ToLower(strings.TrimSpace(parsedSummaryJSON))
	if summary != "" {
		if haystack != "" {
			haystack = haystack + "\n" + summary
		} else {
			haystack = summary
		}
	}
	messageTypeLower := strings.ToLower(strings.TrimSpace(messageType))

	if failureMatch, ok := findFirstMatch(failureDoc.Rules, haystack, messageTypeLower); ok {
		return MatchResult{
			Outcome:   string(MatchOutcomeFailure),
			RuleID:    strings.TrimSpace(failureMatch.ID),
			RuleLabel: strings.TrimSpace(failureMatch.Label),
			Reason:    buildRuleReason(failureMatch),
			Warnings:  []string{},
		}, nil
	}
	if expectedMatch, ok := findFirstMatch(expectedDoc.Rules, haystack, messageTypeLower); ok {
		return MatchResult{
			Outcome:   string(MatchOutcomeExpected),
			RuleID:    strings.TrimSpace(expectedMatch.ID),
			RuleLabel: strings.TrimSpace(expectedMatch.Label),
			Reason:    buildRuleReason(expectedMatch),
			Warnings:  []string{},
		}, nil
	}

	return MatchResult{
		Outcome:  string(MatchOutcomeNoMatch),
		Reason:   "no matcher rule matched",
		Warnings: []string{},
	}, nil
}

func findFirstMatch(rules []MatchRule, haystack string, messageTypeLower string) (MatchRule, bool) {
	for _, rule := range rules {
		if ruleMatches(rule, haystack, messageTypeLower) {
			return rule, true
		}
	}
	return MatchRule{}, false
}

func ruleMatches(rule MatchRule, haystack string, messageTypeLower string) bool {
	ruleType := strings.ToLower(strings.TrimSpace(rule.MessageType))
	if ruleType != "" && ruleType != messageTypeLower {
		return false
	}

	required := append([]string{}, rule.AllContains...)
	required = append(required, rule.Contains...)
	for _, raw := range required {
		token := strings.ToLower(strings.TrimSpace(raw))
		if token == "" {
			continue
		}
		if !strings.Contains(haystack, token) {
			return false
		}
	}

	anyTokens := sanitizeTokens(rule.AnyContains)
	if len(anyTokens) > 0 {
		anyMatched := false
		for _, token := range anyTokens {
			if strings.Contains(haystack, token) {
				anyMatched = true
				break
			}
		}
		if !anyMatched {
			return false
		}
	}

	for _, token := range sanitizeTokens(rule.NotContains) {
		if strings.Contains(haystack, token) {
			return false
		}
	}
	return true
}

func sanitizeTokens(values []string) []string {
	result := make([]string, 0, len(values))
	for _, raw := range values {
		token := strings.ToLower(strings.TrimSpace(raw))
		if token != "" {
			result = append(result, token)
		}
	}
	return result
}

func buildRuleReason(rule MatchRule) string {
	if strings.TrimSpace(rule.Label) != "" {
		return "matched rule " + strings.TrimSpace(rule.Label)
	}
	if strings.TrimSpace(rule.ID) != "" {
		return "matched rule " + strings.TrimSpace(rule.ID)
	}
	return "matched unnamed rule"
}
