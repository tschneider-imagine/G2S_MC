package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

const (
	blockerPolicySourceFile      = "file"
	blockerPolicySourceOverride  = "override"
	blockerPolicyDowngradeMarker = "DOWNGRADED_BY_BLOCKER_POLICY"

	blockerPolicyHistoryLimit          = 25
	blockerPolicyPreflightHistoryLimit = 5

	blockerPolicyActionApprove      = "approve"
	blockerPolicyActionRevoke       = "revoke"
	blockerPolicyActionSaveOverride = "save_override"
)

var blockerPolicyIDPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

var defaultApprovedBlockerIDs = []string{
	"service_readiness",
	"cabinet_profile",
	"profile_source",
	"certificate_mode_requirements",
	"certificate_san_wire_identity",
}

type blockerPolicyResponse struct {
	Effective           config.BlockerPolicy               `json:"effective"`
	PolicySource        string                             `json:"policy_source"`
	PolicyLastUpdatedAt *time.Time                         `json:"policy_last_updated_at,omitempty"`
	OverridePresent     bool                               `json:"override_present"`
	Override            *blockerPolicyOverrideView         `json:"override,omitempty"`
	EscalationHistory   []blockerPolicyEscalationEventView `json:"escalation_history,omitempty"`
}

type blockerPolicyOverrideView struct {
	ApprovedBlockerIDs   []string  `json:"approved_blocker_ids"`
	UpdatedAt            time.Time `json:"updated_at"`
	UpdatedBy            string    `json:"updated_by,omitempty"`
	LastChangeAction     string    `json:"last_change_action,omitempty"`
	LastChangeRationale  string    `json:"last_change_rationale,omitempty"`
	LastChangeActorScope string    `json:"last_change_actor_scope,omitempty"`
}

type blockerPolicyEscalationEventView struct {
	ID         int64     `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	Action     string    `json:"action"`
	FindingID  string    `json:"finding_id"`
	Rationale  string    `json:"rationale,omitempty"`
	ActorScope string    `json:"actor_scope"`
	EGMFocus   string    `json:"egm_focus,omitempty"`
	UpdatedBy  string    `json:"updated_by,omitempty"`
}

type blockerPolicySuggestionItem struct {
	FindingID          string `json:"finding_id"`
	Message            string `json:"message"`
	DowngradedByPolicy bool   `json:"downgraded_by_policy"`
}

type blockerPolicySuggestionsResponse struct {
	GeneratedAt time.Time                     `json:"generated_at"`
	Policy      blockerPolicyResponse         `json:"policy"`
	Suggestions []blockerPolicySuggestionItem `json:"suggestions"`
}

type blockerPolicyActionRequest struct {
	FindingID string `json:"finding_id"`
	Rationale string `json:"rationale"`
}

type resolvedBlockerPolicy struct {
	Effective           config.BlockerPolicy
	File                config.BlockerPolicy
	Override            *store.BlockerPolicyOverride
	PolicySource        string
	PolicyLastUpdatedAt *time.Time
	EscalationHistory   []store.BlockerPolicyEscalationEvent
}

func blockerPolicyHandler(auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			resolved, err := resolveBlockerPolicy(r.Context(), auditStore, cfg.BlockerPolicy)
			if err != nil {
				writeJSON(w, nil, err)
				return
			}
			writeJSON(w, buildBlockerPolicyResponse(resolved), nil)
		case http.MethodPut:
			var payload struct {
				ApprovedBlockerIDs []string `json:"approved_blocker_ids"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&payload); err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.save", "fail", "Blocker policy save rejected", "invalid JSON body")
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			approved, err := normalizeBlockerPolicyIDs(payload.ApprovedBlockerIDs)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.save", "fail", "Blocker policy save rejected", err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			updatedBy := updateActorNameFromRequest(r)
			if err := auditStore.UpsertBlockerPolicyOverrideWithMeta(
				r.Context(),
				approved,
				updatedBy,
				blockerPolicyActionSaveOverride,
				"",
				operatorAuditActorScope(r, cfg),
			); err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.save", "fail", "Blocker policy save failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			resolved, err := resolveBlockerPolicy(r.Context(), auditStore, cfg.BlockerPolicy)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.save", "fail", "Blocker policy save failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(
				r.Context(),
				auditStore,
				r,
				cfg,
				"blocker_policy.save",
				"success",
				"Blocker policy override saved",
				"approved_blocker_ids="+strings.Join(resolved.Effective.ApprovedBlockerIDs, ","),
			)
			writeJSON(w, buildBlockerPolicyResponse(resolved), nil)
		case http.MethodDelete:
			if err := auditStore.ClearBlockerPolicyOverride(r.Context()); err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.clear", "fail", "Blocker policy override clear failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			resolved, err := resolveBlockerPolicy(r.Context(), auditStore, cfg.BlockerPolicy)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.clear", "fail", "Blocker policy override clear failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.clear", "success", "Blocker policy override cleared", "policy_source="+resolved.PolicySource)
			writeJSON(w, buildBlockerPolicyResponse(resolved), nil)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func blockerPolicySuggestionsHandler(eng *engine.Engine, auditStore *store.SQLiteStore, cfg config.Config, runtime runtimeInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		preflight := evaluateCabinetPreflight(r.Context(), eng, auditStore, cfg, runtime)
		approved := blockerPolicyIDSet(preflight.BlockerPolicy.Effective.ApprovedBlockerIDs)
		response := blockerPolicySuggestionsResponse{
			GeneratedAt: time.Now().UTC(),
			Policy:      preflight.BlockerPolicy,
			Suggestions: blockerSuggestionsFromPreflight(preflight, approved),
		}
		writeJSON(w, response, nil)
	}
}

func blockerPolicyApproveHandler(eng *engine.Engine, auditStore *store.SQLiteStore, cfg config.Config, runtime runtimeInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		request, err := decodeBlockerPolicyActionRequest(r)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.approve", "fail", "Blocker policy approve rejected", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(request.Rationale) == "" {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.approve", "fail", "Blocker policy approve rejected", "rationale is required")
			http.Error(w, "rationale is required", http.StatusBadRequest)
			return
		}

		preflight := evaluateCabinetPreflight(r.Context(), eng, auditStore, cfg, runtime)
		approvedSet := blockerPolicyIDSet(preflight.BlockerPolicy.Effective.ApprovedBlockerIDs)
		if _, exists := approvedSet[request.FindingID]; exists {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.approve", "fail", "Blocker policy approve rejected", "finding_id already approved")
			http.Error(w, "finding_id already approved", http.StatusConflict)
			return
		}
		suggestions := blockerSuggestionsFromPreflight(preflight, approvedSet)
		if !blockerSuggestionIncludesID(suggestions, request.FindingID) {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.approve", "fail", "Blocker policy approve rejected", "finding_id is not a current unapproved FAIL preflight finding")
			http.Error(w, "finding_id is not a current unapproved FAIL preflight finding", http.StatusBadRequest)
			return
		}

		newApproved := append([]string{}, preflight.BlockerPolicy.Effective.ApprovedBlockerIDs...)
		newApproved = append(newApproved, request.FindingID)
		normalizedApproved, err := normalizeBlockerPolicyIDs(newApproved)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.approve", "fail", "Blocker policy approve failed", err.Error())
			writeJSON(w, nil, err)
			return
		}

		updatedBy := updateActorNameFromRequest(r)
		actorScope := operatorAuditActorScope(r, cfg)
		if err := auditStore.UpsertBlockerPolicyOverrideWithMeta(
			r.Context(),
			normalizedApproved,
			updatedBy,
			blockerPolicyActionApprove,
			request.Rationale,
			actorScope,
		); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.approve", "fail", "Blocker policy approve failed", err.Error())
			writeJSON(w, nil, err)
			return
		}

		_, err = auditStore.RecordBlockerPolicyEscalationEvent(r.Context(), store.BlockerPolicyEscalationEvent{
			CreatedAt:  time.Now().UTC(),
			Action:     blockerPolicyActionApprove,
			FindingID:  request.FindingID,
			Rationale:  strings.TrimSpace(request.Rationale),
			ActorScope: actorScope,
			EGMFocus:   operatorAuditEGMFocus(r),
			UpdatedBy:  updatedBy,
		})
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.approve", "fail", "Blocker policy approve failed", err.Error())
			writeJSON(w, nil, err)
			return
		}

		resolved, err := resolveBlockerPolicy(r.Context(), auditStore, cfg.BlockerPolicy)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.approve", "fail", "Blocker policy approve failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		recordOperatorAuditEvent(
			r.Context(),
			auditStore,
			r,
			cfg,
			"blocker_policy.approve",
			"success",
			"Blocker escalation approved",
			"finding_id="+request.FindingID+" rationale="+strings.TrimSpace(request.Rationale),
		)
		writeJSON(w, buildBlockerPolicyResponse(resolved), nil)
	}
}

func blockerPolicyRevokeHandler(auditStore *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		request, err := decodeBlockerPolicyActionRequest(r)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.revoke", "fail", "Blocker policy revoke rejected", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resolved, err := resolveBlockerPolicy(r.Context(), auditStore, cfg.BlockerPolicy)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.revoke", "fail", "Blocker policy revoke failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		approvedSet := blockerPolicyIDSet(resolved.Effective.ApprovedBlockerIDs)
		if _, exists := approvedSet[request.FindingID]; !exists {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.revoke", "fail", "Blocker policy revoke rejected", "finding_id is not currently approved")
			http.Error(w, "finding_id is not currently approved", http.StatusBadRequest)
			return
		}
		newApproved := []string{}
		for _, id := range resolved.Effective.ApprovedBlockerIDs {
			if id == request.FindingID {
				continue
			}
			newApproved = append(newApproved, id)
		}
		normalizedApproved, err := normalizeBlockerPolicyIDs(newApproved)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.revoke", "fail", "Blocker policy revoke failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		updatedBy := updateActorNameFromRequest(r)
		actorScope := operatorAuditActorScope(r, cfg)
		if err := auditStore.UpsertBlockerPolicyOverrideWithMeta(
			r.Context(),
			normalizedApproved,
			updatedBy,
			blockerPolicyActionRevoke,
			strings.TrimSpace(request.Rationale),
			actorScope,
		); err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.revoke", "fail", "Blocker policy revoke failed", err.Error())
			writeJSON(w, nil, err)
			return
		}

		_, err = auditStore.RecordBlockerPolicyEscalationEvent(r.Context(), store.BlockerPolicyEscalationEvent{
			CreatedAt:  time.Now().UTC(),
			Action:     blockerPolicyActionRevoke,
			FindingID:  request.FindingID,
			Rationale:  strings.TrimSpace(request.Rationale),
			ActorScope: actorScope,
			EGMFocus:   operatorAuditEGMFocus(r),
			UpdatedBy:  updatedBy,
		})
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.revoke", "fail", "Blocker policy revoke failed", err.Error())
			writeJSON(w, nil, err)
			return
		}

		resolved, err = resolveBlockerPolicy(r.Context(), auditStore, cfg.BlockerPolicy)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), auditStore, r, cfg, "blocker_policy.revoke", "fail", "Blocker policy revoke failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		recordOperatorAuditEvent(
			r.Context(),
			auditStore,
			r,
			cfg,
			"blocker_policy.revoke",
			"success",
			"Blocker escalation revoked",
			"finding_id="+request.FindingID+(func() string {
				rationale := strings.TrimSpace(request.Rationale)
				if rationale == "" {
					return ""
				}
				return " rationale=" + rationale
			}()),
		)
		writeJSON(w, buildBlockerPolicyResponse(resolved), nil)
	}
}

func resolveBlockerPolicy(ctx context.Context, auditStore *store.SQLiteStore, filePolicy config.BlockerPolicy) (resolvedBlockerPolicy, error) {
	return resolveBlockerPolicyWithHistoryLimit(ctx, auditStore, filePolicy, blockerPolicyHistoryLimit)
}

func resolveBlockerPolicyWithHistoryLimit(ctx context.Context, auditStore *store.SQLiteStore, filePolicy config.BlockerPolicy, historyLimit int) (resolvedBlockerPolicy, error) {
	fileApproved, err := normalizeBlockerPolicyIDs(filePolicy.ApprovedBlockerIDs)
	if err != nil {
		return resolvedBlockerPolicy{}, err
	}
	if len(fileApproved) == 0 {
		fileApproved = append([]string{}, defaultApprovedBlockerIDs...)
		sort.Strings(fileApproved)
	}
	resolved := resolvedBlockerPolicy{
		Effective: config.BlockerPolicy{
			ApprovedBlockerIDs: append([]string{}, fileApproved...),
		},
		File: config.BlockerPolicy{
			ApprovedBlockerIDs: append([]string{}, fileApproved...),
		},
		PolicySource: blockerPolicySourceFile,
	}

	override, err := auditStore.GetBlockerPolicyOverride(ctx)
	if err != nil {
		return resolved, err
	}
	if override != nil {
		approved, err := normalizeBlockerPolicyIDs(override.ApprovedBlockerIDs)
		if err != nil {
			return resolvedBlockerPolicy{}, err
		}
		resolved.Override = &store.BlockerPolicyOverride{
			ApprovedBlockerIDs:   approved,
			UpdatedAt:            override.UpdatedAt,
			UpdatedBy:            override.UpdatedBy,
			LastChangeAction:     override.LastChangeAction,
			LastChangeRationale:  override.LastChangeRationale,
			LastChangeActorScope: override.LastChangeActorScope,
		}
		resolved.Effective.ApprovedBlockerIDs = append([]string{}, approved...)
		resolved.PolicySource = blockerPolicySourceOverride
		updatedAt := override.UpdatedAt
		resolved.PolicyLastUpdatedAt = &updatedAt
	}

	history, err := auditStore.ListBlockerPolicyEscalationEvents(ctx, historyLimit)
	if err != nil {
		return resolvedBlockerPolicy{}, err
	}
	resolved.EscalationHistory = history
	return resolved, nil
}

func buildBlockerPolicyResponse(policy resolvedBlockerPolicy) blockerPolicyResponse {
	response := blockerPolicyResponse{
		Effective: config.BlockerPolicy{
			ApprovedBlockerIDs: append([]string{}, policy.Effective.ApprovedBlockerIDs...),
		},
		PolicySource:        policy.PolicySource,
		PolicyLastUpdatedAt: policy.PolicyLastUpdatedAt,
		OverridePresent:     policy.Override != nil,
		EscalationHistory:   buildBlockerPolicyEscalationHistoryView(policy.EscalationHistory),
	}
	if policy.Override != nil {
		response.Override = &blockerPolicyOverrideView{
			ApprovedBlockerIDs:   append([]string{}, policy.Override.ApprovedBlockerIDs...),
			UpdatedAt:            policy.Override.UpdatedAt,
			UpdatedBy:            policy.Override.UpdatedBy,
			LastChangeAction:     policy.Override.LastChangeAction,
			LastChangeRationale:  policy.Override.LastChangeRationale,
			LastChangeActorScope: policy.Override.LastChangeActorScope,
		}
	}
	return response
}

func buildBlockerPolicyEscalationHistoryView(history []store.BlockerPolicyEscalationEvent) []blockerPolicyEscalationEventView {
	rows := []blockerPolicyEscalationEventView{}
	for _, item := range history {
		rows = append(rows, blockerPolicyEscalationEventView{
			ID:         item.ID,
			CreatedAt:  item.CreatedAt,
			Action:     item.Action,
			FindingID:  item.FindingID,
			Rationale:  item.Rationale,
			ActorScope: item.ActorScope,
			EGMFocus:   item.EGMFocus,
			UpdatedBy:  item.UpdatedBy,
		})
	}
	return rows
}

func normalizeBlockerPolicyIDs(raw []string) ([]string, error) {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(raw))
	for _, id := range raw {
		value := strings.TrimSpace(id)
		if value == "" {
			return nil, fmt.Errorf("approved_blocker_ids entries must be non-empty")
		}
		if !blockerPolicyIDPattern.MatchString(value) {
			return nil, fmt.Errorf("approved_blocker_ids entry %q must match ^[a-z0-9_]+$", value)
		}
		if _, ok := seen[value]; ok {
			return nil, fmt.Errorf("approved_blocker_ids entry %q is duplicated", value)
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func blockerPolicyIDSet(ids []string) map[string]struct{} {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		value := strings.TrimSpace(id)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return set
}

func blockerSuggestionsFromPreflight(preflight cabinetPreflightResponse, approved map[string]struct{}) []blockerPolicySuggestionItem {
	downgradedSet := map[string]struct{}{}
	for _, item := range preflight.DowngradedFindings {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		downgradedSet[id] = struct{}{}
	}
	rows := []blockerPolicySuggestionItem{}
	for _, check := range preflight.Checks {
		if check.Result != preflightFail {
			continue
		}
		id := strings.TrimSpace(check.ID)
		if id == "" {
			continue
		}
		if _, exists := approved[id]; exists {
			continue
		}
		_, downgraded := downgradedSet[id]
		rows = append(rows, blockerPolicySuggestionItem{
			FindingID:          id,
			Message:            strings.TrimSpace(check.Message),
			DowngradedByPolicy: downgraded,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].FindingID < rows[j].FindingID
	})
	return rows
}

func blockerSuggestionIncludesID(suggestions []blockerPolicySuggestionItem, findingID string) bool {
	for _, item := range suggestions {
		if item.FindingID == findingID {
			return true
		}
	}
	return false
}

func decodeBlockerPolicyActionRequest(r *http.Request) (blockerPolicyActionRequest, error) {
	payload := blockerPolicyActionRequest{}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return blockerPolicyActionRequest{}, fmt.Errorf("invalid JSON body")
	}
	payload.FindingID = strings.TrimSpace(payload.FindingID)
	payload.Rationale = strings.TrimSpace(payload.Rationale)
	if payload.FindingID == "" {
		return blockerPolicyActionRequest{}, fmt.Errorf("finding_id is required")
	}
	if !blockerPolicyIDPattern.MatchString(payload.FindingID) {
		return blockerPolicyActionRequest{}, fmt.Errorf("finding_id must match ^[a-z0-9_]+$")
	}
	return payload, nil
}

func updateActorNameFromRequest(r *http.Request) string {
	updatedBy := strings.TrimSpace(r.Header.Get("X-Operator"))
	if updatedBy == "" {
		updatedBy = "lab-api"
	}
	return updatedBy
}
