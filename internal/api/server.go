package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actiondispatch"
	"github.com/tschneider-imagine/G2S_MC/internal/actionexecutor"
	"github.com/tschneider-imagine/G2S_MC/internal/actionplanner"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/deliverycheck"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/pendingdelivery"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type Store interface {
	GetInputChannel(ctx context.Context, id string) (*inputs.InputChannel, error)
	UpsertInputChannel(ctx context.Context, channel inputs.InputChannel) error
	ListInputChannels(ctx context.Context) ([]inputs.InputChannel, error)
	GetInputRuntimeState(ctx context.Context, inputID string) (*inputruntime.InputRuntimeState, error)
	UpsertInputRuntimeState(ctx context.Context, state inputruntime.InputRuntimeState) error
	RecordInputTransition(ctx context.Context, transition inputs.InputTransition) (int64, error)
	ListInputTransitions(ctx context.Context, limit int) ([]inputs.InputTransition, error)

	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	UpsertActionDefinition(ctx context.Context, definition actions.ActionDefinition) error
	ListActionDefinitions(ctx context.Context) ([]actions.ActionDefinition, error)
	CreateActionRun(ctx context.Context, run actions.ActionRun) (actions.ActionRun, error)
	GetActionRun(ctx context.Context, id string) (*actions.ActionRun, error)
	ListActionRuns(ctx context.Context, query store.ActionRunListQuery) ([]actions.ActionRun, error)
	UpdateActionRun(ctx context.Context, run actions.ActionRun) error
	CreateActionTargetResult(ctx context.Context, result actions.ActionTargetResult) (actions.ActionTargetResult, error)
	ListActionTargetResults(ctx context.Context, actionRunID string) ([]actions.ActionTargetResult, error)
	UpdateActionTargetResult(ctx context.Context, row actions.ActionTargetResult) error

	GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error)
	GetG2STemplateVersion(ctx context.Context, templateID string, version int) (*templates.G2STemplateVersion, error)
	GetActiveG2STemplateVersion(ctx context.Context, templateID string) (*templates.G2STemplateVersion, error)
	UpsertG2STemplate(ctx context.Context, tpl templates.G2STemplate) error
	ListG2STemplates(ctx context.Context) ([]templates.G2STemplate, error)

	GetEGMRecord(ctx context.Context, egmID string) (*egms.EGMRecord, error)
	UpsertEGMRecord(ctx context.Context, record egms.EGMRecord) error
	ListEGMRecords(ctx context.Context) ([]egms.EGMRecord, error)
	GetEGMGroup(ctx context.Context, id string) (*egms.EGMGroup, error)
	ListEGMGroups(ctx context.Context) ([]egms.EGMGroup, error)

	ListMessageJournalEntries(ctx context.Context, query store.MessageJournalListQuery) ([]g2sengine.MessageJournalEntry, error)
	GetMessageJournalEntry(ctx context.Context, id int64) (*g2sengine.MessageJournalEntry, error)
	RecordMessageJournalEntry(ctx context.Context, entry g2sengine.MessageJournalEntry) (int64, error)
	UpdateMessageJournalResult(ctx context.Context, id int64, result g2sengine.MessageResult, errText string, responseExcerpt string, httpStatusCode int, latencyMS int, transportMode string, sentAt *time.Time, completedAt *time.Time) error
	UpdateMessageJournalOffer(ctx context.Context, id int64, offeredAt time.Time, result g2sengine.MessageResult) (bool, error)
	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)
	ListAuditTimelineEntries(ctx context.Context, query store.AuditTimelineListQuery) ([]audit.AuditTimelineEntry, error)
	ListCertificateInventory(ctx context.Context) ([]model.CertificateInventory, error)
}

type RuntimeInfo struct {
	Version       string
	Revision      string
	RevisionShort string
	Modified      bool
	BuildTime     string
	GoVersion     string
	StartedAt     time.Time
	ConfigPath    string
	DatabasePath  string
	BindAddress   string
}

type Server struct {
	Store                   Store
	AuthorizeMutation       func(http.ResponseWriter, *http.Request) bool
	ActionSender            g2stransport.Sender
	DefaultDeliverySettings g2stransport.DeliverySettings
	EndpointDefaults        g2stransport.EndpointDefaults
	DeliveryClientConfig    g2stransport.HTTPClientConfig
	DeliveryMode            string
	DeliveryTopology        string
	DeliveryCaptureEndpoint string
	G2SHostURL              string
	G2SHostID               string
	RuntimeInfo             RuntimeInfo
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v2/inputs", s.handleInputs)
	mux.HandleFunc("/api/v2/inputs/", s.handleInputByID)
	mux.HandleFunc("/api/v2/inputs/state", s.handleInputRuntimeState)
	mux.HandleFunc("/api/v2/inputs/transitions", s.handleInputTransitions)

	mux.HandleFunc("/api/v2/actions/runs", s.handleActionRuns)
	mux.HandleFunc("/api/v2/actions/runs/", s.handleActionRunByID)
	mux.HandleFunc("/api/v2/actions", s.handleActions)
	mux.HandleFunc("/api/v2/actions/", s.handleActionByID)

	mux.HandleFunc("/api/v2/templates", s.handleTemplates)
	mux.HandleFunc("/api/v2/templates/render-preview", s.handleTemplateRenderPreview)
	mux.HandleFunc("/api/v2/templates/", s.handleTemplateByID)

	mux.HandleFunc("/api/v2/egms", s.handleEGMs)
	mux.HandleFunc("/api/v2/egms/", s.handleEGMByID)

	mux.HandleFunc("/api/v2/comms/messages", s.handleMessages)
	mux.HandleFunc("/api/v2/audit/timeline", s.handleTimeline)
	mux.HandleFunc("/api/v2/runtime", s.handleRuntime)
	mux.HandleFunc("/api/v2/settings/message-delivery-check", s.handleMessageDeliveryCheck)
	mux.HandleFunc("/api/v2/pending-delivery", s.handlePendingDelivery)
	mux.HandleFunc("/api/v2/pending-delivery/sweep", s.handlePendingDeliverySweep)
}

func (s *Server) handleActionRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	query := store.ActionRunListQuery{
		Limit:              queryLimit(r, 50),
		ActionDefinitionID: strings.TrimSpace(r.URL.Query().Get("action_definition_id")),
	}
	if rawStatus := strings.TrimSpace(r.URL.Query().Get("status")); rawStatus != "" {
		query.Status = actions.ActionRunStatus(strings.ToUpper(rawStatus))
	}
	if rawTransition := strings.TrimSpace(r.URL.Query().Get("input_transition_id")); rawTransition != "" {
		value, err := strconv.ParseInt(rawTransition, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid input_transition_id")
			return
		}
		query.InputTransitionID = value
	}

	rows, err := s.Store.ListActionRuns(r.Context(), query)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) handleActionRunByID(w http.ResponseWriter, r *http.Request) {
	id, action, ok := actionRunRoute(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	switch action {
	case "targets":
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		rows, err := s.Store.ListActionTargetResults(r.Context(), id)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rows)
		return
	case "messages":
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		rows, err := s.Store.ListMessageJournalEntries(r.Context(), store.MessageJournalListQuery{
			Limit:       queryLimit(r, 200),
			ActionRunID: id,
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rows)
		return
	case "dispatch-dry-run":
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if !s.authorizeMutation(w, r) {
			return
		}
		var req ActionRunDispatchDryRunRequest
		if r.ContentLength > 0 {
			if !decodeJSON(w, r, &req) {
				return
			}
		}
		dispatcher := actiondispatch.Dispatcher{Store: s.Store}
		result, err := dispatcher.Dispatch(r.Context(), actiondispatch.DispatchRequest{
			ActionRunID: id,
			Mode:        actiondispatch.DispatchModeDryRun,
			Actor:       strings.TrimSpace(req.Actor),
		})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				writeJSONError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	case "send-prepared":
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if !s.authorizeMutation(w, r) {
			return
		}
		var req ActionRunSendPreparedRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		dispatcher := actiondispatch.Dispatcher{Store: s.Store}
		result, err := dispatcher.SendPreparedMessages(r.Context(), actiondispatch.SendPreparedMessagesRequest{
			ActionRunID:     id,
			TransportMode:   g2stransport.Mode(strings.ToUpper(strings.TrimSpace(req.TransportMode))),
			AllowRealSend:   req.AllowRealSend,
			CaptureOnlySend: req.CaptureOnlySend,
			CaptureEndpoint: strings.TrimSpace(req.CaptureEndpoint),
			Actor:           strings.TrimSpace(req.Actor),
		})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				writeJSONError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	case "execute":
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if !s.authorizeMutation(w, r) {
			return
		}
		var req ActionRunExecuteRequest
		hasBody := r.ContentLength > 0
		if r.ContentLength > 0 {
			if !decodeJSON(w, r, &req) {
				return
			}
		}
		delivery := s.DefaultDeliverySettings.Normalize()
		if mode := strings.TrimSpace(req.DeliveryMode); mode != "" {
			delivery.Mode = g2stransport.DeliveryMode(strings.ToUpper(mode))
		}
		if hasBody {
			delivery.AllowDelivery = req.AllowDelivery
			delivery.CaptureOnly = req.CaptureOnly
			if req.TimeoutMS > 0 {
				delivery.TimeoutMS = req.TimeoutMS
			}
		}
		delivery = delivery.Normalize()
		topology := strings.TrimSpace(s.DeliveryTopology)
		if override := strings.TrimSpace(req.DeliveryTopology); override != "" {
			topology = override
		}

		executor := actionexecutor.Executor{
			Store:            s.Store,
			Sender:           s.ActionSender,
			EndpointDefaults: s.EndpointDefaults,
		}
		result, err := executor.Execute(r.Context(), actionexecutor.ExecuteRequest{
			ActionRunID: id,
			Actor:       strings.TrimSpace(req.Actor),
			MaxTargets:  req.MaxTargets,
			Delivery:    delivery,
			Topology:    topology,
		})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				writeJSONError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	case "run":
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		row, err := s.Store.GetActionRun(r.Context(), id)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if row == nil {
			writeJSONError(w, http.StatusNotFound, "action run not found")
			return
		}
		writeJSON(w, http.StatusOK, row)
		return
	default:
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
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
	id, action, ok := inputRoute(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	switch action {
	case "clear-latch":
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if !s.authorizeMutation(w, r) {
			return
		}
		var req InputClearLatchRequest
		if r.ContentLength > 0 {
			if !decodeJSON(w, r, &req) {
				return
			}
		}
		evaluator := inputruntime.Evaluator{Store: s.Store}
		result, err := evaluator.ClearLatchedInput(r.Context(), id, strings.TrimSpace(req.Actor), strings.TrimSpace(req.Reason))
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	case "input":
		if r.Method != http.MethodPut {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if !s.authorizeMutation(w, r) {
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
		return
	default:
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
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

func (s *Server) handleTemplateRenderPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req TemplateRenderPreviewRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	templateID := strings.TrimSpace(req.TemplateID)
	if templateID == "" && strings.TrimSpace(req.EGMID) != "" {
		egmRow, err := s.Store.GetEGMRecord(r.Context(), strings.TrimSpace(req.EGMID))
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if egmRow != nil {
			templateID = strings.TrimSpace(egmRow.TemplateID)
			if strings.TrimSpace(req.IPAddress) == "" {
				req.IPAddress = egmRow.IPAddress
			}
			if strings.TrimSpace(req.EndpointPath) == "" {
				req.EndpointPath = egmRow.EndpointPath
			}
		}
	}
	if templateID == "" {
		writeJSONError(w, http.StatusBadRequest, "template_id is required")
		return
	}
	if strings.TrimSpace(req.TemplateActionKey) == "" {
		writeJSONError(w, http.StatusBadRequest, "template_action_key is required")
		return
	}

	templateRow, err := s.Store.GetG2STemplate(r.Context(), templateID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if templateRow == nil {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("template %q not found", templateID))
		return
	}

	var versionRow *templates.G2STemplateVersion
	if req.Version > 0 {
		versionRow, err = s.Store.GetG2STemplateVersion(r.Context(), templateID, req.Version)
	} else {
		versionRow, err = s.Store.GetActiveG2STemplateVersion(r.Context(), templateID)
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if versionRow == nil {
		if req.Version > 0 {
			writeJSONError(w, http.StatusNotFound, fmt.Sprintf("template %q version %d not found", templateID, req.Version))
		} else {
			writeJSONError(w, http.StatusNotFound, fmt.Sprintf("template %q has no active version", templateID))
		}
		return
	}

	doc, err := g2sengine.ParseActionTemplateDocument(versionRow.ActionsJSON)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	rendered, err := g2sengine.RenderActionMessage(doc, g2sengine.RenderRequest{
		TemplateID:        templateID,
		TemplateVersion:   parseTemplateVersionInt(versionRow.VersionLabel),
		TemplateActionKey: strings.TrimSpace(req.TemplateActionKey),
		ActionID:          strings.TrimSpace(req.ActionID),
		ActionRunID:       strings.TrimSpace(req.ActionRunID),
		ActionStepID:      strings.TrimSpace(req.ActionStepID),
		EGMID:             strings.TrimSpace(req.EGMID),
		HostID:            strings.TrimSpace(req.HostID),
		Timestamp:         time.Now().UTC(),
		IPAddress:         strings.TrimSpace(req.IPAddress),
		EndpointPath:      strings.TrimSpace(req.EndpointPath),
		Variables:         req.Variables,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, TemplateRenderPreviewResponse{
		Rendered: rendered,
		Warnings: rendered.Warnings,
	})
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

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, RuntimeInfoResponse{
		Version:       strings.TrimSpace(s.RuntimeInfo.Version),
		Revision:      strings.TrimSpace(s.RuntimeInfo.Revision),
		RevisionShort: strings.TrimSpace(s.RuntimeInfo.RevisionShort),
		Modified:      s.RuntimeInfo.Modified,
		BuildTime:     strings.TrimSpace(s.RuntimeInfo.BuildTime),
		GoVersion:     strings.TrimSpace(s.RuntimeInfo.GoVersion),
		StartedAt:     s.RuntimeInfo.StartedAt,
		ConfigPath:    strings.TrimSpace(s.RuntimeInfo.ConfigPath),
		DatabasePath:  strings.TrimSpace(s.RuntimeInfo.DatabasePath),
		BindAddress:   strings.TrimSpace(s.RuntimeInfo.BindAddress),
	})
}

func (s *Server) handleMessageDeliveryCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req MessageDeliveryCheckRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.IncludeNetworkCheck || req.IncludeTLSCheck {
		if !s.authorizeMutation(w, r) {
			return
		}
	}
	service := deliverycheck.Service{
		Store: s.Store,
		Options: deliverycheck.Options{
			EndpointDefaults: s.EndpointDefaults,
			ClientConfig:     s.DeliveryClientConfig,
			DeliveryMode:     s.DeliveryMode,
			DeliveryTopology: s.DeliveryTopology,
			CaptureEndpoint:  s.DeliveryCaptureEndpoint,
			ListenerURL:      s.G2SHostURL,
			HostID:           s.G2SHostID,
			DefaultTimeoutMS: s.DefaultDeliverySettings.Normalize().TimeoutMS,
		},
	}
	result, err := service.Check(r.Context(), deliverycheck.CheckRequest{
		EGMID:               strings.TrimSpace(req.EGMID),
		ActionID:            strings.TrimSpace(req.ActionID),
		TemplateID:          strings.TrimSpace(req.TemplateID),
		TemplateActionKey:   strings.TrimSpace(req.TemplateActionKey),
		IncludeNetworkCheck: req.IncludeNetworkCheck,
		IncludeTLSCheck:     req.IncludeTLSCheck,
		TimeoutMS:           req.TimeoutMS,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handlePendingDelivery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	rows, err := s.Store.ListMessageJournalEntries(r.Context(), store.MessageJournalListQuery{
		Limit: queryLimit(r, 200),
		EGMID: strings.TrimSpace(r.URL.Query().Get("egm_id")),
		Results: []g2sengine.MessageResult{
			g2sengine.MessageResultPrepared,
			g2sengine.MessageResultPending,
			g2sengine.MessageResultOffered,
			g2sengine.MessageResultDelivered,
		},
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) handlePendingDeliverySweep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !s.authorizeMutation(w, r) {
		return
	}
	service := pendingdelivery.Service{
		Store: s.Store,
		Clock: time.Now,
	}
	result, err := service.SweepWaitingConfirmations(r.Context(), time.Now().UTC())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
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

func inputRoute(path string) (id string, action string, ok bool) {
	const prefix = "/api/v2/inputs/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	trimmed := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if trimmed == "" {
		return "", "", false
	}
	if strings.HasSuffix(trimmed, "/clear-latch") {
		id = strings.TrimSpace(strings.TrimSuffix(trimmed, "/clear-latch"))
		if id == "" || strings.Contains(id, "/") {
			return "", "", false
		}
		return id, "clear-latch", true
	}
	if strings.Contains(trimmed, "/") {
		return "", "", false
	}
	return trimmed, "input", true
}

func actionRunRoute(path string) (id string, action string, ok bool) {
	const prefix = "/api/v2/actions/runs/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	trimmed := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if trimmed == "" {
		return "", "", false
	}
	if strings.HasSuffix(trimmed, "/targets") {
		id = strings.TrimSpace(strings.TrimSuffix(trimmed, "/targets"))
		if id == "" || strings.Contains(id, "/") {
			return "", "", false
		}
		return id, "targets", true
	}
	if strings.HasSuffix(trimmed, "/messages") {
		id = strings.TrimSpace(strings.TrimSuffix(trimmed, "/messages"))
		if id == "" || strings.Contains(id, "/") {
			return "", "", false
		}
		return id, "messages", true
	}
	if strings.HasSuffix(trimmed, "/dispatch-dry-run") {
		id = strings.TrimSpace(strings.TrimSuffix(trimmed, "/dispatch-dry-run"))
		if id == "" || strings.Contains(id, "/") {
			return "", "", false
		}
		return id, "dispatch-dry-run", true
	}
	if strings.HasSuffix(trimmed, "/send-prepared") {
		id = strings.TrimSpace(strings.TrimSuffix(trimmed, "/send-prepared"))
		if id == "" || strings.Contains(id, "/") {
			return "", "", false
		}
		return id, "send-prepared", true
	}
	if strings.HasSuffix(trimmed, "/execute") {
		id = strings.TrimSpace(strings.TrimSuffix(trimmed, "/execute"))
		if id == "" || strings.Contains(id, "/") {
			return "", "", false
		}
		return id, "execute", true
	}
	if strings.Contains(trimmed, "/") {
		return "", "", false
	}
	return trimmed, "run", true
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

func parseTemplateVersionInt(versionLabel string) int {
	trimmed := strings.TrimSpace(versionLabel)
	if trimmed == "" {
		return 0
	}
	value, err := strconv.Atoi(trimmed)
	if err == nil {
		return value
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "v") {
		value, err = strconv.Atoi(strings.TrimPrefix(lower, "v"))
		if err == nil {
			return value
		}
	}
	return 0
}
