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
	egm, ok := snapshotEGMByID(snapshot, "EGM-1")
	if !ok {
		t.Fatalf("expected EGM-1 in snapshot")
	}
	if got := egm.Status; got != model.EGMGreen {
		t.Fatalf("expected EGM green, got %s", got)
	}
	if egm.Source != model.EGMSourceConfigured {
		t.Fatalf("expected configured source, got %q", egm.Source)
	}
}

func TestSessionOnlineDiscoversUnknownEGM(t *testing.T) {
	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	now := time.Now()
	eng.handle(Event{Type: EventG2SSessionOnline, EGMID: "EGM-2", At: now})

	snapshot := eng.Snapshot()
	egm, ok := snapshotEGMByID(snapshot, "EGM-2")
	if !ok {
		t.Fatalf("expected discovered EGM-2 in snapshot")
	}
	if egm.Source != model.EGMSourceDiscovered {
		t.Fatalf("expected discovered source, got %q", egm.Source)
	}
	if egm.Status != model.EGMGreen {
		t.Fatalf("expected discovered EGM to be GREEN, got %s", egm.Status)
	}
	if !egm.LastSeen.Equal(now) {
		t.Fatalf("expected last seen to equal event time")
	}
}

func TestKeepAliveDiscoversUnknownEGMAndRecordsAudit(t *testing.T) {
	audit := &recordingAudit{}
	eng := NewWithAuditSink("controller", []config.EGM{}, audit)
	now := time.Now()
	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-9", At: now})

	snapshot := eng.Snapshot()
	egm, ok := snapshotEGMByID(snapshot, "EGM-9")
	if !ok {
		t.Fatalf("expected discovered EGM-9 in snapshot")
	}
	if egm.Source != model.EGMSourceDiscovered {
		t.Fatalf("expected discovered source, got %q", egm.Source)
	}
	if egm.Status != model.EGMGreen {
		t.Fatalf("expected discovered EGM to be GREEN, got %s", egm.Status)
	}
	if len(audit.statuses) != 1 {
		t.Fatalf("status records = %d, want 1", len(audit.statuses))
	}
	if audit.statuses[0].EGMID != "EGM-9" {
		t.Fatalf("recorded egm_id = %q, want EGM-9", audit.statuses[0].EGMID)
	}
}

func TestSessionOnlineCapturesLastSeenEndpointMetadata(t *testing.T) {
	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "10.0.0.11", Port: 9443}})
	now := time.Now()
	eng.handle(Event{
		Type:       EventG2SSessionOnline,
		EGMID:      "EGM-1",
		At:         now,
		SourceIP:   "192.168.1.22",
		SourcePort: 60501,
	})

	snapshot := eng.Snapshot()
	egm, ok := snapshotEGMByID(snapshot, "EGM-1")
	if !ok {
		t.Fatalf("expected EGM-1 in snapshot")
	}
	if egm.LastEndpointIP != "192.168.1.22" {
		t.Fatalf("last_endpoint_ip = %q, want 192.168.1.22", egm.LastEndpointIP)
	}
	if egm.LastEndpointPort != 60501 {
		t.Fatalf("last_endpoint_port = %d, want 60501", egm.LastEndpointPort)
	}
	if !egm.LastEndpointSeenAt.Equal(now) {
		t.Fatalf("last_endpoint_seen_at mismatch")
	}
	if egm.EndpointDrift {
		t.Fatal("endpoint_drift_warning = true, want false")
	}
}

func TestKeepAliveSetsEndpointDriftWhenSameEGMIDMovesAcrossIPs(t *testing.T) {
	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "10.0.0.11", Port: 9443}})
	now := time.Now()
	eng.handle(Event{
		Type:       EventKeepAlive,
		EGMID:      "EGM-1",
		At:         now,
		SourceIP:   "10.10.1.15",
		SourcePort: 9443,
	})
	eng.handle(Event{
		Type:       EventKeepAlive,
		EGMID:      "EGM-1",
		At:         now.Add(time.Minute),
		SourceIP:   "10.10.1.16",
		SourcePort: 9443,
	})

	snapshot := eng.Snapshot()
	egm, ok := snapshotEGMByID(snapshot, "EGM-1")
	if !ok {
		t.Fatalf("expected EGM-1 in snapshot")
	}
	if !egm.EndpointDrift {
		t.Fatal("endpoint_drift_warning = false, want true")
	}
	if len(egm.EndpointDriftIPs) != 2 {
		t.Fatalf("endpoint_drift_ips len = %d, want 2", len(egm.EndpointDriftIPs))
	}
	if egm.LastEndpointIP != "10.10.1.16" {
		t.Fatalf("last_endpoint_ip = %q, want 10.10.1.16", egm.LastEndpointIP)
	}
}

func TestEndpointDriftWindowExpiresOldIPs(t *testing.T) {
	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "10.0.0.11", Port: 9443}})
	now := time.Now()
	eng.handle(Event{
		Type:       EventKeepAlive,
		EGMID:      "EGM-1",
		At:         now,
		SourceIP:   "10.0.0.21",
		SourcePort: 9443,
	})
	eng.handle(Event{
		Type:       EventKeepAlive,
		EGMID:      "EGM-1",
		At:         now.Add(endpointDriftWindow + time.Second),
		SourceIP:   "10.0.0.22",
		SourcePort: 9443,
	})

	snapshot := eng.Snapshot()
	egm, ok := snapshotEGMByID(snapshot, "EGM-1")
	if !ok {
		t.Fatalf("expected EGM-1 in snapshot")
	}
	if egm.EndpointDrift {
		t.Fatal("endpoint_drift_warning = true, want false after window expires")
	}
	if len(egm.EndpointDriftIPs) != 0 {
		t.Fatalf("endpoint_drift_ips len = %d, want 0", len(egm.EndpointDriftIPs))
	}
	if egm.LastEndpointIP != "10.0.0.22" {
		t.Fatalf("last_endpoint_ip = %q, want 10.0.0.22", egm.LastEndpointIP)
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

func snapshotEGMByID(snapshot Snapshot, id string) (model.EGM, bool) {
	for _, egm := range snapshot.EGMs {
		if egm.ID == id {
			return egm, true
		}
	}
	return model.EGM{}, false
}
