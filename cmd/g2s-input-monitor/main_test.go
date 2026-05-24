package main

import (
	"context"
	"testing"

	"github.com/tschneider-imagine/G2S_MC/internal/inputpoller"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestSeedDemoActionDefinitionsAndBindings(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := inputpoller.EnsureDefaultPi4InputChannels(ctx, st, true); err != nil {
		t.Fatalf("seed default channels: %v", err)
	}
	if err := seedDemoActionDefinitionsAndBindings(ctx, st); err != nil {
		t.Fatalf("seed demo definitions and bindings: %v", err)
	}

	emergency, err := st.GetInputChannel(ctx, "emergency-broadcast")
	if err != nil {
		t.Fatalf("get emergency-broadcast channel: %v", err)
	}
	if emergency == nil {
		t.Fatal("expected emergency-broadcast channel")
	}
	if emergency.OnTriggerActionID != "emergency-broadcast-trigger" {
		t.Fatalf("emergency trigger action id=%q", emergency.OnTriggerActionID)
	}
	if emergency.OnNormalActionID != "emergency-broadcast-normal" {
		t.Fatalf("emergency normal action id=%q", emergency.OnNormalActionID)
	}

	actionIDs := []string{
		"regular-operation-trigger",
		"general-broadcast-trigger",
		"emergency-broadcast-trigger",
		"local-notice-trigger",
		"emergency-broadcast-normal",
		"general-broadcast-normal",
		"local-notice-normal",
	}
	for _, id := range actionIDs {
		row, getErr := st.GetActionDefinition(ctx, id)
		if getErr != nil {
			t.Fatalf("get action definition %s: %v", id, getErr)
		}
		if row == nil {
			t.Fatalf("expected action definition %s", id)
		}
	}
}

func TestSeedDemoEGMRegistry(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := seedDemoEGMRegistry(ctx, st); err != nil {
		t.Fatalf("seed demo egm registry: %v", err)
	}

	tpl, err := st.GetG2STemplate(ctx, "template-smoke-no-send")
	if err != nil {
		t.Fatalf("get smoke template: %v", err)
	}
	if tpl == nil {
		t.Fatal("expected smoke template")
	}

	for _, id := range []string{"EGM-SMOKE-001", "EGM-SMOKE-002"} {
		row, getErr := st.GetEGMRecord(ctx, id)
		if getErr != nil {
			t.Fatalf("get %s: %v", id, getErr)
		}
		if row == nil {
			t.Fatalf("expected %s", id)
		}
		if !row.Enabled || !row.EmergencyEnabled {
			t.Fatalf("expected enabled emergency-enabled %s: %+v", id, row)
		}
		if row.TemplateID != "template-smoke-no-send" {
			t.Fatalf("template_id=%q for %s", row.TemplateID, id)
		}
	}
}
