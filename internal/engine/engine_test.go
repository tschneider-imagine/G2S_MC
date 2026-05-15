package engine

import (
	"context"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
)

type recordingAudit struct {
	incidents   []model.Incident
	statuses    []model.EGMStatusSnapshot
	compliance  []model.EGMComplianceLog
	transitions []model.StateChange
}

func (r *recordingAudit) RecordIncident(_ context.Context, incident model.Incident) (int64, error) {
	r.incidents = append(r.incidents, incident)
	return int64(len(r.incidents)), nil
}

func (r *recordingAudit) RecordEGMStatus(_ context.Context, snapshot model.EGMStatusSnapshot) error {
	r.statuses = append(r.statuses, snapshot)
	return nil
}

func (r *recordingAudit) RecordEGMComplianceLog(_ context.Context, entry model.EGMComplianceLog) error {
	r.compliance = append(r.compliance, entry)
	return nil
}

func (r *recordingAudit) RecordStateChange(_ context.Context, change model.StateChange) error {
	r.transitions = append(r.transitions, change)
	return nil
}

func TestSecurityLineDropCreatesIncident(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)
	eng.Submit(Event{Type: EventBootComplete})
	eng.Submit(Event{Type: EventSecurityLineDrop, Detail: "test"})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := eng.Snapshot()
		if snapshot.State == model.StateEmergencyActive && snapshot.Incident != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected emergency incident")
}

func TestSessionOnlineMarksEGMGreen(t *testing.T) {
	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.handle(Event{Type: EventG2SSessionOnline, EGMID: "EGM-1", At: time.Now()})

	snapshot := eng.Snapshot()
	if got := snapshot.EGMs[0].Status; got != model.EGMGreen {
		t.Fatalf("expected EGM green, got %s", got)
	}
}

func TestAuditSinkRecordsIncidentAndEGMResult(t *testing.T) {
	audit := &recordingAudit{}
	eng := NewWithAuditSink("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}}, audit)

	now := time.Now()
	eng.handle(Event{Type: EventSecurityLineDrop, Detail: "test", At: now})
	eng.handle(Event{Type: EventEGMResult, EGMID: "EGM-1", OK: true, At: now.Add(time.Second)})

	if len(audit.incidents) != 1 {
		t.Fatalf("incident records = %d, want 1", len(audit.incidents))
	}
	if len(audit.statuses) != 1 {
		t.Fatalf("status records = %d, want 1", len(audit.statuses))
	}
	if len(audit.compliance) != 1 {
		t.Fatalf("compliance records = %d, want 1", len(audit.compliance))
	}
	if audit.compliance[0].IncidentID != 1 {
		t.Fatalf("compliance incident id = %d, want 1", audit.compliance[0].IncidentID)
	}
	if len(audit.transitions) != 1 {
		t.Fatalf("transition records = %d, want 1", len(audit.transitions))
	}
}
