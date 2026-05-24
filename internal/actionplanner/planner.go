package actionplanner

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type Store interface {
	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	ListEGMRecords(ctx context.Context) ([]egms.EGMRecord, error)
	GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error)
	GetEGMGroup(ctx context.Context, id string) (*egms.EGMGroup, error)
	ListEGMGroups(ctx context.Context) ([]egms.EGMGroup, error)
}

type Planner struct {
	Store Store
}

func (p *Planner) BuildPlan(ctx context.Context, actionID string) (*ActionPlan, error) {
	if p.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	id := strings.TrimSpace(actionID)
	if id == "" {
		return nil, fmt.Errorf("action id is required")
	}
	definition, err := p.Store.GetActionDefinition(ctx, id)
	if err != nil {
		return nil, err
	}
	if definition == nil {
		return nil, fmt.Errorf("action %q not found", id)
	}
	return p.BuildPlanForDefinition(ctx, *definition)
}

func (p *Planner) BuildPlanForDefinition(ctx context.Context, definition actions.ActionDefinition) (*ActionPlan, error) {
	if err := definition.Validate(); err != nil {
		return nil, err
	}
	records, err := p.Store.ListEGMRecords(ctx)
	if err != nil {
		return nil, err
	}
	selected, warnings, err := p.selectTargets(ctx, definition.TargetSelector, records)
	if err != nil {
		return nil, err
	}

	targets := make([]ActionPlanTarget, 0, len(selected))
	for _, record := range selected {
		target := ActionPlanTarget{
			EGMID:        record.EGMID,
			DisplayName:  record.DisplayName,
			TemplateID:   record.TemplateID,
			IPAddress:    record.IPAddress,
			EndpointPath: record.EndpointPath,
		}
		if strings.TrimSpace(record.TemplateID) == "" {
			target.MissingTemplate = true
			warnings = append(warnings, PlanningWarning{
				Code:    "MISSING_TEMPLATE",
				Message: fmt.Sprintf("EGM %s has no assigned template", record.EGMID),
			})
		} else {
			tpl, err := p.Store.GetG2STemplate(ctx, record.TemplateID)
			if err != nil {
				return nil, err
			}
			if tpl == nil {
				target.MissingTemplate = true
				warnings = append(warnings, PlanningWarning{
					Code:    "MISSING_TEMPLATE",
					Message: fmt.Sprintf("EGM %s references unknown template %s", record.EGMID, record.TemplateID),
				})
			}
		}
		targets = append(targets, target)
	}

	if len(targets) == 0 {
		warnings = append(warnings, PlanningWarning{Code: "EMPTY_TARGET_SET", Message: "No eligible EGM targets found for selector"})
	}

	sort.Slice(targets, func(i, j int) bool {
		return targets[i].EGMID < targets[j].EGMID
	})

	return &ActionPlan{
		ActionID:    definition.ID,
		ActionName:  definition.Name,
		Version:     definition.Version,
		TargetCount: len(targets),
		Targets:     targets,
		Steps:       planStepsFromDefinition(definition),
		Warnings:    dedupeWarnings(warnings),
	}, nil
}

func (p *Planner) selectTargets(ctx context.Context, selectorRaw string, records []egms.EGMRecord) ([]egms.EGMRecord, []PlanningWarning, error) {
	selector := strings.TrimSpace(selectorRaw)
	if selector == SelectorAllEmergencyEnabled {
		return selectAllEmergencyEnabled(records), nil, nil
	}
	if strings.HasPrefix(selector, SelectorEGMIDsPrefix) {
		ids := splitSelectorCSV(strings.TrimPrefix(selector, SelectorEGMIDsPrefix))
		return selectByEGMIDs(records, ids), nil, nil
	}
	if strings.HasPrefix(selector, SelectorTemplatePrefix) {
		templateID := strings.TrimSpace(strings.TrimPrefix(selector, SelectorTemplatePrefix))
		return selectByTemplate(records, templateID), nil, nil
	}
	if strings.HasPrefix(selector, SelectorGroupPrefix) {
		groupID := strings.TrimSpace(strings.TrimPrefix(selector, SelectorGroupPrefix))
		if groupID == "" {
			return nil, []PlanningWarning{{Code: "GROUP_SELECTOR_INVALID", Message: "GROUP selector is missing a group id"}}, nil
		}
		group, err := p.Store.GetEGMGroup(ctx, groupID)
		if err != nil {
			return nil, nil, err
		}
		if group == nil {
			return nil, []PlanningWarning{{Code: "GROUP_NOT_FOUND", Message: fmt.Sprintf("Group %s not found", groupID)}}, nil
		}
		if len(group.EGMIDs) == 0 {
			return nil, []PlanningWarning{{Code: "GROUP_EMPTY", Message: fmt.Sprintf("Group %s has no EGM membership", groupID)}}, nil
		}
		return selectByEGMIDs(records, group.EGMIDs), nil, nil
	}
	if strings.HasPrefix(selector, SelectorZonePrefix) {
		zoneID := strings.TrimSpace(strings.TrimPrefix(selector, SelectorZonePrefix))
		if zoneID == "" {
			return nil, []PlanningWarning{{Code: "ZONE_SELECTOR_INVALID", Message: "ZONE selector is missing a zone id"}}, nil
		}
		return selectByZone(records, zoneID), nil, nil
	}
	return nil, []PlanningWarning{{Code: "SELECTOR_UNKNOWN", Message: fmt.Sprintf("Unknown target selector: %s", selector)}}, nil
}

func selectAllEmergencyEnabled(records []egms.EGMRecord) []egms.EGMRecord {
	selected := []egms.EGMRecord{}
	for _, record := range records {
		if !record.Enabled || !record.EmergencyEnabled {
			continue
		}
		selected = append(selected, record)
	}
	return selected
}

func selectByEGMIDs(records []egms.EGMRecord, ids []string) []egms.EGMRecord {
	allowed := map[string]struct{}{}
	for _, id := range ids {
		allowed[id] = struct{}{}
	}
	selected := []egms.EGMRecord{}
	for _, record := range records {
		if !record.Enabled {
			continue
		}
		if _, ok := allowed[record.EGMID]; ok {
			selected = append(selected, record)
		}
	}
	return selected
}

func selectByTemplate(records []egms.EGMRecord, templateID string) []egms.EGMRecord {
	selected := []egms.EGMRecord{}
	for _, record := range records {
		if !record.Enabled {
			continue
		}
		if record.TemplateID == templateID {
			selected = append(selected, record)
		}
	}
	return selected
}

func selectByZone(records []egms.EGMRecord, zoneID string) []egms.EGMRecord {
	selected := []egms.EGMRecord{}
	for _, record := range records {
		if !record.Enabled {
			continue
		}
		if strings.TrimSpace(record.Zone) == zoneID {
			selected = append(selected, record)
		}
	}
	return selected
}

func splitSelectorCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func dedupeWarnings(warnings []PlanningWarning) []PlanningWarning {
	seen := map[string]struct{}{}
	result := make([]PlanningWarning, 0, len(warnings))
	for _, warning := range warnings {
		key := warning.Code + "::" + warning.Message
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, warning)
	}
	return result
}
