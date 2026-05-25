package actionexecutor

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type fakeSender struct {
	sendFn func(context.Context, g2stransport.SendRequest) (g2stransport.SendResult, error)
}

func (f *fakeSender) Send(ctx context.Context, req g2stransport.SendRequest) (g2stransport.SendResult, error) {
	if f.sendFn == nil {
		return g2stransport.SendResult{}, fmt.Errorf("sendFn is required")
	}
	return f.sendFn(ctx, req)
}

type fakeStore struct {
	runs       map[string]actions.ActionRun
	defs       map[string]actions.ActionDefinition
	targetRows map[string][]actions.ActionTargetResult
	egmsByID   map[string]egms.EGMRecord
	templates  map[string]templates.G2STemplate
	versions   map[string]templates.G2STemplateVersion
	groups     map[string]egms.EGMGroup

	messages      []g2sengine.MessageJournalEntry
	audits        []audit.AuditTimelineEntry
	runStatusLog  []actions.ActionRunStatus
	nextTargetID  int64
	nextMessageID int64
}

func (s *fakeStore) GetActionRun(_ context.Context, id string) (*actions.ActionRun, error) {
	run, ok := s.runs[id]
	if !ok {
		return nil, nil
	}
	copy := run
	return &copy, nil
}

func (s *fakeStore) UpdateActionRun(_ context.Context, run actions.ActionRun) error {
	s.runs[run.ID] = run
	s.runStatusLog = append(s.runStatusLog, run.Status)
	return nil
}

func (s *fakeStore) GetActionDefinition(_ context.Context, id string) (*actions.ActionDefinition, error) {
	row, ok := s.defs[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (s *fakeStore) ListActionTargetResults(_ context.Context, actionRunID string) ([]actions.ActionTargetResult, error) {
	rows := make([]actions.ActionTargetResult, len(s.targetRows[actionRunID]))
	copy(rows, s.targetRows[actionRunID])
	return rows, nil
}

func (s *fakeStore) UpdateActionTargetResult(_ context.Context, row actions.ActionTargetResult) error {
	rows := s.targetRows[row.ActionRunID]
	for i := range rows {
		if rows[i].ID == row.ID {
			rows[i] = row
			s.targetRows[row.ActionRunID] = rows
			return nil
		}
	}
	return fmt.Errorf("target result id %d not found", row.ID)
}

func (s *fakeStore) GetEGMRecord(_ context.Context, egmID string) (*egms.EGMRecord, error) {
	row, ok := s.egmsByID[egmID]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (s *fakeStore) GetG2STemplate(_ context.Context, id string) (*templates.G2STemplate, error) {
	row, ok := s.templates[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (s *fakeStore) GetActiveG2STemplateVersion(_ context.Context, templateID string) (*templates.G2STemplateVersion, error) {
	row, ok := s.versions[templateID]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (s *fakeStore) RecordMessageJournalEntry(_ context.Context, row g2sengine.MessageJournalEntry) (int64, error) {
	s.nextMessageID++
	row.ID = s.nextMessageID
	s.messages = append(s.messages, row)
	return row.ID, nil
}

func (s *fakeStore) UpdateMessageJournalResult(_ context.Context, id int64, result g2sengine.MessageResult, errText string, responseExcerpt string, httpStatusCode int, latencyMS int, transportMode string, sentAt *time.Time, completedAt *time.Time) error {
	for i := range s.messages {
		if s.messages[i].ID != id {
			continue
		}
		s.messages[i].Result = result
		s.messages[i].Error = errText
		s.messages[i].ResponseExcerpt = responseExcerpt
		s.messages[i].HTTPStatusCode = httpStatusCode
		s.messages[i].LatencyMS = latencyMS
		s.messages[i].TransportMode = transportMode
		s.messages[i].SentAt = sentAt
		s.messages[i].CompletedAt = completedAt
		return nil
	}
	return fmt.Errorf("message %d not found", id)
}

func (s *fakeStore) RecordAuditTimelineEntry(_ context.Context, row audit.AuditTimelineEntry) (int64, error) {
	row.ID = int64(len(s.audits) + 1)
	s.audits = append(s.audits, row)
	return row.ID, nil
}

func (s *fakeStore) ListEGMRecords(_ context.Context) ([]egms.EGMRecord, error) {
	rows := make([]egms.EGMRecord, 0, len(s.egmsByID))
	for _, row := range s.egmsByID {
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *fakeStore) GetEGMGroup(_ context.Context, id string) (*egms.EGMGroup, error) {
	row, ok := s.groups[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (s *fakeStore) ListEGMGroups(_ context.Context) ([]egms.EGMGroup, error) {
	rows := make([]egms.EGMGroup, 0, len(s.groups))
	for _, row := range s.groups {
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *fakeStore) CreateActionRun(_ context.Context, run actions.ActionRun) (actions.ActionRun, error) {
	s.runs[run.ID] = run
	return run, nil
}

func (s *fakeStore) CreateActionTargetResult(_ context.Context, row actions.ActionTargetResult) (actions.ActionTargetResult, error) {
	s.nextTargetID++
	row.ID = s.nextTargetID
	s.targetRows[row.ActionRunID] = append(s.targetRows[row.ActionRunID], row)
	return row, nil
}

func TestExecuteMissingRunReturnsError(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	delete(st.runs, "run-1")
	executor := Executor{Store: st}

	_, err := executor.Execute(context.Background(), ExecuteRequest{ActionRunID: "run-1"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestExecuteRejectsNonPendingRun(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	run := st.runs["run-1"]
	run.Status = actions.RunStatusDispatchPrepared
	st.runs["run-1"] = run

	executor := Executor{Store: st}
	_, err := executor.Execute(context.Background(), ExecuteRequest{ActionRunID: "run-1"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "must be pending") {
		t.Fatalf("expected pending status error, got %v", err)
	}
}

func TestExecuteSuccessConfirmsTargetAndRunSucceeded(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	executor := Executor{
		Store: st,
		Sender: &fakeSender{sendFn: func(_ context.Context, req g2stransport.SendRequest) (g2stransport.SendResult, error) {
			return g2stransport.SendResult{
				MessageID:       req.MessageID,
				EGMID:           req.EGMID,
				TransportMode:   req.TransportMode,
				Sent:            true,
				HTTPStatusCode:  202,
				ResponseExcerpt: "<ack>accepted</ack>",
				CompletedAt:     now,
			}, nil
		}},
		Clock: func() time.Time { return now },
		Sleep: func(time.Duration) {},
	}

	result, err := executor.Execute(context.Background(), ExecuteRequest{
		ActionRunID: "run-1",
		Actor:       "operator",
		Delivery:    enabledDeliverySettings(),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(st.runStatusLog) == 0 || st.runStatusLog[0] != actions.RunStatusRunning {
		t.Fatalf("expected first run status update to RUNNING: %+v", st.runStatusLog)
	}
	if result.ActionRun.Status != actions.RunStatusSucceeded {
		t.Fatalf("run status=%q, want %q", result.ActionRun.Status, actions.RunStatusSucceeded)
	}
	if len(result.TargetResults) != 1 || result.TargetResults[0].Status != actions.TargetStatusConfirmed {
		t.Fatalf("unexpected target rows: %+v", result.TargetResults)
	}
	if len(result.Attempts) == 0 || result.Attempts[0].MatchOutcome != string(g2sengine.MatchOutcomeExpected) {
		t.Fatalf("expected expected-match attempt summary: %+v", result.Attempts)
	}
}

func TestExecuteFailureMatcherMarksRunFailed(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	executor := Executor{
		Store: st,
		Sender: &fakeSender{sendFn: func(_ context.Context, req g2stransport.SendRequest) (g2stransport.SendResult, error) {
			return g2stransport.SendResult{
				MessageID:       req.MessageID,
				EGMID:           req.EGMID,
				TransportMode:   req.TransportMode,
				Sent:            true,
				HTTPStatusCode:  400,
				ResponseExcerpt: "<ack>rejected</ack>",
				CompletedAt:     now,
			}, nil
		}},
		Clock: func() time.Time { return now },
		Sleep: func(time.Duration) {},
	}

	result, err := executor.Execute(context.Background(), ExecuteRequest{
		ActionRunID: "run-1",
		Actor:       "operator",
		Delivery:    enabledDeliverySettings(),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.ActionRun.Status != actions.RunStatusFailed {
		t.Fatalf("run status=%q, want %q", result.ActionRun.Status, actions.RunStatusFailed)
	}
	if len(result.TargetResults) != 1 || result.TargetResults[0].Status != actions.TargetStatusFailed {
		t.Fatalf("unexpected target rows: %+v", result.TargetResults)
	}
}

func TestExecuteDeliveryFailureRetriesAndCreatesEvidence(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	def := st.defs["action-1"]
	def.RetryPolicyJSON = `{"count":2,"delay_ms":0}`
	st.defs["action-1"] = def

	sendCalls := 0
	executor := Executor{
		Store: st,
		Sender: &fakeSender{sendFn: func(_ context.Context, req g2stransport.SendRequest) (g2stransport.SendResult, error) {
			sendCalls++
			return g2stransport.SendResult{
				MessageID:     req.MessageID,
				EGMID:         req.EGMID,
				TransportMode: req.TransportMode,
				Sent:          false,
				Error:         "network unavailable",
				CompletedAt:   now,
			}, nil
		}},
		Clock: func() time.Time { return now },
		Sleep: func(time.Duration) {},
	}

	result, err := executor.Execute(context.Background(), ExecuteRequest{
		ActionRunID: "run-1",
		Delivery:    enabledDeliverySettings(),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if sendCalls != 3 {
		t.Fatalf("send calls=%d, want 3", sendCalls)
	}
	if result.ActionRun.Status != actions.RunStatusFailed {
		t.Fatalf("run status=%q, want %q", result.ActionRun.Status, actions.RunStatusFailed)
	}
	if len(st.messages) != 3 {
		t.Fatalf("message journal rows=%d, want 3", len(st.messages))
	}
	retryAuditCount := 0
	for _, row := range st.audits {
		if row.EventType == audit.EventTypeRetry {
			retryAuditCount++
		}
	}
	if retryAuditCount == 0 {
		t.Fatal("expected retry audit events")
	}
}

func TestExecuteQueuesEscalationAfterFailure(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	base := st.defs["action-1"]
	base.RetryPolicyJSON = `{"count":0,"delay_ms":0}`
	base.EscalationJSON = `{"escalation_action_id":"action-escalate","after_attempts":1}`
	st.defs["action-1"] = base
	st.defs["action-escalate"] = actions.ActionDefinition{
		ID:               "action-escalate",
		Name:             "Escalation Action",
		Severity:         actions.SeverityEmergency,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps: []actions.ActionStep{{
			ID:                "step-escalate",
			Name:              "Escalate",
			Sequence:          0,
			TemplateActionKey: "emergency_broadcast_silence",
		}},
		Version: 1,
	}

	executor := Executor{
		Store: st,
		Sender: &fakeSender{sendFn: func(_ context.Context, req g2stransport.SendRequest) (g2stransport.SendResult, error) {
			return g2stransport.SendResult{
				MessageID:       req.MessageID,
				EGMID:           req.EGMID,
				TransportMode:   req.TransportMode,
				Sent:            true,
				HTTPStatusCode:  400,
				ResponseExcerpt: "<ack>rejected</ack>",
				CompletedAt:     now,
			}, nil
		}},
		Clock: func() time.Time { return now },
		Sleep: func(time.Duration) {},
	}

	result, err := executor.Execute(context.Background(), ExecuteRequest{
		ActionRunID: "run-1",
		Actor:       "operator",
		Delivery:    enabledDeliverySettings(),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.ActionRun.Status != actions.RunStatusEscalated {
		t.Fatalf("run status=%q, want %q", result.ActionRun.Status, actions.RunStatusEscalated)
	}
	if result.EscalationRun == nil {
		t.Fatal("expected escalation run")
	}
	if result.EscalationRun.ActionDefinitionID != "action-escalate" {
		t.Fatalf("unexpected escalation run: %+v", result.EscalationRun)
	}
}

func TestExecuteNoSenderFailsWithoutPretendingSuccess(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	executor := Executor{
		Store: st,
		Clock: func() time.Time { return now },
		Sleep: func(time.Duration) {},
	}

	result, err := executor.Execute(context.Background(), ExecuteRequest{ActionRunID: "run-1"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.ActionRun.Status != actions.RunStatusFailed {
		t.Fatalf("run status=%q, want %q", result.ActionRun.Status, actions.RunStatusFailed)
	}
	if len(result.Attempts) == 0 || !strings.Contains(strings.ToLower(result.Attempts[0].Error), "sender") {
		t.Fatalf("expected sender configuration error in attempts: %+v", result.Attempts)
	}
}

func TestExecuteDefaultDeliverySettingsDoNotSilentlySend(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	calls := 0
	executor := Executor{
		Store: st,
		Sender: &fakeSender{sendFn: func(_ context.Context, req g2stransport.SendRequest) (g2stransport.SendResult, error) {
			calls++
			if req.AllowRealSend {
				t.Fatalf("allow_real_send must be false by default: %+v", req)
			}
			if req.TransportMode != g2stransport.ModeDisabled {
				t.Fatalf("transport mode must be disabled by default: %q", req.TransportMode)
			}
			return g2stransport.SendResult{
				MessageID:     req.MessageID,
				EGMID:         req.EGMID,
				TransportMode: req.TransportMode,
				Blocked:       true,
				Sent:          false,
				Error:         "send blocked: allow_real_send is false",
				CompletedAt:   now,
			}, nil
		}},
		Clock: func() time.Time { return now },
		Sleep: func(time.Duration) {},
	}

	result, err := executor.Execute(context.Background(), ExecuteRequest{ActionRunID: "run-1"})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if calls == 0 {
		t.Fatal("expected sender to be called")
	}
	if result.ActionRun.Status != actions.RunStatusFailed {
		t.Fatalf("run status=%q, want %q", result.ActionRun.Status, actions.RunStatusFailed)
	}
}

func TestExecuteUsesConfiguredDeliverySettings(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	captured := g2stransport.SendRequest{}
	executor := Executor{
		Store: st,
		Sender: &fakeSender{sendFn: func(_ context.Context, req g2stransport.SendRequest) (g2stransport.SendResult, error) {
			captured = req
			return g2stransport.SendResult{
				MessageID:       req.MessageID,
				EGMID:           req.EGMID,
				TransportMode:   req.TransportMode,
				Sent:            true,
				HTTPStatusCode:  200,
				ResponseExcerpt: "<ack>accepted</ack>",
				CompletedAt:     now,
			}, nil
		}},
		Clock: func() time.Time { return now },
		Sleep: func(time.Duration) {},
	}

	settings := g2stransport.DeliverySettings{
		Mode:          g2stransport.DeliveryModeHTTP,
		AllowDelivery: true,
		CaptureOnly:   false,
		TimeoutMS:     4321,
	}
	_, err := executor.Execute(context.Background(), ExecuteRequest{
		ActionRunID: "run-1",
		Delivery:    settings,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if captured.TransportMode != g2stransport.ModeHTTP || !captured.AllowRealSend || captured.CaptureOnlySend || captured.TimeoutMS != 4321 {
		t.Fatalf("send request did not reflect delivery settings: %+v", captured)
	}
}

func TestExecuteRestoreActionUsesSamePath(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	st.runs["run-restore"] = actions.ActionRun{
		ID:                 "run-restore",
		ActionDefinitionID: "action-restore",
		StartedAt:          now,
		Status:             actions.RunStatusPending,
		TargetCount:        1,
	}
	st.targetRows["run-restore"] = []actions.ActionTargetResult{{
		ID:          2,
		ActionRunID: "run-restore",
		TargetEGMID: "EGM-001",
		Status:      actions.TargetStatusPending,
	}}
	st.defs["action-restore"] = actions.ActionDefinition{
		ID:               "action-restore",
		Name:             "Emergency Broadcast Restore",
		Severity:         actions.SeverityRestore,
		Enabled:          true,
		TargetSelector:   "ALL_EMERGENCY_ENABLED",
		TemplateSelector: "template-by-egm",
		Steps: []actions.ActionStep{{
			ID:                "restore-step",
			Name:              "Restore",
			Sequence:          0,
			TemplateActionKey: "emergency_broadcast_restore",
		}},
		Version: 1,
	}

	executor := Executor{
		Store: st,
		Sender: &fakeSender{sendFn: func(_ context.Context, req g2stransport.SendRequest) (g2stransport.SendResult, error) {
			return g2stransport.SendResult{
				MessageID:       req.MessageID,
				EGMID:           req.EGMID,
				TransportMode:   req.TransportMode,
				Sent:            true,
				HTTPStatusCode:  200,
				ResponseExcerpt: "<ack>accepted</ack>",
				CompletedAt:     now,
			}, nil
		}},
		Clock: func() time.Time { return now },
		Sleep: func(time.Duration) {},
	}

	result, err := executor.Execute(context.Background(), ExecuteRequest{
		ActionRunID: "run-restore",
		Delivery:    enabledDeliverySettings(),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.ActionRun.Status != actions.RunStatusSucceeded {
		t.Fatalf("restore run status=%q, want %q", result.ActionRun.Status, actions.RunStatusSucceeded)
	}
}

func enabledDeliverySettings() g2stransport.DeliverySettings {
	return g2stransport.DeliverySettings{
		Mode:          g2stransport.DeliveryModeHTTP,
		AllowDelivery: true,
		CaptureOnly:   false,
		TimeoutMS:     5000,
	}
}

func newFakeStore(now time.Time) *fakeStore {
	return &fakeStore{
		runs: map[string]actions.ActionRun{
			"run-1": {
				ID:                 "run-1",
				ActionDefinitionID: "action-1",
				InputTransitionID:  10,
				StartedAt:          now,
				Status:             actions.RunStatusPending,
				TriggerReason:      "input transition 10",
				TargetCount:        1,
			},
		},
		defs: map[string]actions.ActionDefinition{
			"action-1": {
				ID:               "action-1",
				Name:             "Emergency Broadcast Trigger",
				Severity:         actions.SeverityEmergency,
				Enabled:          true,
				TargetSelector:   "ALL_EMERGENCY_ENABLED",
				TemplateSelector: "template-by-egm",
				Steps: []actions.ActionStep{{
					ID:                "step-1",
					Name:              "Emergency Silence",
					Sequence:          0,
					TemplateActionKey: "emergency_broadcast_silence",
				}},
				Version: 1,
			},
		},
		targetRows: map[string][]actions.ActionTargetResult{
			"run-1": {{
				ID:           1,
				ActionRunID:  "run-1",
				TargetEGMID:  "EGM-001",
				Status:       actions.TargetStatusPending,
				AttemptCount: 0,
			}},
		},
		egmsByID: map[string]egms.EGMRecord{
			"EGM-001": {
				EGMID:              "EGM-001",
				DisplayName:        "Cabinet 001",
				IPAddress:          "127.0.0.1",
				EndpointPath:       "/capture",
				Enabled:            true,
				EmergencyEnabled:   true,
				TemplateID:         "template-generic-g2s-action",
				CurrentActionState: egms.EGMActionStateNormal,
			},
		},
		templates: map[string]templates.G2STemplate{
			"template-generic-g2s-action": {
				ID:               "template-generic-g2s-action",
				Name:             "Generic G2S Action Template",
				Vendor:           "Generic",
				Status:           templates.TemplateStatusActive,
				CurrentVersionID: "1",
			},
		},
		versions: map[string]templates.G2STemplateVersion{
			"template-generic-g2s-action": {
				ID:                    "template-generic-g2s-action-v1",
				TemplateID:            "template-generic-g2s-action",
				VersionLabel:          "1",
				ActionsJSON:           `{"actions":{"emergency_broadcast_silence":{"message_type":"EMERGENCY_SILENCE","content_type":"application/xml","payload_template":"<cmd egm=\"{{.EGMID}}\" run=\"{{.ActionRunID}}\"/>"},"emergency_broadcast_restore":{"message_type":"EMERGENCY_RESTORE","content_type":"application/xml","payload_template":"<restore egm=\"{{.EGMID}}\" run=\"{{.ActionRunID}}\"/>"}}}`,
				ConfirmationRulesJSON: `{"rules":[{"id":"accepted","contains":["accepted"]}]}`,
				FailureRulesJSON:      `{"rules":[{"id":"rejected","contains":["rejected"]}]}`,
			},
		},
		groups:        map[string]egms.EGMGroup{},
		messages:      []g2sengine.MessageJournalEntry{},
		audits:        []audit.AuditTimelineEntry{},
		nextTargetID:  1,
		nextMessageID: 0,
	}
}
