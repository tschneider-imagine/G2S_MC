package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

const runtimeOverridesPresetPathPrefix = "/api/runtime-overrides/presets/"

var runtimeOverridePresetNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

type runtimeOverridesSnapshotResponse struct {
	GeneratedAt             time.Time                    `json:"generated_at"`
	CabinetProfileOverride  *cabinetProfileOverrideView  `json:"cabinet_profile_override,omitempty"`
	HeartbeatPolicyOverride *heartbeatPolicyOverrideView `json:"heartbeat_policy_override,omitempty"`
	EGMRegistryOverrides    []egmRegistryOverrideView    `json:"egm_registry_overrides"`
}

type runtimeOverridesRestoreRequest struct {
	CabinetProfileOverride  *config.CabinetProfile    `json:"cabinet_profile_override"`
	HeartbeatPolicyOverride *heartbeatPolicy          `json:"heartbeat_policy_override"`
	EGMRegistryOverrides    []egmRegistryOverrideView `json:"egm_registry_overrides"`
}

type runtimeOverridesRestoreResponse struct {
	Snapshot        runtimeOverridesSnapshotResponse `json:"snapshot"`
	CabinetProfile  cabinetProfileResponse           `json:"cabinet_profile"`
	HeartbeatPolicy heartbeatPolicyResponse          `json:"heartbeat_policy"`
	EGMRegistry     egmRegistryResponse              `json:"egm_registry"`
}

type runtimeOverridePresetListEntry struct {
	Name      string    `json:"name"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type runtimeOverridePresetListResponse struct {
	GeneratedAt time.Time                        `json:"generated_at"`
	Presets     []runtimeOverridePresetListEntry `json:"presets"`
}

type runtimeOverridePresetSaveRequest struct {
	Name string `json:"name"`
	Note string `json:"note"`
}

type runtimeOverridePresetLoadRequest struct {
	Name string `json:"name"`
}

type runtimeOverridePresetLoadResponse struct {
	Name    string                          `json:"name"`
	Note    string                          `json:"note,omitempty"`
	Restore runtimeOverridesRestoreResponse `json:"restore"`
}

func runtimeOverridesSnapshotHandler(auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snapshot, err := buildRuntimeOverridesSnapshotResponse(r.Context(), auditStore, cfg)
		writeJSON(w, snapshot, err)
	}
}

func runtimeOverridesPresetsHandler(auditStore *store.SQLiteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		response, err := buildRuntimeOverridePresetListResponse(r.Context(), auditStore)
		writeJSON(w, response, err)
	}
}

func runtimeOverridesPresetsSaveHandler(auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		payload := runtimeOverridePresetSaveRequest{}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_save", "fail", "Runtime preset save rejected", "invalid JSON body")
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		name, err := normalizeRuntimeOverridePresetName(payload.Name)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_save", "fail", "Runtime preset save rejected", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		note := strings.TrimSpace(payload.Note)
		if len(note) > 2000 {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_save", "fail", "Runtime preset save rejected", "note must be 2000 characters or fewer")
			http.Error(w, "note must be 2000 characters or fewer", http.StatusBadRequest)
			return
		}
		snapshot, err := buildRuntimeOverridesSnapshotResponse(r.Context(), auditStore, cfg)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_save", "fail", "Runtime preset save failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		serialized, err := json.Marshal(snapshot)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_save", "fail", "Runtime preset save failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		if err := auditStore.UpsertRuntimeOverridePreset(r.Context(), store.RuntimeOverridePreset{
			Name:        name,
			Note:        note,
			PayloadJSON: string(serialized),
		}); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_save", "fail", "Runtime preset save failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_save", "success", "Runtime preset saved", "name="+name)
		response, err := buildRuntimeOverridePresetListResponse(r.Context(), auditStore)
		writeJSON(w, response, err)
	}
}

func runtimeOverridesPresetsLoadHandler(eng *engine.Engine, auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		payload := runtimeOverridePresetLoadRequest{}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_load", "fail", "Runtime preset load rejected", "invalid JSON body")
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		name, err := normalizeRuntimeOverridePresetName(payload.Name)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_load", "fail", "Runtime preset load rejected", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		preset, err := auditStore.GetRuntimeOverridePreset(r.Context(), name)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_load", "fail", "Runtime preset load failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		if preset == nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_load", "fail", "Runtime preset load failed", "preset not found")
			http.Error(w, "preset not found", http.StatusNotFound)
			return
		}
		snapshotPayload := runtimeOverridesSnapshotResponse{}
		if err := json.Unmarshal([]byte(preset.PayloadJSON), &snapshotPayload); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_load", "fail", "Runtime preset load failed", "preset payload is invalid")
			http.Error(w, "preset payload is invalid", http.StatusBadRequest)
			return
		}
		restorePayload := runtimeOverridesRestoreRequestFromSnapshot(snapshotPayload)
		restoreInput, err := runtimeOverridesReplaceInputFromRestoreRequest(restorePayload, updateActorNameFromRequest(r), operatorAuditActorScope(r, cfg))
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_load", "fail", "Runtime preset load failed", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		restoreResponse, err := applyRuntimeOverridesReplaceInput(r.Context(), eng, auditStore, cfg, restoreInput)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_load", "fail", "Runtime preset load failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_load", "success", "Runtime preset loaded", "name="+name)
		writeJSON(w, runtimeOverridePresetLoadResponse{
			Name:    preset.Name,
			Note:    preset.Note,
			Restore: restoreResponse,
		}, nil)
	}
}

func runtimeOverridesPresetByNameHandler(auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name, err := runtimeOverridePresetNameFromPath(r.URL.Path)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_delete", "fail", "Runtime preset delete rejected", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		deleted, err := auditStore.DeleteRuntimeOverridePreset(r.Context(), name)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_delete", "fail", "Runtime preset delete failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		if !deleted {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_delete", "fail", "Runtime preset delete failed", "preset not found")
			http.Error(w, "preset not found", http.StatusNotFound)
			return
		}
		recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.preset_delete", "success", "Runtime preset deleted", "name="+name)
		response, err := buildRuntimeOverridePresetListResponse(r.Context(), auditStore)
		writeJSON(w, response, err)
	}
}

func runtimeOverridesRestoreHandler(eng *engine.Engine, auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		payload := runtimeOverridesRestoreRequest{}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore rejected", "invalid JSON body")
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		restoreInput, err := runtimeOverridesReplaceInputFromRestoreRequest(payload, updateActorNameFromRequest(r), operatorAuditActorScope(r, cfg))
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore rejected", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		response, err := applyRuntimeOverridesReplaceInput(r.Context(), eng, auditStore, cfg, restoreInput)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		recordOperatorAuditEvent(
			r.Context(),
			auditStore,
			r,
			cfg,
			"runtime_overrides.restore",
			"success",
			"Runtime override snapshot restored",
			fmt.Sprintf(
				"cabinet_profile_override=%t heartbeat_policy_override=%t egm_registry_overrides=%d",
				restoreInput.CabinetProfileOverride != nil,
				restoreInput.HeartbeatPolicyOverride != nil,
				len(restoreInput.EGMRegistryOverrides),
			),
		)
		writeJSON(w, response, nil)
	}
}

func normalizeRuntimeOverridePresetName(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("name is required")
	}
	if !runtimeOverridePresetNamePattern.MatchString(value) {
		return "", fmt.Errorf("name must match ^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$")
	}
	return value, nil
}

func runtimeOverridePresetNameFromPath(path string) (string, error) {
	if !strings.HasPrefix(path, runtimeOverridesPresetPathPrefix) {
		return "", fmt.Errorf("invalid runtime preset path")
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, runtimeOverridesPresetPathPrefix), "/")
	if trimmed == "" {
		return "", fmt.Errorf("name is required")
	}
	if strings.Contains(trimmed, "/") {
		return "", fmt.Errorf("invalid runtime preset path")
	}
	value, err := url.PathUnescape(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid preset name")
	}
	return normalizeRuntimeOverridePresetName(value)
}

func buildRuntimeOverridePresetListResponse(ctx context.Context, auditStore *store.SQLiteStore) (runtimeOverridePresetListResponse, error) {
	rows, err := auditStore.ListRuntimeOverridePresets(ctx)
	if err != nil {
		return runtimeOverridePresetListResponse{}, err
	}
	items := make([]runtimeOverridePresetListEntry, 0, len(rows))
	for _, row := range rows {
		items = append(items, runtimeOverridePresetListEntry{
			Name:      row.Name,
			Note:      row.Note,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return runtimeOverridePresetListResponse{
		GeneratedAt: time.Now().UTC(),
		Presets:     items,
	}, nil
}

func runtimeOverridesRestoreRequestFromSnapshot(snapshot runtimeOverridesSnapshotResponse) runtimeOverridesRestoreRequest {
	request := runtimeOverridesRestoreRequest{
		EGMRegistryOverrides: append([]egmRegistryOverrideView{}, snapshot.EGMRegistryOverrides...),
	}
	if snapshot.CabinetProfileOverride != nil {
		profile := snapshot.CabinetProfileOverride.Profile
		request.CabinetProfileOverride = &profile
	}
	if snapshot.HeartbeatPolicyOverride != nil {
		request.HeartbeatPolicyOverride = &heartbeatPolicy{
			IntervalMS:         snapshot.HeartbeatPolicyOverride.IntervalMS,
			WarningAfterMissed: snapshot.HeartbeatPolicyOverride.WarningAfterMissed,
			BlockAfterMissed:   snapshot.HeartbeatPolicyOverride.BlockAfterMissed,
		}
	}
	return request
}

func runtimeOverridesReplaceInputFromRestoreRequest(payload runtimeOverridesRestoreRequest, updatedBy string, actorScope string) (store.RuntimeOverridesReplaceInput, error) {
	if payload.CabinetProfileOverride != nil {
		if err := config.ValidateCabinetProfile(*payload.CabinetProfileOverride); err != nil {
			return store.RuntimeOverridesReplaceInput{}, err
		}
	}
	if payload.HeartbeatPolicyOverride != nil {
		if payload.HeartbeatPolicyOverride.IntervalMS <= 0 {
			return store.RuntimeOverridesReplaceInput{}, fmt.Errorf("heartbeat_policy_override.interval_ms must be greater than zero")
		}
		if payload.HeartbeatPolicyOverride.WarningAfterMissed < 1 {
			return store.RuntimeOverridesReplaceInput{}, fmt.Errorf("heartbeat_policy_override.warning_after_missed must be greater than or equal to 1")
		}
		if payload.HeartbeatPolicyOverride.BlockAfterMissed < payload.HeartbeatPolicyOverride.WarningAfterMissed {
			return store.RuntimeOverridesReplaceInput{}, fmt.Errorf("heartbeat_policy_override.block_after_missed must be greater than or equal to warning_after_missed")
		}
	}

	registryOverrides := make([]store.EGMRegistryOverride, 0, len(payload.EGMRegistryOverrides))
	for i, row := range payload.EGMRegistryOverrides {
		egmID := strings.TrimSpace(row.EGMID)
		if egmID == "" {
			return store.RuntimeOverridesReplaceInput{}, fmt.Errorf("egm_registry_overrides[%d].egm_id is required", i)
		}
		if err := validateEGMRegistryTextFields(row.DisplayName, row.Vendor, row.CabinetFamily, row.GameTitle, row.SoftwareVersion, row.Notes); err != nil {
			return store.RuntimeOverridesReplaceInput{}, fmt.Errorf("egm_registry_overrides[%d] rejected: %s", i, err.Error())
		}
		registryOverrides = append(registryOverrides, store.EGMRegistryOverride{
			EGMID:           egmID,
			DisplayName:     strings.TrimSpace(row.DisplayName),
			Vendor:          strings.TrimSpace(row.Vendor),
			CabinetFamily:   strings.TrimSpace(row.CabinetFamily),
			GameTitle:       strings.TrimSpace(row.GameTitle),
			SoftwareVersion: strings.TrimSpace(row.SoftwareVersion),
			Notes:           strings.TrimSpace(row.Notes),
			UpdatedBy:       updatedBy,
		})
	}

	restore := store.RuntimeOverridesReplaceInput{
		EGMRegistryOverrides: registryOverrides,
	}
	if payload.CabinetProfileOverride != nil {
		restore.CabinetProfileOverride = &store.CabinetProfileOverride{
			Profile:   *payload.CabinetProfileOverride,
			UpdatedBy: updatedBy,
		}
	}
	if payload.HeartbeatPolicyOverride != nil {
		restore.HeartbeatPolicyOverride = &store.HeartbeatPolicyOverride{
			IntervalMS:         payload.HeartbeatPolicyOverride.IntervalMS,
			WarningAfterMissed: payload.HeartbeatPolicyOverride.WarningAfterMissed,
			BlockAfterMissed:   payload.HeartbeatPolicyOverride.BlockAfterMissed,
			UpdatedBy:          updatedBy,
		}
	}
	return restore, nil
}

func applyRuntimeOverridesReplaceInput(ctx context.Context, eng *engine.Engine, auditStore *store.SQLiteStore, cfg config.Config, restore store.RuntimeOverridesReplaceInput) (runtimeOverridesRestoreResponse, error) {
	if err := auditStore.ReplaceRuntimeOverrides(ctx, restore); err != nil {
		return runtimeOverridesRestoreResponse{}, err
	}
	cabinetResolved, err := resolveCabinetProfile(ctx, auditStore, cfg.CabinetProfile)
	if err != nil {
		return runtimeOverridesRestoreResponse{}, err
	}
	heartbeatResolved, err := resolveHeartbeatPolicy(ctx, auditStore, cfg.Timeouts)
	if err != nil {
		return runtimeOverridesRestoreResponse{}, err
	}
	egmRegistryResolved, err := buildEGMRegistryResponse(ctx, eng, auditStore)
	if err != nil {
		return runtimeOverridesRestoreResponse{}, err
	}
	snapshot, err := buildRuntimeOverridesSnapshotResponse(ctx, auditStore, cfg)
	if err != nil {
		return runtimeOverridesRestoreResponse{}, err
	}
	return runtimeOverridesRestoreResponse{
		Snapshot:        snapshot,
		CabinetProfile:  buildCabinetProfileResponse(cabinetResolved),
		HeartbeatPolicy: buildHeartbeatPolicyResponse(heartbeatResolved),
		EGMRegistry:     egmRegistryResolved,
	}, nil
}

func buildRuntimeOverridesSnapshotResponse(ctx context.Context, auditStore *store.SQLiteStore, cfg config.Config) (runtimeOverridesSnapshotResponse, error) {
	response := runtimeOverridesSnapshotResponse{
		GeneratedAt:          time.Now().UTC(),
		EGMRegistryOverrides: []egmRegistryOverrideView{},
	}

	cabinetOverride, err := auditStore.GetCabinetProfileOverride(ctx)
	if err != nil {
		return runtimeOverridesSnapshotResponse{}, err
	}
	if cabinetOverride != nil {
		response.CabinetProfileOverride = &cabinetProfileOverrideView{
			Profile:   cabinetOverride.Profile,
			UpdatedAt: cabinetOverride.UpdatedAt,
			UpdatedBy: cabinetOverride.UpdatedBy,
		}
	}

	heartbeatOverride, err := auditStore.GetHeartbeatPolicyOverride(ctx)
	if err != nil {
		return runtimeOverridesSnapshotResponse{}, err
	}
	if heartbeatOverride != nil {
		response.HeartbeatPolicyOverride = &heartbeatPolicyOverrideView{
			IntervalMS:         heartbeatOverride.IntervalMS,
			WarningAfterMissed: heartbeatOverride.WarningAfterMissed,
			BlockAfterMissed:   heartbeatOverride.BlockAfterMissed,
			UpdatedAt:          heartbeatOverride.UpdatedAt,
			UpdatedBy:          heartbeatOverride.UpdatedBy,
		}
	}

	registryOverrides, err := auditStore.ListEGMRegistryOverrides(ctx)
	if err != nil {
		return runtimeOverridesSnapshotResponse{}, err
	}
	response.EGMRegistryOverrides = buildEGMRegistryOverrideViews(registryOverrides)
	return response, nil
}
