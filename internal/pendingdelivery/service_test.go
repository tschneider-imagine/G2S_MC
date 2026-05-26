package pendingdelivery

import (
	"context"
	"strings"
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
	messages []g2sengine.MessageJournalEntry
	runs     map[string]actions.ActionRun
	defs     map[string]actions.ActionDefinition
	targets  map[string][]actions.ActionTargetResult
	egms     map[string]egms.EGMRecord
	tpls     map[string]templates.G2STemplate
	audits   []audit.AuditTimelineEntry
}

func (f *fakeStore) ListMessageJournalEntries(_ context.Context, query store.MessageJournalListQuery) ([]g2sengine.MessageJournalEntry, error) {
	rows := []g2sengine.MessageJournalEntry{}
	allowed := map[g2sengine.MessageResult]struct{}{}
	for _, value := range query.Results {
		allowed[value] = struct{}{}
	}
	for _, row := range f.messages {
		if strings.TrimSpace(query.EGMID) != "" && !strings.EqualFold(strings.TrimSpace(row.EGMID), strings.TrimSpace(query.EGMID)) {
			continue
		}
		if strings.TrimSpace(query.ActionRunID) != "" && !strings.EqualFold(strings.TrimSpace(row.ActionRunID), strings.TrimSpace(query.ActionRunID)) {
			continue
		}
		if query.Direction != "" && row.Direction != query.Direction {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[row.Result]; !ok {
				continue
			}
		}
		rows = append(rows, row)
	}
	// newest first to match sqlite list order.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	if query.Limit > 0 && len(rows) > query.Limit {
		rows = rows[:query.Limit]
	}
	return rows, nil
}

func (f *fakeStore) GetMessageJournalEntry(_ context.Context, id int64) (*g2sengine.MessageJournalEntry, error) {
	for i := range f.messages {
		if f.messages[i].ID == id {
			copy := f.messages[i]
			return &copy, nil
		}
	}
	return nil, nil
}

func (f *fakeStore) UpdateMessageJournalOffer(_ context.Context, id int64, offeredAt time.Time, result g2sengine.MessageResult) (bool, error) {
	for i := range f.messages {
		if f.messages[i].ID != id {
			continue
		}
		f.messages[i].Result = result
		f.messages[i].OfferCount++
		value := offeredAt
		f.messages[i].OfferedAt = &value
		return true, nil
	}
	return false, nil
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
	return nil
}

func (f *fakeStore) RecordAuditTimelineEntry(_ context.Context, entry audit.AuditTimelineEntry) (int64, error) {
	entry.ID = int64(len(f.audits) + 1)
	f.audits = append(f.audits, entry)
	return entry.ID, nil
}

func (f *fakeStore) ListActionRuns(_ context.Context, query store.ActionRunListQuery) ([]actions.ActionRun, error) {
	rows := []actions.ActionRun{}
	for _, row := range f.runs {
		if query.Status != "" && row.Status != query.Status {
			continue
		}
		if strings.TrimSpace(query.IncidentID) != "" && !strings.EqualFold(strings.TrimSpace(row.IncidentID), strings.TrimSpace(query.IncidentID)) {
			continue
		}
		if strings.TrimSpace(query.ActionDefinitionID) != "" && !strings.EqualFold(strings.TrimSpace(row.ActionDefinitionID), strings.TrimSpace(query.ActionDefinitionID)) {
			continue
		}
		if query.InputTransitionID > 0 && row.InputTransitionID != query.InputTransitionID {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *fakeStore) GetActionDefinition(_ context.Context, id string) (*actions.ActionDefinition, error) {
	row, ok := f.defs[id]
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

func (f *fakeStore) ListEGMRecords(_ context.Context) ([]egms.EGMRecord, error) {
	rows := make([]egms.EGMRecord, 0, len(f.egms))
	for _, row := range f.egms {
		rows = append(rows, row)
	}
	return rows, nil
}

func (f *fakeStore) GetG2STemplate(_ context.Context, id string) (*templates.G2STemplate, error) {
	row, ok := f.tpls[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) GetEGMGroup(_ context.Context, _ string) (*egms.EGMGroup, error) {
	return nil, nil
}

func (f *fakeStore) ListEGMGroups(_ context.Context) ([]egms.EGMGroup, error) {
	return []egms.EGMGroup{}, nil
}

func (f *fakeStore) CreateActionRun(_ context.Context, run actions.ActionRun) (actions.ActionRun, error) {
	f.runs[run.ID] = run
	return run, nil
}

func (f *fakeStore) CreateActionTargetResult(_ context.Context, row actions.ActionTargetResult) (actions.ActionTargetResult, error) {
	row.ID = int64(len(f.targets[row.ActionRunID]) + 1)
	f.targets[row.ActionRunID] = append(f.targets[row.ActionRunID], row)
	return row, nil
}

func TestHandleClientContactOffersPreparedMessageForMatchingEGMOnly(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := newFixture(now)
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	result, err := svc.HandleClientContact(context.Background(), ContactRequest{EGMID: "EGM-001", ContactAt: now})
	if err != nil {
		t.Fatalf("handle contact: %v", err)
	}
	if result.Offered == nil || result.Offered.MessageID != 1 {
		t.Fatalf("expected message 1 offered, got %+v", result.Offered)
	}
	if st.messages[0].Result != g2sengine.MessageResultOffered {
		t.Fatalf("message 1 result=%q want OFFERED", st.messages[0].Result)
	}
	if st.messages[1].Result != g2sengine.MessageResultPrepared {
		t.Fatalf("message 2 should remain PREPARED, got %q", st.messages[1].Result)
	}
}

func TestHandleClientContactWrongEGMDoesNotOfferDifferentEGMMessage(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := newFixture(now)
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	result, err := svc.HandleClientContact(context.Background(), ContactRequest{EGMID: "EGM-999", ContactAt: now})
	if err != nil {
		t.Fatalf("handle contact: %v", err)
	}
	if result.Offered != nil {
		t.Fatalf("did not expect offer, got %+v", result.Offered)
	}
}

func TestHandleClientContactWithActionRunIDOffersRequestedRunMessage(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := &fakeStore{
		messages: []g2sengine.MessageJournalEntry{
			{
				ID:          1,
				Timestamp:   now.Add(-2 * time.Minute),
				Direction:   g2sengine.DirectionOutbound,
				EGMID:       "EGM-001",
				ActionRunID: "run-old",
				Result:      g2sengine.MessageResultPrepared,
				RawPayload:  "<old/>",
			},
			{
				ID:          2,
				Timestamp:   now.Add(-time.Minute),
				Direction:   g2sengine.DirectionOutbound,
				EGMID:       "EGM-001",
				ActionRunID: "run-new",
				Result:      g2sengine.MessageResultPrepared,
				RawPayload:  "<new/>",
			},
		},
		runs:    map[string]actions.ActionRun{},
		defs:    map[string]actions.ActionDefinition{},
		targets: map[string][]actions.ActionTargetResult{},
		egms:    map[string]egms.EGMRecord{},
		tpls:    map[string]templates.G2STemplate{},
		audits:  []audit.AuditTimelineEntry{},
	}
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	result, err := svc.HandleClientContact(context.Background(), ContactRequest{
		EGMID:       "EGM-001",
		ActionRunID: "run-new",
		ContactAt:   now,
	})
	if err != nil {
		t.Fatalf("handle contact: %v", err)
	}
	if result.Offered == nil || result.Offered.MessageID != 2 {
		t.Fatalf("expected run-new message offered, got %+v", result.Offered)
	}
	if st.messages[0].Result != g2sengine.MessageResultPrepared {
		t.Fatalf("run-old message changed unexpectedly: %q", st.messages[0].Result)
	}
	if st.messages[1].Result != g2sengine.MessageResultOffered {
		t.Fatalf("run-new message result=%q want OFFERED", st.messages[1].Result)
	}
	if st.messages[1].OfferCount != 1 {
		t.Fatalf("offer_count=%d want 1", st.messages[1].OfferCount)
	}
}

func TestHandleClientContactWithActionRunIDDoesNotOfferUnrelatedMessage(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := &fakeStore{
		messages: []g2sengine.MessageJournalEntry{
			{
				ID:          1,
				Timestamp:   now.Add(-2 * time.Minute),
				Direction:   g2sengine.DirectionOutbound,
				EGMID:       "EGM-001",
				ActionRunID: "run-old",
				Result:      g2sengine.MessageResultPrepared,
				RawPayload:  "<old/>",
			},
		},
		runs:    map[string]actions.ActionRun{},
		defs:    map[string]actions.ActionDefinition{},
		targets: map[string][]actions.ActionTargetResult{},
		egms:    map[string]egms.EGMRecord{},
		tpls:    map[string]templates.G2STemplate{},
		audits:  []audit.AuditTimelineEntry{},
	}
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	result, err := svc.HandleClientContact(context.Background(), ContactRequest{
		EGMID:       "EGM-001",
		ActionRunID: "run-new",
		ContactAt:   now,
	})
	if err != nil {
		t.Fatalf("handle contact: %v", err)
	}
	if result.Offered != nil {
		t.Fatalf("did not expect offered message, got %+v", result.Offered)
	}
	if len(result.Warnings) == 0 || result.Warnings[0] != "No pending message for requested action run." {
		t.Fatalf("unexpected warnings: %+v", result.Warnings)
	}
	if st.messages[0].Result != g2sengine.MessageResultPrepared {
		t.Fatalf("unrelated message should stay PREPARED, got %q", st.messages[0].Result)
	}
}

func TestHandleClientContactWithoutActionRunIDUsesOldestMessageForEGM(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := &fakeStore{
		messages: []g2sengine.MessageJournalEntry{
			{
				ID:          1,
				Timestamp:   now.Add(-2 * time.Minute),
				Direction:   g2sengine.DirectionOutbound,
				EGMID:       "EGM-001",
				ActionRunID: "run-old",
				Result:      g2sengine.MessageResultPrepared,
				RawPayload:  "<old/>",
			},
			{
				ID:          2,
				Timestamp:   now.Add(-time.Minute),
				Direction:   g2sengine.DirectionOutbound,
				EGMID:       "EGM-001",
				ActionRunID: "run-new",
				Result:      g2sengine.MessageResultPrepared,
				RawPayload:  "<new/>",
			},
		},
		runs:    map[string]actions.ActionRun{},
		defs:    map[string]actions.ActionDefinition{},
		targets: map[string][]actions.ActionTargetResult{},
		egms:    map[string]egms.EGMRecord{},
		tpls:    map[string]templates.G2STemplate{},
		audits:  []audit.AuditTimelineEntry{},
	}
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	result, err := svc.HandleClientContact(context.Background(), ContactRequest{EGMID: "EGM-001", ContactAt: now})
	if err != nil {
		t.Fatalf("handle contact: %v", err)
	}
	if result.Offered == nil || result.Offered.MessageID != 1 {
		t.Fatalf("expected oldest EGM message offered, got %+v", result.Offered)
	}
	if st.messages[0].Result != g2sengine.MessageResultOffered {
		t.Fatalf("oldest message should become OFFERED, got %q", st.messages[0].Result)
	}
	if st.messages[1].Result != g2sengine.MessageResultPrepared {
		t.Fatalf("newer message should remain PREPARED, got %q", st.messages[1].Result)
	}
}

func TestSweepWaitingConfirmationsExpiresMessageAndFailsRun(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := newFixture(now)
	st.defs["action-1"] = actions.ActionDefinition{
		ID:               "action-1",
		Name:             "Emergency",
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
		RetryPolicyJSON: `{"count":0,"delay_ms":0}`,
		Version:         1,
	}
	st.runs["run-1"] = actions.ActionRun{
		ID:                 "run-1",
		ActionDefinitionID: "action-1",
		StartedAt:          now.Add(-time.Minute),
		Status:             actions.RunStatusWaitingConfirmation,
		TargetCount:        1,
	}
	st.targets["run-1"] = []actions.ActionTargetResult{
		{ID: 1, ActionRunID: "run-1", TargetEGMID: "EGM-001", Status: actions.TargetStatusPending},
	}
	st.messages[0].ActionRunID = "run-1"
	st.messages[0].Result = g2sengine.MessageResultOffered
	st.messages[0].OfferedAt = ptrTime(now.Add(-time.Minute))
	st.messages[0].OfferCount = 1

	svc := &Service{Store: st, Clock: func() time.Time { return now }}
	result, err := svc.SweepWaitingConfirmations(context.Background(), now)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if result.MessagesExpired != 1 {
		t.Fatalf("messages_expired=%d want 1", result.MessagesExpired)
	}
	if st.messages[0].Result != g2sengine.MessageResultExpired {
		t.Fatalf("message result=%q want EXPIRED", st.messages[0].Result)
	}
	run := st.runs["run-1"]
	if run.Status != actions.RunStatusFailed {
		t.Fatalf("run status=%q want FAILED", run.Status)
	}
}

func TestExpireMessagePreparedUpdatesLifecycleAndAudit(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := newFixture(now)
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	result, err := svc.ExpireMessage(context.Background(), 1, "operator", "manual expire")
	if err != nil {
		t.Fatalf("expire message: %v", err)
	}
	if result.Result != g2sengine.MessageResultExpired {
		t.Fatalf("result=%q want EXPIRED", result.Result)
	}
	if st.messages[0].Result != g2sengine.MessageResultExpired {
		t.Fatalf("message result=%q want EXPIRED", st.messages[0].Result)
	}
	if st.messages[0].RawPayload != "<cmd/>" {
		t.Fatalf("raw payload changed unexpectedly: %q", st.messages[0].RawPayload)
	}
	found := false
	for _, row := range st.audits {
		if row.EventType == audit.EventTypeMessageExpired && strings.Contains(strings.ToLower(row.Summary), "expired by operator") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected message expired audit row, rows=%+v", st.audits)
	}
}

func TestExpireMessageConfirmedRejected(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := newFixture(now)
	st.messages[0].Result = g2sengine.MessageResultConfirmed
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	if _, err := svc.ExpireMessage(context.Background(), 1, "operator", "manual expire"); err == nil {
		t.Fatal("expected expire confirmed message to fail")
	}
}

func TestExpireMessageOfferedUpdatesLifecycle(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := newFixture(now)
	st.messages[0].Result = g2sengine.MessageResultOffered
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	result, err := svc.ExpireMessage(context.Background(), 1, "operator", "manual expire offered")
	if err != nil {
		t.Fatalf("expire offered message: %v", err)
	}
	if result.Result != g2sengine.MessageResultExpired {
		t.Fatalf("result=%q want EXPIRED", result.Result)
	}
}

func TestSupersedeMessageOfferedUpdatesLifecycleAndAudit(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := newFixture(now)
	st.messages[0].Result = g2sengine.MessageResultOffered
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	result, err := svc.SupersedeMessage(context.Background(), 1, "operator", "manual supersede")
	if err != nil {
		t.Fatalf("supersede message: %v", err)
	}
	if result.Result != g2sengine.MessageResultSuperseded {
		t.Fatalf("result=%q want SUPERSEDED", result.Result)
	}
	if st.messages[0].Result != g2sengine.MessageResultSuperseded {
		t.Fatalf("message result=%q want SUPERSEDED", st.messages[0].Result)
	}
}

func TestReprepareMessageOfferedUpdatesLifecycleAndAudit(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := newFixture(now)
	st.messages[0].Result = g2sengine.MessageResultOffered
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	result, err := svc.ReprepareMessage(context.Background(), 1, "operator", "manual reprepare")
	if err != nil {
		t.Fatalf("reprepare message: %v", err)
	}
	if result.Result != g2sengine.MessageResultPrepared {
		t.Fatalf("result=%q want PREPARED", result.Result)
	}
	if st.messages[0].Result != g2sengine.MessageResultPrepared {
		t.Fatalf("message result=%q want PREPARED", st.messages[0].Result)
	}
	found := false
	for _, row := range st.audits {
		if row.EventType == audit.EventTypeRetry && strings.Contains(strings.ToLower(row.Summary), "returned to prepared") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected retry audit row, rows=%+v", st.audits)
	}
}

func TestReprepareMessageConfirmedRejected(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := newFixture(now)
	st.messages[0].Result = g2sengine.MessageResultConfirmed
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	if _, err := svc.ReprepareMessage(context.Background(), 1, "operator", "manual reprepare"); err == nil {
		t.Fatal("expected reprepare confirmed message to fail")
	}
}

func TestSupersedeMessageFailedRejected(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := newFixture(now)
	st.messages[0].Result = g2sengine.MessageResultFailed
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	if _, err := svc.SupersedeMessage(context.Background(), 1, "operator", "manual supersede"); err == nil {
		t.Fatal("expected supersede failed message to be rejected")
	}
}

func TestResolveIncidentAfterReturnSuccessSupersedesOlderUnresolvedRuns(t *testing.T) {
	now := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	st := &fakeStore{
		messages: []g2sengine.MessageJournalEntry{
			{
				ID:          1,
				Timestamp:   now.Add(-5 * time.Minute),
				Direction:   g2sengine.DirectionOutbound,
				EGMID:       "EGM-001",
				ActionRunID: "run-trigger",
				Result:      g2sengine.MessageResultPrepared,
				RawPayload:  "<trigger/>",
			},
			{
				ID:          2,
				Timestamp:   now.Add(-4 * time.Minute),
				Direction:   g2sengine.DirectionOutbound,
				EGMID:       "EGM-002",
				ActionRunID: "run-trigger",
				Result:      g2sengine.MessageResultOffered,
				RawPayload:  "<trigger/>",
			},
			{
				ID:          3,
				Timestamp:   now.Add(-time.Minute),
				Direction:   g2sengine.DirectionOutbound,
				EGMID:       "EGM-001",
				ActionRunID: "run-return",
				Result:      g2sengine.MessageResultConfirmed,
				RawPayload:  "<return/>",
			},
		},
		runs: map[string]actions.ActionRun{
			"run-trigger": {
				ID:                 "run-trigger",
				ActionDefinitionID: "emergency-broadcast-trigger",
				IncidentID:         "101",
				StartedAt:          now.Add(-10 * time.Minute),
				Status:             actions.RunStatusWaitingConfirmation,
				TargetCount:        2,
			},
			"run-return": {
				ID:                 "run-return",
				ActionDefinitionID: "emergency-broadcast-normal",
				IncidentID:         "101",
				StartedAt:          now.Add(-2 * time.Minute),
				Status:             actions.RunStatusSucceeded,
				TargetCount:        2,
				ConfirmedCount:     2,
			},
			"run-unrelated": {
				ID:                 "run-unrelated",
				ActionDefinitionID: "general-broadcast-trigger",
				IncidentID:         "202",
				StartedAt:          now.Add(-11 * time.Minute),
				Status:             actions.RunStatusWaitingConfirmation,
				TargetCount:        1,
			},
		},
		defs: map[string]actions.ActionDefinition{
			"emergency-broadcast-trigger": {
				ID:             "emergency-broadcast-trigger",
				Name:           "Emergency Broadcast Trigger",
				Severity:       actions.SeverityEmergency,
				Enabled:        true,
				TargetSelector: "ALL_EMERGENCY_ENABLED", TemplateSelector: "template-by-egm",
				Steps:          []actions.ActionStep{{ID: "step-1", Name: "Silence", Sequence: 0, TemplateActionKey: "emergency_broadcast_silence"}},
				ReturnActionID: "emergency-broadcast-normal",
				Version:        1,
			},
			"emergency-broadcast-normal": {
				ID:             "emergency-broadcast-normal",
				Name:           "Emergency Broadcast Return",
				Severity:       actions.SeverityRestore,
				Enabled:        true,
				TargetSelector: "ALL_EMERGENCY_ENABLED", TemplateSelector: "template-by-egm",
				Steps:   []actions.ActionStep{{ID: "step-1", Name: "Restore", Sequence: 0, TemplateActionKey: "emergency_broadcast_restore"}},
				Version: 1,
			},
			"general-broadcast-trigger": {
				ID:             "general-broadcast-trigger",
				Name:           "General Broadcast Trigger",
				Severity:       actions.SeverityBroadcast,
				Enabled:        true,
				TargetSelector: "ALL_EMERGENCY_ENABLED", TemplateSelector: "template-by-egm",
				Steps:   []actions.ActionStep{{ID: "step-1", Name: "Notice", Sequence: 0, TemplateActionKey: "general_broadcast_notice"}},
				Version: 1,
			},
		},
		targets: map[string][]actions.ActionTargetResult{
			"run-trigger": {
				{ID: 1, ActionRunID: "run-trigger", TargetEGMID: "EGM-001", Status: actions.TargetStatusPending},
				{ID: 2, ActionRunID: "run-trigger", TargetEGMID: "EGM-002", Status: actions.TargetStatusConfirmed},
			},
			"run-return": {
				{ID: 3, ActionRunID: "run-return", TargetEGMID: "EGM-001", Status: actions.TargetStatusConfirmed},
				{ID: 4, ActionRunID: "run-return", TargetEGMID: "EGM-002", Status: actions.TargetStatusConfirmed},
			},
			"run-unrelated": {
				{ID: 5, ActionRunID: "run-unrelated", TargetEGMID: "EGM-001", Status: actions.TargetStatusPending},
			},
		},
		egms:   map[string]egms.EGMRecord{},
		tpls:   map[string]templates.G2STemplate{},
		audits: []audit.AuditTimelineEntry{},
	}

	result, err := ResolveIncidentAfterReturnSuccess(context.Background(), st, "101", "run-return", now)
	if err != nil {
		t.Fatalf("resolve after return: %v", err)
	}
	if result.RunsSuperseded != 1 {
		t.Fatalf("runs_superseded=%d want 1", result.RunsSuperseded)
	}
	if st.runs["run-trigger"].Status != actions.RunStatusSuperseded {
		t.Fatalf("trigger run status=%q want SUPERSEDED", st.runs["run-trigger"].Status)
	}
	if st.runs["run-trigger"].Status == actions.RunStatusSucceeded {
		t.Fatal("trigger run must not become SUCCEEDED")
	}
	if st.targets["run-trigger"][0].Status != actions.TargetStatusSuperseded {
		t.Fatalf("target status=%q want SUPERSEDED", st.targets["run-trigger"][0].Status)
	}
	if st.messages[0].Result != g2sengine.MessageResultSuperseded || st.messages[1].Result != g2sengine.MessageResultSuperseded {
		t.Fatalf("expected trigger messages superseded, got %q and %q", st.messages[0].Result, st.messages[1].Result)
	}
	if st.messages[2].Result != g2sengine.MessageResultConfirmed {
		t.Fatalf("return confirmation message changed unexpectedly: %q", st.messages[2].Result)
	}
	if st.runs["run-unrelated"].Status != actions.RunStatusWaitingConfirmation {
		t.Fatalf("unrelated run changed unexpectedly: %q", st.runs["run-unrelated"].Status)
	}
	foundAudit := false
	for _, row := range st.audits {
		if row.EventType == audit.EventTypeReturnToNormal && strings.Contains(row.Summary, "superseded by return-to-normal") {
			foundAudit = true
		}
	}
	if !foundAudit {
		t.Fatalf("expected return-to-normal supersede audit, rows=%+v", st.audits)
	}
}

func newFixture(now time.Time) *fakeStore {
	return &fakeStore{
		messages: []g2sengine.MessageJournalEntry{
			{
				ID:         1,
				Timestamp:  now.Add(-2 * time.Minute),
				Direction:  g2sengine.DirectionOutbound,
				EGMID:      "EGM-001",
				Result:     g2sengine.MessageResultPrepared,
				RawPayload: "<cmd/>",
			},
			{
				ID:         2,
				Timestamp:  now.Add(-time.Minute),
				Direction:  g2sengine.DirectionOutbound,
				EGMID:      "EGM-002",
				Result:     g2sengine.MessageResultPrepared,
				RawPayload: "<cmd/>",
			},
		},
		runs:    map[string]actions.ActionRun{},
		defs:    map[string]actions.ActionDefinition{},
		targets: map[string][]actions.ActionTargetResult{},
		egms: map[string]egms.EGMRecord{
			"EGM-001": {EGMID: "EGM-001", Enabled: true, EmergencyEnabled: true, TemplateID: "template-generic-g2s-action", CurrentActionState: egms.EGMActionStateNormal},
		},
		tpls: map[string]templates.G2STemplate{
			"template-generic-g2s-action": {
				ID:               "template-generic-g2s-action",
				Name:             "Generic G2S Action Template",
				Vendor:           "Generic",
				Status:           templates.TemplateStatusActive,
				CurrentVersionID: "1",
			},
		},
		audits: []audit.AuditTimelineEntry{},
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
