package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actiondispatch"
	"github.com/tschneider-imagine/G2S_MC/internal/actionruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/inputpoller"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestSeedLabActionDefinitionsAndBindings(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := inputpoller.EnsureDefaultPi4InputChannels(ctx, st, true); err != nil {
		t.Fatalf("seed default channels: %v", err)
	}
	if err := seedLabActionDefinitionsAndBindings(ctx, st); err != nil {
		t.Fatalf("seed lab definitions and bindings: %v", err)
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
	if emergency.OnNormalActionID != "emergency-broadcast-restore" {
		t.Fatalf("emergency normal action id=%q", emergency.OnNormalActionID)
	}

	actionIDs := []string{
		"regular-operation-trigger",
		"general-broadcast-trigger",
		"emergency-broadcast-trigger",
		"local-notice-trigger",
		"emergency-broadcast-restore",
		"general-broadcast-restore",
		"local-notice-restore",
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

func TestSeedLabEGMRegistry(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := seedLabEGMRegistry(ctx, st, "http://127.0.0.1:18080/capture"); err != nil {
		t.Fatalf("seed lab egm registry: %v", err)
	}

	tpl, err := st.GetG2STemplate(ctx, "template-generic-g2s-action")
	if err != nil {
		t.Fatalf("get generic template: %v", err)
	}
	if tpl == nil {
		t.Fatal("expected generic template")
	}
	if tpl.CurrentVersionID != "1" {
		t.Fatalf("current version id=%q, want 1", tpl.CurrentVersionID)
	}
	active, err := st.GetActiveG2STemplateVersion(ctx, "template-generic-g2s-action")
	if err != nil {
		t.Fatalf("get active generic template version: %v", err)
	}
	if active == nil {
		t.Fatal("expected active generic template version")
	}
	doc, err := g2sengine.ParseActionTemplateDocument(active.ActionsJSON)
	if err != nil {
		t.Fatalf("parse seeded actions json: %v", err)
	}
	rendered, err := g2sengine.RenderActionMessage(doc, g2sengine.RenderRequest{
		TemplateID:        "template-generic-g2s-action",
		TemplateVersion:   1,
		TemplateActionKey: "emergency_broadcast_silence",
		ActionID:          "emergency-broadcast-trigger",
		ActionRunID:       "run-seed-test",
		EGMID:             "EGM-001",
		Timestamp:         time.Date(2026, 5, 24, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("render seeded action: %v", err)
	}
	if !strings.Contains(rendered.RawPayload, `egm="EGM-001"`) {
		t.Fatalf("rendered payload missing egm id: %s", rendered.RawPayload)
	}

	for _, id := range []string{"EGM-001", "EGM-002"} {
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
		if row.TemplateID != "template-generic-g2s-action" {
			t.Fatalf("template_id=%q for %s", row.TemplateID, id)
		}
		if row.EndpointPath != "http://127.0.0.1:18080/capture" {
			t.Fatalf("endpoint_path=%q for %s", row.EndpointPath, id)
		}
	}
}

func TestParseTransportMode(t *testing.T) {
	cases := []struct {
		raw    string
		expect g2stransport.Mode
		ok     bool
	}{
		{raw: "", expect: g2stransport.ModeDisabled, ok: true},
		{raw: "disabled", expect: g2stransport.ModeDisabled, ok: true},
		{raw: "dry-run", expect: g2stransport.ModeDryRun, ok: true},
		{raw: "http", expect: g2stransport.ModeHTTP, ok: true},
		{raw: "udp", expect: "", ok: false},
	}
	for _, tc := range cases {
		got, err := parseTransportMode(tc.raw)
		if tc.ok && err != nil {
			t.Fatalf("parseTransportMode(%q) err=%v", tc.raw, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("parseTransportMode(%q) expected error", tc.raw)
		}
		if tc.ok && got != tc.expect {
			t.Fatalf("parseTransportMode(%q)=%q want %q", tc.raw, got, tc.expect)
		}
	}
}

func TestValidateCaptureSendConfig(t *testing.T) {
	cases := []struct {
		name            string
		mode            g2stransport.Mode
		allowRealSend   bool
		captureOnlySend bool
		endpoint        string
		wantErr         bool
	}{
		{
			name:            "default no send",
			mode:            g2stransport.ModeDisabled,
			allowRealSend:   false,
			captureOnlySend: false,
			endpoint:        "",
			wantErr:         false,
		},
		{
			name:            "allow send requires http mode",
			mode:            g2stransport.ModeDryRun,
			allowRealSend:   true,
			captureOnlySend: true,
			endpoint:        "http://127.0.0.1:18080/capture",
			wantErr:         true,
		},
		{
			name:            "allow send requires capture only",
			mode:            g2stransport.ModeHTTP,
			allowRealSend:   true,
			captureOnlySend: false,
			endpoint:        "http://127.0.0.1:18080/capture",
			wantErr:         true,
		},
		{
			name:            "allow send requires local endpoint",
			mode:            g2stransport.ModeHTTP,
			allowRealSend:   true,
			captureOnlySend: true,
			endpoint:        "http://10.20.30.40:18080/capture",
			wantErr:         true,
		},
		{
			name:            "allow send localhost ok",
			mode:            g2stransport.ModeHTTP,
			allowRealSend:   true,
			captureOnlySend: true,
			endpoint:        "http://localhost:18080/capture",
			wantErr:         false,
		},
	}

	for _, tc := range cases {
		err := validateCaptureSendConfig(tc.mode, tc.allowRealSend, tc.captureOnlySend, tc.endpoint)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
	}
}

func TestRunClearLatchPath(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	channel := inputs.InputChannel{
		ID:                "emergency-broadcast",
		Name:              "Emergency Broadcast",
		GPIOChannel:       "GPIO21",
		Enabled:           true,
		NormalState:       inputs.InputStateHigh,
		CurrentState:      inputs.InputStateLow,
		DerivedState:      inputs.DerivedStateTriggered,
		DebounceMS:        30,
		Priority:          400,
		OnTriggerActionID: "emergency-broadcast-trigger",
		OnNormalActionID:  "emergency-broadcast-restore",
		LatchingMode:      inputs.LatchingManualClear,
	}
	if err := st.UpsertInputChannel(ctx, channel); err != nil {
		t.Fatalf("upsert input channel: %v", err)
	}
	now := time.Now().UTC()
	if err := st.UpsertInputRuntimeState(ctx, inputruntime.InputRuntimeState{
		InputID:              channel.ID,
		StableRawState:       inputs.InputStateHigh,
		DerivedState:         inputs.DerivedStateTriggered,
		LatchActive:          true,
		StableSince:          now.Add(-2 * time.Second),
		LastObservedRawState: inputs.InputStateHigh,
		LastObservedAt:       now.Add(-1 * time.Second),
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("upsert input runtime state: %v", err)
	}
	if err := st.UpsertActionDefinition(ctx, actions.ActionDefinition{
		ID:               "emergency-broadcast-restore",
		Name:             "Emergency Broadcast Restore",
		Severity:         actions.SeverityRestore,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps: []actions.ActionStep{{
			ID:                "step-1",
			Name:              "Primary message",
			Sequence:          0,
			TemplateActionKey: "emergency_broadcast_restore",
		}},
		Version: 1,
	}); err != nil {
		t.Fatalf("upsert action definition: %v", err)
	}

	evaluator := &inputruntime.Evaluator{Store: st, Clock: time.Now}
	queuer := &actionruntime.Queuer{Store: st, Clock: time.Now}
	dispatcher := &actiondispatch.Dispatcher{Store: st, Clock: time.Now}
	if err := runClearLatch(ctx, evaluator, queuer, dispatcher, channel.ID, true, false, false, g2stransport.ModeDisabled, false, false, ""); err != nil {
		t.Fatalf("runClearLatch: %v", err)
	}
	rows, err := st.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list action runs: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("action run count=%d want 1", len(rows))
	}
}
