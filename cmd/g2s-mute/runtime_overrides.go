package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

const runtimeOverridesHistorySummaryLimit = 5

type runtimeOverridesSnapshotResponse struct {
	GeneratedAt                           time.Time                          `json:"generated_at"`
	CabinetProfileOverride                *cabinetProfileOverrideView        `json:"cabinet_profile_override,omitempty"`
	HeartbeatPolicyOverride               *heartbeatPolicyOverrideView       `json:"heartbeat_policy_override,omitempty"`
	BlockerPolicyOverride                 *blockerPolicyOverrideView         `json:"blocker_policy_override,omitempty"`
	BlockerPolicyEscalationHistorySummary []blockerPolicyEscalationEventView `json:"blocker_policy_escalation_history_summary,omitempty"`
	EGMRegistryOverrides                  []egmRegistryOverrideView          `json:"egm_registry_overrides"`
}

type runtimeOverridesRestoreRequest struct {
	CabinetProfileOverride  *config.CabinetProfile `json:"cabinet_profile_override"`
	HeartbeatPolicyOverride *heartbeatPolicy       `json:"heartbeat_policy_override"`
	BlockerPolicyOverride   *struct {
		ApprovedBlockerIDs []string `json:"approved_blocker_ids"`
	} `json:"blocker_policy_override"`
	EGMRegistryOverrides []egmRegistryOverrideView `json:"egm_registry_overrides"`
}

type runtimeOverridesRestoreResponse struct {
	Snapshot        runtimeOverridesSnapshotResponse `json:"snapshot"`
	CabinetProfile  cabinetProfileResponse           `json:"cabinet_profile"`
	HeartbeatPolicy heartbeatPolicyResponse          `json:"heartbeat_policy"`
	BlockerPolicy   blockerPolicyResponse            `json:"blocker_policy"`
	EGMRegistry     egmRegistryResponse              `json:"egm_registry"`
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

		if payload.CabinetProfileOverride != nil {
			if err := config.ValidateCabinetProfile(*payload.CabinetProfileOverride); err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore rejected", err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if payload.HeartbeatPolicyOverride != nil {
			if payload.HeartbeatPolicyOverride.IntervalMS <= 0 {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore rejected", "heartbeat_policy_override.interval_ms must be greater than zero")
				http.Error(w, "heartbeat_policy_override.interval_ms must be greater than zero", http.StatusBadRequest)
				return
			}
			if payload.HeartbeatPolicyOverride.WarningAfterMissed < 1 {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore rejected", "heartbeat_policy_override.warning_after_missed must be greater than or equal to 1")
				http.Error(w, "heartbeat_policy_override.warning_after_missed must be greater than or equal to 1", http.StatusBadRequest)
				return
			}
			if payload.HeartbeatPolicyOverride.BlockAfterMissed < payload.HeartbeatPolicyOverride.WarningAfterMissed {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore rejected", "heartbeat_policy_override.block_after_missed must be greater than or equal to warning_after_missed")
				http.Error(w, "heartbeat_policy_override.block_after_missed must be greater than or equal to warning_after_missed", http.StatusBadRequest)
				return
			}
		}

		approvedBlockerIDs := []string{}
		if payload.BlockerPolicyOverride != nil {
			normalized, err := normalizeBlockerPolicyIDs(payload.BlockerPolicyOverride.ApprovedBlockerIDs)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore rejected", err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			approvedBlockerIDs = normalized
		}

		registryOverrides := make([]store.EGMRegistryOverride, 0, len(payload.EGMRegistryOverrides))
		for i, row := range payload.EGMRegistryOverrides {
			egmID := strings.TrimSpace(row.EGMID)
			if egmID == "" {
				detail := fmt.Sprintf("egm_registry_overrides[%d].egm_id is required", i)
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore rejected", detail)
				http.Error(w, detail, http.StatusBadRequest)
				return
			}
			if err := validateEGMRegistryTextFields(row.DisplayName, row.Vendor, row.CabinetFamily, row.GameTitle, row.SoftwareVersion, row.Notes); err != nil {
				detail := fmt.Sprintf("egm_registry_overrides[%d] rejected: %s", i, err.Error())
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore rejected", detail)
				http.Error(w, detail, http.StatusBadRequest)
				return
			}
			registryOverrides = append(registryOverrides, store.EGMRegistryOverride{
				EGMID:           egmID,
				DisplayName:     strings.TrimSpace(row.DisplayName),
				Vendor:          strings.TrimSpace(row.Vendor),
				CabinetFamily:   strings.TrimSpace(row.CabinetFamily),
				GameTitle:       strings.TrimSpace(row.GameTitle),
				SoftwareVersion: strings.TrimSpace(row.SoftwareVersion),
				Notes:           strings.TrimSpace(row.Notes),
				UpdatedBy:       updateActorNameFromRequest(r),
			})
		}

		restore := store.RuntimeOverridesReplaceInput{
			EGMRegistryOverrides: registryOverrides,
		}
		if payload.CabinetProfileOverride != nil {
			restore.CabinetProfileOverride = &store.CabinetProfileOverride{
				Profile:   *payload.CabinetProfileOverride,
				UpdatedBy: updateActorNameFromRequest(r),
			}
		}
		if payload.HeartbeatPolicyOverride != nil {
			restore.HeartbeatPolicyOverride = &store.HeartbeatPolicyOverride{
				IntervalMS:         payload.HeartbeatPolicyOverride.IntervalMS,
				WarningAfterMissed: payload.HeartbeatPolicyOverride.WarningAfterMissed,
				BlockAfterMissed:   payload.HeartbeatPolicyOverride.BlockAfterMissed,
				UpdatedBy:          updateActorNameFromRequest(r),
			}
		}
		if payload.BlockerPolicyOverride != nil {
			restore.BlockerPolicyOverride = &store.BlockerPolicyOverride{
				ApprovedBlockerIDs:   approvedBlockerIDs,
				UpdatedBy:            updateActorNameFromRequest(r),
				LastChangeAction:     "restore_snapshot",
				LastChangeRationale:  "runtime override snapshot restore",
				LastChangeActorScope: operatorAuditActorScope(r, cfg),
			}
		}

		if err := auditStore.ReplaceRuntimeOverrides(r.Context(), restore); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore failed", err.Error())
			writeJSON(w, nil, err)
			return
		}

		cabinetResolved, err := resolveCabinetProfile(r.Context(), auditStore, cfg.CabinetProfile)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		heartbeatResolved, err := resolveHeartbeatPolicy(r.Context(), auditStore, cfg.Timeouts)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		blockerResolved, err := resolveBlockerPolicy(r.Context(), auditStore, cfg.BlockerPolicy)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		egmRegistryResolved, err := buildEGMRegistryResponse(r.Context(), eng, auditStore)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "runtime_overrides.restore", "fail", "Runtime override restore failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		snapshot, err := buildRuntimeOverridesSnapshotResponse(r.Context(), auditStore, cfg)
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
				"cabinet_profile_override=%t heartbeat_policy_override=%t blocker_policy_override=%t egm_registry_overrides=%d",
				restore.CabinetProfileOverride != nil,
				restore.HeartbeatPolicyOverride != nil,
				restore.BlockerPolicyOverride != nil,
				len(restore.EGMRegistryOverrides),
			),
		)
		writeJSON(w, runtimeOverridesRestoreResponse{
			Snapshot:        snapshot,
			CabinetProfile:  buildCabinetProfileResponse(cabinetResolved),
			HeartbeatPolicy: buildHeartbeatPolicyResponse(heartbeatResolved),
			BlockerPolicy:   buildBlockerPolicyResponse(blockerResolved),
			EGMRegistry:     egmRegistryResolved,
		}, nil)
	}
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

	blockerResolved, err := resolveBlockerPolicyWithHistoryLimit(ctx, auditStore, cfg.BlockerPolicy, runtimeOverridesHistorySummaryLimit)
	if err != nil {
		return runtimeOverridesSnapshotResponse{}, err
	}
	if blockerResolved.Override != nil {
		response.BlockerPolicyOverride = &blockerPolicyOverrideView{
			ApprovedBlockerIDs:   append([]string{}, blockerResolved.Override.ApprovedBlockerIDs...),
			UpdatedAt:            blockerResolved.Override.UpdatedAt,
			UpdatedBy:            blockerResolved.Override.UpdatedBy,
			LastChangeAction:     blockerResolved.Override.LastChangeAction,
			LastChangeRationale:  blockerResolved.Override.LastChangeRationale,
			LastChangeActorScope: blockerResolved.Override.LastChangeActorScope,
		}
	}
	response.BlockerPolicyEscalationHistorySummary = buildBlockerPolicyEscalationHistoryView(blockerResolved.EscalationHistory)

	registryOverrides, err := auditStore.ListEGMRegistryOverrides(ctx)
	if err != nil {
		return runtimeOverridesSnapshotResponse{}, err
	}
	response.EGMRegistryOverrides = buildEGMRegistryOverrideViews(registryOverrides)
	return response, nil
}
