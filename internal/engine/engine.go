package engine

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
)

type EventType string

const (
	EventBootComplete     EventType = "BOOT_COMPLETE"
	EventSecurityLineDrop EventType = "SECURITY_LINE_DROP"
	EventG2SSessionOnline EventType = "G2S_SESSION_ONLINE"
	EventKeepAlive        EventType = "G2S_KEEPALIVE"
	EventEGMResult        EventType = "EGM_RESULT"
	EventDegraded         EventType = "DEGRADED"
)

const (
	endpointDriftWindow   = 5 * time.Minute
	recentEndpointHistory = 10
)

type Event struct {
	Type       EventType
	At         time.Time
	EGMID      string
	OK         bool
	Detail     string
	SourceIP   string
	SourcePort int
}

type Snapshot struct {
	ControllerID string                `json:"controller_id"`
	State        model.ControllerState `json:"state"`
	UpdatedAt    time.Time             `json:"updated_at"`
	Incident     *model.Incident       `json:"incident,omitempty"`
	EGMs         []model.EGM           `json:"egms"`
	LastEvent    string                `json:"last_event,omitempty"`
	AuditError   string                `json:"audit_error,omitempty"`
}

type AuditSink interface {
	RecordIncident(context.Context, model.Incident) (int64, error)
	RecordEGMStatus(context.Context, model.EGMStatusSnapshot) error
	RecordEGMComplianceLog(context.Context, model.EGMComplianceLog) error
	RecordStateChange(context.Context, model.StateChange) error
}

type Engine struct {
	controllerID string
	events       chan Event
	audit        AuditSink

	mu             sync.RWMutex
	state          model.ControllerState
	updatedAt      time.Time
	incident       *model.Incident
	nextIncidentID int64
	egms           map[string]model.EGM
	endpointSeen   map[string]map[string]time.Time
	lastEvent      string
	auditError     string
}

func New(controllerID string, roster []config.EGM) *Engine {
	return NewWithAuditSink(controllerID, roster, nil)
}

func NewWithAuditSink(controllerID string, roster []config.EGM, audit AuditSink) *Engine {
	egms := make(map[string]model.EGM, len(roster))
	for _, item := range roster {
		egms[item.EGMID] = model.EGM{
			ID:              item.EGMID,
			IPAddress:       item.IPAddress,
			Port:            item.Port,
			Vendor:          item.Vendor,
			CabinetFamily:   item.CabinetFamily,
			GameTitle:       item.GameTitle,
			SoftwareVersion: item.SoftwareVersion,
			Source:          model.EGMSourceConfigured,
			Status:          model.EGMGreen,
		}
	}

	return &Engine{
		controllerID: controllerID,
		events:       make(chan Event, 64),
		audit:        audit,
		state:        model.StateBooting,
		updatedAt:    time.Now(),
		egms:         egms,
		endpointSeen: make(map[string]map[string]time.Time),
	}
}

func (e *Engine) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-e.events:
				e.handle(event)
			}
		}
	}()
}

func (e *Engine) Submit(event Event) {
	if event.At.IsZero() {
		event.At = time.Now()
	}
	e.events <- event
}

func (e *Engine) Snapshot() Snapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()

	egms := make([]model.EGM, 0, len(e.egms))
	for _, egm := range e.egms {
		if len(egm.EndpointDriftIPs) > 0 {
			egm.EndpointDriftIPs = append([]string(nil), egm.EndpointDriftIPs...)
		}
		if len(egm.RecentEndpoints) > 0 {
			egm.RecentEndpoints = append([]model.EGMEndpointObservation(nil), egm.RecentEndpoints...)
		}
		egms = append(egms, egm)
	}

	var incident *model.Incident
	if e.incident != nil {
		copy := *e.incident
		incident = &copy
	}

	return Snapshot{
		ControllerID: e.controllerID,
		State:        e.state,
		UpdatedAt:    e.updatedAt,
		Incident:     incident,
		EGMs:         egms,
		LastEvent:    e.lastEvent,
		AuditError:   e.auditError,
	}
}

func (e *Engine) handle(event Event) {
	e.mu.Lock()
	defer e.mu.Unlock()

	oldState := e.state
	e.updatedAt = event.At
	e.lastEvent = string(event.Type)

	switch event.Type {
	case EventBootComplete:
		if e.state == model.StateBooting {
			e.state = model.StateHealthy
		}
	case EventSecurityLineDrop:
		e.openIncident(event)
		e.state = model.StateEmergencyActive
	case EventG2SSessionOnline, EventKeepAlive:
		egmID := strings.TrimSpace(event.EGMID)
		if egmID == "" {
			break
		}
		event.EGMID = egmID
		egm, ok := e.egms[egmID]
		if !ok {
			egm = model.EGM{
				ID:     egmID,
				Source: model.EGMSourceDiscovered,
			}
		}
		if egm.Source == "" {
			egm.Source = model.EGMSourceConfigured
		}
		egm.Status = model.EGMGreen
		egm.LastSeen = event.At
		egm.LastError = ""
		e.observeEndpoint(&egm, event)
		e.egms[egmID] = egm
		e.recordEGMStatus(event, egm)
	case EventEGMResult:
		egmID := strings.TrimSpace(event.EGMID)
		if egmID == "" {
			break
		}
		event.EGMID = egmID
		if egm, ok := e.egms[egmID]; ok {
			egm.LastSeen = event.At
			if event.OK {
				egm.Status = model.EGMRed
				egm.LastError = ""
			} else {
				egm.Status = model.EGMGrey
				egm.LastError = event.Detail
			}
			e.egms[egmID] = egm
			e.recordEGMStatus(event, egm)
			e.recordEGMCompliance(event, egm)
		}
	case EventDegraded:
		e.state = model.StateDegraded
	}

	if oldState != e.state {
		e.recordStateChange(oldState, e.state, event)
	}
}

func (e *Engine) observeEndpoint(egm *model.EGM, event Event) {
	if egm == nil {
		return
	}
	sourceIP := strings.TrimSpace(event.SourceIP)
	sourcePort := event.SourcePort
	if sourceIP != "" {
		egm.LastEndpointIP = sourceIP
	}
	if sourcePort > 0 {
		egm.LastEndpointPort = sourcePort
	}
	if sourceIP != "" || sourcePort > 0 {
		egm.LastEndpointSeenAt = event.At
		egm.RecentEndpoints = updateRecentEndpoints(egm.RecentEndpoints, sourceIP, sourcePort, event.At, recentEndpointHistory)
	}
	if sourceIP == "" {
		return
	}
	observed := e.endpointSeen[egm.ID]
	if observed == nil {
		observed = make(map[string]time.Time)
	}
	observed[sourceIP] = event.At
	cutoff := event.At.Add(-endpointDriftWindow)
	for ip, seenAt := range observed {
		if seenAt.Before(cutoff) {
			delete(observed, ip)
		}
	}
	if len(observed) == 0 {
		delete(e.endpointSeen, egm.ID)
		egm.EndpointDrift = false
		egm.EndpointDriftIPs = nil
		return
	}
	e.endpointSeen[egm.ID] = observed
	ips := make([]string, 0, len(observed))
	for ip := range observed {
		ips = append(ips, ip)
	}
	sort.Strings(ips)
	egm.EndpointDrift = len(ips) > 1
	if egm.EndpointDrift {
		egm.EndpointDriftIPs = ips
	} else {
		egm.EndpointDriftIPs = nil
	}
}

func updateRecentEndpoints(history []model.EGMEndpointObservation, ip string, port int, at time.Time, limit int) []model.EGMEndpointObservation {
	if limit <= 0 || at.IsZero() {
		return history
	}
	ip = strings.TrimSpace(ip)
	if ip == "" && port <= 0 {
		return history
	}
	records := make([]model.EGMEndpointObservation, 0, len(history)+1)
	records = append(records, history...)
	idx := -1
	for i := range records {
		if records[i].IPAddress == ip && records[i].Port == port {
			idx = i
			break
		}
	}
	if idx >= 0 {
		entry := records[idx]
		if entry.FirstSeenAt.IsZero() || at.Before(entry.FirstSeenAt) {
			entry.FirstSeenAt = at
		}
		if entry.LastSeenAt.IsZero() || at.After(entry.LastSeenAt) {
			entry.LastSeenAt = at
		}
		if entry.SeenCount < 0 {
			entry.SeenCount = 0
		}
		entry.SeenCount++
		records = append(records[:idx], records[idx+1:]...)
		records = append([]model.EGMEndpointObservation{entry}, records...)
	} else {
		records = append([]model.EGMEndpointObservation{{
			IPAddress:   ip,
			Port:        port,
			FirstSeenAt: at,
			LastSeenAt:  at,
			SeenCount:   1,
		}}, records...)
	}
	if len(records) > limit {
		records = records[:limit]
	}
	return records
}

func (e *Engine) openIncident(event Event) {
	if e.incident != nil && e.incident.ResolvedAt == nil {
		return
	}

	e.nextIncidentID++
	e.incident = &model.Incident{
		ID:            e.nextIncidentID,
		TriggerType:   string(event.Type),
		TriggerSource: event.Detail,
		CreatedAt:     event.At,
		FinalState:    model.StateEmergencyActive,
	}
	if e.audit != nil {
		id, err := e.audit.RecordIncident(context.Background(), *e.incident)
		if err != nil {
			e.setAuditError("record incident", err)
			return
		}
		e.incident.ID = id
	}
}

func (e *Engine) recordEGMStatus(event Event, egm model.EGM) {
	if e.audit == nil {
		return
	}
	if err := e.audit.RecordEGMStatus(context.Background(), model.EGMStatusSnapshot{
		EGMID:     egm.ID,
		Status:    egm.Status,
		EventType: string(event.Type),
		Detail:    event.Detail,
		LastError: egm.LastError,
		CreatedAt: event.At,
	}); err != nil {
		e.setAuditError("record egm status", err)
	}
}

func (e *Engine) recordEGMCompliance(event Event, egm model.EGM) {
	if e.audit == nil || e.incident == nil || e.incident.ID == 0 {
		return
	}
	status := "SUCCESS"
	if !event.OK {
		status = "GREY_TIMEOUT"
		if event.Detail != "" {
			status = event.Detail
		}
	}
	if err := e.audit.RecordEGMComplianceLog(context.Background(), model.EGMComplianceLog{
		IncidentID:   e.incident.ID,
		EGMID:        egm.ID,
		IPAddress:    egm.IPAddress,
		ActionSent:   string(event.Type),
		StatusResult: status,
		CreatedAt:    event.At,
	}); err != nil {
		e.setAuditError("record egm compliance", err)
	}
}

func (e *Engine) recordStateChange(oldState model.ControllerState, newState model.ControllerState, event Event) {
	if e.audit == nil {
		return
	}
	if err := e.audit.RecordStateChange(context.Background(), model.StateChange{
		OldState:  oldState,
		NewState:  newState,
		Reason:    string(event.Type),
		CreatedAt: event.At,
	}); err != nil {
		e.setAuditError("record state change", err)
	}
}

func (e *Engine) setAuditError(action string, err error) {
	e.auditError = action + ": " + err.Error()
	log.Printf("audit error: %s", e.auditError)
}
