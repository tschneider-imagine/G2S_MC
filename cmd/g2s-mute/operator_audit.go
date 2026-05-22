package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func operatorAuditHandler(store *store.SQLiteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		resultFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("result")))
		if resultFilter != "" && resultFilter != "success" && resultFilter != "fail" {
			http.Error(w, "result must be success or fail", http.StatusBadRequest)
			return
		}

		query := model.OperatorAuditQuery{
			Limit:  queryLimit(r, 50),
			Action: strings.TrimSpace(r.URL.Query().Get("action")),
			Result: resultFilter,
			Search: strings.TrimSpace(r.URL.Query().Get("q")),
		}
		events, err := store.ListOperatorAuditEvents(r.Context(), query)
		writeJSON(w, events, err)
	}
}

func operatorAuditActorScope(r *http.Request, cfg config.Config) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		return "token"
	}
	if isLoopbackRequest(r) {
		return "local"
	}
	if trustedPrivateNetworkMutationBypassAllowed(r, cfg) {
		return "trusted"
	}
	if cfg.WebUI.RequireLogin {
		return "authenticated"
	}
	return "authenticated"
}

func operatorAuditEGMFocus(r *http.Request) string {
	if r == nil {
		return ""
	}
	if focus := strings.TrimSpace(r.Header.Get("X-EGM-Focus")); focus != "" {
		return focus
	}
	if focus := strings.TrimSpace(r.URL.Query().Get("egm_focus")); focus != "" {
		return focus
	}
	return ""
}

func recordOperatorAuditEvent(ctx context.Context, store *store.SQLiteStore, r *http.Request, cfg config.Config, action string, result string, summary string, detail string) {
	if store == nil || strings.TrimSpace(action) == "" {
		return
	}
	normalizedResult := strings.ToLower(strings.TrimSpace(result))
	if normalizedResult != "success" {
		normalizedResult = "fail"
	}
	event := model.OperatorAuditEvent{
		Timestamp:  time.Now().UTC(),
		Action:     strings.TrimSpace(action),
		Result:     normalizedResult,
		ActorScope: operatorAuditActorScope(r, cfg),
		EGMFocus:   operatorAuditEGMFocus(r),
		Summary:    strings.TrimSpace(summary),
		Detail:     strings.TrimSpace(detail),
	}
	if _, err := store.RecordOperatorAuditEvent(ctx, event); err != nil {
		log.Printf("operator audit record failed action=%s result=%s err=%v", event.Action, event.Result, err)
	}
}
