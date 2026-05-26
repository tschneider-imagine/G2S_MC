package configvalidation

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actionplanner"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

const (
	StatusOK    = "OK"
	StatusWarn  = "WARN"
	StatusError = "ERROR"
)

type ItemResult struct {
	ID       string   `json:"id"`
	Name     string   `json:"name,omitempty"`
	Status   string   `json:"status"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

type Result struct {
	GeneratedAt time.Time    `json:"generated_at"`
	Status      string       `json:"status"`
	Actions     []ItemResult `json:"actions"`
	Templates   []ItemResult `json:"templates"`
	EGMs        []ItemResult `json:"egms"`
	Groups      []ItemResult `json:"groups"`
}

type Options struct {
	DeliveryTopology string
}

type Store interface {
	GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error)
	GetG2STemplateVersion(ctx context.Context, templateID string, version int) (*templates.G2STemplateVersion, error)
	GetActiveG2STemplateVersion(ctx context.Context, templateID string) (*templates.G2STemplateVersion, error)
	ListG2STemplates(ctx context.Context) ([]templates.G2STemplate, error)
	ListG2STemplateVersions(ctx context.Context, templateID string) ([]templates.G2STemplateVersion, error)

	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	ListActionDefinitions(ctx context.Context) ([]actions.ActionDefinition, error)
	ListEGMRecords(ctx context.Context) ([]egms.EGMRecord, error)
	GetEGMGroup(ctx context.Context, id string) (*egms.EGMGroup, error)
	ListEGMGroups(ctx context.Context) ([]egms.EGMGroup, error)
}

type Service struct {
	Store   Store
	Options Options
	Now     func() time.Time
}

func (s *Service) Validate(ctx context.Context) (Result, error) {
	if s.Store == nil {
		return Result{}, fmt.Errorf("store is required")
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}

	templatesList, err := s.Store.ListG2STemplates(ctx)
	if err != nil {
		return Result{}, err
	}
	actionsList, err := s.Store.ListActionDefinitions(ctx)
	if err != nil {
		return Result{}, err
	}
	egmList, err := s.Store.ListEGMRecords(ctx)
	if err != nil {
		return Result{}, err
	}
	groupList, err := s.Store.ListEGMGroups(ctx)
	if err != nil {
		return Result{}, err
	}

	templateByID := map[string]templates.G2STemplate{}
	activeVersionByTemplate := map[string]*templates.G2STemplateVersion{}
	actionKeysByTemplate := map[string]map[string]struct{}{}
	templateResults := make([]ItemResult, 0, len(templatesList))
	for _, row := range templatesList {
		templateByID[row.ID] = row
		item := ItemResult{ID: row.ID, Name: row.Name}
		if err := row.Validate(); err != nil {
			item.Errors = append(item.Errors, err.Error())
		}

		active, activeErr := s.Store.GetActiveG2STemplateVersion(ctx, row.ID)
		if activeErr != nil {
			return Result{}, activeErr
		}
		activeVersionByTemplate[row.ID] = active

		if row.Status == templates.TemplateStatusActive && active == nil {
			item.Errors = append(item.Errors, "active template requires an active version")
		}
		if active != nil {
			keys, keyErr := parseActionKeys(active.ActionsJSON)
			if keyErr != nil {
				item.Errors = append(item.Errors, "active template ActionsJSON is invalid")
			} else {
				actionKeysByTemplate[row.ID] = keys
			}
			if _, matcherErr := g2sengine.ParseMatcherDocument(active.ConfirmationRulesJSON); matcherErr != nil {
				item.Errors = append(item.Errors, "expected response matcher JSON is invalid")
			}
			if _, matcherErr := g2sengine.ParseMatcherDocument(active.FailureRulesJSON); matcherErr != nil {
				item.Errors = append(item.Errors, "failure matcher JSON is invalid")
			}
			if err := validateOptionalJSON(active.EndpointQuirksJSON); err != nil {
				item.Errors = append(item.Errors, "endpoint quirks JSON is invalid")
			}
		}

		versions, versionsErr := s.Store.ListG2STemplateVersions(ctx, row.ID)
		if versionsErr != nil {
			return Result{}, versionsErr
		}
		for _, version := range versions {
			if err := validateOptionalJSON(version.ActionsJSON); err != nil {
				item.Errors = append(item.Errors, "template version "+version.VersionLabel+" ActionsJSON is invalid")
			}
			if _, matcherErr := g2sengine.ParseMatcherDocument(version.ConfirmationRulesJSON); matcherErr != nil {
				item.Errors = append(item.Errors, "template version "+version.VersionLabel+" expected response matcher JSON is invalid")
			}
			if _, matcherErr := g2sengine.ParseMatcherDocument(version.FailureRulesJSON); matcherErr != nil {
				item.Errors = append(item.Errors, "template version "+version.VersionLabel+" failure matcher JSON is invalid")
			}
			if err := validateOptionalJSON(version.EndpointQuirksJSON); err != nil {
				item.Errors = append(item.Errors, "template version "+version.VersionLabel+" endpoint quirks JSON is invalid")
			}
		}

		item.Status = statusFromItem(item)
		templateResults = append(templateResults, item)
	}

	egmByID := map[string]egms.EGMRecord{}
	egmResults := make([]ItemResult, 0, len(egmList))
	topology, _ := g2stransport.NormalizeDeliveryTopology(s.Options.DeliveryTopology)
	for _, row := range egmList {
		egmByID[row.EGMID] = row
		item := ItemResult{ID: row.EGMID, Name: row.DisplayName}
		if err := row.Validate(); err != nil {
			item.Errors = append(item.Errors, err.Error())
		}
		if !row.Enabled && row.EmergencyEnabled {
			item.Warnings = append(item.Warnings, "Emergency participation requires Enabled")
		}
		templateID := strings.TrimSpace(row.TemplateID)
		if templateID == "" {
			item.Errors = append(item.Errors, "Missing Template")
		} else {
			tpl := templateByID[templateID]
			if strings.TrimSpace(tpl.ID) == "" {
				item.Errors = append(item.Errors, "assigned template not found")
			} else if activeVersionByTemplate[templateID] == nil {
				item.Errors = append(item.Errors, "assigned template has no active version")
			}
		}
		if topology == g2stransport.DeliveryTopologyOutboundEndpoint || topology == g2stransport.DeliveryTopologyCaptureEndpoint {
			if strings.TrimSpace(row.EndpointPath) == "" {
				item.Errors = append(item.Errors, "missing outbound endpoint")
			}
		}
		item.Status = statusFromItem(item)
		egmResults = append(egmResults, item)
	}

	groupByID := map[string]egms.EGMGroup{}
	groupResults := make([]ItemResult, 0, len(groupList))
	for _, row := range groupList {
		groupByID[row.ID] = row
		item := ItemResult{ID: row.ID, Name: row.Name}
		if err := row.Validate(); err != nil {
			item.Errors = append(item.Errors, err.Error())
		}
		if len(row.EGMIDs) == 0 {
			item.Warnings = append(item.Warnings, "group has no members")
		}
		unknown := []string{}
		for _, egmID := range row.EGMIDs {
			if _, ok := egmByID[strings.TrimSpace(egmID)]; !ok {
				unknown = append(unknown, strings.TrimSpace(egmID))
			}
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			item.Warnings = append(item.Warnings, "unknown group members: "+strings.Join(unknown, ", "))
		}
		item.Status = statusFromItem(item)
		groupResults = append(groupResults, item)
	}

	actionByID := map[string]actions.ActionDefinition{}
	for _, row := range actionsList {
		actionByID[row.ID] = row
	}
	planner := actionplanner.Planner{Store: plannerStoreAdapter{s.Store}}
	actionResults := make([]ItemResult, 0, len(actionsList))
	for _, row := range actionsList {
		item := ItemResult{ID: row.ID, Name: row.Name}
		if err := row.Validate(); err != nil {
			item.Errors = append(item.Errors, err.Error())
			item.Status = statusFromItem(item)
			actionResults = append(actionResults, item)
			continue
		}

		retry, retryErr := parseRetryPolicy(row.RetryPolicyJSON)
		if retryErr != nil {
			item.Errors = append(item.Errors, retryErr.Error())
		} else if retry.Count < 0 || retry.DelayMS < 0 {
			item.Errors = append(item.Errors, "retry policy values must be >= 0")
		}
		escalation, escalationErr := parseEscalationPolicy(row.EscalationJSON)
		if escalationErr != nil {
			item.Errors = append(item.Errors, escalationErr.Error())
		} else if strings.TrimSpace(escalation.ActionID) != "" {
			if _, ok := actionByID[escalation.ActionID]; !ok {
				item.Errors = append(item.Errors, "escalation action not found")
			}
		}
		if row.Severity == actions.SeverityEmergency && strings.TrimSpace(row.ReturnActionID) == "" {
			item.Errors = append(item.Errors, "Return Action is required for emergency action")
		}
		if strings.TrimSpace(row.ReturnActionID) != "" {
			if _, ok := actionByID[row.ReturnActionID]; !ok {
				item.Errors = append(item.Errors, "Return Action is not defined")
			}
		}

		plan, planErr := planner.BuildPlanForDefinition(ctx, row)
		if planErr != nil {
			item.Errors = append(item.Errors, planErr.Error())
		} else {
			for _, warning := range plan.Warnings {
				switch warning.Code {
				case "GROUP_EMPTY", "GROUP_NOT_FOUND", "SELECTOR_UNKNOWN", "GROUP_SELECTOR_INVALID", "ZONE_SELECTOR_INVALID":
					item.Errors = append(item.Errors, warning.Message)
				default:
					item.Warnings = append(item.Warnings, warning.Message)
				}
			}
			if row.Enabled && plan.TargetCount == 0 {
				item.Errors = append(item.Errors, "Target Selection resolved no EGMs")
			}
			for _, target := range plan.Targets {
				targetEGM := egmByID[target.EGMID]
				if row.Severity == actions.SeverityEmergency && !targetEGM.EmergencyEnabled {
					item.Errors = append(item.Errors, "target "+target.EGMID+" is not emergency-enabled")
				}
				templateID, templateErr := resolveTemplateForTarget(row.TemplateSelector, targetEGM)
				if templateErr != nil {
					item.Errors = append(item.Errors, "target "+target.EGMID+": "+templateErr.Error())
					continue
				}
				keys := actionKeysByTemplate[templateID]
				if len(keys) == 0 {
					if activeVersionByTemplate[templateID] == nil {
						item.Errors = append(item.Errors, "target "+target.EGMID+": template "+templateID+" has no active version")
					} else {
						item.Errors = append(item.Errors, "target "+target.EGMID+": template "+templateID+" has invalid action keys")
					}
					continue
				}
				for _, step := range row.Steps {
					if _, ok := keys[strings.TrimSpace(step.TemplateActionKey)]; !ok {
						item.Errors = append(item.Errors, "Missing Action Key "+strings.TrimSpace(step.TemplateActionKey)+" for template "+templateID+" target "+target.EGMID)
					}
				}
			}
		}

		if strings.HasPrefix(strings.TrimSpace(row.TargetSelector), actionplanner.SelectorGroupPrefix) {
			groupID := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(row.TargetSelector), actionplanner.SelectorGroupPrefix))
			group := groupByID[groupID]
			if strings.TrimSpace(group.ID) == "" {
				item.Errors = append(item.Errors, "Target Selection group not found: "+groupID)
			} else if len(group.EGMIDs) == 0 {
				item.Errors = append(item.Errors, "Target Selection group has no members: "+groupID)
			}
		}

		item.Status = statusFromItem(item)
		actionResults = append(actionResults, item)
	}

	result := Result{
		GeneratedAt: now,
		Actions:     actionResults,
		Templates:   templateResults,
		EGMs:        egmResults,
		Groups:      groupResults,
	}
	result.Status = StatusOK
	for _, row := range actionResults {
		result.Status = mergeStatus(result.Status, row.Status)
	}
	for _, row := range templateResults {
		result.Status = mergeStatus(result.Status, row.Status)
	}
	for _, row := range egmResults {
		result.Status = mergeStatus(result.Status, row.Status)
	}
	for _, row := range groupResults {
		result.Status = mergeStatus(result.Status, row.Status)
	}
	return result, nil
}

func statusFromItem(item ItemResult) string {
	if len(item.Errors) > 0 {
		return StatusError
	}
	if len(item.Warnings) > 0 {
		return StatusWarn
	}
	return StatusOK
}

func mergeStatus(current string, next string) string {
	if current == StatusError || next == StatusError {
		return StatusError
	}
	if current == StatusWarn || next == StatusWarn {
		return StatusWarn
	}
	return StatusOK
}

func parseActionKeys(raw string) (map[string]struct{}, error) {
	doc, err := g2sengine.ParseActionTemplateDocument(strings.TrimSpace(raw))
	if err != nil {
		return nil, err
	}
	keys := map[string]struct{}{}
	for key := range doc.Actions {
		keys[strings.TrimSpace(key)] = struct{}{}
	}
	return keys, nil
}

func validateOptionalJSON(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var payload any
	return json.Unmarshal([]byte(trimmed), &payload)
}

type retryPolicy struct {
	Count   int `json:"count"`
	DelayMS int `json:"delay_ms"`
}

func parseRetryPolicy(raw string) (retryPolicy, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return retryPolicy{}, nil
	}
	var policy retryPolicy
	if err := json.Unmarshal([]byte(trimmed), &policy); err != nil {
		return retryPolicy{}, fmt.Errorf("retry policy JSON is invalid")
	}
	if policy.Count < 0 || policy.DelayMS < 0 {
		return retryPolicy{}, fmt.Errorf("retry policy values must be >= 0")
	}
	return policy, nil
}

type escalationPolicy struct {
	ActionID      string `json:"escalation_action_id"`
	AfterAttempts int    `json:"after_attempts"`
}

func parseEscalationPolicy(raw string) (escalationPolicy, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return escalationPolicy{}, nil
	}
	var policy escalationPolicy
	if err := json.Unmarshal([]byte(trimmed), &policy); err != nil {
		return escalationPolicy{}, fmt.Errorf("escalation policy JSON is invalid")
	}
	policy.ActionID = strings.TrimSpace(policy.ActionID)
	if policy.AfterAttempts < 0 {
		return escalationPolicy{}, fmt.Errorf("escalation after attempts must be >= 0")
	}
	if policy.ActionID == "" && policy.AfterAttempts > 0 {
		return escalationPolicy{}, fmt.Errorf("escalation action is required when escalation attempts are set")
	}
	if policy.ActionID != "" && policy.AfterAttempts == 0 {
		return escalationPolicy{}, fmt.Errorf("escalation after attempts is required when escalation action is set")
	}
	return policy, nil
}

func resolveTemplateForTarget(templateSelector string, target egms.EGMRecord) (string, error) {
	selector := strings.TrimSpace(templateSelector)
	if selector == "" {
		return "", fmt.Errorf("template selector is required")
	}
	if strings.EqualFold(selector, "template-by-egm") {
		id := strings.TrimSpace(target.TemplateID)
		if id == "" {
			return "", fmt.Errorf("Missing Template")
		}
		return id, nil
	}
	if strings.HasPrefix(selector, actionplanner.SelectorTemplatePrefix) {
		id := strings.TrimSpace(strings.TrimPrefix(selector, actionplanner.SelectorTemplatePrefix))
		if id == "" {
			return "", fmt.Errorf("template selector is missing template id")
		}
		return id, nil
	}
	if strings.HasPrefix(selector, "TEMPLATE_VERSION:") {
		return "", fmt.Errorf("template selector TEMPLATE_VERSION is not supported")
	}
	if strings.Contains(selector, ":") {
		return "", fmt.Errorf("unsupported template selector %s", selector)
	}
	return selector, nil
}

type plannerStoreAdapter struct {
	store Store
}

func (p plannerStoreAdapter) GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error) {
	return p.store.GetActionDefinition(ctx, id)
}

func (p plannerStoreAdapter) ListEGMRecords(ctx context.Context) ([]egms.EGMRecord, error) {
	return p.store.ListEGMRecords(ctx)
}

func (p plannerStoreAdapter) GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error) {
	return p.store.GetG2STemplate(ctx, id)
}

func (p plannerStoreAdapter) GetEGMGroup(ctx context.Context, id string) (*egms.EGMGroup, error) {
	return p.store.GetEGMGroup(ctx, id)
}

func (p plannerStoreAdapter) ListEGMGroups(ctx context.Context) ([]egms.EGMGroup, error) {
	return p.store.ListEGMGroups(ctx)
}

func activeTemplateVersionInUse(ctx context.Context, st Store, templateID string, versionLabel string) (bool, error) {
	egmRows, err := st.ListEGMRecords(ctx)
	if err != nil {
		return false, err
	}
	for _, row := range egmRows {
		if strings.TrimSpace(row.TemplateID) == strings.TrimSpace(templateID) {
			return true, nil
		}
	}
	actionRows, err := st.ListActionDefinitions(ctx)
	if err != nil {
		return false, err
	}
	for _, row := range actionRows {
		selector := strings.TrimSpace(row.TemplateSelector)
		if strings.EqualFold(selector, "template-by-egm") {
			return true, nil
		}
		if strings.EqualFold(selector, strings.TrimSpace(templateID)) {
			return true, nil
		}
		if strings.HasPrefix(selector, actionplanner.SelectorTemplatePrefix) {
			selected := strings.TrimSpace(strings.TrimPrefix(selector, actionplanner.SelectorTemplatePrefix))
			if strings.EqualFold(selected, strings.TrimSpace(templateID)) {
				return true, nil
			}
		}
	}
	return false, nil
}

func ActiveTemplateVersionInUse(ctx context.Context, st Store, templateID string, versionLabel string) (bool, error) {
	return activeTemplateVersionInUse(ctx, st, templateID, versionLabel)
}
