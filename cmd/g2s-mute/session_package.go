package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

const (
	sessionPackageSchemaVersion  = "session-package.v1"
	sessionPackageOperatorAuditN = 500
)

type sessionPackageExportPayload struct {
	SchemaVersion        string                       `json:"schema_version"`
	GeneratedAt          time.Time                    `json:"generated_at"`
	Status               applianceStatus              `json:"status"`
	CabinetPreflight     cabinetPreflightResponse     `json:"cabinet_preflight"`
	SessionWorkflow      sessionWorkflowResponse      `json:"session_workflow"`
	HeartbeatPolicy      heartbeatPolicyResponse      `json:"heartbeat_policy"`
	OperatorAudit        []model.OperatorAuditEvent   `json:"operator_audit"`
	SessionEvidenceIndex sessionEvidenceArchiveIndex  `json:"session_evidence_index"`
	SavedCapturesMeta    []sessionEvidenceArchiveItem `json:"saved_captures_metadata"`
}

func sessionPackageExportHandler(eng *engine.Engine, store *store.SQLiteStore, cfg config.Config, runtime runtimeInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		statusSnapshot, err := computeApplianceStatus(r.Context(), eng, store, cfg, runtime, r)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}

		cabinetPreflight := evaluateCabinetPreflight(r.Context(), eng, store, cfg, runtime)

		workflowProgress, err := store.GetSessionWorkflowProgress(r.Context())
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		workflowSnapshot := buildSessionWorkflowResponse(workflowProgress)

		heartbeat, err := resolveHeartbeatPolicy(r.Context(), store, cfg.Timeouts)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		heartbeatSnapshot := buildHeartbeatPolicyResponse(heartbeat)

		operatorAudit, err := store.ListOperatorAuditEvents(r.Context(), model.OperatorAuditQuery{Limit: sessionPackageOperatorAuditN})
		if err != nil {
			writeJSON(w, nil, err)
			return
		}

		sessionEvidence, err := store.ListAllSessionEvidence(r.Context())
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		evidenceIndex := buildSessionEvidenceArchiveIndex(sessionEvidence)

		payload := sessionPackageExportPayload{
			SchemaVersion:        sessionPackageSchemaVersion,
			GeneratedAt:          time.Now().UTC(),
			Status:               statusSnapshot,
			CabinetPreflight:     cabinetPreflight,
			SessionWorkflow:      workflowSnapshot,
			HeartbeatPolicy:      heartbeatSnapshot,
			OperatorAudit:        operatorAudit,
			SessionEvidenceIndex: evidenceIndex,
			SavedCapturesMeta:    append([]sessionEvidenceArchiveItem{}, evidenceIndex.Captures...),
		}

		filename := "session-package-" + payload.GeneratedAt.Format("20060102T150405Z") + ".json"
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
