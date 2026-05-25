package incidents

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

type fakeStore struct {
	transitions map[int64]inputs.InputTransition
	channels    map[string]inputs.InputChannel
	incidents   map[int64]IncidentRecord
	runs        map[string]actions.ActionRun
	audits      []audit.AuditTimelineEntry
	nextID      int64
}

func (f *fakeStore) GetInputTransition(_ context.Context, id int64) (*inputs.InputTransition, error) {
	row, ok := f.transitions[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}
func (f *fakeStore) GetInputChannel(_ context.Context, id string) (*inputs.InputChannel, error) {
	row, ok := f.channels[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}
func (f *fakeStore) CreateIncidentRecord(_ context.Context, record IncidentRecord) (IncidentRecord, error) {
	f.nextID++
	record.ID = f.nextID
	f.incidents[record.ID] = record
	return record, nil
}
func (f *fakeStore) GetIncidentRecord(_ context.Context, id int64) (*IncidentRecord, error) {
	row, ok := f.incidents[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}
func (f *fakeStore) GetOpenIncidentByInput(_ context.Context, inputID string) (*IncidentRecord, error) {
	for _, row := range f.incidents {
		if row.PrimaryInputID == inputID && row.Status == StatusOpen {
			copy := row
			return &copy, nil
		}
	}
	return nil, nil
}
func (f *fakeStore) CloseIncidentRecord(_ context.Context, id int64, closedAt time.Time, closedByTransitionID int64, closeReason string) (*IncidentRecord, error) {
	row := f.incidents[id]
	row.Status = StatusClosed
	row.ClosedAt = &closedAt
	row.ClosedByTransitionID = closedByTransitionID
	row.CloseReason = closeReason
	f.incidents[id] = row
	copy := row
	return &copy, nil
}
func (f *fakeStore) UpdateIncidentPrimaryActionRun(_ context.Context, id int64, actionRunID string) error {
	row := f.incidents[id]
	row.PrimaryActionRunID = actionRunID
	f.incidents[id] = row
	return nil
}
func (f *fakeStore) GetActionRun(_ context.Context, id string) (*actions.ActionRun, error) {
	row, ok := f.runs[id]
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
func (f *fakeStore) RecordAuditTimelineEntry(_ context.Context, entry audit.AuditTimelineEntry) (int64, error) {
	entry.ID = int64(len(f.audits) + 1)
	f.audits = append(f.audits, entry)
	return entry.ID, nil
}

func TestHandleTransitionOpensAndClosesIncident(t *testing.T) {
	now := time.Now().UTC()
	st := &fakeStore{
		transitions: map[int64]inputs.InputTransition{
			11: {ID: 11, InputChannelID: "emergency-broadcast", NewDerived: inputs.DerivedStateTriggered, TransitionAt: now},
			12: {ID: 12, InputChannelID: "emergency-broadcast", NewDerived: inputs.DerivedStateNormal, TransitionAt: now.Add(2 * time.Minute)},
		},
		channels: map[string]inputs.InputChannel{
			"emergency-broadcast": {ID: "emergency-broadcast", Name: "Emergency Broadcast", Enabled: true},
		},
		incidents: map[int64]IncidentRecord{},
		runs:      map[string]actions.ActionRun{},
	}
	svc := &Service{Store: st, Clock: func() time.Time { return now }}

	openResult, err := svc.HandleTransition(context.Background(), 11, "operator", now)
	if err != nil {
		t.Fatalf("open transition: %v", err)
	}
	if !openResult.Opened || openResult.Incident == nil {
		t.Fatalf("expected incident open result: %+v", openResult)
	}
	id := openResult.Incident.ID
	dupResult, err := svc.HandleTransition(context.Background(), 11, "operator", now)
	if err != nil {
		t.Fatalf("duplicate transition: %v", err)
	}
	if dupResult.Opened {
		t.Fatalf("expected duplicate trigger to reuse incident: %+v", dupResult)
	}
	if len(st.incidents) != 1 {
		t.Fatalf("incident count=%d, want 1", len(st.incidents))
	}
	closeResult, err := svc.HandleTransition(context.Background(), 12, "operator", now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("close transition: %v", err)
	}
	if !closeResult.Closed || closeResult.Incident == nil || closeResult.Incident.ID != id {
		t.Fatalf("expected incident close result: %+v", closeResult)
	}
}

func TestLinkActionRunLinksOpenIncidentAndSetsPrimary(t *testing.T) {
	now := time.Now().UTC()
	st := &fakeStore{
		transitions: map[int64]inputs.InputTransition{
			21: {ID: 21, InputChannelID: "emergency-broadcast", NewDerived: inputs.DerivedStateTriggered, TransitionAt: now},
		},
		channels: map[string]inputs.InputChannel{
			"emergency-broadcast": {ID: "emergency-broadcast", Name: "Emergency Broadcast", Enabled: true},
		},
		incidents: map[int64]IncidentRecord{},
		runs: map[string]actions.ActionRun{
			"run-1": {ID: "run-1", ActionDefinitionID: "action-1", StartedAt: now, Status: actions.RunStatusPending},
		},
	}
	svc := &Service{Store: st, Clock: func() time.Time { return now }}
	openResult, err := svc.HandleTransition(context.Background(), 21, "operator", now)
	if err != nil {
		t.Fatalf("open transition: %v", err)
	}
	if openResult.Incident == nil {
		t.Fatal("expected incident")
	}
	incident, err := svc.LinkActionRun(context.Background(), "run-1", 21, "emergency-broadcast", "operator", now)
	if err != nil {
		t.Fatalf("link action run: %v", err)
	}
	if incident == nil {
		t.Fatal("expected incident link")
	}
	updatedRun := st.runs["run-1"]
	if updatedRun.IncidentID != strconv.FormatInt(incident.ID, 10) {
		t.Fatalf("run incident_id=%q", updatedRun.IncidentID)
	}
	updatedIncident := st.incidents[incident.ID]
	if updatedIncident.PrimaryActionRunID != "run-1" {
		t.Fatalf("incident primary action run=%q", updatedIncident.PrimaryActionRunID)
	}
	linkedAudit := false
	for _, row := range st.audits {
		if row.EventType == audit.EventTypeIncidentLinked {
			linkedAudit = true
		}
	}
	if !linkedAudit {
		t.Fatal("expected incident linked audit event")
	}
}

func TestHandleTransitionNormalWithoutOpenIncidentNoClose(t *testing.T) {
	now := time.Now().UTC()
	st := &fakeStore{
		transitions: map[int64]inputs.InputTransition{
			31: {ID: 31, InputChannelID: "local-notice", NewDerived: inputs.DerivedStateNormal, TransitionAt: now},
		},
		channels: map[string]inputs.InputChannel{
			"local-notice": {ID: "local-notice", Name: "Local Notice", Enabled: true},
		},
		incidents: map[int64]IncidentRecord{},
		runs:      map[string]actions.ActionRun{},
	}
	svc := &Service{Store: st}
	result, err := svc.HandleTransition(context.Background(), 31, "operator", now)
	if err != nil {
		t.Fatalf("handle transition: %v", err)
	}
	if result.Closed {
		t.Fatalf("expected no close: %+v", result)
	}
	if !strings.Contains(result.Reason, "no open incident") {
		t.Fatalf("reason=%q", result.Reason)
	}
}
