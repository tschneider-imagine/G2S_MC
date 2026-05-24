package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/tschneider-imagine/G2S_MC/internal/actionplanner"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type Store interface {
	GetInputChannel(ctx context.Context, id string) (*inputs.InputChannel, error)
	UpsertInputChannel(ctx context.Context, channel inputs.InputChannel) error
	ListInputChannels(ctx context.Context) ([]inputs.InputChannel, error)
	GetInputRuntimeState(ctx context.Context, inputID string) (*inputruntime.InputRuntimeState, error)
	ListInputTransitions(ctx context.Context, limit int) ([]inputs.InputTransition, error)

	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	UpsertActionDefinition(ctx context.Context, definition actions.ActionDefinition) error
	ListActionDefinitions(ctx context.Context) ([]actions.ActionDefinition, error)

	GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error)
	UpsertG2STemplate(ctx context.Context, tpl templates.G2STemplate) error
	ListG2STemplates(ctx context.Context) ([]templates.G2STemplate, error)

	GetEGMRecord(ctx context.Context, egmID string) (*egms.EGMRecord, error)
	UpsertEGMRecord(ctx context.Context, record egms.EGMRecord) error
	ListEGMRecords(ctx context.Context) ([]egms.EGMRecord, error)
	GetEGMGroup(ctx context.Context, id string) (*egms.EGMGroup, error)
	ListEGMGroups(ctx context.Context) ([]egms.EGMGroup, error)

	ListMessageJournalEntries(ctx context.Context, query store.MessageJournalListQuery) ([]g2sengine.MessageJournalEntry, error)
	ListAuditTimelineEntries(ctx context.Context, query store.AuditTimelineListQuery) ([]audit.AuditTimelineEntry, error)
}

type Server struct {
	Store             Store
	AuthorizeMutation func(http.ResponseWriter, *http.Request) bool
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v2/inputs", s.handleInputs)
	mux.HandleFunc("/api/v2/inputs/", s.handleInputByID)
	mux.HandleFunc("/api/v2/inputs/state", s.handleInputRuntimeState)
	mux.HandleFunc("/api/v2/inputs/transitions", s.handleInputTransitions)

	mux.HandleFunc("/api/v2/actions", s.handleActions)
	mux.HandleFunc("/api/v2/actions/", s.handleActionByID)

	mux.HandleFunc("/api/v2/templates", s.handleTemplates)
	mux.HandleFunc("/api/v2/templates/", s.handleTemplateByID)

	mux.HandleFunc("/api/v2/egms", s.handleEGMs)
	mux.HandleFunc("/api/v2/egms/", s.handleEGMByID)

	mux.HandleFunc("/api/v2/comms/messages", s.handleMessages)
	mux.HandleFunc("/api/v2/audit/timeline", s.handleTimeline)
}

func (s *Server) handleInputRuntimeState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	channels, err := s.Store.ListInputChannels(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	result := make([]InputStateEnvelope, 0, len(channels))
	for _, channel := range channels {
		runtimeState, err := s.Store.GetInputRuntimeState(r.Context(), channel.ID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		result = append(result, InputStateEnvelope{
			Channel:      channel,
			RuntimeState: runtimeState,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleInputTransitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	transitions, err := s.Store.ListInputTransitions(r.Context(), queryLimit(r, 50))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, transitions)
}

func (s *Server) handleInputs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	channels, err := s.Store.ListInputChannels(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, channels)
}

func (s *Server) handleInputByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.authorizeMutation(w, r) {
		return
	}
	id, ok := pathID(r.URL.Path, "/api/v2/inputs/")
	if !ok {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	var channel inputs.InputChannel
	if !decodeJSON(w, r, &channel) {
		return
	}
	if strings.TrimSpace(channel.ID) == "" {
		channel.ID = id
	} else if channel.ID != id {
		writeJSONError(w, http.StatusBadRequest, "path id must match body id")
		return
	}
	if err := channel.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Store.UpsertInputChannel(r.Context(), channel); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, channel)
}

func (s *Server) handleActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	definitions, err := s.Store.ListActionDefinitions(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, definitions)
}

func (s *Server) handleActionByID(w http.ResponseWriter, r *http.Request) {
	id, preview, ok := actionRoute(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	if preview {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleActionPreviewByID(w, r, id)
		return
	}
	if r.Method != http.MethodPut {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.authorizeMutation(w, r) {
		return
	}
	var definition actions.ActionDefinition
	if !decodeJSON(w, r, &definition) {
		return
	}
	if strings.TrimSpace(definition.ID) == "" {
		definition.ID = id
	} else if definition.ID != id {
		writeJSONError(w, http.StatusBadRequest, "path id must match body id")
		return
	}
	if err := definition.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Store.UpsertActionDefinition(r.Context(), definition); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, definition)
}

func (s *Server) handleActionPreviewByID(w http.ResponseWriter, r *http.Request, actionID string) {
	definition, err := s.Store.GetActionDefinition(r.Context(), actionID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if definition == nil {
		writeJSONError(w, http.StatusNotFound, "action not found")
		return
	}
	planner := actionplanner.Planner{Store: s.Store}
	plan, err := planner.BuildPlanForDefinition(r.Context(), *definition)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	rows, err := s.Store.ListG2STemplates(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) handleTemplateByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.authorizeMutation(w, r) {
		return
	}
	id, ok := pathID(r.URL.Path, "/api/v2/templates/")
	if !ok {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	var tpl templates.G2STemplate
	if !decodeJSON(w, r, &tpl) {
		return
	}
	if strings.TrimSpace(tpl.ID) == "" {
		tpl.ID = id
	} else if tpl.ID != id {
		writeJSONError(w, http.StatusBadRequest, "path id must match body id")
		return
	}
	if err := tpl.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Store.UpsertG2STemplate(r.Context(), tpl); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tpl)
}

func (s *Server) handleEGMs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	records, err := s.Store.ListEGMRecords(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) handleEGMByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.authorizeMutation(w, r) {
		return
	}
	id, ok := pathID(r.URL.Path, "/api/v2/egms/")
	if !ok {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	var record egms.EGMRecord
	if !decodeJSON(w, r, &record) {
		return
	}
	if strings.TrimSpace(record.EGMID) == "" {
		record.EGMID = id
	} else if record.EGMID != id {
		writeJSONError(w, http.StatusBadRequest, "path id must match body id")
		return
	}
	if err := record.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Store.UpsertEGMRecord(r.Context(), record); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	query := store.MessageJournalListQuery{
		Limit: queryLimit(r, 50),
		EGMID: strings.TrimSpace(r.URL.Query().Get("egm_id")),
	}
	if rawDirection := strings.TrimSpace(r.URL.Query().Get("direction")); rawDirection != "" {
		query.Direction = g2sengine.MessageDirection(strings.ToUpper(rawDirection))
	}
	entries, err := s.Store.ListMessageJournalEntries(r.Context(), query)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	query := store.AuditTimelineListQuery{
		Limit:     queryLimit(r, 50),
		EventType: strings.TrimSpace(r.URL.Query().Get("event_type")),
	}
	if rawSeverity := strings.TrimSpace(r.URL.Query().Get("severity")); rawSeverity != "" {
		query.Severity = audit.AuditSeverity(strings.ToUpper(rawSeverity))
	}
	entries, err := s.Store.ListAuditTimelineEntries(r.Context(), query)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) authorizeMutation(w http.ResponseWriter, r *http.Request) bool {
	if s.AuthorizeMutation == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	tw := &trackingResponseWriter{ResponseWriter: w}
	if s.AuthorizeMutation(tw, r) {
		return true
	}
	if !tw.wroteHeader {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
	}
	return false
}

type trackingResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (t *trackingResponseWriter) WriteHeader(code int) {
	t.wroteHeader = true
	t.ResponseWriter.WriteHeader(code)
}

func (t *trackingResponseWriter) Write(data []byte) (int, error) {
	if !t.wroteHeader {
		t.wroteHeader = true
	}
	return t.ResponseWriter.Write(data)
}

func pathID(path string, prefix string) (string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	id := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

func actionRoute(path string) (id string, preview bool, ok bool) {
	const prefix = "/api/v2/actions/"
	if !strings.HasPrefix(path, prefix) {
		return "", false, false
	}
	trimmed := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if trimmed == "" {
		return "", false, false
	}
	if strings.HasSuffix(trimmed, "/preview") {
		id = strings.TrimSpace(strings.TrimSuffix(trimmed, "/preview"))
		if id == "" || strings.Contains(id, "/") {
			return "", false, false
		}
		return id, true, true
	}
	if strings.Contains(trimmed, "/") {
		return "", false, false
	}
	return trimmed, false, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	if decoder.More() {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func queryLimit(r *http.Request, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value <= 0 {
		return fallback
	}
	return value
}
