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

	assertMessageJournalSendColumns(t, store)
	assertHandlerRulesColumns(t, store)
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

	row.Status = actions.TargetStatusFailed
	row.AttemptCount = 2
	row.LastError = "delivery failed"
	lastResultAt := time.Now().UTC()
	row.LastResultAt = &lastResultAt
	if err := store.UpdateActionTargetResult(ctx, row); err != nil {
		t.Fatalf("update action target result: %v", err)
	}
	updatedList, err := store.ListActionTargetResults(ctx, "run-22")
	if err != nil {
		t.Fatalf("list updated action target results: %v", err)
	}
	if len(updatedList) != 1 || updatedList[0].Status != actions.TargetStatusFailed || updatedList[0].AttemptCount != 2 {
		t.Fatalf("unexpected updated target row: %+v", updatedList)
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

func TestG2STemplateVersionUpsertGetListAndActive(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	tpl := templates.G2STemplate{ID: "tpl-1", Name: "IGT Lab v1", Vendor: "IGT", Status: templates.TemplateStatusActive}
	if err := store.UpsertG2STemplate(ctx, tpl); err != nil {
		t.Fatalf("upsert g2s template: %v", err)
	}
	v1 := templates.G2STemplateVersion{
		ID:           "tpl-1-v1",
		TemplateID:   "tpl-1",
		VersionLabel: "1",
		ActionsJSON:  `{"actions":{"queue_only_no_send":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<x/>"}}}`,
	}
	v2 := templates.G2STemplateVersion{
		ID:           "tpl-1-v2",
		TemplateID:   "tpl-1",
		VersionLabel: "2",
		ActionsJSON:  `{"actions":{"queue_only_no_send":{"message_type":"DRY_RUN_NO_SEND","payload_template":"<y/>"}}}`,
	}
	if err := store.UpsertG2STemplateVersion(ctx, v1); err != nil {
		t.Fatalf("upsert version 1: %v", err)
	}
	if err := store.UpsertG2STemplateVersion(ctx, v2); err != nil {
		t.Fatalf("upsert version 2: %v", err)
	}

	fetchedV1, err := store.GetG2STemplateVersion(ctx, "tpl-1", 1)
	if err != nil {
		t.Fatalf("get version 1: %v", err)
	}
	if fetchedV1 == nil || fetchedV1.ID != "tpl-1-v1" {
		t.Fatalf("unexpected version 1 row: %+v", fetchedV1)
	}

	versions, err := store.ListG2STemplateVersions(ctx, "tpl-1")
	if err != nil {
		t.Fatalf("list template versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("version count=%d, want 2", len(versions))
	}

	if err := store.SetActiveG2STemplateVersion(ctx, "tpl-1", 2); err != nil {
		t.Fatalf("set active template version: %v", err)
	}
	active, err := store.GetActiveG2STemplateVersion(ctx, "tpl-1")
	if err != nil {
		t.Fatalf("get active template version: %v", err)
	}
	if active == nil || active.VersionLabel != "2" {
		t.Fatalf("unexpected active template version: %+v", active)
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

func TestMessageJournalUpdateResultPersistsSendFields(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	messageID, err := store.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:   time.Now().UTC(),
		Direction:   g2sengine.DirectionOutbound,
		EGMID:       "EGM-1",
		ActionRunID: "run-1",
		MessageType: "mute",
		RawPayload:  "<mute/>",
		Result:      g2sengine.MessageResultDryRun,
	})
	if err != nil {
		t.Fatalf("record message journal entry: %v", err)
	}
	sentAt := time.Now().UTC()
	completedAt := sentAt.Add(20 * time.Millisecond)
	if err := store.UpdateMessageJournalResult(
		ctx,
		messageID,
		g2sengine.MessageResultSendBlocked,
		"blocked by transport gate",
		"no response",
		0,
		20,
		"DISABLED",
		&sentAt,
		&completedAt,
	); err != nil {
		t.Fatalf("update message result: %v", err)
	}

	entries, err := store.ListMessageJournalEntries(ctx, MessageJournalListQuery{Limit: 10, ActionRunID: "run-1"})
	if err != nil {
		t.Fatalf("list message journal entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len=%d, want 1", len(entries))
	}
	row := entries[0]
	if row.Result != g2sengine.MessageResultSendBlocked {
		t.Fatalf("result=%q, want %q", row.Result, g2sengine.MessageResultSendBlocked)
	}
	if row.TransportMode != "DISABLED" {
		t.Fatalf("transport_mode=%q", row.TransportMode)
	}
	if row.LatencyMS != 20 {
		t.Fatalf("latency_ms=%d, want 20", row.LatencyMS)
	}
	if row.CompletedAt == nil {
		t.Fatal("expected completed_at")
	}
}

func TestMessageJournalOfferUpdateAndResultFilter(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	messageID, err := store.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:   time.Now().UTC(),
		Direction:   g2sengine.DirectionOutbound,
		EGMID:       "EGM-1",
		ActionRunID: "run-1",
		MessageType: "mute",
		RawPayload:  "<mute/>",
		Result:      g2sengine.MessageResultPrepared,
	})
	if err != nil {
		t.Fatalf("record message journal entry: %v", err)
	}
	offeredAt := time.Now().UTC()
	updated, err := store.UpdateMessageJournalOffer(ctx, messageID, offeredAt, g2sengine.MessageResultOffered)
	if err != nil {
		t.Fatalf("update offer: %v", err)
	}
	if !updated {
		t.Fatal("expected offer update to modify a row")
	}
	row, err := store.GetMessageJournalEntry(ctx, messageID)
	if err != nil {
		t.Fatalf("get message journal entry: %v", err)
	}
	if row == nil {
		t.Fatal("expected row")
	}
	if row.Result != g2sengine.MessageResultOffered {
		t.Fatalf("result=%q want OFFERED", row.Result)
	}
	if row.OfferCount != 1 {
		t.Fatalf("offer_count=%d want 1", row.OfferCount)
	}
	if row.OfferedAt == nil {
		t.Fatal("expected offered_at")
	}
	filtered, err := store.ListMessageJournalEntries(ctx, MessageJournalListQuery{
		Limit:   10,
		Results: []g2sengine.MessageResult{g2sengine.MessageResultOffered},
	})
	if err != nil {
		t.Fatalf("list filtered messages: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != messageID {
		t.Fatalf("unexpected filtered rows: %+v", filtered)
	}
}

func TestMessageJournalUpdateHandlerRule(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	messageID, err := store.RecordMessageJournalEntry(ctx, g2sengine.MessageJournalEntry{
		Timestamp:   time.Now().UTC(),
		Direction:   g2sengine.DirectionInbound,
		EGMID:       "EGM-1",
		ActionRunID: "run-1",
		MessageType: "ACK",
		RawPayload:  "<ack/>",
		Result:      g2sengine.MessageResultReceived,
	})
	if err != nil {
		t.Fatalf("record message: %v", err)
	}
	if err := store.UpdateMessageJournalHandlerRule(ctx, messageID, "rule-1"); err != nil {
		t.Fatalf("update message handler_rule_id: %v", err)
	}
	row, err := store.GetMessageJournalEntry(ctx, messageID)
	if err != nil {
		t.Fatalf("get message: %v", err)
	}
	if row == nil || row.HandlerRuleID != "rule-1" {
		t.Fatalf("unexpected handler rule linkage: %+v", row)
	}
}

func TestHandlerRuleUpsertGetListAndDisable(t *testing.T) {
	ctx := context.Background()
	store := newPhaseStore(t, ctx)
	defer store.Close()

	rule := g2sengine.HandlerRule{
		ID:          "handler-1",
		Name:        "Accept ACK",
		Enabled:     true,
		Direction:   g2sengine.HandlerRuleDirectionInbound,
		TemplateID:  "template-generic-g2s-action",
		MessageType: "ACK",
		EGMID:       "EGM-001",
		ActionID:    "emergency-broadcast-trigger",
		MatchJSON:   `{"contains":["accepted"]}`,
		Outcome:     g2sengine.HandlerRuleOutcomeConfirmation,
		Notes:       "operator note",
	}
	if err := store.UpsertHandlerRule(ctx, rule); err != nil {
		t.Fatalf("upsert handler rule: %v", err)
	}

	fetched, err := store.GetHandlerRule(ctx, "handler-1")
	if err != nil {
		t.Fatalf("get handler rule: %v", err)
	}
	if fetched == nil || fetched.Name != "Accept ACK" || fetched.Outcome != g2sengine.HandlerRuleOutcomeConfirmation {
		t.Fatalf("unexpected fetched handler rule: %+v", fetched)
	}

	allRules, err := store.ListHandlerRules(ctx, HandlerRuleListQuery{Limit: 20})
	if err != nil {
		t.Fatalf("list handler rules: %v", err)
	}
	if len(allRules) != 1 {
		t.Fatalf("handler rule count=%d want=1", len(allRules))
	}

	enabledRules, err := store.ListEnabledHandlerRules(ctx, 20)
	if err != nil {
		t.Fatalf("list enabled handler rules: %v", err)
	}
	if len(enabledRules) != 1 {
		t.Fatalf("enabled rule count=%d want=1", len(enabledRules))
	}

	if err := store.DisableHandlerRule(ctx, "handler-1"); err != nil {
		t.Fatalf("disable handler rule: %v", err)
	}
	enabledRules, err = store.ListEnabledHandlerRules(ctx, 20)
	if err != nil {
		t.Fatalf("list enabled handler rules after disable: %v", err)
	}
	if len(enabledRules) != 0 {
		t.Fatalf("enabled rule count=%d want=0", len(enabledRules))
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
	if err := store.UpsertG2STemplateVersion(ctx, templates.G2STemplateVersion{}); err == nil {
		t.Fatal("expected template version validation error")
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
	assertCount(t, store, "g2s_template_versions", 0)
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

func assertMessageJournalSendColumns(t *testing.T, store *SQLiteStore) {
	t.Helper()
	rows, err := store.db.Query(`PRAGMA table_info(message_journal)`)
	if err != nil {
		t.Fatalf("table_info(message_journal): %v", err)
	}
	defer rows.Close()

	found := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table_info row: %v", err)
		}
		found[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("table_info rows: %v", err)
	}
	for _, column := range []string{"http_status_code", "latency_ms", "response_excerpt", "offered_at", "offer_count", "sent_at", "completed_at", "transport_mode"} {
		if !found[column] {
			t.Fatalf("expected message_journal column %q", column)
		}
	}
}

func assertHandlerRulesColumns(t *testing.T, store *SQLiteStore) {
	t.Helper()
	rows, err := store.db.Query(`PRAGMA table_info(handler_rules)`)
	if err != nil {
		t.Fatalf("table_info(handler_rules): %v", err)
	}
	defer rows.Close()

	found := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table_info row: %v", err)
		}
		found[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("table_info rows: %v", err)
	}
	for _, column := range []string{"direction", "template_id", "message_type", "egm_id", "action_id", "action_step_id", "outcome"} {
		if !found[column] {
			t.Fatalf("expected handler_rules column %q", column)
		}
	}
}
