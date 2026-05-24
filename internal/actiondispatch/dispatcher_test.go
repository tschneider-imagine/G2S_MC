package actiondispatch

import (
	"context"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type fakeStore struct {
	runs       map[string]actions.ActionRun
	defs       map[string]actions.ActionDefinition
	targetRows map[string][]actions.ActionTargetResult
	egmsByID   map[string]egms.EGMRecord
	templates  map[string]templates.G2STemplate

	messages []g2sengine.MessageJournalEntry
	audits   []audit.AuditTimelineEntry
}

func (f *fakeStore) GetActionRun(_ context.Context, id string) (*actions.ActionRun, error) {
	row, ok := f.runs[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) GetActionDefinition(_ context.Context, id string) (*actions.ActionDefinition, error) {
	row, ok := f.defs[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) ListActionTargetResults(_ context.Context, actionRunID string) ([]actions.ActionTargetResult, error) {
	rows := make([]actions.ActionTargetResult, 0, len(f.targetRows[actionRunID]))
	rows = append(rows, f.targetRows[actionRunID]...)
	return rows, nil
}

func (f *fakeStore) GetEGMRecord(_ context.Context, egmID string) (*egms.EGMRecord, error) {
	row, ok := f.egmsByID[egmID]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) GetG2STemplate(_ context.Context, id string) (*templates.G2STemplate, error) {
	row, ok := f.templates[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) UpdateActionRun(_ context.Context, run actions.ActionRun) error {
	f.runs[run.ID] = run
	return nil
}

func (f *fakeStore) RecordMessageJournalEntry(_ context.Context, entry g2sengine.MessageJournalEntry) (int64, error) {
	entry.ID = int64(len(f.messages) + 1)
	f.messages = append(f.messages, entry)
	return entry.ID, nil
}

func (f *fakeStore) RecordAuditTimelineEntry(_ context.Context, entry audit.AuditTimelineEntry) (int64, error) {
	entry.ID = int64(len(f.audits) + 1)
	f.audits = append(f.audits, entry)
	return entry.ID, nil
}

func TestDispatchRejectsInvalidMode(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	dispatcher := &Dispatcher{Store: st}

	_, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "run-1",
		Mode:        DispatchMode("SEND"),
	})
	if err == nil {
		t.Fatal("expected invalid mode error")
	}
}

func TestDispatchRejectsBlankMode(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	dispatcher := &Dispatcher{Store: st}

	_, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "run-1",
	})
	if err == nil {
		t.Fatal("expected blank mode error")
	}
}

func TestDispatchMissingRun(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	dispatcher := &Dispatcher{Store: st}

	_, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "missing",
		Mode:        DispatchModeDryRun,
	})
	if err == nil {
		t.Fatal("expected missing run error")
	}
}

func TestDispatchRejectsNonPendingRun(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	row := st.runs["run-1"]
	row.Status = actions.RunStatusDispatchPrepared
	st.runs["run-1"] = row

	dispatcher := &Dispatcher{Store: st}
	_, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "run-1",
		Mode:        DispatchModeDryRun,
	})
	if err == nil {
		t.Fatal("expected non-pending run status error")
	}
}

func TestDispatchDryRunUpdatesRunAndRecordsMessages(t *testing.T) {
	now := time.Date(2026, 5, 24, 13, 0, 0, 0, time.UTC)
	st := newFakeStore(now)
	dispatcher := &Dispatcher{Store: st}

	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "run-1",
		Mode:        DispatchModeDryRun,
		Actor:       "test",
		RequestedAt: now,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if result.ActionRunID != "run-1" || result.Mode != DispatchModeDryRun {
		t.Fatalf("unexpected dispatch result: %+v", result)
	}
	if len(result.PreparedMessages) != 2 {
		t.Fatalf("prepared messages=%d, want 2", len(result.PreparedMessages))
	}
	for _, row := range result.PreparedMessages {
		if row.Result != g2sengine.MessageResultDryRun {
			t.Fatalf("message result=%q, want %q", row.Result, g2sengine.MessageResultDryRun)
		}
		if row.Direction != g2sengine.DirectionOutbound {
			t.Fatalf("direction=%q, want %q", row.Direction, g2sengine.DirectionOutbound)
		}
	}
	updated := st.runs["run-1"]
	if updated.Status != actions.RunStatusDispatchPrepared {
		t.Fatalf("run status=%q, want %q", updated.Status, actions.RunStatusDispatchPrepared)
	}
	if len(st.audits) != 1 || st.audits[0].EventType != audit.EventTypeActionDispatchPrepared {
		t.Fatalf("unexpected audit entries: %+v", st.audits)
	}
}

func TestDispatchNoTargetsStillRecordsAudit(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	st.targetRows["run-1"] = []actions.ActionTargetResult{}
	dispatcher := &Dispatcher{Store: st}

	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "run-1",
		Mode:        DispatchModeDryRun,
		RequestedAt: now,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if len(result.PreparedMessages) != 0 {
		t.Fatalf("prepared messages=%d, want 0", len(result.PreparedMessages))
	}
	if result.WarningCount == 0 {
		t.Fatalf("warning count=%d, want >0", result.WarningCount)
	}
	if len(st.audits) != 1 {
		t.Fatalf("audit entries=%d, want 1", len(st.audits))
	}
}

func newFakeStore(now time.Time) *fakeStore {
	return &fakeStore{
		runs: map[string]actions.ActionRun{
			"run-1": {
				ID:                 "run-1",
				ActionDefinitionID: "action-1",
				InputTransitionID:  9,
				StartedAt:          now,
				Status:             actions.RunStatusPending,
				TriggerReason:      "input transition 9",
				TargetCount:        2,
			},
		},
		defs: map[string]actions.ActionDefinition{
			"action-1": {
				ID:               "action-1",
				Name:             "Emergency",
				Severity:         actions.SeverityEmergency,
				Enabled:          true,
				TargetSelector:   "ALL_EMERGENCY_ENABLED",
				TemplateSelector: "template-by-egm",
				Steps: []actions.ActionStep{{
					ID:                "step-1",
					Name:              "Queue only no send",
					Sequence:          0,
					TemplateActionKey: "queue_only_no_send",
				}},
				Version: 1,
			},
		},
		targetRows: map[string][]actions.ActionTargetResult{
			"run-1": {
				{ID: 1, ActionRunID: "run-1", TargetEGMID: "EGM-1", Status: actions.TargetStatusPending},
				{ID: 2, ActionRunID: "run-1", TargetEGMID: "EGM-2", Status: actions.TargetStatusPending},
			},
		},
		egmsByID: map[string]egms.EGMRecord{
			"EGM-1": {EGMID: "EGM-1", Enabled: true, EmergencyEnabled: true, TemplateID: "template-1", CurrentActionState: egms.EGMActionStateNormal},
			"EGM-2": {EGMID: "EGM-2", Enabled: true, EmergencyEnabled: true, TemplateID: "template-1", CurrentActionState: egms.EGMActionStateNormal},
		},
		templates: map[string]templates.G2STemplate{
			"template-1": {ID: "template-1", Name: "Smoke", Vendor: "Test", Status: templates.TemplateStatusActive},
		},
		messages: []g2sengine.MessageJournalEntry{},
		audits:   []audit.AuditTimelineEntry{},
	}
}
