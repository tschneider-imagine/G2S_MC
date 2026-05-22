package main

import (
	"context"
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

const egmRegistryPathPrefix = "/api/egm-registry/"

type egmRegistryOverrideView struct {
	EGMID           string    `json:"egm_id"`
	DisplayName     string    `json:"display_name,omitempty"`
	Vendor          string    `json:"vendor,omitempty"`
	CabinetFamily   string    `json:"cabinet_family,omitempty"`
	GameTitle       string    `json:"game_title,omitempty"`
	SoftwareVersion string    `json:"software_version,omitempty"`
	Notes           string    `json:"notes,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
	UpdatedBy       string    `json:"updated_by,omitempty"`
}

type egmRegistryResponse struct {
	GeneratedAt time.Time                 `json:"generated_at"`
	Overrides   []egmRegistryOverrideView `json:"overrides"`
	EGMs        []model.EGM               `json:"egms"`
}

type egmRegistryPromoteRequest struct {
	EGMID           string `json:"egm_id"`
	DisplayName     string `json:"display_name"`
	Vendor          string `json:"vendor"`
	CabinetFamily   string `json:"cabinet_family"`
	GameTitle       string `json:"game_title"`
	SoftwareVersion string `json:"software_version"`
	Notes           string `json:"notes"`
}

type egmRegistryUpdateRequest struct {
	DisplayName     string `json:"display_name"`
	Vendor          string `json:"vendor"`
	CabinetFamily   string `json:"cabinet_family"`
	GameTitle       string `json:"game_title"`
	SoftwareVersion string `json:"software_version"`
	Notes           string `json:"notes"`
}

func egmRegistryHandler(eng *engine.Engine, auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		payload, err := buildEGMRegistryResponse(r.Context(), eng, auditStore)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		writeJSON(w, payload, nil)
	}
}

func egmRegistryPromoteHandler(eng *engine.Engine, auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		payload := egmRegistryPromoteRequest{}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.promote", "fail", "EGM registry promote rejected", "invalid JSON body")
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		egmID := strings.TrimSpace(payload.EGMID)
		if egmID == "" {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.promote", "fail", "EGM registry promote rejected", "egm_id is required")
			http.Error(w, "egm_id is required", http.StatusBadRequest)
			return
		}
		if err := validateEGMRegistryTextFields(payload.DisplayName, payload.Vendor, payload.CabinetFamily, payload.GameTitle, payload.SoftwareVersion, payload.Notes); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.promote", "fail", "EGM registry promote rejected", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		snapshot := eng.Snapshot()
		found, discovered := snapshotEGMByID(snapshot, egmID)
		if !found {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.promote", "fail", "EGM registry promote rejected", "egm_id not found in runtime snapshot")
			http.Error(w, "egm_id not found in runtime snapshot", http.StatusNotFound)
			return
		}

		override := store.EGMRegistryOverride{
			EGMID:           egmID,
			DisplayName:     firstNonEmpty(payload.DisplayName, discovered.DisplayName, discovered.ID),
			Vendor:          firstNonEmpty(payload.Vendor, discovered.Vendor),
			CabinetFamily:   firstNonEmpty(payload.CabinetFamily, discovered.CabinetFamily),
			GameTitle:       firstNonEmpty(payload.GameTitle, discovered.GameTitle),
			SoftwareVersion: firstNonEmpty(payload.SoftwareVersion, discovered.SoftwareVersion),
			Notes:           strings.TrimSpace(payload.Notes),
			UpdatedBy:       updateActorNameFromRequest(r),
		}
		if err := auditStore.UpsertEGMRegistryOverride(r.Context(), override); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.promote", "fail", "EGM registry promote failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.promote", "success", "Discovered EGM promoted to registry override", "egm_id="+egmID)

		response, err := buildEGMRegistryResponse(r.Context(), eng, auditStore)
		writeJSON(w, response, err)
	}
}

func egmRegistryByIDHandler(eng *engine.Engine, auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		egmID, err := parseEGMRegistryIDFromPath(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodPut:
			payload := egmRegistryUpdateRequest{}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&payload); err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.update", "fail", "EGM registry update rejected", "invalid JSON body")
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			if err := validateEGMRegistryTextFields(payload.DisplayName, payload.Vendor, payload.CabinetFamily, payload.GameTitle, payload.SoftwareVersion, payload.Notes); err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.update", "fail", "EGM registry update rejected", err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			override := store.EGMRegistryOverride{
				EGMID:           egmID,
				DisplayName:     strings.TrimSpace(payload.DisplayName),
				Vendor:          strings.TrimSpace(payload.Vendor),
				CabinetFamily:   strings.TrimSpace(payload.CabinetFamily),
				GameTitle:       strings.TrimSpace(payload.GameTitle),
				SoftwareVersion: strings.TrimSpace(payload.SoftwareVersion),
				Notes:           strings.TrimSpace(payload.Notes),
				UpdatedBy:       updateActorNameFromRequest(r),
			}
			if err := auditStore.UpsertEGMRegistryOverride(r.Context(), override); err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.update", "fail", "EGM registry update failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.update", "success", "EGM registry override saved", "egm_id="+egmID)
			response, err := buildEGMRegistryResponse(r.Context(), eng, auditStore)
			writeJSON(w, response, err)
		case http.MethodDelete:
			deleted, err := auditStore.DeleteEGMRegistryOverride(r.Context(), egmID)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.delete", "fail", "EGM registry override delete failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			if !deleted {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.delete", "fail", "EGM registry override delete failed", "egm_id not found")
				http.Error(w, "egm_id not found", http.StatusNotFound)
				return
			}
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "egm_registry.delete", "success", "EGM registry override deleted", "egm_id="+egmID)
			response, err := buildEGMRegistryResponse(r.Context(), eng, auditStore)
			writeJSON(w, response, err)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func parseEGMRegistryIDFromPath(path string) (string, error) {
	if !strings.HasPrefix(path, egmRegistryPathPrefix) {
		return "", fmt.Errorf("invalid egm registry path")
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, egmRegistryPathPrefix), "/")
	if trimmed == "" {
		return "", fmt.Errorf("egm_id is required")
	}
	if strings.Contains(trimmed, "/") {
		return "", fmt.Errorf("invalid egm registry path")
	}
	value, err := url.PathUnescape(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid egm_id")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("egm_id is required")
	}
	return value, nil
}

func validateEGMRegistryTextFields(displayName string, vendor string, cabinetFamily string, gameTitle string, softwareVersion string, notes string) error {
	if len(strings.TrimSpace(displayName)) > 120 {
		return fmt.Errorf("display_name must be 120 characters or fewer")
	}
	if len(strings.TrimSpace(vendor)) > 120 {
		return fmt.Errorf("vendor must be 120 characters or fewer")
	}
	if len(strings.TrimSpace(cabinetFamily)) > 120 {
		return fmt.Errorf("cabinet_family must be 120 characters or fewer")
	}
	if len(strings.TrimSpace(gameTitle)) > 160 {
		return fmt.Errorf("game_title must be 160 characters or fewer")
	}
	if len(strings.TrimSpace(softwareVersion)) > 120 {
		return fmt.Errorf("software_version must be 120 characters or fewer")
	}
	if len(strings.TrimSpace(notes)) > 2000 {
		return fmt.Errorf("notes must be 2000 characters or fewer")
	}
	return nil
}

func snapshotEGMByID(snapshot engine.Snapshot, egmID string) (bool, model.EGM) {
	target := strings.TrimSpace(egmID)
	for _, row := range snapshot.EGMs {
		if strings.TrimSpace(row.ID) == target {
			return true, row
		}
	}
	return false, model.EGM{}
}

func buildEGMRegistryResponse(ctx context.Context, eng *engine.Engine, auditStore *store.SQLiteStore) (egmRegistryResponse, error) {
	overrides, err := auditStore.ListEGMRegistryOverrides(ctx)
	if err != nil {
		return egmRegistryResponse{}, err
	}
	snapshot := eng.Snapshot()
	snapshot = applyEGMRegistryOverrides(snapshot, overrides)
	return egmRegistryResponse{
		GeneratedAt: time.Now().UTC(),
		Overrides:   buildEGMRegistryOverrideViews(overrides),
		EGMs:        snapshot.EGMs,
	}, nil
}

func buildEGMRegistryOverrideViews(overrides []store.EGMRegistryOverride) []egmRegistryOverrideView {
	rows := make([]egmRegistryOverrideView, 0, len(overrides))
	for _, row := range overrides {
		rows = append(rows, egmRegistryOverrideView{
			EGMID:           row.EGMID,
			DisplayName:     row.DisplayName,
			Vendor:          row.Vendor,
			CabinetFamily:   row.CabinetFamily,
			GameTitle:       row.GameTitle,
			SoftwareVersion: row.SoftwareVersion,
			Notes:           row.Notes,
			UpdatedAt:       row.UpdatedAt,
			UpdatedBy:       row.UpdatedBy,
		})
	}
	return rows
}

func applyEGMRegistryOverrides(snapshot engine.Snapshot, overrides []store.EGMRegistryOverride) engine.Snapshot {
	overrideByID := map[string]store.EGMRegistryOverride{}
	for _, override := range overrides {
		id := strings.TrimSpace(override.EGMID)
		if id == "" {
			continue
		}
		overrideByID[id] = override
	}
	for i := range snapshot.EGMs {
		id := strings.TrimSpace(snapshot.EGMs[i].ID)
		override, ok := overrideByID[id]
		if !ok {
			continue
		}
		snapshot.EGMs[i].DisplayName = firstNonEmpty(strings.TrimSpace(override.DisplayName), snapshot.EGMs[i].DisplayName, snapshot.EGMs[i].ID)
		snapshot.EGMs[i].Vendor = firstNonEmpty(strings.TrimSpace(override.Vendor), snapshot.EGMs[i].Vendor)
		snapshot.EGMs[i].CabinetFamily = firstNonEmpty(strings.TrimSpace(override.CabinetFamily), snapshot.EGMs[i].CabinetFamily)
		snapshot.EGMs[i].GameTitle = firstNonEmpty(strings.TrimSpace(override.GameTitle), snapshot.EGMs[i].GameTitle)
		snapshot.EGMs[i].SoftwareVersion = firstNonEmpty(strings.TrimSpace(override.SoftwareVersion), snapshot.EGMs[i].SoftwareVersion)
		snapshot.EGMs[i].Notes = strings.TrimSpace(override.Notes)
		snapshot.EGMs[i].RegistryOverride = true
		if snapshot.EGMs[i].Source == model.EGMSourceDiscovered {
			snapshot.EGMs[i].Source = model.EGMSourceConfigured
		}
	}
	sort.Slice(snapshot.EGMs, func(i, j int) bool {
		return snapshot.EGMs[i].ID < snapshot.EGMs[j].ID
	})
	return snapshot
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
