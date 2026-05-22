package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

const (
	endpointIntegrityAlertsPathPrefix = "/api/endpoint-integrity/alerts/"
	endpointIntegritySnoozeMaxMinutes = 7 * 24 * 60

	endpointIntegritySeverityAlert   = "ALERT"
	endpointIntegritySeverityWarning = "WARNING"
)

type endpointIntegrityAlertView struct {
	ID           string                      `json:"id"`
	Type         model.EndpointCollisionType `json:"type"`
	EGMIDs       []string                    `json:"egm_ids"`
	Endpoint     string                      `json:"endpoint"`
	FirstSeenAt  time.Time                   `json:"first_seen_at,omitempty"`
	LastSeenAt   time.Time                   `json:"last_seen_at,omitempty"`
	Severity     string                      `json:"severity"`
	AckedAt      *time.Time                  `json:"acked_at,omitempty"`
	AckedByScope string                      `json:"acked_by_scope,omitempty"`
	SnoozedUntil *time.Time                  `json:"snoozed_until,omitempty"`
	SnoozeReason string                      `json:"snooze_reason,omitempty"`
}

type endpointIntegrityAlertsSummary struct {
	Total           int `json:"total"`
	ActiveCount     int `json:"active_count"`
	AckedCount      int `json:"acked_count"`
	SnoozedCount    int `json:"snoozed_count"`
	SuppressedCount int `json:"suppressed_count"`
}

type endpointIntegrityAlertsResponse struct {
	GeneratedAt   time.Time                      `json:"generated_at"`
	Summary       endpointIntegrityAlertsSummary `json:"summary"`
	Alerts        []endpointIntegrityAlertView   `json:"alerts"`
	ActiveAlerts  []endpointIntegrityAlertView   `json:"active_alerts"`
	AckedAlerts   []endpointIntegrityAlertView   `json:"acked_alerts"`
	SnoozedAlerts []endpointIntegrityAlertView   `json:"snoozed_alerts"`
}

type endpointIntegritySnoozeRequest struct {
	Minutes      int    `json:"minutes"`
	SnoozeReason string `json:"snooze_reason"`
}

type endpointIntegrityAlertState string

const (
	endpointIntegrityAlertActive  endpointIntegrityAlertState = "active"
	endpointIntegrityAlertAcked   endpointIntegrityAlertState = "acked"
	endpointIntegrityAlertSnoozed endpointIntegrityAlertState = "snoozed"
)

func endpointIntegrityAlertsHandler(eng *engine.Engine, auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		reference := time.Now().UTC()
		if _, err := auditStore.ClearExpiredEndpointIntegrityAlertSnoozes(r.Context(), reference); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "endpoint_integrity_alerts.list", "fail", "Endpoint integrity alerts load failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		response, err := buildEndpointIntegrityAlertsResponse(r.Context(), eng, auditStore, reference)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "endpoint_integrity_alerts.list", "fail", "Endpoint integrity alerts load failed", err.Error())
		}
		writeJSON(w, response, err)
	}
}

func endpointIntegrityAlertActionHandler(eng *engine.Engine, auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		alertID, action, err := parseEndpointIntegrityAlertActionPath(r.URL.Path)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "endpoint_integrity_alert.invalid", "fail", "Endpoint integrity alert action rejected", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		reference := time.Now().UTC()
		if _, err := auditStore.ClearExpiredEndpointIntegrityAlertSnoozes(r.Context(), reference); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "endpoint_integrity_alert."+action, "fail", "Endpoint integrity alert action failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		currentAlerts, err := loadEndpointIntegrityAlerts(r.Context(), eng, auditStore, reference)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "endpoint_integrity_alert."+action, "fail", "Endpoint integrity alert action failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		alertView, ok := currentAlerts[alertID]
		if !ok {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "endpoint_integrity_alert."+action, "fail", "Endpoint integrity alert action rejected", "alert id not found in current collisions")
			http.Error(w, "alert id not found in current endpoint integrity collisions", http.StatusNotFound)
			return
		}

		switch action {
		case "ack":
			if err := auditStore.UpsertEndpointIntegrityAlertAck(
				r.Context(),
				alertID,
				reference,
				operatorAuditActorScope(r, cfg),
				updateActorNameFromRequest(r),
			); err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "endpoint_integrity_alert.ack", "fail", "Endpoint integrity alert ack failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(
				r.Context(),
				auditStore,
				r,
				cfg,
				"endpoint_integrity_alert.ack",
				"success",
				"Endpoint integrity alert acknowledged",
				"alert_id="+alertID+" type="+string(alertView.Type)+" endpoint="+alertView.Endpoint,
			)
		case "snooze":
			request, err := decodeEndpointIntegritySnoozeRequest(r)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "endpoint_integrity_alert.snooze", "fail", "Endpoint integrity alert snooze rejected", err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			snoozedUntil := reference.Add(time.Duration(request.Minutes) * time.Minute)
			if err := auditStore.UpsertEndpointIntegrityAlertSnooze(
				r.Context(),
				alertID,
				snoozedUntil,
				request.SnoozeReason,
				updateActorNameFromRequest(r),
			); err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "endpoint_integrity_alert.snooze", "fail", "Endpoint integrity alert snooze failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(
				r.Context(),
				auditStore,
				r,
				cfg,
				"endpoint_integrity_alert.snooze",
				"success",
				"Endpoint integrity alert snoozed",
				fmt.Sprintf("alert_id=%s minutes=%d endpoint=%s reason=%s", alertID, request.Minutes, alertView.Endpoint, strings.TrimSpace(request.SnoozeReason)),
			)
		case "unsnooze":
			if err := auditStore.ClearEndpointIntegrityAlertSnooze(r.Context(), alertID, updateActorNameFromRequest(r)); err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "endpoint_integrity_alert.unsnooze", "fail", "Endpoint integrity alert unsnooze failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(
				r.Context(),
				auditStore,
				r,
				cfg,
				"endpoint_integrity_alert.unsnooze",
				"success",
				"Endpoint integrity alert unsnoozed",
				"alert_id="+alertID+" endpoint="+alertView.Endpoint,
			)
		default:
			http.Error(w, "action not found", http.StatusNotFound)
			return
		}

		response, err := buildEndpointIntegrityAlertsResponse(r.Context(), eng, auditStore, time.Now().UTC())
		writeJSON(w, response, err)
	}
}

func parseEndpointIntegrityAlertActionPath(path string) (string, string, error) {
	if !strings.HasPrefix(path, endpointIntegrityAlertsPathPrefix) {
		return "", "", fmt.Errorf("invalid endpoint integrity alert action path")
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, endpointIntegrityAlertsPathPrefix), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("endpoint integrity alert action path must be /api/endpoint-integrity/alerts/{id}/{action}")
	}
	alertID, err := url.PathUnescape(strings.TrimSpace(parts[0]))
	if err != nil {
		return "", "", fmt.Errorf("invalid alert id")
	}
	action := strings.ToLower(strings.TrimSpace(parts[1]))
	if alertID == "" {
		return "", "", fmt.Errorf("alert id is required")
	}
	if action != "ack" && action != "snooze" && action != "unsnooze" {
		return "", "", fmt.Errorf("action must be ack, snooze, or unsnooze")
	}
	return alertID, action, nil
}

func decodeEndpointIntegritySnoozeRequest(r *http.Request) (endpointIntegritySnoozeRequest, error) {
	payload := endpointIntegritySnoozeRequest{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return endpointIntegritySnoozeRequest{}, fmt.Errorf("invalid JSON body")
	}
	if payload.Minutes <= 0 {
		return endpointIntegritySnoozeRequest{}, fmt.Errorf("minutes must be greater than zero")
	}
	if payload.Minutes > endpointIntegritySnoozeMaxMinutes {
		return endpointIntegritySnoozeRequest{}, fmt.Errorf("minutes must be less than or equal to %d", endpointIntegritySnoozeMaxMinutes)
	}
	payload.SnoozeReason = strings.TrimSpace(payload.SnoozeReason)
	return payload, nil
}

func buildEndpointIntegrityAlertsResponse(ctx context.Context, eng *engine.Engine, auditStore *store.SQLiteStore, reference time.Time) (endpointIntegrityAlertsResponse, error) {
	rowsByID, err := loadEndpointIntegrityAlerts(ctx, eng, auditStore, reference)
	if err != nil {
		return endpointIntegrityAlertsResponse{}, err
	}
	rows := make([]endpointIntegrityAlertView, 0, len(rowsByID))
	for _, row := range rowsByID {
		rows = append(rows, row)
	}
	sortEndpointIntegrityAlerts(rows)

	active := []endpointIntegrityAlertView{}
	acked := []endpointIntegrityAlertView{}
	snoozed := []endpointIntegrityAlertView{}
	for _, row := range rows {
		switch endpointIntegrityAlertPresentationStateFor(row, reference) {
		case endpointIntegrityAlertSnoozed:
			snoozed = append(snoozed, row)
		case endpointIntegrityAlertAcked:
			acked = append(acked, row)
		default:
			active = append(active, row)
		}
	}
	return endpointIntegrityAlertsResponse{
		GeneratedAt: reference,
		Summary: endpointIntegrityAlertsSummary{
			Total:           len(rows),
			ActiveCount:     len(active),
			AckedCount:      len(acked),
			SnoozedCount:    len(snoozed),
			SuppressedCount: len(acked) + len(snoozed),
		},
		Alerts:        rows,
		ActiveAlerts:  active,
		AckedAlerts:   acked,
		SnoozedAlerts: snoozed,
	}, nil
}

func loadEndpointIntegrityAlerts(ctx context.Context, eng *engine.Engine, auditStore *store.SQLiteStore, reference time.Time) (map[string]endpointIntegrityAlertView, error) {
	states, err := auditStore.ListEndpointIntegrityAlertStates(ctx)
	if err != nil {
		return nil, err
	}
	stateByID := make(map[string]store.EndpointIntegrityAlertState, len(states))
	for _, row := range states {
		stateByID[row.AlertID] = row
	}

	snapshot := eng.Snapshot()
	rows := map[string]endpointIntegrityAlertView{}
	for _, collision := range snapshot.EndpointCollisions {
		alertID := endpointIntegrityAlertID(collision)
		ids := normalizedCollisionEGMIDs(collision.InvolvedEGMIDs)
		row := endpointIntegrityAlertView{
			ID:          alertID,
			Type:        collision.CollisionType,
			EGMIDs:      ids,
			Endpoint:    strings.TrimSpace(collision.Endpoint),
			FirstSeenAt: collision.FirstSeenAt,
			LastSeenAt:  collision.LastSeenAt,
			Severity:    endpointIntegrityAlertSeverity(collision.CollisionType),
		}
		if state, ok := stateByID[alertID]; ok {
			if state.AckedAt != nil {
				ackedAt := state.AckedAt.UTC()
				row.AckedAt = &ackedAt
				row.AckedByScope = strings.TrimSpace(state.AckedByScope)
			}
			if state.SnoozedUntil != nil && state.SnoozedUntil.After(reference) {
				snoozedUntil := state.SnoozedUntil.UTC()
				row.SnoozedUntil = &snoozedUntil
				row.SnoozeReason = strings.TrimSpace(state.SnoozeReason)
			}
		}
		rows[row.ID] = row
	}
	return rows, nil
}

func endpointIntegrityAlertID(collision model.EndpointCollision) string {
	sum := sha256.Sum256([]byte(endpointIntegrityAlertSignature(collision)))
	return "eia-" + hex.EncodeToString(sum[:8])
}

func endpointIntegrityAlertSignature(collision model.EndpointCollision) string {
	ids := normalizedCollisionEGMIDs(collision.InvolvedEGMIDs)
	return strings.ToUpper(strings.TrimSpace(string(collision.CollisionType))) + "|" +
		strings.TrimSpace(collision.Endpoint) + "|" +
		strings.Join(ids, ",")
}

func normalizedCollisionEGMIDs(ids []string) []string {
	normalized := make([]string, 0, len(ids))
	for _, item := range ids {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}

func endpointIntegrityAlertSeverity(collisionType model.EndpointCollisionType) string {
	if collisionType == model.EndpointCollisionSharedEndpoint {
		return endpointIntegritySeverityAlert
	}
	return endpointIntegritySeverityWarning
}

func endpointIntegrityAlertPresentationStateFor(alert endpointIntegrityAlertView, reference time.Time) endpointIntegrityAlertState {
	if alert.SnoozedUntil != nil && alert.SnoozedUntil.After(reference) {
		return endpointIntegrityAlertSnoozed
	}
	if alert.AckedAt != nil {
		return endpointIntegrityAlertAcked
	}
	return endpointIntegrityAlertActive
}

func sortEndpointIntegrityAlerts(rows []endpointIntegrityAlertView) {
	sort.Slice(rows, func(i, j int) bool {
		iTime := rows[i].LastSeenAt
		jTime := rows[j].LastSeenAt
		if !iTime.Equal(jTime) {
			return iTime.After(jTime)
		}
		if rows[i].Severity != rows[j].Severity {
			return rows[i].Severity > rows[j].Severity
		}
		return rows[i].ID < rows[j].ID
	})
}
