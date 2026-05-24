package actionruntime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type fakeStore struct {
	actionDefs    map[string]actions.ActionDefinition
	egmRecords    []egms.EGMRecord
	templatesByID map[string]templates.G2STemplate
	groupsByID    map[string]egms.EGMGroup
	actionRuns    []actions.ActionRun
	targetResults []actions.ActionTargetResult
	auditEntries  []audit.AuditTimelineEntry
	nextTargetID  int64
	nextAuditID   int64
}

func (f *fakeStore) GetActionDefinition(_ context.Context, id string) (*actions.ActionDefinition, error) {
	row, ok := f.actionDefs[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) ListEGMRecords(_ context.Context) ([]egms.EGMRecord, error) {
	rows := make([]egms.EGMRecord, 0, len(f.egmRecords))
	rows = append(rows, f.egmRecords...)
	return rows, nil
}

func (f *fakeStore) GetG2STemplate(_ context.Context, id string) (*templates.G2STemplate, error) {
	row, ok := f.templatesByID[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) GetEGMGroup(_ context.Context, id string) (*egms.EGMGroup, error) {
	row, ok := f.groupsByID[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) ListEGMGroups(_ context.Context) ([]egms.EGMGroup, error) {
	rows := make([]egms.EGMGroup, 0, len(f.groupsByID))
	for _, row := range f.groupsByID {
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *fakeStore) CreateActionRun(_ context.Context, run actions.ActionRun) (actions.ActionRun, error) {
	f.actionRuns = append(f.actionRuns, run)
	return run, nil
}

func (f *fakeStore) CreateActionTargetResult(_ context.Context, result actions.ActionTargetResult) (actions.ActionTargetResult, error) {
	f.nextTargetID++
	result.ID = f.nextTargetID
	f.targetResults = append(f.targetResults, result)
	return result, nil
}

func (f *fakeStore) RecordAuditTimelineEntry(_ context.Context, entry audit.AuditTimelineEntry) (int64, error) {
	f.nextAuditID++
	entry.ID = f.nextAuditID
	f.auditEntries = append(f.auditEntries, entry)
	return entry.ID, nil
}

func TestQueueActionRunBlankActionID(t *testing.T) {
	store := newFakeStore()
	queuer := &Queuer{Store: store}

	result, err := queuer.QueueActionRun(context.Background(), QueueRequest{})
	if err != nil {
		t.Fatalf("queue action run: %v", err)
	}
	if result.Queued {
		t.Fatal("expected queued=false for blank action id")
	}
	if result.Reason != "no action id" {
		t.Fatalf("reason=%q, want %q", result.Reason, "no action id")
	}
	if len(store.actionRuns) != 0 {
		t.Fatalf("unexpected action runs: %+v", store.actionRuns)
	}
}

func TestQueueActionRunMissingAction(t *testing.T) {
	store := newFakeStore()
	queuer := &Queuer{Store: store}

	result, err := queuer.QueueActionRun(context.Background(), QueueRequest{
		ActionID: "missing",
		QueuedAt: time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC),
		InputTransition: inputs.InputTransition{
			ID:              7,
			InputChannelID:  "emergency-broadcast",
			PreviousDerived: inputs.DerivedStateNormal,
			NewDerived:      inputs.DerivedStateTriggered,
			TransitionAt:    time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("queue action run: %v", err)
	}
	if result.Queued {
		t.Fatal("expected queued=false when action definition is missing")
	}
	if len(store.auditEntries) != 1 {
		t.Fatalf("expected one audit entry, got %d", len(store.auditEntries))
	}
	if store.auditEntries[0].EventType != audit.EventTypeSystemWarning {
		t.Fatalf("event type=%q, want %q", store.auditEntries[0].EventType, audit.EventTypeSystemWarning)
	}
}

func TestQueueActionRunCreatesPendingRunAndTargets(t *testing.T) {
	store := newFakeStore()
	queuer := &Queuer{Store: store}
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	result, err := queuer.QueueActionRun(context.Background(), QueueRequest{
		ActionID:      "emergency-broadcast-trigger",
		TriggerReason: "input transition 11",
		QueuedAt:      now,
		Actor:         "monitor",
		InputTransition: inputs.InputTransition{
			ID:              11,
			InputChannelID:  "emergency-broadcast",
			PreviousDerived: inputs.DerivedStateNormal,
			NewDerived:      inputs.DerivedStateTriggered,
			TransitionAt:    now,
		},
	})
	if err != nil {
		t.Fatalf("queue action run: %v", err)
	}
	if !result.Queued {
		t.Fatalf("expected queued=true, result=%+v", result)
	}
	if result.ActionRun == nil {
		t.Fatal("expected action run")
	}
	if result.ActionRun.Status != actions.RunStatusPending {
		t.Fatalf("run status=%q, want %q", result.ActionRun.Status, actions.RunStatusPending)
	}
	if result.ActionRun.TargetCount != 2 {
		t.Fatalf("target count=%d, want 2", result.ActionRun.TargetCount)
	}
	if len(result.TargetResults) != 2 {
		t.Fatalf("target rows=%d, want 2", len(result.TargetResults))
	}
	for _, row := range result.TargetResults {
		if row.Status != actions.TargetStatusPending {
			t.Fatalf("target status=%q, want %q", row.Status, actions.TargetStatusPending)
		}
		if row.AttemptCount != 0 {
			t.Fatalf("attempt_count=%d, want 0", row.AttemptCount)
		}
	}
	if len(store.auditEntries) != 1 {
		t.Fatalf("expected one audit entry, got %d", len(store.auditEntries))
	}
	entry := store.auditEntries[0]
	if entry.EventType != audit.EventTypeActionQueued {
		t.Fatalf("event type=%q, want %q", entry.EventType, audit.EventTypeActionQueued)
	}
	if entry.Severity != audit.AuditSeverityEmergency {
		t.Fatalf("severity=%q, want %q", entry.Severity, audit.AuditSeverityEmergency)
	}
	if entry.ActionRunID == "" || entry.ActionRunID != result.ActionRun.ID {
		t.Fatalf("audit action_run_id=%q, run id=%q", entry.ActionRunID, result.ActionRun.ID)
	}
}

func TestQueueActionRunPreservesPlanWarningsInAuditMetadata(t *testing.T) {
	store := newFakeStore()
	queuer := &Queuer{Store: store}
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	result, err := queuer.QueueActionRun(context.Background(), QueueRequest{
		ActionID: "missing-template-trigger",
		QueuedAt: now,
		InputTransition: inputs.InputTransition{
			ID:              44,
			InputChannelID:  "general-broadcast",
			PreviousDerived: inputs.DerivedStateNormal,
			NewDerived:      inputs.DerivedStateTriggered,
			TransitionAt:    now,
		},
	})
	if err != nil {
		t.Fatalf("queue action run: %v", err)
	}
	if !result.Queued {
		t.Fatalf("expected queued=true, got %+v", result)
	}
	if len(result.PlanWarnings) == 0 {
		t.Fatal("expected plan warnings")
	}

	if len(store.auditEntries) != 1 {
		t.Fatalf("audit entries=%d, want 1", len(store.auditEntries))
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(store.auditEntries[0].DetailJSON), &metadata); err != nil {
		t.Fatalf("decode detail json: %v", err)
	}
	rawWarnings, ok := metadata["plan_warnings"].([]any)
	if !ok || len(rawWarnings) == 0 {
		t.Fatalf("expected plan_warnings in metadata, got %+v", metadata)
	}
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		actionDefs: map[string]actions.ActionDefinition{
			"emergency-broadcast-trigger": {
				ID:               "emergency-broadcast-trigger",
				Name:             "Emergency Broadcast Trigger",
				Severity:         actions.SeverityEmergency,
				Enabled:          true,
				TargetSelector:   "ALL_EMERGENCY_ENABLED",
				TemplateSelector: "template-by-egm",
				Steps: []actions.ActionStep{{
					ID:                "step-1",
					Name:              "Queue only",
					Sequence:          0,
					TemplateActionKey: "queue_only_no_send",
				}},
				Version: 1,
			},
			"missing-template-trigger": {
				ID:               "missing-template-trigger",
				Name:             "Missing Template Trigger",
				Severity:         actions.SeverityBroadcast,
				Enabled:          true,
				TargetSelector:   "EGM_IDS:EGM-004",
				TemplateSelector: "template-by-egm",
				Steps: []actions.ActionStep{{
					ID:                "step-1",
					Name:              "Queue only",
					Sequence:          0,
					TemplateActionKey: "queue_only_no_send",
				}},
				Version: 1,
			},
		},
		egmRecords: []egms.EGMRecord{
			{EGMID: "EGM-003", Enabled: true, EmergencyEnabled: true, TemplateID: "tpl-b", CurrentActionState: egms.EGMActionStateNormal},
			{EGMID: "EGM-001", Enabled: true, EmergencyEnabled: true, TemplateID: "tpl-a", CurrentActionState: egms.EGMActionStateNormal},
			{EGMID: "EGM-002", Enabled: false, EmergencyEnabled: true, TemplateID: "tpl-a", CurrentActionState: egms.EGMActionStateNormal},
			{EGMID: "EGM-004", Enabled: true, EmergencyEnabled: false, TemplateID: "", CurrentActionState: egms.EGMActionStateNormal},
		},
		templatesByID: map[string]templates.G2STemplate{
			"tpl-a": {ID: "tpl-a", Name: "A", Vendor: "IGT", Status: templates.TemplateStatusActive},
			"tpl-b": {ID: "tpl-b", Name: "B", Vendor: "Bally", Status: templates.TemplateStatusActive},
		},
		groupsByID:   map[string]egms.EGMGroup{},
		nextTargetID: 0,
		nextAuditID:  0,
	}
}
