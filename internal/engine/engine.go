package engine

import (
	"context"
	"log"
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

type Event struct {
	Type   EventType
	At     time.Time
	EGMID  string
	OK     bool
	Detail string
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
		if egm, ok := e.egms[event.EGMID]; ok {
			egm.Status = model.EGMGreen
			egm.LastSeen = event.At
			egm.LastError = ""
			e.egms[event.EGMID] = egm
			e.recordEGMStatus(event, egm)
		}
	case EventEGMResult:
		if egm, ok := e.egms[event.EGMID]; ok {
			egm.LastSeen = event.At
			if event.OK {
				egm.Status = model.EGMRed
				egm.LastError = ""
			} else {
				egm.Status = model.EGMGrey
				egm.LastError = event.Detail
			}
			e.egms[event.EGMID] = egm
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
