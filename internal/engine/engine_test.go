package engine

import (
	"context"
	"fmt"
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
	if len(egm.RecentEndpoints) != 1 {
		t.Fatalf("recent_endpoints len = %d, want 1", len(egm.RecentEndpoints))
	}
	if egm.RecentEndpoints[0].IPAddress != "192.168.1.22" || egm.RecentEndpoints[0].Port != 60501 {
		t.Fatalf("unexpected recent endpoint %+v", egm.RecentEndpoints[0])
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
	if !egm.EndpointCollisionWarning {
		t.Fatal("endpoint_collision_warning = false, want true for ID endpoint drift")
	}
	if !containsEndpointCollisionType(egm.EndpointCollisionTypes, model.EndpointCollisionIDEndpointDrift) {
		t.Fatalf("endpoint_collision_types = %#v, want ID_ENDPOINT_DRIFT", egm.EndpointCollisionTypes)
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

func TestSharedEndpointCollisionAcrossMultipleEGMIDs(t *testing.T) {
	eng := New("controller", []config.EGM{
		{EGMID: "EGM-1", IPAddress: "10.0.0.11", Port: 9443},
		{EGMID: "EGM-2", IPAddress: "10.0.0.12", Port: 9444},
	})
	now := time.Now()
	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-1", At: now, SourceIP: "192.168.10.40", SourcePort: 9500})
	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-2", At: now.Add(time.Second), SourceIP: "192.168.10.40", SourcePort: 9500})

	snapshot := eng.Snapshot()
	if snapshot.EndpointCollisionSummary.Total != 1 {
		t.Fatalf("collision total = %d, want 1", snapshot.EndpointCollisionSummary.Total)
	}
	if snapshot.EndpointCollisionSummary.SharedEndpointCount != 1 {
		t.Fatalf("shared endpoint count = %d, want 1", snapshot.EndpointCollisionSummary.SharedEndpointCount)
	}
	if snapshot.EndpointCollisionSummary.IDEndpointDriftCount != 0 {
		t.Fatalf("id endpoint drift count = %d, want 0", snapshot.EndpointCollisionSummary.IDEndpointDriftCount)
	}
	if len(snapshot.EndpointCollisions) != 1 {
		t.Fatalf("endpoint collisions len = %d, want 1", len(snapshot.EndpointCollisions))
	}
	row := snapshot.EndpointCollisions[0]
	if row.CollisionType != model.EndpointCollisionSharedEndpoint {
		t.Fatalf("collision type = %q, want %q", row.CollisionType, model.EndpointCollisionSharedEndpoint)
	}
	if row.Endpoint != "192.168.10.40:9500" {
		t.Fatalf("collision endpoint = %q, want 192.168.10.40:9500", row.Endpoint)
	}
	if len(row.InvolvedEGMIDs) != 2 || row.InvolvedEGMIDs[0] != "EGM-1" || row.InvolvedEGMIDs[1] != "EGM-2" {
		t.Fatalf("collision involved ids = %#v, want [EGM-1 EGM-2]", row.InvolvedEGMIDs)
	}

	egm1, ok := snapshotEGMByID(snapshot, "EGM-1")
	if !ok {
		t.Fatalf("expected EGM-1 in snapshot")
	}
	if !egm1.EndpointCollisionWarning {
		t.Fatal("EGM-1 endpoint_collision_warning = false, want true")
	}
	if !containsEndpointCollisionType(egm1.EndpointCollisionTypes, model.EndpointCollisionSharedEndpoint) {
		t.Fatalf("EGM-1 endpoint_collision_types = %#v, want SHARED_ENDPOINT", egm1.EndpointCollisionTypes)
	}

	egm2, ok := snapshotEGMByID(snapshot, "EGM-2")
	if !ok {
		t.Fatalf("expected EGM-2 in snapshot")
	}
	if !egm2.EndpointCollisionWarning {
		t.Fatal("EGM-2 endpoint_collision_warning = false, want true")
	}
	if !containsEndpointCollisionType(egm2.EndpointCollisionTypes, model.EndpointCollisionSharedEndpoint) {
		t.Fatalf("EGM-2 endpoint_collision_types = %#v, want SHARED_ENDPOINT", egm2.EndpointCollisionTypes)
	}
}

func TestIDEndpointDriftCollisionIncludesEachActiveEndpoint(t *testing.T) {
	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "10.0.0.11", Port: 9443}})
	now := time.Now()
	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-1", At: now, SourceIP: "10.10.1.15", SourcePort: 9443})
	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-1", At: now.Add(time.Minute), SourceIP: "10.10.1.16", SourcePort: 9443})

	snapshot := eng.Snapshot()
	if snapshot.EndpointCollisionSummary.SharedEndpointCount != 0 {
		t.Fatalf("shared endpoint count = %d, want 0", snapshot.EndpointCollisionSummary.SharedEndpointCount)
	}
	if snapshot.EndpointCollisionSummary.IDEndpointDriftCount != 2 {
		t.Fatalf("id endpoint drift count = %d, want 2", snapshot.EndpointCollisionSummary.IDEndpointDriftCount)
	}
	if snapshot.EndpointCollisionSummary.Total != 2 {
		t.Fatalf("collision total = %d, want 2", snapshot.EndpointCollisionSummary.Total)
	}
	egm, ok := snapshotEGMByID(snapshot, "EGM-1")
	if !ok {
		t.Fatalf("expected EGM-1 in snapshot")
	}
	if !egm.EndpointDrift {
		t.Fatal("endpoint_drift_warning = false, want true")
	}
	if !egm.EndpointCollisionWarning {
		t.Fatal("endpoint_collision_warning = false, want true")
	}
	if !containsEndpointCollisionType(egm.EndpointCollisionTypes, model.EndpointCollisionIDEndpointDrift) {
		t.Fatalf("endpoint_collision_types = %#v, want ID_ENDPOINT_DRIFT", egm.EndpointCollisionTypes)
	}
}

func TestSharedEndpointCollisionAgesOutAfterWindow(t *testing.T) {
	eng := New("controller", []config.EGM{
		{EGMID: "EGM-1", IPAddress: "10.0.0.11", Port: 9443},
		{EGMID: "EGM-2", IPAddress: "10.0.0.12", Port: 9444},
	})
	now := time.Now()
	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-1", At: now, SourceIP: "192.168.10.50", SourcePort: 9550})
	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-2", At: now.Add(time.Second), SourceIP: "192.168.10.50", SourcePort: 9550})
	initial := eng.Snapshot()
	if initial.EndpointCollisionSummary.SharedEndpointCount == 0 {
		t.Fatalf("expected initial shared endpoint collision")
	}

	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-1", At: now.Add(endpointDriftWindow + 2*time.Second), SourceIP: "192.168.10.51", SourcePort: 9551})
	snapshot := eng.Snapshot()
	if snapshot.EndpointCollisionSummary.SharedEndpointCount != 0 {
		t.Fatalf("shared endpoint count = %d, want 0 after stale claims age out", snapshot.EndpointCollisionSummary.SharedEndpointCount)
	}
}

func TestRecentEndpointHistoryAggregatesAndMovesToFront(t *testing.T) {
	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "10.0.0.11", Port: 9443}})
	now := time.Now()
	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-1", At: now, SourceIP: "10.0.0.21", SourcePort: 9443})
	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-1", At: now.Add(time.Second), SourceIP: "10.0.0.22", SourcePort: 9444})
	eng.handle(Event{Type: EventKeepAlive, EGMID: "EGM-1", At: now.Add(2 * time.Second), SourceIP: "10.0.0.21", SourcePort: 9443})

	egm, ok := snapshotEGMByID(eng.Snapshot(), "EGM-1")
	if !ok {
		t.Fatalf("expected EGM-1 in snapshot")
	}
	if len(egm.RecentEndpoints) != 2 {
		t.Fatalf("recent_endpoints len = %d, want 2", len(egm.RecentEndpoints))
	}
	first := egm.RecentEndpoints[0]
	second := egm.RecentEndpoints[1]
	if first.IPAddress != "10.0.0.21" || first.Port != 9443 {
		t.Fatalf("recent_endpoints[0] = %+v, want 10.0.0.21:9443", first)
	}
	if first.SeenCount != 2 {
		t.Fatalf("recent_endpoints[0].seen_count = %d, want 2", first.SeenCount)
	}
	if !first.FirstSeenAt.Equal(now) {
		t.Fatalf("recent_endpoints[0].first_seen_at mismatch")
	}
	if !first.LastSeenAt.Equal(now.Add(2 * time.Second)) {
		t.Fatalf("recent_endpoints[0].last_seen_at mismatch")
	}
	if second.IPAddress != "10.0.0.22" || second.Port != 9444 || second.SeenCount != 1 {
		t.Fatalf("recent_endpoints[1] = %+v, want single 10.0.0.22:9444", second)
	}
}

func TestRecentEndpointHistoryBoundedToLimit(t *testing.T) {
	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "10.0.0.11", Port: 9443}})
	base := time.Now()
	for i := 0; i < recentEndpointHistory+2; i++ {
		ip := fmt.Sprintf("10.20.30.%d", i+1)
		eng.handle(Event{
			Type:       EventKeepAlive,
			EGMID:      "EGM-1",
			At:         base.Add(time.Duration(i) * time.Second),
			SourceIP:   ip,
			SourcePort: 9000 + i,
		})
	}

	egm, ok := snapshotEGMByID(eng.Snapshot(), "EGM-1")
	if !ok {
		t.Fatalf("expected EGM-1 in snapshot")
	}
	if len(egm.RecentEndpoints) != recentEndpointHistory {
		t.Fatalf("recent_endpoints len = %d, want %d", len(egm.RecentEndpoints), recentEndpointHistory)
	}
	latest := egm.RecentEndpoints[0]
	if latest.Port != 9000+recentEndpointHistory+1 {
		t.Fatalf("latest port = %d, want %d", latest.Port, 9000+recentEndpointHistory+1)
	}
	oldest := egm.RecentEndpoints[len(egm.RecentEndpoints)-1]
	if oldest.Port != 9000+2 {
		t.Fatalf("oldest retained port = %d, want %d", oldest.Port, 9000+2)
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

func containsEndpointCollisionType(types []model.EndpointCollisionType, target model.EndpointCollisionType) bool {
	for _, entry := range types {
		if entry == target {
			return true
		}
	}
	return false
}
