package inbound

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type fakeStore struct {
	runs      map[string]actions.ActionRun
	targets   map[string][]actions.ActionTargetResult
	defs      map[string]actions.ActionDefinition
	egms      map[string]egms.EGMRecord
	versions  map[string]templates.G2STemplateVersion
	rules     []g2sengine.HandlerRule
	messages  []g2sengine.MessageJournalEntry
	auditRows []audit.AuditTimelineEntry
}

func (f *fakeStore) RecordMessageJournalEntry(_ context.Context, entry g2sengine.MessageJournalEntry) (int64, error) {
	entry.ID = int64(len(f.messages) + 1)
	f.messages = append(f.messages, entry)
	return entry.ID, nil
}

func (f *fakeStore) UpdateMessageJournalHandlerRule(_ context.Context, id int64, handlerRuleID string) error {
	for i := range f.messages {
		if f.messages[i].ID == id {
			f.messages[i].HandlerRuleID = handlerRuleID
		}
	}
	return nil
}

func (f *fakeStore) RecordAuditTimelineEntry(_ context.Context, entry audit.AuditTimelineEntry) (int64, error) {
	entry.ID = int64(len(f.auditRows) + 1)
	f.auditRows = append(f.auditRows, entry)
	return entry.ID, nil
}

func (f *fakeStore) GetActionRun(_ context.Context, id string) (*actions.ActionRun, error) {
	run, ok := f.runs[id]
	if !ok {
		return nil, nil
	}
	copy := run
	return &copy, nil
}

func (f *fakeStore) UpdateActionRun(_ context.Context, run actions.ActionRun) error {
	f.runs[run.ID] = run
	return nil
}

func (f *fakeStore) ListActionRuns(_ context.Context, query store.ActionRunListQuery) ([]actions.ActionRun, error) {
	rows := []actions.ActionRun{}
	for _, run := range f.runs {
		if query.Status != "" && run.Status != query.Status {
			continue
		}
		rows = append(rows, run)
	}
	return rows, nil
}

func (f *fakeStore) ListActionTargetResults(_ context.Context, actionRunID string) ([]actions.ActionTargetResult, error) {
	rows := append([]actions.ActionTargetResult{}, f.targets[actionRunID]...)
	return rows, nil
}

func (f *fakeStore) UpdateActionTargetResult(_ context.Context, row actions.ActionTargetResult) error {
	rows := f.targets[row.ActionRunID]
	for i := range rows {
		if rows[i].ID == row.ID {
			rows[i] = row
			f.targets[row.ActionRunID] = rows
			return nil
		}
	}
	return nil
}

func (f *fakeStore) GetActionDefinition(_ context.Context, id string) (*actions.ActionDefinition, error) {
	def, ok := f.defs[id]
	if !ok {
		return nil, nil
	}
	copy := def
	return &copy, nil
}

func (f *fakeStore) GetEGMRecord(_ context.Context, egmID string) (*egms.EGMRecord, error) {
	row, ok := f.egms[egmID]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) GetActiveG2STemplateVersion(_ context.Context, templateID string) (*templates.G2STemplateVersion, error) {
	row, ok := f.versions[templateID]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) ListEnabledHandlerRules(_ context.Context, _ int) ([]g2sengine.HandlerRule, error) {
	rows := []g2sengine.HandlerRule{}
	for _, row := range f.rules {
		if row.Enabled {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func TestInboundJournalsMessageEvenWhenParsingFails(t *testing.T) {
	store := newInboundStoreFixture()
	svc := &Service{Store: store, Clock: fixedClock()}

	result, err := svc.Process(context.Background(), InboundMessage{
		RawPayload:   `<<not-xml`,
		RemoteAddr:   "10.0.0.5:9999",
		FromEndpoint: "10.0.0.5:9999",
		ToEndpoint:   "/g2s",
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if result.MessageID == 0 {
		t.Fatal("expected message id")
	}
	if len(store.messages) != 1 {
		t.Fatalf("message rows=%d want 1", len(store.messages))
	}
	if store.messages[0].Direction != g2sengine.DirectionInbound {
		t.Fatalf("direction=%q", store.messages[0].Direction)
	}
}

func TestInboundXMLParsesEGMIDAndJournals(t *testing.T) {
	store := newInboundStoreFixture()
	svc := &Service{Store: store, Clock: fixedClock()}

	_, err := svc.Process(context.Background(), InboundMessage{
		RawPayload: `<g2sBody egmId="EGM-001"><keepAlive/></g2sBody>`,
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	row := store.messages[0]
	if row.EGMID != "EGM-001" {
		t.Fatalf("egm_id=%q", row.EGMID)
	}
	if row.MessageType != "keepAlive" {
		t.Fatalf("message_type=%q", row.MessageType)
	}
}

func TestInboundJSONParsesEGMIDAndActionRunID(t *testing.T) {
	store := newInboundStoreFixture()
	svc := &Service{Store: store, Clock: fixedClock()}

	_, err := svc.Process(context.Background(), InboundMessage{
		RawPayload: `{"egm_id":"EGM-001","action_run_id":"run-1","message_type":"ACK","status":"accepted"}`,
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	row := store.messages[0]
	if row.EGMID != "EGM-001" || row.ActionRunID != "run-1" {
		t.Fatalf("unexpected row: %+v", row)
	}
}

func TestInboundExpectedMatcherConfirmsTargetAndRunSucceeded(t *testing.T) {
	store := newInboundStoreFixture()
	// second target stays pending for now
	store.targets["run-1"][1].Status = actions.TargetStatusPending
	svc := &Service{Store: store, Clock: fixedClock()}

	_, err := svc.Process(context.Background(), InboundMessage{
		RawPayload:  `<ack status="accepted"/>`,
		EGMID:       "EGM-001",
		ActionRunID: "run-1",
		MessageType: "ACK",
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	first := store.targets["run-1"][0]
	if first.Status != actions.TargetStatusConfirmed {
		t.Fatalf("target status=%q", first.Status)
	}
	run := store.runs["run-1"]
	if run.Status == actions.RunStatusSucceeded {
		t.Fatalf("run should not be succeeded until all targets confirmed")
	}

	_, err = svc.Process(context.Background(), InboundMessage{
		RawPayload:  `<ack status="accepted"/>`,
		EGMID:       "EGM-002",
		ActionRunID: "run-1",
		MessageType: "ACK",
	})
	if err != nil {
		t.Fatalf("process second: %v", err)
	}
	run = store.runs["run-1"]
	if run.Status != actions.RunStatusSucceeded {
		t.Fatalf("run status=%q want SUCCEEDED", run.Status)
	}
}

func TestInboundFailureMatcherMarksTargetFailed(t *testing.T) {
	store := newInboundStoreFixture()
	svc := &Service{Store: store, Clock: fixedClock()}

	_, err := svc.Process(context.Background(), InboundMessage{
		RawPayload:  `<ack status="rejected"/>`,
		EGMID:       "EGM-001",
		ActionRunID: "run-1",
		MessageType: "ACK",
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	target := store.targets["run-1"][0]
	if target.Status != actions.TargetStatusFailed {
		t.Fatalf("target status=%q want FAILED", target.Status)
	}
}

func TestInboundNoMatchLeavesTargetUnchanged(t *testing.T) {
	store := newInboundStoreFixture()
	svc := &Service{Store: store, Clock: fixedClock()}
	before := store.targets["run-1"][0].Status

	result, err := svc.Process(context.Background(), InboundMessage{
		RawPayload:  `<ack status="pending"/>`,
		EGMID:       "EGM-001",
		ActionRunID: "run-1",
		MessageType: "ACK",
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	after := store.targets["run-1"][0].Status
	if before != after {
		t.Fatalf("target changed: before=%q after=%q", before, after)
	}
	if result.MatchOutcome != string(g2sengine.MatchOutcomeNoMatch) {
		t.Fatalf("match outcome=%q", result.MatchOutcome)
	}
}

func TestInboundAmbiguousCorrelationJournalsButDoesNotUpdate(t *testing.T) {
	store := newInboundStoreFixture()
	store.runs["run-2"] = actions.ActionRun{
		ID:                 "run-2",
		ActionDefinitionID: "emergency-broadcast-trigger",
		StartedAt:          fixedClock()(),
		Status:             actions.RunStatusRunning,
		TargetCount:        1,
	}
	store.targets["run-2"] = []actions.ActionTargetResult{
		{ID: 3, ActionRunID: "run-2", TargetEGMID: "EGM-001", Status: actions.TargetStatusPending},
	}
	svc := &Service{Store: store, Clock: fixedClock()}

	result, err := svc.Process(context.Background(), InboundMessage{
		RawPayload: `<ack status="accepted"/>`,
		EGMID:      "EGM-001",
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if result.TargetUpdated {
		t.Fatal("target should not update for ambiguous correlation")
	}
	if len(store.messages) != 1 {
		t.Fatalf("message rows=%d want 1", len(store.messages))
	}
	if store.targets["run-1"][0].Status != actions.TargetStatusPending {
		t.Fatalf("target unexpectedly changed: %q", store.targets["run-1"][0].Status)
	}
}

func TestInboundHandlerRuleConfirmationConfirmsTarget(t *testing.T) {
	store := newInboundStoreFixture()
	store.rules = []g2sengine.HandlerRule{
		{
			ID:        "rule-confirm",
			Name:      "Confirm ACK",
			Enabled:   true,
			Direction: g2sengine.HandlerRuleDirectionInbound,
			MatchJSON: `{"contains":["accepted"]}`,
			Outcome:   g2sengine.HandlerRuleOutcomeConfirmation,
		},
	}
	svc := &Service{Store: store, Clock: fixedClock()}

	result, err := svc.Process(context.Background(), InboundMessage{
		RawPayload:  `<ack status="accepted"/>`,
		EGMID:       "EGM-001",
		ActionRunID: "run-1",
		MessageType: "ACK",
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !result.TargetUpdated || result.TargetStatus != string(actions.TargetStatusConfirmed) {
		t.Fatalf("expected confirmed target update, got %+v", result)
	}
	if store.messages[0].HandlerRuleID != "rule-confirm" {
		t.Fatalf("handler_rule_id=%q", store.messages[0].HandlerRuleID)
	}
}

func TestInboundHandlerRuleFailureFailsTarget(t *testing.T) {
	store := newInboundStoreFixture()
	store.rules = []g2sengine.HandlerRule{
		{
			ID:        "rule-fail",
			Name:      "Fail ACK",
			Enabled:   true,
			Direction: g2sengine.HandlerRuleDirectionInbound,
			MatchJSON: `{"contains":["rejected"]}`,
			Outcome:   g2sengine.HandlerRuleOutcomeFailure,
		},
	}
	svc := &Service{Store: store, Clock: fixedClock()}

	result, err := svc.Process(context.Background(), InboundMessage{
		RawPayload:  `<ack status="rejected"/>`,
		EGMID:       "EGM-001",
		ActionRunID: "run-1",
		MessageType: "ACK",
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !result.TargetUpdated || result.TargetStatus != string(actions.TargetStatusFailed) {
		t.Fatalf("expected failed target update, got %+v", result)
	}
}

func TestInboundHandlerRuleIgnoreDoesNotMutateTarget(t *testing.T) {
	store := newInboundStoreFixture()
	store.rules = []g2sengine.HandlerRule{
		{
			ID:        "rule-ignore",
			Name:      "Ignore pending",
			Enabled:   true,
			Direction: g2sengine.HandlerRuleDirectionInbound,
			MatchJSON: `{"contains":["pending"]}`,
			Outcome:   g2sengine.HandlerRuleOutcomeIgnore,
		},
	}
	svc := &Service{Store: store, Clock: fixedClock()}
	before := store.targets["run-1"][0].Status

	result, err := svc.Process(context.Background(), InboundMessage{
		RawPayload:  `<ack status="pending"/>`,
		EGMID:       "EGM-001",
		ActionRunID: "run-1",
		MessageType: "ACK",
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if result.TargetUpdated {
		t.Fatalf("target unexpectedly updated: %+v", result)
	}
	after := store.targets["run-1"][0].Status
	if before != after {
		t.Fatalf("target status changed: before=%q after=%q", before, after)
	}
	if store.messages[0].HandlerRuleID != "rule-ignore" {
		t.Fatalf("handler_rule_id=%q", store.messages[0].HandlerRuleID)
	}
}

func TestInboundAuditEntriesRecordedForReceiveAndOutcome(t *testing.T) {
	store := newInboundStoreFixture()
	svc := &Service{Store: store, Clock: fixedClock()}

	result, err := svc.Process(context.Background(), InboundMessage{
		RawPayload:  `<ack status="accepted"/>`,
		EGMID:       "EGM-001",
		ActionRunID: "run-1",
		MessageType: "ACK",
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(result.AuditEntryIDs) < 2 {
		t.Fatalf("audit ids=%d want >=2", len(result.AuditEntryIDs))
	}
	foundReceive := false
	foundConfirmation := false
	for _, row := range store.auditRows {
		if row.EventType == audit.EventTypeMessageReceived {
			foundReceive = true
		}
		if row.EventType == audit.EventTypeConfirmation {
			foundConfirmation = true
		}
	}
	if !foundReceive || !foundConfirmation {
		t.Fatalf("missing receive/confirmation audit rows: %+v", store.auditRows)
	}
}

func newInboundStoreFixture() *fakeStore {
	now := fixedClock()()
	return &fakeStore{
		runs: map[string]actions.ActionRun{
			"run-1": {
				ID:                 "run-1",
				ActionDefinitionID: "emergency-broadcast-trigger",
				StartedAt:          now,
				Status:             actions.RunStatusRunning,
				TargetCount:        2,
			},
		},
		targets: map[string][]actions.ActionTargetResult{
			"run-1": {
				{ID: 1, ActionRunID: "run-1", TargetEGMID: "EGM-001", Status: actions.TargetStatusPending},
				{ID: 2, ActionRunID: "run-1", TargetEGMID: "EGM-002", Status: actions.TargetStatusPending},
			},
		},
		defs: map[string]actions.ActionDefinition{
			"emergency-broadcast-trigger": {
				ID:               "emergency-broadcast-trigger",
				Name:             "Emergency Broadcast Trigger",
				Severity:         actions.SeverityEmergency,
				Enabled:          true,
				TargetSelector:   "ALL_EMERGENCY_ENABLED",
				TemplateSelector: "template-by-egm",
				Steps: []actions.ActionStep{{
					ID:                "step-1",
					Name:              "Silence",
					Sequence:          0,
					TemplateActionKey: "emergency_broadcast_silence",
				}},
				Version: 1,
			},
		},
		egms: map[string]egms.EGMRecord{
			"EGM-001": {EGMID: "EGM-001", TemplateID: "template-generic-g2s-action", Enabled: true, EmergencyEnabled: true},
			"EGM-002": {EGMID: "EGM-002", TemplateID: "template-generic-g2s-action", Enabled: true, EmergencyEnabled: true},
		},
		versions: map[string]templates.G2STemplateVersion{
			"template-generic-g2s-action": {
				ID:                    "template-generic-g2s-action-v1",
				TemplateID:            "template-generic-g2s-action",
				VersionLabel:          "1",
				ActionsJSON:           `{"actions":{"emergency_broadcast_silence":{"message_type":"NOTICE","payload_template":"<x/>"}}}`,
				ConfirmationRulesJSON: `{"rules":[{"id":"ok","contains":["accepted"]}]}`,
				FailureRulesJSON:      `{"rules":[{"id":"bad","contains":["rejected"]}]}`,
			},
		},
		rules: []g2sengine.HandlerRule{},
	}
}

func fixedClock() func() time.Time {
	return func() time.Time {
		return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	}
}

func TestSummaryJSONProduced(t *testing.T) {
	store := newInboundStoreFixture()
	svc := &Service{Store: store, Clock: fixedClock()}
	_, err := svc.Process(context.Background(), InboundMessage{
		RawPayload: `{"egm_id":"EGM-001","action_run_id":"run-1","message_type":"ACK","status":"accepted"}`,
	})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(store.messages[0].ParsedSummaryJSON), &payload); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if payload["egm_id"] != "EGM-001" {
		t.Fatalf("summary egm_id=%v", payload["egm_id"])
	}
}
