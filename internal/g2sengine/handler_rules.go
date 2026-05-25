package g2sengine

import (
	"encoding/json"
	"fmt"
	"strings"
)

type HandlerRuleMatchDocument struct {
	Contains    []string `json:"contains,omitempty"`
	AllContains []string `json:"all_contains,omitempty"`
	AnyContains []string `json:"any_contains,omitempty"`
	NotContains []string `json:"not_contains,omitempty"`
}

type HandlerRuleMatchResult struct {
	Matched bool
	Reason  string
}

type HandlerRuleEvaluation struct {
	Rule    HandlerRule
	Outcome HandlerRuleOutcome
	Reason  string
}

func ParseHandlerRuleMatchDocument(raw string) (HandlerRuleMatchDocument, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return HandlerRuleMatchDocument{}, nil
	}
	var doc HandlerRuleMatchDocument
	if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
		return HandlerRuleMatchDocument{}, fmt.Errorf("invalid handler rule match JSON")
	}
	return doc, nil
}

func MatchHandlerRule(rule HandlerRule, direction MessageDirection, templateID string, messageType string, egmID string, actionID string, actionStepID string, rawPayload string, parsedSummaryJSON string) (HandlerRuleMatchResult, error) {
	if !rule.Enabled {
		return HandlerRuleMatchResult{Matched: false, Reason: "rule disabled"}, nil
	}
	ruleDirection := normalizeHandlerRuleDirection(rule.Direction)
	if ruleDirection != HandlerRuleDirectionAny && !strings.EqualFold(string(ruleDirection), string(direction)) {
		return HandlerRuleMatchResult{Matched: false, Reason: "direction mismatch"}, nil
	}
	if strings.TrimSpace(rule.TemplateID) != "" && !strings.EqualFold(strings.TrimSpace(rule.TemplateID), strings.TrimSpace(templateID)) {
		return HandlerRuleMatchResult{Matched: false, Reason: "template mismatch"}, nil
	}
	if strings.TrimSpace(rule.MessageType) != "" && !strings.EqualFold(strings.TrimSpace(rule.MessageType), strings.TrimSpace(messageType)) {
		return HandlerRuleMatchResult{Matched: false, Reason: "message type mismatch"}, nil
	}
	if strings.TrimSpace(rule.EGMID) != "" && !strings.EqualFold(strings.TrimSpace(rule.EGMID), strings.TrimSpace(egmID)) {
		return HandlerRuleMatchResult{Matched: false, Reason: "egm mismatch"}, nil
	}
	if strings.TrimSpace(rule.ActionID) != "" && !strings.EqualFold(strings.TrimSpace(rule.ActionID), strings.TrimSpace(actionID)) {
		return HandlerRuleMatchResult{Matched: false, Reason: "action mismatch"}, nil
	}
	if strings.TrimSpace(rule.ActionStepID) != "" && !strings.EqualFold(strings.TrimSpace(rule.ActionStepID), strings.TrimSpace(actionStepID)) {
		return HandlerRuleMatchResult{Matched: false, Reason: "action step mismatch"}, nil
	}

	doc, err := ParseHandlerRuleMatchDocument(rule.MatchJSON)
	if err != nil {
		return HandlerRuleMatchResult{}, err
	}
	haystack := strings.ToLower(strings.TrimSpace(rawPayload))
	summary := strings.ToLower(strings.TrimSpace(parsedSummaryJSON))
	if summary != "" {
		if haystack == "" {
			haystack = summary
		} else {
			haystack = haystack + "\n" + summary
		}
	}

	required := append([]string{}, doc.AllContains...)
	required = append(required, doc.Contains...)
	for _, raw := range required {
		token := strings.ToLower(strings.TrimSpace(raw))
		if token == "" {
			continue
		}
		if !strings.Contains(haystack, token) {
			return HandlerRuleMatchResult{Matched: false, Reason: "required token missing"}, nil
		}
	}

	anyTokens := sanitizeTokens(doc.AnyContains)
	if len(anyTokens) > 0 {
		anyMatch := false
		for _, token := range anyTokens {
			if strings.Contains(haystack, token) {
				anyMatch = true
				break
			}
		}
		if !anyMatch {
			return HandlerRuleMatchResult{Matched: false, Reason: "any token missing"}, nil
		}
	}

	for _, token := range sanitizeTokens(doc.NotContains) {
		if strings.Contains(haystack, token) {
			return HandlerRuleMatchResult{Matched: false, Reason: "excluded token present"}, nil
		}
	}
	return HandlerRuleMatchResult{Matched: true, Reason: "rule matched"}, nil
}

func EvaluateHandlerRules(rules []HandlerRule, direction MessageDirection, templateID string, messageType string, egmID string, actionID string, actionStepID string, rawPayload string, parsedSummaryJSON string) (*HandlerRuleEvaluation, error) {
	matches := []HandlerRuleEvaluation{}
	for _, rule := range rules {
		match, err := MatchHandlerRule(rule, direction, templateID, messageType, egmID, actionID, actionStepID, rawPayload, parsedSummaryJSON)
		if err != nil {
			return nil, err
		}
		if !match.Matched {
			continue
		}
		matches = append(matches, HandlerRuleEvaluation{
			Rule:    rule,
			Outcome: normalizeHandlerRuleOutcome(rule.Outcome),
			Reason:  match.Reason,
		})
	}
	if len(matches) == 0 {
		return nil, nil
	}
	precedence := []HandlerRuleOutcome{
		HandlerRuleOutcomeFailure,
		HandlerRuleOutcomeConfirmation,
		HandlerRuleOutcomeIgnore,
		HandlerRuleOutcomeNote,
	}
	for _, outcome := range precedence {
		for _, matched := range matches {
			if matched.Outcome == outcome {
				value := matched
				return &value, nil
			}
		}
	}
	value := matches[0]
	return &value, nil
}
