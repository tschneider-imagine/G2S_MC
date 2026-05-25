package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actiondispatch"
	"github.com/tschneider-imagine/G2S_MC/internal/actionexecutor"
	"github.com/tschneider-imagine/G2S_MC/internal/actionruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/inputpoller"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func TestSeedActionDefinitionsAndBindings(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := inputpoller.EnsureDefaultPi4InputChannels(ctx, st, true); err != nil {
		t.Fatalf("seed default channels: %v", err)
	}
	if err := seedActionDefinitionsAndBindings(ctx, st); err != nil {
		t.Fatalf("seed definitions and bindings: %v", err)
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

func TestSeedEGMRegistry(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := seedEGMRegistry(ctx, st, "http://127.0.0.1:18080/capture"); err != nil {
		t.Fatalf("seed egm registry: %v", err)
	}

	tpl, err := st.GetG2STemplate(ctx, "template-generic-g2s-action")
	if err != nil {
		t.Fatalf("get template: %v", err)
	}
	if tpl == nil {
		t.Fatal("expected template")
	}
	if tpl.CurrentVersionID != "1" {
		t.Fatalf("current version id=%q, want 1", tpl.CurrentVersionID)
	}
	active, err := st.GetActiveG2STemplateVersion(ctx, "template-generic-g2s-action")
	if err != nil {
		t.Fatalf("get active template version: %v", err)
	}
	if active == nil {
		t.Fatal("expected active template version")
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
		OnNormalActionID:  "emergency-broadcast-normal",
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
		ID:               "emergency-broadcast-normal",
		Name:             "Emergency Broadcast Restore",
		Severity:         actions.SeverityRestore,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps: []actions.ActionStep{{
			ID:                "step-1",
			Name:              "Primary Notification",
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
	if err := runClearLatch(ctx, evaluator, queuer, dispatcher, nil, channel.ID, true, false, g2stransport.DeliverySettings{}, false, false, g2stransport.ModeDisabled, false, false, ""); err != nil {
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

func TestParseDeliveryMode(t *testing.T) {
	cases := []struct {
		raw    string
		expect g2stransport.DeliveryMode
		ok     bool
	}{
		{raw: "", expect: g2stransport.DeliveryModeDisabled, ok: true},
		{raw: "disabled", expect: g2stransport.DeliveryModeDisabled, ok: true},
		{raw: "http", expect: g2stransport.DeliveryModeHTTP, ok: true},
		{raw: "dry-run", expect: "", ok: false},
	}
	for _, tc := range cases {
		got, err := parseDeliveryMode(tc.raw)
		if tc.ok && err != nil {
			t.Fatalf("parseDeliveryMode(%q) err=%v", tc.raw, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("parseDeliveryMode(%q) expected error", tc.raw)
		}
		if tc.ok && got != tc.expect {
			t.Fatalf("parseDeliveryMode(%q)=%q want %q", tc.raw, got, tc.expect)
		}
	}
}

func TestValidateExecuteDeliveryConfig(t *testing.T) {
	if err := validateExecuteDeliveryConfig(g2stransport.DeliveryModeDisabled, false, 5000); err != nil {
		t.Fatalf("unexpected err for disabled mode: %v", err)
	}
	if err := validateExecuteDeliveryConfig(g2stransport.DeliveryModeDisabled, true, 5000); err == nil {
		t.Fatal("expected error when allow-delivery with non-http mode")
	}
	if err := validateExecuteDeliveryConfig(g2stransport.DeliveryModeHTTP, true, -1); err == nil {
		t.Fatal("expected error for negative timeout")
	}
}

type fakeMonitorExecutor struct {
	results []actionexecutor.ExecuteResult
	errs    []error
	calls   []actionexecutor.ExecuteRequest
}

func (f *fakeMonitorExecutor) Execute(_ context.Context, request actionexecutor.ExecuteRequest) (actionexecutor.ExecuteResult, error) {
	f.calls = append(f.calls, request)
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		return actionexecutor.ExecuteResult{}, err
	}
	if len(f.results) == 0 {
		return actionexecutor.ExecuteResult{
			ActionRun: actions.ActionRun{
				ID:             request.ActionRunID,
				Status:         actions.RunStatusSucceeded,
				ConfirmedCount: 1,
				FailedCount:    0,
			},
		}, nil
	}
	result := f.results[0]
	f.results = f.results[1:]
	return result, nil
}

func TestRunClearLatchWithoutExecuteQueuesButDoesNotExecute(t *testing.T) {
	ctx := context.Background()
	st := newMonitorStore(t, ctx)
	defer st.Close()
	setupManualClearScenario(t, ctx, st)

	evaluator := &inputruntime.Evaluator{Store: st, Clock: time.Now}
	queuer := &actionruntime.Queuer{Store: st, Clock: time.Now}
	dispatcher := &actiondispatch.Dispatcher{Store: st, Clock: time.Now}
	executor := &fakeMonitorExecutor{}

	if err := runClearLatch(ctx, evaluator, queuer, dispatcher, executor, "emergency-broadcast", true, false, g2stransport.DeliverySettings{}, false, false, g2stransport.ModeDisabled, false, false, ""); err != nil {
		t.Fatalf("runClearLatch: %v", err)
	}
	if len(executor.calls) != 0 {
		t.Fatalf("executor calls=%d, want 0", len(executor.calls))
	}
	runs, err := st.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list action runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("action runs=%d, want 1", len(runs))
	}
	if runs[0].Status != actions.RunStatusPending {
		t.Fatalf("status=%q want PENDING", runs[0].Status)
	}
}

func TestRunClearLatchWithExecuteRunsQueuedAction(t *testing.T) {
	ctx := context.Background()
	st := newMonitorStore(t, ctx)
	defer st.Close()
	setupManualClearScenario(t, ctx, st)

	evaluator := &inputruntime.Evaluator{Store: st, Clock: time.Now}
	queuer := &actionruntime.Queuer{Store: st, Clock: time.Now}
	dispatcher := &actiondispatch.Dispatcher{Store: st, Clock: time.Now}
	executor := &fakeMonitorExecutor{}
	delivery := g2stransport.DeliverySettings{Mode: g2stransport.DeliveryModeHTTP, AllowDelivery: true, TimeoutMS: 5000}

	if err := runClearLatch(ctx, evaluator, queuer, dispatcher, executor, "emergency-broadcast", true, true, delivery, false, false, g2stransport.ModeDisabled, false, false, ""); err != nil {
		t.Fatalf("runClearLatch: %v", err)
	}
	if len(executor.calls) != 1 {
		t.Fatalf("executor calls=%d, want 1", len(executor.calls))
	}
	if executor.calls[0].Delivery.Mode != g2stransport.DeliveryModeHTTP || !executor.calls[0].Delivery.AllowDelivery {
		t.Fatalf("unexpected delivery settings: %+v", executor.calls[0].Delivery)
	}
}

func TestRunClearLatchExecutesOnlyCurrentProcessQueuedRun(t *testing.T) {
	ctx := context.Background()
	st := newMonitorStore(t, ctx)
	defer st.Close()
	setupManualClearScenario(t, ctx, st)

	oldRun, err := st.CreateActionRun(ctx, actions.ActionRun{
		ID:                 "run-historical",
		ActionDefinitionID: "emergency-broadcast-normal",
		StartedAt:          time.Now().UTC().Add(-time.Minute),
		Status:             actions.RunStatusPending,
		TriggerReason:      "historical",
		TargetCount:        1,
	})
	if err != nil {
		t.Fatalf("create historical run: %v", err)
	}
	if _, err := st.CreateActionTargetResult(ctx, actions.ActionTargetResult{ActionRunID: oldRun.ID, TargetEGMID: "EGM-001", Status: actions.TargetStatusPending}); err != nil {
		t.Fatalf("create historical target: %v", err)
	}

	evaluator := &inputruntime.Evaluator{Store: st, Clock: time.Now}
	queuer := &actionruntime.Queuer{Store: st, Clock: time.Now}
	dispatcher := &actiondispatch.Dispatcher{Store: st, Clock: time.Now}
	executor := &fakeMonitorExecutor{}

	if err := runClearLatch(ctx, evaluator, queuer, dispatcher, executor, "emergency-broadcast", true, true, g2stransport.DeliverySettings{}, false, false, g2stransport.ModeDisabled, false, false, ""); err != nil {
		t.Fatalf("runClearLatch: %v", err)
	}
	if len(executor.calls) != 1 {
		t.Fatalf("executor calls=%d, want 1", len(executor.calls))
	}
	if executor.calls[0].ActionRunID == oldRun.ID {
		t.Fatalf("historical run %q should not execute", oldRun.ID)
	}
}

func TestRunClearLatchExecuteWithDisabledDeliveryRecordsFailureEvidence(t *testing.T) {
	ctx := context.Background()
	st := newMonitorStore(t, ctx)
	defer st.Close()
	setupManualClearScenario(t, ctx, st)

	evaluator := &inputruntime.Evaluator{Store: st, Clock: time.Now}
	queuer := &actionruntime.Queuer{Store: st, Clock: time.Now}
	dispatcher := &actiondispatch.Dispatcher{Store: st, Clock: time.Now}
	executor := &actionexecutor.Executor{Store: st, Clock: time.Now}

	if err := runClearLatch(ctx, evaluator, queuer, dispatcher, executor, "emergency-broadcast", true, true, g2stransport.DeliverySettings{Mode: g2stransport.DeliveryModeDisabled, AllowDelivery: false}, false, false, g2stransport.ModeDisabled, false, false, ""); err != nil {
		t.Fatalf("runClearLatch: %v", err)
	}

	runs, err := st.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list action runs: %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("expected action run")
	}
	if runs[0].Status == actions.RunStatusSucceeded {
		t.Fatalf("unexpected succeeded run with disabled delivery: %+v", runs[0])
	}
	rows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 20, ActionRunID: runs[0].ID})
	if err != nil {
		t.Fatalf("list message journal: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected message journal attempt row")
	}
	if rows[0].Result != g2sengine.MessageResultSendFailed && rows[0].Result != g2sengine.MessageResultSendBlocked {
		t.Fatalf("unexpected message result: %q", rows[0].Result)
	}
	auditRows, err := st.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{Limit: 50})
	if err != nil {
		t.Fatalf("list audit timeline: %v", err)
	}
	if len(auditRows) == 0 {
		t.Fatal("expected audit evidence rows")
	}
	foundRunAudit := false
	for _, row := range auditRows {
		if row.ActionRunID == runs[0].ID {
			foundRunAudit = true
			break
		}
	}
	if !foundRunAudit {
		t.Fatalf("expected audit entries for run_id=%s", runs[0].ID)
	}
}

func TestRunClearLatchExecutePrintsEscalationQueuedWithoutAutoExecution(t *testing.T) {
	ctx := context.Background()
	st := newMonitorStore(t, ctx)
	defer st.Close()
	setupManualClearScenario(t, ctx, st)

	evaluator := &inputruntime.Evaluator{Store: st, Clock: time.Now}
	queuer := &actionruntime.Queuer{Store: st, Clock: time.Now}
	dispatcher := &actiondispatch.Dispatcher{Store: st, Clock: time.Now}
	executor := &fakeMonitorExecutor{
		results: []actionexecutor.ExecuteResult{{
			ActionRun: actions.ActionRun{
				ID:             "run-new",
				Status:         actions.RunStatusEscalated,
				ConfirmedCount: 0,
				FailedCount:    1,
			},
			EscalationRun: &actions.ActionRun{ID: "run-escalation", ActionDefinitionID: "general-broadcast-trigger"},
		}},
	}
	output := captureStdout(t, func() {
		if err := runClearLatch(ctx, evaluator, queuer, dispatcher, executor, "emergency-broadcast", true, true, g2stransport.DeliverySettings{}, false, false, g2stransport.ModeDisabled, false, false, ""); err != nil {
			t.Fatalf("runClearLatch: %v", err)
		}
	})
	if !strings.Contains(output, "escalation_queued run_id=run-escalation action_id=general-broadcast-trigger") {
		t.Fatalf("missing escalation output:\n%s", output)
	}
	if strings.Contains(strings.ToLower(output), "demo") || strings.Contains(strings.ToLower(output), "smoke") || strings.Contains(strings.ToLower(output), "fake") {
		t.Fatalf("non-product wording found in output:\n%s", output)
	}
	if len(executor.calls) != 1 {
		t.Fatalf("executor calls=%d, want 1", len(executor.calls))
	}
}

func newMonitorStore(t *testing.T, ctx context.Context) *store.SQLiteStore {
	t.Helper()
	st, err := store.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return st
}

func setupManualClearScenario(t *testing.T, ctx context.Context, st *store.SQLiteStore) {
	t.Helper()
	if err := inputpoller.EnsureDefaultPi4InputChannels(ctx, st, true); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	if err := seedActionDefinitionsAndBindings(ctx, st); err != nil {
		t.Fatalf("seed actions: %v", err)
	}
	if err := seedEGMRegistry(ctx, st, "http://127.0.0.1:18080/capture"); err != nil {
		t.Fatalf("seed egm registry: %v", err)
	}
	channel, err := st.GetInputChannel(ctx, "emergency-broadcast")
	if err != nil {
		t.Fatalf("get emergency channel: %v", err)
	}
	if channel == nil {
		t.Fatal("missing emergency channel")
	}
	channel.LatchingMode = inputs.LatchingManualClear
	channel.NormalState = inputs.InputStateHigh
	channel.CurrentState = inputs.InputStateLow
	channel.DerivedState = inputs.DerivedStateTriggered
	if err := st.UpsertInputChannel(ctx, *channel); err != nil {
		t.Fatalf("upsert emergency channel: %v", err)
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
		t.Fatalf("upsert runtime state: %v", err)
	}

	// Keep action templates/matchers product-neutral and deterministic for execution tests.
	active, err := st.GetActiveG2STemplateVersion(ctx, "template-generic-g2s-action")
	if err != nil {
		t.Fatalf("get active template version: %v", err)
	}
	if active == nil {
		t.Fatal("missing active template version")
	}
	active.ConfirmationRulesJSON = `{"rules":[{"id":"accepted","contains":["accepted"]}]}`
	active.FailureRulesJSON = `{"rules":[{"id":"rejected","contains":["rejected"]}]}`
	if err := st.UpsertG2STemplateVersion(ctx, *active); err != nil {
		t.Fatalf("upsert active template version: %v", err)
	}

	egm, err := st.GetEGMRecord(ctx, "EGM-001")
	if err != nil {
		t.Fatalf("get egm: %v", err)
	}
	if egm == nil {
		t.Fatal("missing EGM-001")
	}
	egm.IPAddress = "127.0.0.1"
	egm.EndpointPath = "/capture"
	if err := st.UpsertEGMRecord(ctx, *egm); err != nil {
		t.Fatalf("upsert egm endpoint: %v", err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		_, _ = io.Copy(&b, r)
		done <- b.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

func TestRunClearLatchExecuteUsesConfiguredDeliveryAndCanSucceed(t *testing.T) {
	ctx := context.Background()
	st := newMonitorStore(t, ctx)
	defer st.Close()
	setupManualClearScenario(t, ctx, st)

	evaluator := &inputruntime.Evaluator{Store: st, Clock: time.Now}
	queuer := &actionruntime.Queuer{Store: st, Clock: time.Now}
	dispatcher := &actiondispatch.Dispatcher{Store: st, Clock: time.Now}
	sender := &monitorSenderOK{}
	executor := &actionexecutor.Executor{Store: st, Sender: sender, Clock: time.Now}
	delivery := g2stransport.DeliverySettings{Mode: g2stransport.DeliveryModeHTTP, AllowDelivery: true, TimeoutMS: 5000}

	if err := runClearLatch(ctx, evaluator, queuer, dispatcher, executor, "emergency-broadcast", true, true, delivery, false, false, g2stransport.ModeDisabled, false, false, ""); err != nil {
		t.Fatalf("runClearLatch: %v", err)
	}
	if sender.calls == 0 {
		t.Fatal("expected sender calls")
	}
	runs, err := st.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list action runs: %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("expected action run")
	}
	if runs[0].Status != actions.RunStatusSucceeded {
		t.Fatalf("status=%q want SUCCEEDED", runs[0].Status)
	}
	rows, err := st.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 20, ActionRunID: runs[0].ID})
	if err != nil {
		t.Fatalf("list message journal: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("expected message row")
	}
	if rows[0].Result != g2sengine.MessageResultSendSucceeded {
		t.Fatalf("result=%q want SEND_SUCCEEDED", rows[0].Result)
	}
}

type monitorSenderOK struct {
	calls int
}

func (s *monitorSenderOK) Send(_ context.Context, request g2stransport.SendRequest) (g2stransport.SendResult, error) {
	s.calls++
	summary := map[string]any{"accepted": true, "message_id": request.MessageID}
	rawSummary, _ := json.Marshal(summary)
	return g2stransport.SendResult{
		MessageID:       request.MessageID,
		EGMID:           request.EGMID,
		TransportMode:   request.TransportMode,
		Sent:            true,
		Blocked:         false,
		HTTPStatusCode:  202,
		ResponseExcerpt: fmt.Sprintf("accepted %s", rawSummary),
		CompletedAt:     time.Now().UTC(),
	}, nil
}
