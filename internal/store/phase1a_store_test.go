package store

import (
	"context"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

func TestPhase1AMigrationIdempotent(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	for i := 0; i < 3; i++ {
		if err := store.Migrate(ctx); err != nil {
			t.Fatalf("migrate pass %d: %v", i+1, err)
		}
	}

	for _, table := range []string{
		"input_channels",
		"input_transitions",
		"action_definitions",
		"action_runs",
		"action_target_results",
		"g2s_templates",
		"g2s_template_versions",
		"message_journal",
		"handler_rules",
		"egm_records",
		"egm_groups",
		"audit_timeline",
	} {
		assertCount(t, store, table, 0)
	}
}

func TestInputChannelUpsertGetList(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	channel := inputs.InputChannel{
		ID:           "input-1",
		Name:         "Emergency Broadcast",
		GPIOChannel:  "GPIO17",
		Enabled:      true,
		NormalState:  inputs.InputStateHigh,
		CurrentState: inputs.InputStateLow,
		DerivedState: inputs.DerivedStateTriggered,
		DebounceMS:   25,
		Priority:     4,
		LatchingMode: inputs.LatchingManualClear,
	}
	if err := store.UpsertInputChannel(ctx, channel); err != nil {
		t.Fatalf("upsert input channel: %v", err)
	}
	fetched, err := store.GetInputChannel(ctx, "input-1")
	if err != nil {
		t.Fatalf("get input channel: %v", err)
	}
	if fetched == nil || fetched.Name != channel.Name {
		t.Fatalf("unexpected fetched input channel: %+v", fetched)
	}
	channels, err := store.ListInputChannels(ctx)
	if err != nil {
		t.Fatalf("list input channels: %v", err)
	}
	if len(channels) != 1 || channels[0].ID != "input-1" {
		t.Fatalf("unexpected input channels: %+v", channels)
	}
}

func TestActionDefinitionUpsertGetList(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	definition := actions.ActionDefinition{
		ID:               "action-1",
		Name:             "Emergency Silence",
		Severity:         actions.SeverityEmergency,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps: []actions.ActionStep{{
			ID:                "step-1",
			Name:              "Send mute",
			Sequence:          0,
			TemplateActionKey: "mute_primary",
		}},
		Version: 1,
	}
	if err := store.UpsertActionDefinition(ctx, definition); err != nil {
		t.Fatalf("upsert action definition: %v", err)
	}
	fetched, err := store.GetActionDefinition(ctx, "action-1")
	if err != nil {
		t.Fatalf("get action definition: %v", err)
	}
	if fetched == nil || len(fetched.Steps) != 1 {
		t.Fatalf("unexpected fetched action definition: %+v", fetched)
	}
	definitions, err := store.ListActionDefinitions(ctx)
	if err != nil {
		t.Fatalf("list action definitions: %v", err)
	}
	if len(definitions) != 1 || definitions[0].ID != "action-1" {
		t.Fatalf("unexpected action definitions: %+v", definitions)
	}
}

func TestActionRunCreateGetList(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	if err := store.UpsertActionDefinition(ctx, actions.ActionDefinition{
		ID:               "action-1",
		Name:             "Queue Action",
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
	}); err != nil {
		t.Fatalf("seed action definition: %v", err)
	}

	startedAt := time.Now().UTC()
	run, err := store.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-1",
		ActionDefinitionID: "action-1",
		InputTransitionID:  99,
		StartedAt:          startedAt,
		Status:             actions.RunStatusPending,
		TriggerReason:      "input transition 99",
		TargetCount:        2,
		ConfirmedCount:     0,
		FailedCount:        0,
		EscalatedCount:     0,
	})
	if err != nil {
		t.Fatalf("create action run: %v", err)
	}
	if run.ID != "run-1" {
		t.Fatalf("created run id = %q, want %q", run.ID, "run-1")
	}

	got, err := store.GetActionRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("get action run: %v", err)
	}
	if got == nil || got.ActionDefinitionID != "action-1" {
		t.Fatalf("unexpected action run: %+v", got)
	}

	rows, err := store.ListActionRuns(ctx, ActionRunListQuery{
		Limit:              10,
		Status:             actions.RunStatusPending,
		ActionDefinitionID: "action-1",
		InputTransitionID:  99,
	})
	if err != nil {
		t.Fatalf("list action runs: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "run-1" {
		t.Fatalf("unexpected action runs: %+v", rows)
	}

	run.Status = actions.RunStatusDispatchPrepared
	run.ConfirmedCount = 1
	if err := store.UpdateActionRun(ctx, run); err != nil {
		t.Fatalf("update action run: %v", err)
	}
	updated, err := store.GetActionRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("get updated action run: %v", err)
	}
	if updated == nil || updated.Status != actions.RunStatusDispatchPrepared || updated.ConfirmedCount != 1 {
		t.Fatalf("unexpected updated action run: %+v", updated)
	}
}

func TestActionTargetResultCreateList(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	row, err := store.CreateActionTargetResult(ctx, actions.ActionTargetResult{
		ActionRunID:  "run-22",
		TargetEGMID:  "EGM-001",
		Status:       actions.TargetStatusPending,
		AttemptCount: 0,
	})
	if err != nil {
		t.Fatalf("create action target result: %v", err)
	}
	if row.ID == 0 {
		t.Fatalf("expected non-zero row id: %+v", row)
	}

	list, err := store.ListActionTargetResults(ctx, "run-22")
	if err != nil {
		t.Fatalf("list action target results: %v", err)
	}
	if len(list) != 1 || list[0].TargetEGMID != "EGM-001" {
		t.Fatalf("unexpected target results: %+v", list)
	}
}

func TestG2STemplateUpsertGetList(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	tpl := templates.G2STemplate{ID: "tpl-1", Name: "IGT Lab v1", Vendor: "IGT", Status: templates.TemplateStatusDraft}
	if err := store.UpsertG2STemplate(ctx, tpl); err != nil {
		t.Fatalf("upsert g2s template: %v", err)
	}
	fetched, err := store.GetG2STemplate(ctx, "tpl-1")
	if err != nil {
		t.Fatalf("get g2s template: %v", err)
	}
	if fetched == nil || fetched.ID != "tpl-1" {
		t.Fatalf("unexpected fetched g2s template: %+v", fetched)
	}
	templatesList, err := store.ListG2STemplates(ctx)
	if err != nil {
		t.Fatalf("list g2s templates: %v", err)
	}
	if len(templatesList) != 1 || templatesList[0].ID != "tpl-1" {
		t.Fatalf("unexpected g2s templates: %+v", templatesList)
	}
}

func TestEGMRecordUpsertGetList(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	record := egms.EGMRecord{
		EGMID:              "EGM-1",
		DisplayName:        "Cabinet 1",
		IPAddress:          "10.10.0.10",
		Enabled:            true,
		EmergencyEnabled:   true,
		TemplateID:         "tpl-1",
		CurrentActionState: egms.EGMActionStateNormal,
	}
	if err := store.UpsertEGMRecord(ctx, record); err != nil {
		t.Fatalf("upsert egm record: %v", err)
	}
	fetched, err := store.GetEGMRecord(ctx, "EGM-1")
	if err != nil {
		t.Fatalf("get egm record: %v", err)
	}
	if fetched == nil || fetched.DisplayName != "Cabinet 1" {
		t.Fatalf("unexpected fetched egm record: %+v", fetched)
	}
	records, err := store.ListEGMRecords(ctx)
	if err != nil {
		t.Fatalf("list egm records: %v", err)
	}
	if len(records) != 1 || records[0].EGMID != "EGM-1" {
		t.Fatalf("unexpected egm records: %+v", records)
	}
}

func TestMessageJournalRecordAndList(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	messageID, err := store.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:         time.Now().UTC(),
		Direction:         g2sengine.DirectionOutbound,
		EGMID:             "EGM-1",
		ActionRunID:       "run-1",
		InputTransitionID: 1,
		MessageType:       "mute",
		RawPayload:        "<mute/>",
		Result:            g2sengine.MessageResultSent,
	})
	if err != nil {
		t.Fatalf("record message journal entry: %v", err)
	}
	entries, err := store.ListMessageJournalEntries(ctx, MessageJournalListQuery{Limit: 10, EGMID: "EGM-1"})
	if err != nil {
		t.Fatalf("list message journal entries: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != messageID {
		t.Fatalf("unexpected message entries: %+v", entries)
	}

	entriesByRun, err := store.ListMessageJournalEntries(ctx, MessageJournalListQuery{Limit: 10, ActionRunID: "run-1"})
	if err != nil {
		t.Fatalf("list message journal entries by action run: %v", err)
	}
	if len(entriesByRun) != 1 || entriesByRun[0].ID != messageID {
		t.Fatalf("unexpected message entries by action run: %+v", entriesByRun)
	}
}

func TestAuditTimelineRecordAndList(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	auditID, err := store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt: time.Now().UTC(),
		Severity:   audit.AuditSeverityEmergency,
		EventType:  "ACTION_START",
		Summary:    "Emergency action started",
	})
	if err != nil {
		t.Fatalf("record audit timeline entry: %v", err)
	}
	auditEntries, err := store.ListAuditTimelineEntries(ctx, AuditTimelineListQuery{Limit: 10, EventType: "ACTION_START"})
	if err != nil {
		t.Fatalf("list audit timeline entries: %v", err)
	}
	if len(auditEntries) != 1 || auditEntries[0].ID != auditID {
		t.Fatalf("unexpected audit timeline entries: %+v", auditEntries)
	}
}

func TestPhase1AValidationErrorsAreNotStored(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	if err := store.UpsertInputChannel(ctx, inputs.InputChannel{}); err == nil {
		t.Fatal("expected input channel validation error")
	}
	if err := store.UpsertActionDefinition(ctx, actions.ActionDefinition{}); err == nil {
		t.Fatal("expected action definition validation error")
	}
	if err := store.UpsertG2STemplate(ctx, templates.G2STemplate{}); err == nil {
		t.Fatal("expected template validation error")
	}
	if err := store.UpsertEGMRecord(ctx, egms.EGMRecord{}); err == nil {
		t.Fatal("expected egm record validation error")
	}
	if _, err := store.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{}); err == nil {
		t.Fatal("expected message journal validation error")
	}
	if _, err := store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{}); err == nil {
		t.Fatal("expected audit timeline validation error")
	}

	assertCount(t, store, "input_channels", 0)
	assertCount(t, store, "action_definitions", 0)
	assertCount(t, store, "g2s_templates", 0)
	assertCount(t, store, "egm_records", 0)
	assertCount(t, store, "message_journal", 0)
	assertCount(t, store, "audit_timeline", 0)
}

func TestEGMGroupUpsertGetList(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	group := egms.EGMGroup{
		ID:          "zone-a",
		Name:        "Zone A",
		Description: "High priority floor",
		EGMIDs:      []string{"EGM-3", "EGM-1"},
	}
	if err := store.UpsertEGMGroup(ctx, group); err != nil {
		t.Fatalf("upsert egm group: %v", err)
	}
	fetched, err := store.GetEGMGroup(ctx, "zone-a")
	if err != nil {
		t.Fatalf("get egm group: %v", err)
	}
	if fetched == nil || fetched.Name != "Zone A" {
		t.Fatalf("unexpected fetched group: %+v", fetched)
	}
	if len(fetched.EGMIDs) != 2 || fetched.EGMIDs[0] != "EGM-3" || fetched.EGMIDs[1] != "EGM-1" {
		t.Fatalf("unexpected fetched group membership: %+v", fetched.EGMIDs)
	}
	groups, err := store.ListEGMGroups(ctx)
	if err != nil {
		t.Fatalf("list egm groups: %v", err)
	}
	if len(groups) != 1 || groups[0].ID != "zone-a" {
		t.Fatalf("unexpected groups: %+v", groups)
	}
	if len(groups[0].EGMIDs) != 2 || groups[0].EGMIDs[0] != "EGM-3" || groups[0].EGMIDs[1] != "EGM-1" {
		t.Fatalf("unexpected listed group membership: %+v", groups[0].EGMIDs)
	}
}

func newPhaseStore(t *testing.T, ctx context.Context) *SQLiteStore {
	t.Helper()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}
