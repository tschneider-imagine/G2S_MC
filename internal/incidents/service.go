package incidents

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

type Store interface {
	GetInputTransition(ctx context.Context, id int64) (*inputs.InputTransition, error)
	GetInputChannel(ctx context.Context, id string) (*inputs.InputChannel, error)

	CreateIncidentRecord(ctx context.Context, record IncidentRecord) (IncidentRecord, error)
	GetIncidentRecord(ctx context.Context, id int64) (*IncidentRecord, error)
	GetOpenIncidentByInput(ctx context.Context, inputID string) (*IncidentRecord, error)
	CloseIncidentRecord(ctx context.Context, id int64, closedAt time.Time, closedByTransitionID int64, closeReason string) (*IncidentRecord, error)
	UpdateIncidentPrimaryActionRun(ctx context.Context, id int64, actionRunID string) error

	GetActionRun(ctx context.Context, id string) (*actions.ActionRun, error)
	UpdateActionRun(ctx context.Context, run actions.ActionRun) error

	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)
}

type Service struct {
	Store Store
	Clock func() time.Time
}

func (s *Service) HandleTransition(ctx context.Context, transitionID int64, actor string, occurredAt time.Time) (TransitionResult, error) {
	if s.Store == nil {
		return TransitionResult{}, fmt.Errorf("store is required")
	}
	if transitionID <= 0 {
		return TransitionResult{}, fmt.Errorf("transition_id is required")
	}
	transition, err := s.Store.GetInputTransition(ctx, transitionID)
	if err != nil {
		return TransitionResult{}, err
	}
	if transition == nil {
		return TransitionResult{}, fmt.Errorf("transition %d not found", transitionID)
	}
	channel, err := s.Store.GetInputChannel(ctx, transition.InputChannelID)
	if err != nil {
		return TransitionResult{}, err
	}
	if channel == nil || !channel.Enabled {
		return TransitionResult{Reason: "input not enabled"}, nil
	}

	now := occurredAt.UTC()
	if now.IsZero() {
		now = s.now()
	}
	operator := strings.TrimSpace(actor)

	openIncident, err := s.Store.GetOpenIncidentByInput(ctx, channel.ID)
	if err != nil {
		return TransitionResult{}, err
	}

	switch transition.NewDerived {
	case inputs.DerivedStateTriggered:
		if openIncident != nil {
			return TransitionResult{Incident: openIncident, Reason: "incident already open"}, nil
		}
		record := IncidentRecord{
			OpenedAt:             now,
			Status:               StatusOpen,
			Severity:             severityForChannel(channel),
			PrimaryInputID:       channel.ID,
			OpenedByTransitionID: transition.ID,
			Summary:              fmt.Sprintf("%s triggered", defaultString(channel.Name, channel.ID)),
		}
		record, err = s.Store.CreateIncidentRecord(ctx, record)
		if err != nil {
			return TransitionResult{}, err
		}
		if _, err := s.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
			OccurredAt:        now,
			Severity:          mapIncidentSeverity(record.Severity),
			EventType:         audit.EventTypeIncidentOpened,
			Summary:           fmt.Sprintf("Incident %d opened for input %s", record.ID, record.PrimaryInputID),
			DetailJSON:        incidentAuditDetail(record),
			InputTransitionID: transition.ID,
			Operator:          operator,
		}); err != nil {
			return TransitionResult{}, err
		}
		return TransitionResult{Incident: &record, Opened: true, Reason: "opened"}, nil

	case inputs.DerivedStateNormal:
		if openIncident == nil {
			return TransitionResult{Reason: "no open incident"}, nil
		}
		closeReason := "Return to Normal"
		closed, err := s.Store.CloseIncidentRecord(ctx, openIncident.ID, now, transition.ID, closeReason)
		if err != nil {
			return TransitionResult{}, err
		}
		if _, err := s.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
			OccurredAt:        now,
			Severity:          audit.AuditSeverityInfo,
			EventType:         audit.EventTypeIncidentClosed,
			Summary:           fmt.Sprintf("Incident %d closed for input %s", openIncident.ID, channel.ID),
			DetailJSON:        incidentAuditDetail(*closed),
			InputTransitionID: transition.ID,
			Operator:          operator,
		}); err != nil {
			return TransitionResult{}, err
		}
		return TransitionResult{Incident: closed, Closed: true, Reason: closeReason}, nil
	default:
		return TransitionResult{Incident: openIncident, Reason: "no incident change"}, nil
	}
}

func (s *Service) LinkActionRun(ctx context.Context, actionRunID string, transitionID int64, inputID string, actor string, occurredAt time.Time) (*IncidentRecord, error) {
	if s.Store == nil {
		return nil, fmt.Errorf("store is required")
	}
	runID := strings.TrimSpace(actionRunID)
	if runID == "" {
		return nil, fmt.Errorf("action_run_id is required")
	}
	run, err := s.Store.GetActionRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("action run %q not found", runID)
	}
	if strings.TrimSpace(run.IncidentID) != "" {
		id, parseErr := strconv.ParseInt(strings.TrimSpace(run.IncidentID), 10, 64)
		if parseErr == nil && id > 0 {
			return s.Store.GetIncidentRecord(ctx, id)
		}
		return nil, nil
	}

	trimmedInput := strings.TrimSpace(inputID)
	if trimmedInput == "" && transitionID > 0 {
		transition, err := s.Store.GetInputTransition(ctx, transitionID)
		if err != nil {
			return nil, err
		}
		if transition != nil {
			trimmedInput = strings.TrimSpace(transition.InputChannelID)
		}
	}
	if trimmedInput == "" {
		return nil, nil
	}

	incident, err := s.Store.GetOpenIncidentByInput(ctx, trimmedInput)
	if err != nil {
		return nil, err
	}
	if incident == nil {
		return nil, nil
	}

	run.IncidentID = strconv.FormatInt(incident.ID, 10)
	if err := s.Store.UpdateActionRun(ctx, *run); err != nil {
		return nil, err
	}
	if strings.TrimSpace(incident.PrimaryActionRunID) == "" {
		if err := s.Store.UpdateIncidentPrimaryActionRun(ctx, incident.ID, run.ID); err != nil {
			return nil, err
		}
		incident.PrimaryActionRunID = run.ID
	}

	now := occurredAt.UTC()
	if now.IsZero() {
		now = s.now()
	}
	operator := strings.TrimSpace(actor)
	if _, err := s.Store.RecordAuditTimelineEntry(ctx, audit.AuditTimelineEntry{
		OccurredAt:  now,
		Severity:    audit.AuditSeverityInfo,
		EventType:   audit.EventTypeIncidentLinked,
		Summary:     fmt.Sprintf("Action run %s linked to incident %d", run.ID, incident.ID),
		DetailJSON:  incidentAuditDetail(*incident),
		ActionRunID: run.ID,
		Operator:    operator,
	}); err != nil {
		return nil, err
	}
	return incident, nil
}

func (s *Service) now() time.Time {
	clock := s.Clock
	if clock == nil {
		clock = time.Now
	}
	return clock().UTC()
}

func severityForChannel(channel *inputs.InputChannel) string {
	if channel == nil {
		return "WARNING"
	}
	id := strings.ToLower(strings.TrimSpace(channel.ID))
	switch {
	case strings.Contains(id, "emergency"):
		return "EMERGENCY"
	case channel.Priority >= 300:
		return "WARNING"
	default:
		return "INFO"
	}
}

func mapIncidentSeverity(value string) audit.AuditSeverity {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "EMERGENCY":
		return audit.AuditSeverityEmergency
	case "WARNING":
		return audit.AuditSeverityWarning
	default:
		return audit.AuditSeverityInfo
	}
}

func incidentAuditDetail(incident IncidentRecord) string {
	payload, _ := json.Marshal(map[string]any{
		"incident_id":             incident.ID,
		"status":                  incident.Status,
		"severity":                incident.Severity,
		"primary_input_id":        incident.PrimaryInputID,
		"primary_action_run_id":   incident.PrimaryActionRunID,
		"opened_by_transition_id": incident.OpenedByTransitionID,
		"closed_by_transition_id": incident.ClosedByTransitionID,
		"close_reason":            incident.CloseReason,
		"summary":                 incident.Summary,
	})
	return string(payload)
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
