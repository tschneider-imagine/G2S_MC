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
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

const (
	blockerPolicySourceFile      = "file"
	blockerPolicySourceOverride  = "override"
	blockerPolicyDowngradeMarker = "DOWNGRADED_BY_BLOCKER_POLICY"
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
	Effective           config.BlockerPolicy       `json:"effective"`
	PolicySource        string                     `json:"policy_source"`
	PolicyLastUpdatedAt *time.Time                 `json:"policy_last_updated_at,omitempty"`
	OverridePresent     bool                       `json:"override_present"`
	Override            *blockerPolicyOverrideView `json:"override,omitempty"`
}

type blockerPolicyOverrideView struct {
	ApprovedBlockerIDs []string  `json:"approved_blocker_ids"`
	UpdatedAt          time.Time `json:"updated_at"`
	UpdatedBy          string    `json:"updated_by,omitempty"`
}

type resolvedBlockerPolicy struct {
	Effective           config.BlockerPolicy
	File                config.BlockerPolicy
	Override            *store.BlockerPolicyOverride
	PolicySource        string
	PolicyLastUpdatedAt *time.Time
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
			updatedBy := strings.TrimSpace(r.Header.Get("X-Operator"))
			if updatedBy == "" {
				updatedBy = "lab-api"
			}
			if err := auditStore.UpsertBlockerPolicyOverride(r.Context(), approved, updatedBy); err != nil {
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

func resolveBlockerPolicy(ctx context.Context, auditStore *store.SQLiteStore, filePolicy config.BlockerPolicy) (resolvedBlockerPolicy, error) {
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
	if override == nil {
		return resolved, nil
	}
	approved, err := normalizeBlockerPolicyIDs(override.ApprovedBlockerIDs)
	if err != nil {
		return resolvedBlockerPolicy{}, err
	}
	resolved.Override = &store.BlockerPolicyOverride{
		ApprovedBlockerIDs: approved,
		UpdatedAt:          override.UpdatedAt,
		UpdatedBy:          override.UpdatedBy,
	}
	resolved.Effective.ApprovedBlockerIDs = append([]string{}, approved...)
	resolved.PolicySource = blockerPolicySourceOverride
	updatedAt := override.UpdatedAt
	resolved.PolicyLastUpdatedAt = &updatedAt
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
	}
	if policy.Override != nil {
		response.Override = &blockerPolicyOverrideView{
			ApprovedBlockerIDs: append([]string{}, policy.Override.ApprovedBlockerIDs...),
			UpdatedAt:          policy.Override.UpdatedAt,
			UpdatedBy:          policy.Override.UpdatedBy,
		}
	}
	return response
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
