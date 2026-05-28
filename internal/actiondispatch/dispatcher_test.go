package actiondispatch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type fakeStore struct {
	runs       map[string]actions.ActionRun
	defs       map[string]actions.ActionDefinition
	targetRows map[string][]actions.ActionTargetResult
	egmsByID   map[string]egms.EGMRecord
	templates  map[string]templates.G2STemplate
	versions   map[string]templates.G2STemplateVersion

	messages []g2sengine.MessageJournalEntry
	audits   []audit.AuditTimelineEntry
}

type fakeSender struct {
	sendFn func(context.Context, g2stransport.SendRequest) (g2stransport.SendResult, error)
}

func (s *fakeSender) Send(ctx context.Context, request g2stransport.SendRequest) (g2stransport.SendResult, error) {
	if s.sendFn == nil {
		return g2stransport.SendResult{}, fmt.Errorf("sendFn not configured")
	}
	return s.sendFn(ctx, request)
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

func (f *fakeStore) GetActiveG2STemplateVersion(_ context.Context, templateID string) (*templates.G2STemplateVersion, error) {
	row, ok := f.versions[templateID]
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

func (f *fakeStore) ListMessageJournalEntries(_ context.Context, query store.MessageJournalListQuery) ([]g2sengine.MessageJournalEntry, error) {
	rows := []g2sengine.MessageJournalEntry{}
	for _, row := range f.messages {
		if query.ActionRunID != "" && row.ActionRunID != query.ActionRunID {
			continue
		}
		if query.Direction != "" && row.Direction != query.Direction {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *fakeStore) UpdateMessageJournalResult(_ context.Context, id int64, result g2sengine.MessageResult, errText string, responseExcerpt string, httpStatusCode int, latencyMS int, transportMode string, sentAt *time.Time, completedAt *time.Time) error {
	for i := range f.messages {
		if f.messages[i].ID != id {
			continue
		}
		f.messages[i].Result = result
		f.messages[i].Error = errText
		f.messages[i].ResponseExcerpt = responseExcerpt
		f.messages[i].HTTPStatusCode = httpStatusCode
		f.messages[i].LatencyMS = latencyMS
		f.messages[i].TransportMode = transportMode
		f.messages[i].SentAt = sentAt
		f.messages[i].CompletedAt = completedAt
		return nil
	}
	return fmt.Errorf("message id %d not found", id)
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
		if !strings.Contains(row.RawPayload, `egm="EGM-`) {
			t.Fatalf("expected rendered payload with egm marker: %s", row.RawPayload)
		}
		if !strings.Contains(row.RawPayload, `run="run-1"`) {
			t.Fatalf("expected rendered payload with action run id: %s", row.RawPayload)
		}
		if !strings.Contains(row.ParsedSummaryJSON, `"rendered":true`) {
			t.Fatalf("expected rendered summary marker: %s", row.ParsedSummaryJSON)
		}
		if !strings.Contains(row.ParsedSummaryJSON, `"dry_run":true`) || !strings.Contains(row.ParsedSummaryJSON, `"no_send":true`) {
			t.Fatalf("expected dry-run no-send summary markers: %s", row.ParsedSummaryJSON)
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

func TestDispatchMissingTemplateVersionStillRecordsJournalWarning(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	delete(st.versions, "template-1")
	dispatcher := &Dispatcher{Store: st}

	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "run-1",
		Mode:        DispatchModeDryRun,
		RequestedAt: now,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if len(result.PreparedMessages) != 2 {
		t.Fatalf("prepared messages=%d, want 2", len(result.PreparedMessages))
	}
	if result.WarningCount == 0 {
		t.Fatalf("warning count=%d, want >0", result.WarningCount)
	}
	for _, row := range result.PreparedMessages {
		if row.Error == "" {
			t.Fatalf("expected journal error detail when template version missing: %+v", row)
		}
		if !strings.Contains(row.ParsedSummaryJSON, `"rendered":false`) {
			t.Fatalf("expected rendered=false summary marker: %s", row.ParsedSummaryJSON)
		}
	}
}

func TestSendPreparedMessagesFailedUpdatesJournal(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	dispatcher := &Dispatcher{Store: st, Sender: &fakeSender{sendFn: func(_ context.Context, request g2stransport.SendRequest) (g2stransport.SendResult, error) {
		return g2stransport.SendResult{
			MessageID:     request.MessageID,
			EGMID:         request.EGMID,
			TransportMode: request.TransportMode,
			Blocked:       true,
			Sent:          false,
			Error:         "blocked",
			CompletedAt:   now,
		}, nil
	}}}

	_, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "run-1",
		Mode:        DispatchModeDryRun,
		RequestedAt: now,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	result, err := dispatcher.SendPreparedMessages(context.Background(), SendPreparedMessagesRequest{
		ActionRunID:     "run-1",
		TransportMode:   g2stransport.ModeHTTP,
		RequestedAt:     now,
	})
	if err != nil {
		t.Fatalf("send prepared: %v", err)
	}
	if result.FailedCount == 0 {
		t.Fatalf("failed count=%d, want >0", result.FailedCount)
	}
	for _, row := range st.messages {
		if row.ActionRunID != "run-1" {
			continue
		}
		if row.Result != g2sengine.MessageResultSendFailed {
			t.Fatalf("expected send-failed result, got %q", row.Result)
		}
	}
}

func TestSendPreparedMessagesAllowedMarksSucceeded(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	dispatcher := &Dispatcher{Store: st, Sender: &fakeSender{sendFn: func(_ context.Context, request g2stransport.SendRequest) (g2stransport.SendResult, error) {
		return g2stransport.SendResult{
			MessageID:       request.MessageID,
			EGMID:           request.EGMID,
			TransportMode:   request.TransportMode,
			Sent:            true,
			Blocked:         false,
			HTTPStatusCode:  200,
			ResponseExcerpt: "ok",
			CompletedAt:     now,
		}, nil
	}}}

	_, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "run-1",
		Mode:        DispatchModeDryRun,
		RequestedAt: now,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	result, err := dispatcher.SendPreparedMessages(context.Background(), SendPreparedMessagesRequest{
		ActionRunID:     "run-1",
		TransportMode:   g2stransport.ModeHTTP,
		CaptureEndpoint: "http://127.0.0.1:18080/capture",
		RequestedAt:     now,
	})
	if err != nil {
		t.Fatalf("send prepared: %v", err)
	}
	if result.SentCount == 0 {
		t.Fatalf("sent count=%d, want >0", result.SentCount)
	}
	for _, row := range st.messages {
		if row.ActionRunID != "run-1" {
			continue
		}
		if row.Result != g2sengine.MessageResultSendSucceeded {
			t.Fatalf("expected send-succeeded result, got %q", row.Result)
		}
	}
	updated := st.runs["run-1"]
	if updated.Status == actions.RunStatusSucceeded {
		t.Fatalf("run status should not be succeeded in phase 2F: %q", updated.Status)
	}
}

func TestSendPreparedMessagesRealSenderAttemptsDeliveryWithoutCaptureOnly(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	dispatcher := &Dispatcher{Store: st}

	_, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "run-1",
		Mode:        DispatchModeDryRun,
		RequestedAt: now,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	result, err := dispatcher.SendPreparedMessages(context.Background(), SendPreparedMessagesRequest{
		ActionRunID:     "run-1",
		TransportMode:   g2stransport.ModeHTTP,
		CaptureEndpoint: "http://127.0.0.1:18080/capture",
		RequestedAt:     now,
	})
	if err != nil {
		t.Fatalf("send prepared: %v", err)
	}
	if result.FailedCount == 0 {
		t.Fatalf("failed count=%d want >0", result.FailedCount)
	}
}

// This test covers the current capture-proof mode. It does not imply that
// loopback-only endpoints are a permanent production policy.
func TestSendPreparedMessagesRealSenderLocalhostCaptureSucceeds(t *testing.T) {
	now := time.Now().UTC()
	st := newFakeStore(now)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("captured"))
	}))
	defer server.Close()

	dispatcher := &Dispatcher{Store: st}
	_, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		ActionRunID: "run-1",
		Mode:        DispatchModeDryRun,
		RequestedAt: now,
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	result, err := dispatcher.SendPreparedMessages(context.Background(), SendPreparedMessagesRequest{
		ActionRunID:     "run-1",
		TransportMode:   g2stransport.ModeHTTP,
		CaptureEndpoint: server.URL + "/capture",
		RequestedAt:     now,
	})
	if err != nil {
		t.Fatalf("send prepared: %v", err)
	}
	if result.SentCount == 0 {
		t.Fatalf("sent count=%d want >0", result.SentCount)
	}
	for _, row := range st.messages {
		if row.ActionRunID != "run-1" {
			continue
		}
		if row.Result != g2sengine.MessageResultSendSucceeded {
			t.Fatalf("expected send succeeded result, got %q", row.Result)
		}
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
			"template-1": {
				ID:               "template-1",
				Name:             "Smoke",
				Vendor:           "Test",
				Status:           templates.TemplateStatusActive,
				CurrentVersionID: "1",
			},
		},
		versions: map[string]templates.G2STemplateVersion{
			"template-1": {
				ID:           "template-1-v1",
				TemplateID:   "template-1",
				VersionLabel: "1",
				ActionsJSON:  `{"actions":{"queue_only_no_send":{"message_type":"DRY_RUN_NO_SEND","content_type":"application/xml","payload_template":"<dryRunG2SMessage noSend=\"true\" action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\" step=\"{{.TemplateActionKey}}\" timestamp=\"{{.TimestampRFC3339}}\"/>"}}}`,
			},
		},
		messages: []g2sengine.MessageJournalEntry{},
		audits:   []audit.AuditTimelineEntry{},
	}
}

