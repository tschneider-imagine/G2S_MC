package store

import (
	"context"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
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

func TestPhase1AStoreMethods(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
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
	channels, err := store.ListInputChannels(ctx)
	if err != nil {
		t.Fatalf("list input channels: %v", err)
	}
	if len(channels) != 1 || channels[0].ID != "input-1" {
		t.Fatalf("unexpected input channels: %+v", channels)
	}

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
	definitions, err := store.ListActionDefinitions(ctx)
	if err != nil {
		t.Fatalf("list action definitions: %v", err)
	}
	if len(definitions) != 1 || len(definitions[0].Steps) != 1 {
		t.Fatalf("unexpected action definitions: %+v", definitions)
	}

	tpl := templates.G2STemplate{ID: "tpl-1", Name: "IGT Lab v1", Vendor: "IGT", Status: templates.TemplateStatusDraft}
	if err := store.UpsertG2STemplate(ctx, tpl); err != nil {
		t.Fatalf("upsert g2s template: %v", err)
	}
	templatesList, err := store.ListG2STemplates(ctx)
	if err != nil {
		t.Fatalf("list g2s templates: %v", err)
	}
	if len(templatesList) != 1 || templatesList[0].ID != "tpl-1" {
		t.Fatalf("unexpected g2s templates: %+v", templatesList)
	}

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
	entries, err := store.ListMessageJournalEntries(ctx, 10)
	if err != nil {
		t.Fatalf("list message journal entries: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != messageID {
		t.Fatalf("unexpected message entries: %+v", entries)
	}

	auditID, err := store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:       time.Now().UTC(),
		Severity:         audit.AuditSeverityEmergency,
		EventType:        "ACTION_START",
		Summary:          "Emergency action started",
		MessageJournalID: messageID,
	})
	if err != nil {
		t.Fatalf("record audit timeline entry: %v", err)
	}
	auditEntries, err := store.ListAuditTimelineEntries(ctx, 10)
	if err != nil {
		t.Fatalf("list audit timeline entries: %v", err)
	}
	if len(auditEntries) != 1 || auditEntries[0].ID != auditID {
		t.Fatalf("unexpected audit timeline entries: %+v", auditEntries)
	}
}
