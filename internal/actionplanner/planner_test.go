package actionplanner

import (
	"context"
	"testing"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type mockStore struct {
	action    *actions.ActionDefinition
	records   []egms.EGMRecord
	templates map[string]templates.G2STemplate
	groups    map[string]egms.EGMGroup
}

func (m *mockStore) GetActionDefinition(_ context.Context, _ string) (*actions.ActionDefinition, error) {
	if m.action == nil {
		return nil, nil
	}
	copy := *m.action
	return &copy, nil
}

func (m *mockStore) ListEGMRecords(_ context.Context) ([]egms.EGMRecord, error) {
	result := make([]egms.EGMRecord, 0, len(m.records))
	result = append(result, m.records...)
	return result, nil
}

func (m *mockStore) GetG2STemplate(_ context.Context, id string) (*templates.G2STemplate, error) {
	row, ok := m.templates[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (m *mockStore) GetEGMGroup(_ context.Context, id string) (*egms.EGMGroup, error) {
	row, ok := m.groups[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (m *mockStore) ListEGMGroups(_ context.Context) ([]egms.EGMGroup, error) {
	rows := make([]egms.EGMGroup, 0, len(m.groups))
	for _, row := range m.groups {
		rows = append(rows, row)
	}
	return rows, nil
}

func TestPlannerAllEmergencyEnabledSelector(t *testing.T) {
	store := baseMockStore()
	store.action.TargetSelector = SelectorAllEmergencyEnabled
	planner := &Planner{Store: store}

	plan, err := planner.BuildPlan(context.Background(), store.action.ID)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.TargetCount != 2 {
		t.Fatalf("target count = %d, want 2", plan.TargetCount)
	}
	if plan.Targets[0].EGMID != "EGM-001" || plan.Targets[1].EGMID != "EGM-003" {
		t.Fatalf("unexpected targets: %+v", plan.Targets)
	}
}

func TestPlannerExplicitEGMIDSelector(t *testing.T) {
	store := baseMockStore()
	store.action.TargetSelector = "EGM_IDS:EGM-003,EGM-001"
	planner := &Planner{Store: store}

	plan, err := planner.BuildPlan(context.Background(), store.action.ID)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.TargetCount != 2 {
		t.Fatalf("target count = %d, want 2", plan.TargetCount)
	}
	if plan.Targets[0].EGMID != "EGM-001" || plan.Targets[1].EGMID != "EGM-003" {
		t.Fatalf("expected deterministic sorted targets, got %+v", plan.Targets)
	}
}

func TestPlannerTemplateSelector(t *testing.T) {
	store := baseMockStore()
	store.action.TargetSelector = "TEMPLATE:tpl-b"
	planner := &Planner{Store: store}

	plan, err := planner.BuildPlan(context.Background(), store.action.ID)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.TargetCount != 1 || plan.Targets[0].EGMID != "EGM-003" {
		t.Fatalf("unexpected template selector targets: %+v", plan.Targets)
	}
}

func TestPlannerDisabledEGMExcluded(t *testing.T) {
	store := baseMockStore()
	store.action.TargetSelector = "EGM_IDS:EGM-002"
	planner := &Planner{Store: store}

	plan, err := planner.BuildPlan(context.Background(), store.action.ID)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.TargetCount != 0 {
		t.Fatalf("expected disabled egm excluded, got %+v", plan.Targets)
	}
}

func TestPlannerMissingTemplateWarning(t *testing.T) {
	store := baseMockStore()
	store.action.TargetSelector = "EGM_IDS:EGM-004"
	planner := &Planner{Store: store}

	plan, err := planner.BuildPlan(context.Background(), store.action.ID)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.TargetCount != 1 {
		t.Fatalf("target count = %d, want 1", plan.TargetCount)
	}
	if len(plan.Warnings) == 0 {
		t.Fatal("expected missing template warning")
	}
}

func TestPlannerEmptyTargetWarning(t *testing.T) {
	store := baseMockStore()
	store.action.TargetSelector = "EGM_IDS:DOES-NOT-EXIST"
	planner := &Planner{Store: store}

	plan, err := planner.BuildPlan(context.Background(), store.action.ID)
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.TargetCount != 0 {
		t.Fatalf("target count = %d, want 0", plan.TargetCount)
	}
	if len(plan.Warnings) == 0 || plan.Warnings[0].Code != "EMPTY_TARGET_SET" {
		t.Fatalf("expected empty target warning, got %+v", plan.Warnings)
	}
}

func TestPlannerDeterministicSort(t *testing.T) {
	store := baseMockStore()
	store.action.TargetSelector = "EGM_IDS:EGM-003,EGM-001"
	planner := &Planner{Store: store}

	plan1, err := planner.BuildPlan(context.Background(), store.action.ID)
	if err != nil {
		t.Fatalf("build plan1: %v", err)
	}
	plan2, err := planner.BuildPlan(context.Background(), store.action.ID)
	if err != nil {
		t.Fatalf("build plan2: %v", err)
	}
	if plan1.Targets[0].EGMID != plan2.Targets[0].EGMID || plan1.Targets[1].EGMID != plan2.Targets[1].EGMID {
		t.Fatalf("plans are not deterministic: plan1=%+v plan2=%+v", plan1.Targets, plan2.Targets)
	}
}

func baseMockStore() *mockStore {
	return &mockStore{
		action: &actions.ActionDefinition{
			ID:               "action-1",
			Name:             "Emergency Silence",
			Severity:         actions.SeverityEmergency,
			Enabled:          true,
			TargetSelector:   SelectorAllEmergencyEnabled,
			TemplateSelector: "template-by-egm",
			Steps: []actions.ActionStep{{
				ID:                "step-1",
				Name:              "Send mute",
				Sequence:          0,
				TemplateActionKey: "mute_primary",
			}},
			Version: 1,
		},
		records: []egms.EGMRecord{
			{EGMID: "EGM-003", Enabled: true, EmergencyEnabled: true, TemplateID: "tpl-b", CurrentActionState: egms.EGMActionStateNormal},
			{EGMID: "EGM-001", Enabled: true, EmergencyEnabled: true, TemplateID: "tpl-a", CurrentActionState: egms.EGMActionStateNormal},
			{EGMID: "EGM-002", Enabled: false, EmergencyEnabled: true, TemplateID: "tpl-a", CurrentActionState: egms.EGMActionStateNormal},
			{EGMID: "EGM-004", Enabled: true, EmergencyEnabled: false, TemplateID: "", CurrentActionState: egms.EGMActionStateNormal},
		},
		templates: map[string]templates.G2STemplate{
			"tpl-a": {ID: "tpl-a", Name: "A", Vendor: "IGT", Status: templates.TemplateStatusActive},
			"tpl-b": {ID: "tpl-b", Name: "B", Vendor: "Bally", Status: templates.TemplateStatusActive},
		},
		groups: map[string]egms.EGMGroup{
			"zone-a": {ID: "zone-a", Name: "Zone A"},
		},
	}
}
