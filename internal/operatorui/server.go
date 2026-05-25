package operatorui

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actionplanner"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

const (
	operatorRouteBase = "/operator"
	operatorCSSRoute  = operatorRouteBase + "/static/operator.css"
)

func operatorRoute(path string) string {
	if path == "" {
		return operatorRouteBase
	}
	if strings.HasPrefix(path, "/") {
		return operatorRouteBase + path
	}
	return operatorRouteBase + "/" + path
}

type Store interface {
	GetInputChannel(ctx context.Context, id string) (*inputs.InputChannel, error)
	UpsertInputChannel(ctx context.Context, channel inputs.InputChannel) error
	ListInputChannels(ctx context.Context) ([]inputs.InputChannel, error)
	GetInputRuntimeState(ctx context.Context, inputID string) (*inputruntime.InputRuntimeState, error)
	UpsertInputRuntimeState(ctx context.Context, state inputruntime.InputRuntimeState) error
	RecordInputTransition(ctx context.Context, transition inputs.InputTransition) (int64, error)
	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)
	ListInputTransitions(ctx context.Context, limit int) ([]inputs.InputTransition, error)

	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	UpsertActionDefinition(ctx context.Context, definition actions.ActionDefinition) error
	ListActionDefinitions(ctx context.Context) ([]actions.ActionDefinition, error)
	ListActionRuns(ctx context.Context, query store.ActionRunListQuery) ([]actions.ActionRun, error)

	GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error)
	UpsertG2STemplate(ctx context.Context, tpl templates.G2STemplate) error
	ListG2STemplates(ctx context.Context) ([]templates.G2STemplate, error)
	UpsertG2STemplateVersion(ctx context.Context, version templates.G2STemplateVersion) error
	GetG2STemplateVersion(ctx context.Context, templateID string, version int) (*templates.G2STemplateVersion, error)
	GetActiveG2STemplateVersion(ctx context.Context, templateID string) (*templates.G2STemplateVersion, error)
	ListG2STemplateVersions(ctx context.Context, templateID string) ([]templates.G2STemplateVersion, error)
	SetActiveG2STemplateVersion(ctx context.Context, templateID string, version int) error

	GetEGMRecord(ctx context.Context, egmID string) (*egms.EGMRecord, error)
	UpsertEGMRecord(ctx context.Context, record egms.EGMRecord) error
	ListEGMRecords(ctx context.Context) ([]egms.EGMRecord, error)
	GetEGMGroup(ctx context.Context, id string) (*egms.EGMGroup, error)
	ListEGMGroups(ctx context.Context) ([]egms.EGMGroup, error)

	ListMessageJournalEntries(ctx context.Context, query store.MessageJournalListQuery) ([]g2sengine.MessageJournalEntry, error)
	ListAuditTimelineEntries(ctx context.Context, query store.AuditTimelineListQuery) ([]audit.AuditTimelineEntry, error)

	ListCertificateInventory(ctx context.Context) ([]model.CertificateInventory, error)
}

type Options struct {
	AppVersion              string
	ControllerID            string
	SiteName                string
	DatabasePath            string
	ConfigPath              string
	BindAddress             string
	G2SHostURL              string
	G2SEndpointPath         string
	G2SHostID               string
	TLSRequired             bool
	ClientCertRequired      bool
	WebLoginRequired        bool
	AdminClientCertRequired bool
	CAConfigured            bool
	ClientCertConfigured    bool
	ServerCertConfigured    bool
	StartedAt               time.Time
}

type Server struct {
	Store             Store
	Options           Options
	AuthorizeMutation func(http.ResponseWriter, *http.Request) bool
}

func NewServer(store Store, options Options, authorizeMutation func(http.ResponseWriter, *http.Request) bool) *Server {
	return &Server{
		Store:             store,
		Options:           options,
		AuthorizeMutation: authorizeMutation,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(operatorRoute(""), s.handleHome)
	mux.HandleFunc(operatorRoute("/inputs"), s.handleInputs)
	mux.HandleFunc(operatorRoute("/inputs/"), s.handleInputByID)
	mux.HandleFunc(operatorRoute("/actions"), s.handleActions)
	mux.HandleFunc(operatorRoute("/actions/"), s.handleActionByID)
	mux.HandleFunc(operatorRoute("/egms"), s.handleEGMs)
	mux.HandleFunc(operatorRoute("/egms/"), s.handleEGMByID)
	mux.HandleFunc(operatorRoute("/templates"), s.handleTemplates)
	mux.HandleFunc(operatorRoute("/templates/"), s.handleTemplateByID)
	mux.HandleFunc(operatorRoute("/comms"), s.handleComms)
	mux.HandleFunc(operatorRoute("/comms/export"), s.handleCommsExport)
	mux.HandleFunc(operatorRoute("/audit"), s.handleAudit)
	mux.HandleFunc(operatorRoute("/audit/export"), s.handleAuditExport)
	mux.HandleFunc(operatorRoute("/settings"), s.handleSettings)
	mux.HandleFunc(operatorCSSRoute, s.handleStyles)
}

func (s *Server) handleStyles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write([]byte(operatorCSS))
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	channels, err := s.Store.ListInputChannels(r.Context())
	if err != nil {
		s.renderError(w, operatorRoute(""), "Operator Console", err)
		return
	}
	triggered := 0
	latches := []string{}
	for _, channel := range channels {
		runtimeState, err := s.Store.GetInputRuntimeState(r.Context(), channel.ID)
		if err != nil {
			s.renderError(w, operatorRoute(""), "Operator Console", err)
			return
		}
		if runtimeState != nil && runtimeState.DerivedState == inputs.DerivedStateTriggered {
			triggered++
		}
		if runtimeState != nil && runtimeState.LatchActive {
			latches = append(latches, channel.ID)
		}
	}
	sort.Strings(latches)

	actionRuns, err := s.Store.ListActionRuns(r.Context(), store.ActionRunListQuery{Limit: 8})
	if err != nil {
		s.renderError(w, operatorRoute(""), "Operator Console", err)
		return
	}
	messages, err := s.Store.ListMessageJournalEntries(r.Context(), store.MessageJournalListQuery{Limit: 8})
	if err != nil {
		s.renderError(w, operatorRoute(""), "Operator Console", err)
		return
	}

	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Input Summary</h2>`)
	body.WriteString(fmt.Sprintf(`<p>Configured inputs: <strong>%d</strong> | Triggered now: <strong>%d</strong></p>`, len(channels), triggered))
	if len(latches) > 0 {
		body.WriteString(`<p>Active manual latches: <span class="mono">` + esc(strings.Join(latches, ", ")) + `</span></p>`)
	} else {
		body.WriteString(`<p>Active manual latches: none</p>`)
	}
	body.WriteString(`</div>`)

	body.WriteString(`<div class="panel"><h2>Latest Action Runs</h2>`)
	body.WriteString(`<table><tr><th>Started</th><th>Run ID</th><th>Action ID</th><th>Status</th><th>Targets</th></tr>`)
	for _, run := range actionRuns {
		body.WriteString(`<tr>`)
		body.WriteString(`<td>` + esc(fmtTime(run.StartedAt)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(run.ID) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(run.ActionDefinitionID) + `</td>`)
		body.WriteString(`<td>` + esc(string(run.Status)) + `</td>`)
		body.WriteString(`<td>` + esc(strconv.Itoa(run.TargetCount)) + `</td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h2>Latest Message Journal Entries</h2>`)
	body.WriteString(`<table><tr><th>Timestamp</th><th>Direction</th><th>EGM</th><th>Run</th><th>Type</th><th>Result</th></tr>`)
	for _, row := range messages {
		body.WriteString(`<tr>`)
		body.WriteString(`<td>` + esc(fmtTime(row.Timestamp)) + `</td>`)
		body.WriteString(`<td>` + esc(string(row.Direction)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.EGMID) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.ActionRunID) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.MessageType) + `</td>`)
		body.WriteString(`<td>` + esc(string(row.Result)) + `</td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)

	s.renderPage(w, operatorRoute(""), "Operator Console", body.String(), "", "")
}

func (s *Server) handleInputs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderInputsPage(w, r, "", "")
	case http.MethodPost:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleInputByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorizeMutation(w, r) {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, operatorRoute("/inputs/")))
	action := "update"
	inputID := strings.TrimSpace(strings.TrimSuffix(path, "/"))
	if strings.HasSuffix(path, "/clear-latch") {
		action = "clear-latch"
		inputID = strings.TrimSpace(strings.TrimSuffix(path, "/clear-latch"))
		inputID = strings.TrimSpace(strings.TrimSuffix(inputID, "/"))
	}
	if inputID == "" || strings.Contains(inputID, "/") {
		http.NotFound(w, r)
		return
	}
	if action == "clear-latch" {
		evaluator := inputruntime.Evaluator{Store: s.Store}
		result, err := evaluator.ClearLatchedInput(r.Context(), inputID, "operator-console", "operator clear latch")
		if err != nil {
			s.renderInputsPage(w, r, "", err.Error())
			return
		}
		msg := "Latch clear: " + inputID + " " + defaultString(strings.TrimSpace(result.Reason), "updated")
		if strings.TrimSpace(result.ActionQueuedID) != "" {
			msg += " action queued: " + strings.TrimSpace(result.ActionQueuedID)
		}
		s.renderInputsPage(w, r, msg, "")
		return
	}

	existing, err := s.Store.GetInputChannel(r.Context(), inputID)
	if err != nil {
		s.renderInputsPage(w, r, "", err.Error())
		return
	}
	if existing == nil {
		s.renderInputsPage(w, r, "", "input not found")
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderInputsPage(w, r, "", "invalid form payload")
		return
	}
	normalState, err := parseInputStateField(r.FormValue("normal_state"))
	if err != nil {
		s.renderInputsPage(w, r, "", err.Error())
		return
	}
	debounceMS, err := parseNonNegativeIntField(r.FormValue("debounce_ms"), "debounce")
	if err != nil {
		s.renderInputsPage(w, r, "", err.Error())
		return
	}
	priority, err := parseNonNegativeIntField(r.FormValue("priority"), "priority")
	if err != nil {
		s.renderInputsPage(w, r, "", err.Error())
		return
	}
	latchingMode, err := parseLatchingModeField(r.FormValue("latching_mode"))
	if err != nil {
		s.renderInputsPage(w, r, "", err.Error())
		return
	}
	updated := *existing
	updated.NormalState = normalState
	updated.DebounceMS = debounceMS
	updated.LatchingMode = latchingMode
	updated.Priority = priority
	updated.OnTriggerActionID = strings.TrimSpace(r.FormValue("on_trigger_action_id"))
	updated.OnNormalActionID = strings.TrimSpace(r.FormValue("on_normal_action_id"))
	if err := updated.Validate(); err != nil {
		s.renderInputsPage(w, r, "", err.Error())
		return
	}
	if err := s.Store.UpsertInputChannel(r.Context(), updated); err != nil {
		s.renderInputsPage(w, r, "", err.Error())
		return
	}
	s.renderInputsPage(w, r, "Input updated: "+updated.ID, "")
}

func (s *Server) renderInputsPage(w http.ResponseWriter, r *http.Request, message string, errText string) {
	channels, err := s.Store.ListInputChannels(r.Context())
	if err != nil {
		s.renderError(w, "/operator/inputs", "Operator Console Inputs", err)
		return
	}
	transitions, err := s.Store.ListInputTransitions(r.Context(), 200)
	if err != nil {
		s.renderError(w, "/operator/inputs", "Operator Console Inputs", err)
		return
	}
	lastTransitionByInput := map[string]inputs.InputTransition{}
	for _, transition := range transitions {
		if _, exists := lastTransitionByInput[transition.InputChannelID]; !exists {
			lastTransitionByInput[transition.InputChannelID] = transition
		}
	}
	auditRows, err := s.Store.ListAuditTimelineEntries(r.Context(), store.AuditTimelineListQuery{Limit: 300})
	if err != nil {
		s.renderError(w, "/operator/inputs", "Operator Console Inputs", err)
		return
	}
	auditByTransitionID := map[int64]audit.AuditTimelineEntry{}
	for _, row := range auditRows {
		if row.InputTransitionID > 0 {
			if _, exists := auditByTransitionID[row.InputTransitionID]; !exists {
				auditByTransitionID[row.InputTransitionID] = row
			}
		}
	}

	body := strings.Builder{}
	expectedInputs := []string{
		"regular-operation",
		"general-broadcast",
		"emergency-broadcast",
		"local-notice",
	}
	presentInputs := map[string]bool{}
	for _, channel := range channels {
		presentInputs[channel.ID] = true
	}
	missingInputs := []string{}
	for _, id := range expectedInputs {
		if !presentInputs[id] {
			missingInputs = append(missingInputs, id)
		}
	}
	if len(missingInputs) > 0 {
		body.WriteString(`<div class="panel"><h3>Input Coverage</h3><p>Missing configured input channels: <span class="mono">` + esc(strings.Join(missingInputs, ", ")) + `</span></p></div>`)
	}

	body.WriteString(`<div class="panel"><h2>Configured Inputs</h2><table>`)
	body.WriteString(`<tr><th>ID</th><th>Name</th><th>GPIO</th><th>Raw</th><th>Derived</th><th>Latch Mode</th><th>Latch Active</th><th>Normal</th><th>Debounce</th><th>Priority</th><th>On Trigger</th><th>On Normal</th><th>Last Transition</th><th>Edit</th></tr>`)
	for _, channel := range channels {
		runtimeState, err := s.Store.GetInputRuntimeState(r.Context(), channel.ID)
		if err != nil {
			s.renderError(w, "/operator/inputs", "Operator Console Inputs", err)
			return
		}
		lastTransition := "-"
		if transition, ok := lastTransitionByInput[channel.ID]; ok {
			actionQueued := actionQueuedFromTransition(transition, auditByTransitionID[transition.ID])
			lastTransition = fmt.Sprintf("%s %s->%s action=%s", fmtTime(transition.TransitionAt), transition.PreviousDerived, transition.NewDerived, defaultString(strings.TrimSpace(actionQueued), "-"))
		}
		rawState := string(channel.CurrentState)
		derivedState := string(channel.DerivedState)
		latchActive := "no"
		if runtimeState != nil {
			if runtimeState.LastObservedRawState != "" {
				rawState = string(runtimeState.LastObservedRawState)
			}
			if runtimeState.DerivedState != "" {
				derivedState = string(runtimeState.DerivedState)
			}
			if runtimeState.LatchActive {
				latchActive = "yes"
			}
		}
		body.WriteString(`<tr>`)
		body.WriteString(`<td class="mono">` + esc(channel.ID) + `</td>`)
		body.WriteString(`<td>` + esc(channel.Name) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(channel.GPIOChannel) + `</td>`)
		body.WriteString(`<td>` + esc(rawState) + `</td>`)
		body.WriteString(`<td>` + esc(derivedState) + `</td>`)
		body.WriteString(`<td>` + esc(string(channel.LatchingMode)) + `</td>`)
		body.WriteString(`<td>` + esc(latchActive) + `</td>`)
		body.WriteString(`<td>` + esc(string(channel.NormalState)) + `</td>`)
		body.WriteString(`<td>` + esc(strconv.Itoa(channel.DebounceMS)) + `</td>`)
		body.WriteString(`<td>` + esc(strconv.Itoa(channel.Priority)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(channel.OnTriggerActionID) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(channel.OnNormalActionID) + `</td>`)
		body.WriteString(`<td>` + esc(lastTransition) + `</td>`)
		body.WriteString(`<td><form class="inline-form" method="post" action="/operator/inputs/` + esc(channel.ID) + `">`)
		body.WriteString(`normal <select name="normal_state"><option value="HIGH"` + selected(string(channel.NormalState), string(inputs.InputStateHigh)) + `>HIGH</option><option value="LOW"` + selected(string(channel.NormalState), string(inputs.InputStateLow)) + `>LOW</option></select> `)
		body.WriteString(`debounce <input type="number" name="debounce_ms" value="` + esc(strconv.Itoa(channel.DebounceMS)) + `" style="width:76px"> `)
		body.WriteString(`latch <select name="latching_mode"><option value="AUTO_CLEAR"` + selected(string(channel.LatchingMode), string(inputs.LatchingAutoClear)) + `>AUTO_CLEAR</option><option value="MANUAL_CLEAR"` + selected(string(channel.LatchingMode), string(inputs.LatchingManualClear)) + `>MANUAL_CLEAR</option></select> `)
		body.WriteString(`priority <input type="number" name="priority" value="` + esc(strconv.Itoa(channel.Priority)) + `" style="width:66px"> `)
		body.WriteString(`<br>on-trigger <input type="text" name="on_trigger_action_id" value="` + esc(channel.OnTriggerActionID) + `" style="width:180px"> `)
		body.WriteString(`on-normal <input type="text" name="on_normal_action_id" value="` + esc(channel.OnNormalActionID) + `" style="width:180px"> `)
		if channel.LatchingMode == inputs.LatchingManualClear {
			body.WriteString(`<br><button type="submit" formaction="/operator/inputs/` + esc(channel.ID) + `/clear-latch">Clear Latch</button>`)
		}
		body.WriteString(`<button type="submit">Save</button></form></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h3>Recent Input Transitions</h3><table>`)
	body.WriteString(`<tr><th>Timestamp</th><th>Input ID</th><th>From</th><th>To</th><th>Raw From</th><th>Raw To</th><th>Action Queued</th><th>Transition ID</th><th>Related Audit</th><th>Notes</th></tr>`)
	for _, transition := range transitions {
		auditRow := auditByTransitionID[transition.ID]
		rawTo := "-"
		if strings.TrimSpace(auditRow.DetailJSON) != "" {
			rawTo = defaultString(extractJSONValue(auditRow.DetailJSON, "raw_state"), "-")
		}
		actionQueued := actionQueuedFromTransition(transition, auditRow)
		note := defaultString(strings.TrimSpace(transition.Reason), "-")
		if strings.TrimSpace(auditRow.Summary) != "" {
			note = auditRow.Summary
		}
		body.WriteString(`<tr>`)
		body.WriteString(`<td>` + esc(fmtTime(transition.TransitionAt)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(transition.InputChannelID) + `</td>`)
		body.WriteString(`<td>` + esc(string(transition.PreviousDerived)) + `</td>`)
		body.WriteString(`<td>` + esc(string(transition.NewDerived)) + `</td>`)
		body.WriteString(`<td>-</td>`)
		body.WriteString(`<td>` + esc(rawTo) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(defaultString(strings.TrimSpace(actionQueued), "-")) + `</td>`)
		body.WriteString(`<td>` + esc(zeroDash64(transition.ID)) + `</td>`)
		body.WriteString(`<td>` + esc(zeroDash64(auditRow.ID)) + `</td>`)
		body.WriteString(`<td>` + esc(note) + `</td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)
	s.renderPage(w, operatorRoute("/inputs"), "Operator Inputs", body.String(), message, errText)
}

func (s *Server) handleActions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderActionsPage(w, r, "", "")
	case http.MethodPost:
		if !s.authorizeMutation(w, r) {
			return
		}
		if err := r.ParseForm(); err != nil {
			s.renderActionsPage(w, r, "", "invalid form payload")
			return
		}
		actionID := strings.TrimSpace(r.FormValue("id"))
		if actionID == "" {
			s.renderActionsPage(w, r, "", "action id is required")
			return
		}
		if err := s.upsertActionFromForm(r.Context(), actionID, r); err != nil {
			s.renderActionsPage(w, r, "", err.Error())
			return
		}
		s.renderActionsPage(w, r, "Action upserted: "+actionID, "")
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleActionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorizeMutation(w, r) {
		return
	}
	actionID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, operatorRoute("/actions/")))
	if actionID == "" || strings.Contains(actionID, "/") {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderActionsPage(w, r, "", "invalid form payload")
		return
	}
	if err := s.upsertActionFromForm(r.Context(), actionID, r); err != nil {
		s.renderActionsPage(w, r, "", err.Error())
		return
	}
	s.renderActionsPage(w, r, "Action updated: "+actionID, "")
}

func (s *Server) upsertActionFromForm(ctx context.Context, actionID string, r *http.Request) error {
	existing, err := s.Store.GetActionDefinition(ctx, actionID)
	if err != nil {
		return err
	}
	version := 1
	currentEnabled := true
	if existing != nil {
		version = existing.Version
		currentEnabled = existing.Enabled
	}
	enabled := currentEnabled
	if r.Form.Has("enabled") {
		enabled = parseFormBool(r.FormValue("enabled"))
	}

	targetSelector, err := parseTargetSelectorForm(r.FormValue("target_selector_type"), r.FormValue("target_selector_value"))
	if err != nil {
		return err
	}
	templateSelector := strings.TrimSpace(r.FormValue("template_selector"))
	if templateSelector == "" {
		return fmt.Errorf("template selector is required")
	}

	stepKey := strings.TrimSpace(r.FormValue("step_template_action_key"))
	if stepKey == "" && existing != nil && len(existing.Steps) > 0 {
		stepKey = existing.Steps[0].TemplateActionKey
	}
	if stepKey == "" {
		stepKey = "regular_operation_notice"
	}
	retryPolicy, err := parseRetryPolicyForm(r.FormValue("retry_count"), r.FormValue("retry_delay_ms"))
	if err != nil {
		return err
	}
	escalationPolicy, err := parseEscalationPolicyForm(r.FormValue("escalation_action_id"), r.FormValue("escalation_after_attempts"))
	if err != nil {
		return err
	}

	definition := actions.ActionDefinition{
		ID:               actionID,
		Name:             strings.TrimSpace(r.FormValue("name")),
		Severity:         actions.ActionSeverity(strings.ToUpper(strings.TrimSpace(r.FormValue("severity")))),
		Enabled:          enabled,
		TargetSelector:   targetSelector,
		TemplateSelector: templateSelector,
		Steps: []actions.ActionStep{{
			ID:                "step-1",
			Name:              "Step 1",
			Sequence:          0,
			TemplateActionKey: stepKey,
		}},
		RetryPolicyJSON: retryPolicy.toJSON(),
		EscalationJSON:  escalationPolicy.toJSON(),
		ReturnActionID:  strings.TrimSpace(r.FormValue("return_action_id")),
		Version:         version,
	}
	if existing != nil {
		if definition.Name == "" {
			definition.Name = existing.Name
		}
		if definition.Severity == "" {
			definition.Severity = existing.Severity
		}
		if definition.TargetSelector == "" {
			definition.TargetSelector = existing.TargetSelector
		}
		if definition.TemplateSelector == "" {
			definition.TemplateSelector = existing.TemplateSelector
		}
		if definition.ReturnActionID == "" {
			definition.ReturnActionID = existing.ReturnActionID
		}
	}
	if err := validateActionTargetSelector(definition.TargetSelector); err != nil {
		return err
	}
	return s.Store.UpsertActionDefinition(ctx, definition)
}

func (s *Server) renderActionsPage(w http.ResponseWriter, r *http.Request, message string, errText string) {
	definitions, err := s.Store.ListActionDefinitions(r.Context())
	if err != nil {
		s.renderError(w, "/operator/actions", "Operator Console Actions", err)
		return
	}
	planner := actionplanner.Planner{Store: s.Store}
	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Action Definitions</h2><table>`)
	body.WriteString(`<tr><th>ID</th><th>Name</th><th>Severity</th><th>Enabled</th><th>Target Selector</th><th>Template Selector</th><th>Template Action Key</th><th>Return Action</th><th>Retry Count</th><th>Retry Delay (ms)</th><th>Escalation Action</th><th>Escalation After Attempts</th><th>Target Preview</th><th>Edit</th></tr>`)
	for _, definition := range definitions {
		stepKeys := []string{}
		for _, step := range definition.Steps {
			stepKeys = append(stepKeys, step.TemplateActionKey)
		}
		firstStep := ""
		if len(stepKeys) > 0 {
			firstStep = stepKeys[0]
		}
		retryPolicy := parseRetryPolicyJSON(definition.RetryPolicyJSON)
		escalationPolicy := parseEscalationPolicyJSON(definition.EscalationJSON)
		selectorType, selectorValue, selectorReadable := selectorTypeValueAndReadable(definition.TargetSelector)

		plan, planErr := planner.BuildPlanForDefinition(r.Context(), definition)
		previewSummary := "target preview unavailable"
		if planErr == nil {
			targets := make([]string, 0, len(plan.Targets))
			for _, target := range plan.Targets {
				if strings.TrimSpace(target.DisplayName) != "" {
					targets = append(targets, target.DisplayName+" ("+target.EGMID+")")
				} else {
					targets = append(targets, target.EGMID)
				}
			}
			targetText := "-"
			if len(targets) > 0 {
				targetText = strings.Join(targets, ", ")
			}
			warnings := []string{}
			for _, warning := range plan.Warnings {
				warnings = append(warnings, warning.Code+": "+warning.Message)
			}
			if definition.Severity == actions.SeverityEmergency && strings.TrimSpace(definition.ReturnActionID) == "" {
				warnings = append(warnings, "EMERGENCY_RETURN_MISSING: Emergency action has no return action configured")
			}
			previewSummary = fmt.Sprintf("targets=%d [%s]", plan.TargetCount, targetText)
			if len(warnings) > 0 {
				previewSummary += " warnings=" + strings.Join(warnings, " | ")
			}
		}
		body.WriteString(`<tr>`)
		body.WriteString(`<td class="mono">` + esc(definition.ID) + `</td>`)
		body.WriteString(`<td>` + esc(definition.Name) + `</td>`)
		body.WriteString(`<td>` + esc(string(definition.Severity)) + `</td>`)
		body.WriteString(`<td>` + yesNo(definition.Enabled) + `</td>`)
		body.WriteString(`<td><div>` + esc(selectorReadable) + `</div><div class="mono">` + esc(definition.TargetSelector) + `</div></td>`)
		body.WriteString(`<td class="mono">` + esc(definition.TemplateSelector) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(strings.Join(stepKeys, ", ")) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(definition.ReturnActionID) + `</td>`)
		body.WriteString(`<td>` + esc(strconv.Itoa(retryPolicy.Count)) + `</td>`)
		body.WriteString(`<td>` + esc(strconv.Itoa(retryPolicy.DelayMS)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(escalationPolicy.ActionID) + `</td>`)
		body.WriteString(`<td>` + esc(strconv.Itoa(escalationPolicy.AfterAttempts)) + `</td>`)
		body.WriteString(`<td><details><summary>summary</summary><pre>` + esc(previewSummary) + `</pre></details><a href="/api/v2/actions/` + esc(definition.ID) + `/preview" target="_blank" rel="noreferrer">View API Preview</a></td>`)
		body.WriteString(`<td><form class="inline-form" method="post" action="/operator/actions/` + esc(definition.ID) + `">`)
		body.WriteString(`<input type="hidden" name="id" value="` + esc(definition.ID) + `">`)
		body.WriteString(`name <input type="text" name="name" value="` + esc(definition.Name) + `" style="width:140px"> `)
		body.WriteString(`severity <select name="severity">` + severityOptions(definition.Severity) + `</select> `)
		body.WriteString(`enabled <input type="checkbox" name="enabled" value="true"` + checked(definition.Enabled) + `> `)
		body.WriteString(`<br>target type <select name="target_selector_type">` + targetSelectorTypeOptions(selectorType) + `</select> `)
		body.WriteString(`target value <input type="text" name="target_selector_value" value="` + esc(selectorValue) + `" style="width:220px"> `)
		body.WriteString(`template <input type="text" name="template_selector" value="` + esc(definition.TemplateSelector) + `" style="width:150px"> `)
		body.WriteString(`<br>step key <input type="text" name="step_template_action_key" value="` + esc(firstStep) + `" style="width:180px"> `)
		body.WriteString(`return <input type="text" name="return_action_id" value="` + esc(definition.ReturnActionID) + `" style="width:180px"> `)
		body.WriteString(`<br>retry count <input type="number" name="retry_count" min="0" value="` + esc(strconv.Itoa(retryPolicy.Count)) + `" style="width:90px"> `)
		body.WriteString(`retry delay ms <input type="number" name="retry_delay_ms" min="0" value="` + esc(strconv.Itoa(retryPolicy.DelayMS)) + `" style="width:110px"> `)
		body.WriteString(`<br>escalation action <input type="text" name="escalation_action_id" value="` + esc(escalationPolicy.ActionID) + `" style="width:180px"> `)
		body.WriteString(`escalation after attempts <input type="number" name="escalation_after_attempts" min="0" value="` + esc(strconv.Itoa(escalationPolicy.AfterAttempts)) + `" style="width:110px"> `)
		body.WriteString(`<button type="submit">Save</button></form></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)
	body.WriteString(`<div class="panel"><p>Retry, escalation, and return fields are configuration only in this console. Execution behavior is not available here.</p></div>`)

	body.WriteString(`<div class="panel"><h3>Add / Upsert Action</h3>`)
	body.WriteString(`<form method="post" action="/operator/actions">`)
	body.WriteString(`<label>ID <input type="text" name="id"></label>`)
	body.WriteString(`<label>Name <input type="text" name="name"></label>`)
	body.WriteString(`<label>Severity <select name="severity">` + severityOptions("") + `</select></label>`)
	body.WriteString(`<label>Enabled <input type="checkbox" name="enabled" value="true" checked></label><br>`)
	body.WriteString(`<label>Target Selector Type <select name="target_selector_type">` + targetSelectorTypeOptions(targetSelectorTypeAllEmergencyEnabled) + `</select></label>`)
	body.WriteString(`<label>Target Selector Value <input type="text" name="target_selector_value" style="width:220px"></label>`)
	body.WriteString(`<label>Template Selector <input type="text" name="template_selector" value="template-by-egm" style="width:200px"></label><br>`)
	body.WriteString(`<label>Step Template Action Key <input type="text" name="step_template_action_key" value="regular_operation_notice" style="width:220px"></label>`)
	body.WriteString(`<label>Return Action ID <input type="text" name="return_action_id" style="width:220px"></label><br>`)
	body.WriteString(`<label>Retry Count <input type="number" name="retry_count" min="0" value="0" style="width:100px"></label>`)
	body.WriteString(`<label>Retry Delay Milliseconds <input type="number" name="retry_delay_ms" min="0" value="0" style="width:120px"></label><br>`)
	body.WriteString(`<label>Escalation Action ID <input type="text" name="escalation_action_id" style="width:220px"></label>`)
	body.WriteString(`<label>Escalation After Attempts <input type="number" name="escalation_after_attempts" min="0" value="0" style="width:120px"></label><br>`)
	body.WriteString(`<button type="submit">Upsert Action</button></form></div>`)

	s.renderPage(w, operatorRoute("/actions"), "Operator Actions", body.String(), message, errText)
}

func (s *Server) handleEGMs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderEGMsPage(w, r, "", "", nil)
	case http.MethodPost:
		if !s.authorizeMutation(w, r) {
			return
		}
		if err := r.ParseForm(); err != nil {
			s.renderEGMsPage(w, r, "", "invalid form payload", nil)
			return
		}
		egmID := strings.TrimSpace(r.FormValue("egm_id"))
		draft := buildEGMFormData(r, egmID)
		if egmID == "" {
			s.renderEGMsPage(w, r, "", "egm_id is required", &draft)
			return
		}
		if err := s.upsertEGMFromForm(r.Context(), draft); err != nil {
			s.renderEGMsPage(w, r, "", err.Error(), &draft)
			return
		}
		s.renderEGMsPage(w, r, "EGM upserted: "+egmID, "", nil)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleEGMByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorizeMutation(w, r) {
		return
	}
	egmID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, operatorRoute("/egms/")))
	if egmID == "" || strings.Contains(egmID, "/") {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderEGMsPage(w, r, "", "invalid form payload", nil)
		return
	}
	draft := buildEGMFormData(r, egmID)
	if err := s.upsertEGMFromForm(r.Context(), draft); err != nil {
		s.renderEGMsPage(w, r, "", err.Error(), &draft)
		return
	}
	s.renderEGMsPage(w, r, "EGM updated: "+egmID, "", nil)
}

type egmFormData struct {
	EGMID            string
	DisplayName      string
	IPAddress        string
	EndpointPath     string
	Vendor           string
	CabinetFamily    string
	GameTitle        string
	SoftwareVersion  string
	Zone             string
	Enabled          bool
	EmergencyEnabled bool
	TemplateID       string
	Notes            string
}

func buildEGMFormData(r *http.Request, egmID string) egmFormData {
	return egmFormData{
		EGMID:            strings.TrimSpace(egmID),
		DisplayName:      strings.TrimSpace(r.FormValue("display_name")),
		IPAddress:        strings.TrimSpace(r.FormValue("ip_address")),
		EndpointPath:     strings.TrimSpace(r.FormValue("endpoint_path")),
		Vendor:           strings.TrimSpace(r.FormValue("vendor")),
		CabinetFamily:    strings.TrimSpace(r.FormValue("cabinet_family")),
		GameTitle:        strings.TrimSpace(r.FormValue("game_title")),
		SoftwareVersion:  strings.TrimSpace(r.FormValue("software_version")),
		Zone:             strings.TrimSpace(r.FormValue("zone")),
		Enabled:          parseFormBool(r.FormValue("enabled")),
		EmergencyEnabled: parseFormBool(r.FormValue("emergency_enabled")),
		TemplateID:       strings.TrimSpace(r.FormValue("template_id")),
		Notes:            strings.TrimSpace(r.FormValue("notes")),
	}
}

func (s *Server) upsertEGMFromForm(ctx context.Context, form egmFormData) error {
	if form.EGMID == "" {
		return fmt.Errorf("egm_id is required")
	}
	existing, err := s.Store.GetEGMRecord(ctx, form.EGMID)
	if err != nil {
		return err
	}
	state := egms.EGMActionStateNormal
	if existing != nil && existing.CurrentActionState != "" {
		state = existing.CurrentActionState
	}
	record := egms.EGMRecord{
		EGMID:              form.EGMID,
		DisplayName:        form.DisplayName,
		IPAddress:          form.IPAddress,
		EndpointPath:       form.EndpointPath,
		Vendor:             form.Vendor,
		CabinetFamily:      form.CabinetFamily,
		GameTitle:          form.GameTitle,
		SoftwareVersion:    form.SoftwareVersion,
		Zone:               form.Zone,
		Enabled:            form.Enabled,
		EmergencyEnabled:   form.EmergencyEnabled,
		TemplateID:         form.TemplateID,
		Notes:              form.Notes,
		CurrentActionState: state,
	}
	if existing != nil {
		record.HeartbeatOverrideJSON = existing.HeartbeatOverrideJSON
		record.LastSeenAt = existing.LastSeenAt
	}
	return s.Store.UpsertEGMRecord(ctx, record)
}

func (s *Server) renderEGMsPage(w http.ResponseWriter, r *http.Request, message string, errText string, draft *egmFormData) {
	records, err := s.Store.ListEGMRecords(r.Context())
	if err != nil {
		s.renderError(w, "/operator/egms", "Operator Console EGMs", err)
		return
	}
	templatesList, err := s.Store.ListG2STemplates(r.Context())
	if err != nil {
		s.renderError(w, "/operator/egms", "Operator Console EGMs", err)
		return
	}
	templateExists := make(map[string]bool, len(templatesList))
	templateIDs := make([]string, 0, len(templatesList))
	for _, tpl := range templatesList {
		templateExists[tpl.ID] = true
		templateIDs = append(templateIDs, tpl.ID)
	}
	sort.Strings(templateIDs)

	groups, err := s.Store.ListEGMGroups(r.Context())
	if err != nil {
		s.renderError(w, "/operator/egms", "Operator Console EGMs", err)
		return
	}

	formDefaults := egmFormData{Enabled: true, EmergencyEnabled: true}
	if draft != nil {
		formDefaults = *draft
	}

	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>EGM Registry</h2><table>`)
	body.WriteString(`<tr><th>EGM ID</th><th>Cabinet</th><th>IP Address</th><th>Endpoint</th><th>Vendor</th><th>Cabinet Family</th><th>Game Title</th><th>Software Version</th><th>Zone</th><th>Enabled</th><th>Emergency Enabled</th><th>Template</th><th>Current Action State</th><th>Last Seen</th><th>Notes</th><th>Status</th><th>Edit</th></tr>`)
	for _, record := range records {
		warnings := make([]string, 0, 2)
		if record.TemplateID != "" && !templateExists[record.TemplateID] {
			warnings = append(warnings, "Template not found")
		}
		if !record.Enabled && record.EmergencyEnabled {
			warnings = append(warnings, "Emergency participation requires Enabled.")
		}
		body.WriteString(`<tr>`)
		body.WriteString(`<td class="mono">` + esc(record.EGMID) + `</td>`)
		body.WriteString(`<td>` + esc(record.DisplayName) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(record.IPAddress) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(record.EndpointPath) + `</td>`)
		body.WriteString(`<td>` + esc(record.Vendor) + `</td>`)
		body.WriteString(`<td>` + esc(record.CabinetFamily) + `</td>`)
		body.WriteString(`<td>` + esc(record.GameTitle) + `</td>`)
		body.WriteString(`<td>` + esc(record.SoftwareVersion) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(record.Zone) + `</td>`)
		body.WriteString(`<td>` + yesNo(record.Enabled) + `</td>`)
		body.WriteString(`<td>` + yesNo(record.EmergencyEnabled) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(record.TemplateID) + `</td>`)
		body.WriteString(`<td>` + esc(string(record.CurrentActionState)) + `</td>`)
		body.WriteString(`<td>` + esc(fmtMaybeTime(record.LastSeenAt)) + `</td>`)
		body.WriteString(`<td>` + esc(record.Notes) + `</td>`)
		body.WriteString(`<td>`)
		if len(warnings) == 0 {
			body.WriteString(`<span class="status-ok">OK</span>`)
		} else {
			for i, warning := range warnings {
				if i > 0 {
					body.WriteString(`<br>`)
				}
				body.WriteString(`<span class="status-warn">` + esc(warning) + `</span>`)
			}
		}
		body.WriteString(`</td>`)
		body.WriteString(`<td><form class="inline-form" method="post" action="/operator/egms/` + esc(record.EGMID) + `">`)
		body.WriteString(`<label>Display Name <input type="text" name="display_name" value="` + esc(record.DisplayName) + `" style="width:130px"></label>`)
		body.WriteString(`<label>IP Address <input type="text" name="ip_address" value="` + esc(record.IPAddress) + `" style="width:110px"></label>`)
		body.WriteString(`<label>Endpoint Path <input type="text" name="endpoint_path" value="` + esc(record.EndpointPath) + `" style="width:110px"></label><br>`)
		body.WriteString(`<label>Vendor <input type="text" name="vendor" value="` + esc(record.Vendor) + `" style="width:120px"></label>`)
		body.WriteString(`<label>Cabinet Family <input type="text" name="cabinet_family" value="` + esc(record.CabinetFamily) + `" style="width:120px"></label>`)
		body.WriteString(`<label>Game Title <input type="text" name="game_title" value="` + esc(record.GameTitle) + `" style="width:120px"></label>`)
		body.WriteString(`<label>Software Version <input type="text" name="software_version" value="` + esc(record.SoftwareVersion) + `" style="width:100px"></label><br>`)
		body.WriteString(`<label>Zone <input type="text" name="zone" value="` + esc(record.Zone) + `" style="width:90px"></label>`)
		body.WriteString(`<label>Enabled <input type="checkbox" name="enabled" value="true"` + checked(record.Enabled) + `></label>`)
		body.WriteString(`<label>Emergency Enabled <input type="checkbox" name="emergency_enabled" value="true"` + checked(record.EmergencyEnabled) + `></label>`)
		body.WriteString(`<label>Template ID <input type="text" name="template_id" value="` + esc(record.TemplateID) + `" style="width:140px"></label><br>`)
		body.WriteString(`<label>Notes <input type="text" name="notes" value="` + esc(record.Notes) + `" style="width:280px"></label>`)
		body.WriteString(`<button type="submit">Save</button></form></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h3>Available Template IDs</h3>`)
	if len(templateIDs) == 0 {
		body.WriteString(`<p>None</p>`)
	} else {
		body.WriteString(`<p class="mono">` + esc(strings.Join(templateIDs, ", ")) + `</p>`)
	}
	body.WriteString(`</div>`)

	body.WriteString(`<div class="panel"><h3>EGM Groups</h3>`)
	if len(groups) == 0 {
		body.WriteString(`<p>No groups configured.</p>`)
	} else {
		body.WriteString(`<table><tr><th>Group ID</th><th>Name</th><th>Member Count</th><th>Members</th></tr>`)
		for _, group := range groups {
			members := append([]string(nil), group.EGMIDs...)
			sort.Strings(members)
			body.WriteString(`<tr>`)
			body.WriteString(`<td class="mono">` + esc(group.ID) + `</td>`)
			body.WriteString(`<td>` + esc(group.Name) + `</td>`)
			body.WriteString(`<td>` + esc(strconv.Itoa(len(members))) + `</td>`)
			body.WriteString(`<td class="mono">` + esc(strings.Join(members, ", ")) + `</td>`)
			body.WriteString(`</tr>`)
		}
		body.WriteString(`</table>`)
	}
	body.WriteString(`</div>`)

	body.WriteString(`<div class="panel"><h3>Add / Upsert EGM</h3>`)
	body.WriteString(`<form method="post" action="/operator/egms">`)
	body.WriteString(`<label>EGM ID <input type="text" name="egm_id" value="` + esc(formDefaults.EGMID) + `"></label>`)
	body.WriteString(`<label>Display Name <input type="text" name="display_name" value="` + esc(formDefaults.DisplayName) + `"></label>`)
	body.WriteString(`<label>IP Address <input type="text" name="ip_address" value="` + esc(formDefaults.IPAddress) + `"></label>`)
	body.WriteString(`<label>Endpoint Path <input type="text" name="endpoint_path" value="` + esc(formDefaults.EndpointPath) + `"></label><br>`)
	body.WriteString(`<label>Vendor <input type="text" name="vendor" value="` + esc(formDefaults.Vendor) + `"></label>`)
	body.WriteString(`<label>Cabinet Family <input type="text" name="cabinet_family" value="` + esc(formDefaults.CabinetFamily) + `"></label>`)
	body.WriteString(`<label>Game Title <input type="text" name="game_title" value="` + esc(formDefaults.GameTitle) + `"></label>`)
	body.WriteString(`<label>Software Version <input type="text" name="software_version" value="` + esc(formDefaults.SoftwareVersion) + `"></label><br>`)
	body.WriteString(`<label>Zone <input type="text" name="zone" value="` + esc(formDefaults.Zone) + `"></label>`)
	body.WriteString(`<label>Enabled <input type="checkbox" name="enabled" value="true"` + checked(formDefaults.Enabled) + `></label>`)
	body.WriteString(`<label>Emergency Enabled <input type="checkbox" name="emergency_enabled" value="true"` + checked(formDefaults.EmergencyEnabled) + `></label>`)
	body.WriteString(`<label>Template ID <input type="text" name="template_id" value="` + esc(formDefaults.TemplateID) + `"></label><br>`)
	body.WriteString(`<label>Notes <input type="text" name="notes" value="` + esc(formDefaults.Notes) + `" style="width:320px"></label>`)
	body.WriteString(`<button type="submit">Upsert EGM</button></form></div>`)

	s.renderPage(w, operatorRoute("/egms"), "Operator EGM Registry", body.String(), message, errText)
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderTemplatesPage(w, r, "", "", nil)
	case http.MethodPost:
		if !s.authorizeMutation(w, r) {
			return
		}
		if err := r.ParseForm(); err != nil {
			s.renderTemplatesPage(w, r, "", "invalid form payload", nil)
			return
		}
		templateID := strings.TrimSpace(r.FormValue("id"))
		if templateID == "" {
			s.renderTemplatesPage(w, r, "", "template id is required", nil)
			return
		}
		templateRow := templates.G2STemplate{
			ID:                   templateID,
			Name:                 strings.TrimSpace(r.FormValue("name")),
			Vendor:               strings.TrimSpace(r.FormValue("vendor")),
			CabinetFamily:        strings.TrimSpace(r.FormValue("cabinet_family")),
			SoftwareVersionMatch: strings.TrimSpace(r.FormValue("software_version_match")),
			Status:               templates.TemplateStatus(strings.ToUpper(strings.TrimSpace(r.FormValue("status")))),
			Notes:                strings.TrimSpace(r.FormValue("notes")),
		}
		existing, err := s.Store.GetG2STemplate(r.Context(), templateID)
		if err != nil {
			s.renderTemplatesPage(w, r, "", err.Error(), nil)
			return
		}
		if existing != nil {
			templateRow.CurrentVersionID = existing.CurrentVersionID
		}
		if err := s.Store.UpsertG2STemplate(r.Context(), templateRow); err != nil {
			s.renderTemplatesPage(w, r, "", err.Error(), nil)
			return
		}
		s.renderTemplatesPage(w, r, "Template upserted: "+templateID, "", nil)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTemplateByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, operatorRoute("/templates/")))
	if path == "" {
		http.NotFound(w, r)
		return
	}
	if path == "render-preview" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			s.renderTemplatesPage(w, r, "", "invalid form payload", nil)
			return
		}
		preview, err := s.renderTemplatePreview(r.Context(), r)
		if err != nil {
			s.renderTemplatesPage(w, r, "", err.Error(), nil)
			return
		}
		s.renderTemplatesPage(w, r, "Render preview complete", "", &templatePreviewState{Render: preview})
		return
	}
	if path == "match-preview" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			s.renderTemplatesPage(w, r, "", "invalid form payload", nil)
			return
		}
		preview, err := s.matchTemplatePreview(r.Context(), r)
		if err != nil {
			s.renderTemplatesPage(w, r, "", err.Error(), nil)
			return
		}
		s.renderTemplatesPage(w, r, "Match preview complete", "", &templatePreviewState{Match: preview})
		return
	}

	if strings.HasSuffix(path, "/versions") {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !s.authorizeMutation(w, r) {
			return
		}
		templateID := strings.TrimSuffix(path, "/versions")
		templateID = strings.TrimSpace(strings.TrimSuffix(templateID, "/"))
		if templateID == "" || strings.Contains(templateID, "/") {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			s.renderTemplatesPage(w, r, "", "invalid form payload", nil)
			return
		}
		versionLabel := strings.TrimSpace(r.FormValue("version_label"))
		if versionLabel == "" {
			s.renderTemplatesPage(w, r, "", "version_label is required", nil)
			return
		}
		versionValue, err := strconv.Atoi(versionLabel)
		if err != nil || versionValue <= 0 {
			s.renderTemplatesPage(w, r, "", "version_label must be a positive integer", nil)
			return
		}
		versionID := strings.TrimSpace(r.FormValue("version_id"))
		if versionID == "" {
			versionID = fmt.Sprintf("%s-v%d", templateID, versionValue)
		}
		row := templates.G2STemplateVersion{
			ID:                    versionID,
			TemplateID:            templateID,
			VersionLabel:          versionLabel,
			EndpointQuirksJSON:    strings.TrimSpace(r.FormValue("endpoint_quirks_json")),
			ActionsJSON:           strings.TrimSpace(r.FormValue("actions_json")),
			ConfirmationRulesJSON: strings.TrimSpace(r.FormValue("confirmation_rules_json")),
			FailureRulesJSON:      strings.TrimSpace(r.FormValue("failure_rules_json")),
			HeartbeatProfileJSON:  strings.TrimSpace(r.FormValue("heartbeat_profile_json")),
			Notes:                 strings.TrimSpace(r.FormValue("notes")),
		}
		if err := validateTemplateVersionPayload(row); err != nil {
			s.renderTemplatesPage(w, r, "", err.Error(), nil)
			return
		}
		if err := s.Store.UpsertG2STemplateVersion(r.Context(), row); err != nil {
			s.renderTemplatesPage(w, r, "", err.Error(), nil)
			return
		}
		s.renderTemplatesPage(w, r, "Template version upserted: "+versionID, "", nil)
		return
	}

	if strings.HasSuffix(path, "/active-version") {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !s.authorizeMutation(w, r) {
			return
		}
		templateID := strings.TrimSuffix(path, "/active-version")
		templateID = strings.TrimSpace(strings.TrimSuffix(templateID, "/"))
		if templateID == "" || strings.Contains(templateID, "/") {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			s.renderTemplatesPage(w, r, "", "invalid form payload", nil)
			return
		}
		version, err := strconv.Atoi(strings.TrimSpace(r.FormValue("active_version")))
		if err != nil || version <= 0 {
			s.renderTemplatesPage(w, r, "", "active_version must be a positive integer", nil)
			return
		}
		if err := s.Store.SetActiveG2STemplateVersion(r.Context(), templateID, version); err != nil {
			s.renderTemplatesPage(w, r, "", err.Error(), nil)
			return
		}
		s.renderTemplatesPage(w, r, "Active template version set", "", nil)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authorizeMutation(w, r) {
		return
	}
	templateID := strings.TrimSpace(strings.TrimSuffix(path, "/"))
	if templateID == "" || strings.Contains(templateID, "/") {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderTemplatesPage(w, r, "", "invalid form payload", nil)
		return
	}
	templateRow := templates.G2STemplate{
		ID:                   templateID,
		Name:                 strings.TrimSpace(r.FormValue("name")),
		Vendor:               strings.TrimSpace(r.FormValue("vendor")),
		CabinetFamily:        strings.TrimSpace(r.FormValue("cabinet_family")),
		SoftwareVersionMatch: strings.TrimSpace(r.FormValue("software_version_match")),
		Status:               templates.TemplateStatus(strings.ToUpper(strings.TrimSpace(r.FormValue("status")))),
		Notes:                strings.TrimSpace(r.FormValue("notes")),
	}
	existing, err := s.Store.GetG2STemplate(r.Context(), templateID)
	if err != nil {
		s.renderTemplatesPage(w, r, "", err.Error(), nil)
		return
	}
	if existing != nil {
		templateRow.CurrentVersionID = existing.CurrentVersionID
	}
	if err := s.Store.UpsertG2STemplate(r.Context(), templateRow); err != nil {
		s.renderTemplatesPage(w, r, "", err.Error(), nil)
		return
	}
	s.renderTemplatesPage(w, r, "Template updated: "+templateID, "", nil)
}

type renderPreviewResult struct {
	MessageType string
	ContentType string
	Headers     map[string]string
	RawPayload  string
	SummaryJSON string
	Warnings    []string
}

type matchPreviewResult struct {
	Outcome   string
	RuleID    string
	RuleLabel string
	Reason    string
	Warnings  []string
}

type templatePreviewState struct {
	Render *renderPreviewResult
	Match  *matchPreviewResult
}

func (s *Server) renderTemplatePreview(ctx context.Context, r *http.Request) (*renderPreviewResult, error) {
	templateID := strings.TrimSpace(r.FormValue("template_id"))
	actionKey := strings.TrimSpace(r.FormValue("template_action_key"))
	if templateID == "" || actionKey == "" {
		return nil, fmt.Errorf("template_id and template_action_key are required")
	}
	templateRow, err := s.Store.GetG2STemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if templateRow == nil {
		return nil, fmt.Errorf("template not found")
	}
	versionRaw := strings.TrimSpace(r.FormValue("version"))
	var versionRow *templates.G2STemplateVersion
	if versionRaw == "" {
		versionRow, err = s.Store.GetActiveG2STemplateVersion(ctx, templateID)
	} else {
		value, parseErr := strconv.Atoi(versionRaw)
		if parseErr != nil {
			return nil, fmt.Errorf("version must be a number")
		}
		versionRow, err = s.Store.GetG2STemplateVersion(ctx, templateID, value)
	}
	if err != nil {
		return nil, err
	}
	if versionRow == nil {
		return nil, fmt.Errorf("template version not found")
	}

	doc, err := g2sengine.ParseActionTemplateDocument(versionRow.ActionsJSON)
	if err != nil {
		return nil, err
	}
	versionInt := parseVersionLabelInt(versionRow.VersionLabel)
	rendered, err := g2sengine.RenderActionMessage(doc, g2sengine.RenderRequest{
		TemplateID:        templateID,
		TemplateVersion:   versionInt,
		TemplateActionKey: actionKey,
		ActionID:          strings.TrimSpace(r.FormValue("action_id")),
		ActionRunID:       strings.TrimSpace(r.FormValue("action_run_id")),
		ActionStepID:      strings.TrimSpace(r.FormValue("action_step_id")),
		EGMID:             strings.TrimSpace(r.FormValue("egm_id")),
		HostID:            strings.TrimSpace(r.FormValue("host_id")),
		Timestamp:         time.Now().UTC(),
		IPAddress:         strings.TrimSpace(r.FormValue("ip_address")),
		EndpointPath:      strings.TrimSpace(r.FormValue("endpoint_path")),
	})
	if err != nil {
		return nil, err
	}
	return &renderPreviewResult{
		MessageType: rendered.MessageType,
		ContentType: rendered.ContentType,
		Headers:     rendered.Headers,
		RawPayload:  rendered.RawPayload,
		SummaryJSON: rendered.SummaryJSON,
		Warnings:    rendered.Warnings,
	}, nil
}

func (s *Server) matchTemplatePreview(ctx context.Context, r *http.Request) (*matchPreviewResult, error) {
	templateID := strings.TrimSpace(r.FormValue("template_id"))
	if templateID == "" {
		return nil, fmt.Errorf("template_id is required")
	}
	templateRow, err := s.Store.GetG2STemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if templateRow == nil {
		return nil, fmt.Errorf("template not found")
	}
	versionRaw := strings.TrimSpace(r.FormValue("version"))
	var versionRow *templates.G2STemplateVersion
	if versionRaw == "" {
		versionRow, err = s.Store.GetActiveG2STemplateVersion(ctx, templateID)
	} else {
		value, parseErr := strconv.Atoi(versionRaw)
		if parseErr != nil {
			return nil, fmt.Errorf("version must be a number")
		}
		versionRow, err = s.Store.GetG2STemplateVersion(ctx, templateID, value)
	}
	if err != nil {
		return nil, err
	}
	if versionRow == nil {
		return nil, fmt.Errorf("template version not found")
	}
	result, err := g2sengine.MatchMessage(
		strings.TrimSpace(r.FormValue("raw_payload")),
		strings.TrimSpace(r.FormValue("parsed_summary_json")),
		strings.TrimSpace(r.FormValue("message_type")),
		versionRow.ConfirmationRulesJSON,
		versionRow.FailureRulesJSON,
	)
	if err != nil {
		return nil, err
	}
	return &matchPreviewResult{
		Outcome:   result.Outcome,
		RuleID:    result.RuleID,
		RuleLabel: result.RuleLabel,
		Reason:    result.Reason,
		Warnings:  result.Warnings,
	}, nil
}

func (s *Server) renderTemplatesPage(w http.ResponseWriter, r *http.Request, message string, errText string, preview *templatePreviewState) {
	templateRows, err := s.Store.ListG2STemplates(r.Context())
	if err != nil {
		s.renderError(w, "/operator/templates", "Operator Console Templates", err)
		return
	}
	versionsByTemplate := map[string][]templates.G2STemplateVersion{}
	for _, tpl := range templateRows {
		rows, err := s.Store.ListG2STemplateVersions(r.Context(), tpl.ID)
		if err != nil {
			s.renderError(w, "/operator/templates", "Operator Console Templates", err)
			return
		}
		versionsByTemplate[tpl.ID] = rows
	}

	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Templates</h2><table>`)
	body.WriteString(`<tr><th>Template ID</th><th>Name</th><th>Vendor</th><th>Status</th><th>Active Version</th><th>Version Count</th><th>ActionsJSON</th><th>Expected Response Matcher</th><th>Failure Matcher</th><th>Action Keys</th><th>Edit</th></tr>`)
	for _, tpl := range templateRows {
		versionRows := versionsByTemplate[tpl.ID]
		versionLabels := make([]string, 0, len(versionRows))
		for _, version := range versionRows {
			versionLabels = append(versionLabels, version.VersionLabel)
		}
		activeVersion := findActiveTemplateVersion(tpl, versionRows)
		activeHasActions := false
		activeHasExpectedMatcher := false
		activeHasFailureMatcher := false
		activeActionKeys := []string{}
		if activeVersion != nil {
			activeHasActions = strings.TrimSpace(activeVersion.ActionsJSON) != ""
			activeHasExpectedMatcher = strings.TrimSpace(activeVersion.ConfirmationRulesJSON) != ""
			activeHasFailureMatcher = strings.TrimSpace(activeVersion.FailureRulesJSON) != ""
			keys, keyErr := actionKeysFromActionsJSON(activeVersion.ActionsJSON)
			if keyErr != nil {
				activeActionKeys = []string{"invalid_actions_json"}
			} else {
				activeActionKeys = keys
			}
		}

		body.WriteString(`<tr>`)
		body.WriteString(`<td class="mono">` + esc(tpl.ID) + `</td>`)
		body.WriteString(`<td>` + esc(tpl.Name) + `</td>`)
		body.WriteString(`<td>` + esc(tpl.Vendor) + `</td>`)
		body.WriteString(`<td>` + esc(string(tpl.Status)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(defaultString(tpl.CurrentVersionID, "-")) + `</td>`)
		body.WriteString(`<td>` + esc(strconv.Itoa(len(versionRows))) + `</td>`)
		body.WriteString(`<td>` + yesNo(activeHasActions) + `</td>`)
		body.WriteString(`<td>` + yesNo(activeHasExpectedMatcher) + `</td>`)
		body.WriteString(`<td>` + yesNo(activeHasFailureMatcher) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(defaultString(strings.Join(activeActionKeys, ", "), "-")) + `</td>`)
		body.WriteString(`<td><form class="inline-form" method="post" action="/operator/templates/` + esc(tpl.ID) + `">`)
		body.WriteString(`<label>Name <input type="text" name="name" value="` + esc(tpl.Name) + `" style="width:120px"></label>`)
		body.WriteString(`<label>Vendor <input type="text" name="vendor" value="` + esc(tpl.Vendor) + `" style="width:100px"></label>`)
		body.WriteString(`<label>Cabinet Family <input type="text" name="cabinet_family" value="` + esc(tpl.CabinetFamily) + `" style="width:100px"></label>`)
		body.WriteString(`<label>Software Match <input type="text" name="software_version_match" value="` + esc(tpl.SoftwareVersionMatch) + `" style="width:100px"></label>`)
		body.WriteString(`<label>Status <select name="status">` + templateStatusOptions(tpl.Status) + `</select></label>`)
		body.WriteString(`<label>Notes <input type="text" name="notes" value="` + esc(tpl.Notes) + `" style="width:120px"></label>`)
		body.WriteString(`<button type="submit">Save</button></form>`)
		body.WriteString(`<form class="inline-form" method="post" action="/operator/templates/` + esc(tpl.ID) + `/active-version">`)
		body.WriteString(`<label>Set Active Version <input type="number" name="active_version" min="1" style="width:70px"></label> <button type="submit">Set</button></form>`)
		if len(versionLabels) > 0 {
			body.WriteString(`<div>Versions: <span class="mono">` + esc(strings.Join(versionLabels, ", ")) + `</span></div>`)
		}
		body.WriteString(`<form class="inline-form" method="post" action="/operator/templates/` + esc(tpl.ID) + `/versions">`)
		body.WriteString(`<label>Version Number <input type="number" name="version_label" min="1" style="width:90px"></label>`)
		body.WriteString(`<label>Version ID <input type="text" name="version_id" style="width:150px"></label>`)
		body.WriteString(`<label>Endpoint Quirks JSON <input type="text" name="endpoint_quirks_json" style="width:180px"></label>`)
		body.WriteString(`<label>Heartbeat Profile JSON <input type="text" name="heartbeat_profile_json" style="width:180px"></label>`)
		body.WriteString(`<label>Notes <input type="text" name="notes" style="width:150px"></label><br>`)
		body.WriteString(`<label style="display:block;">Expected Response Matcher JSON <textarea name="confirmation_rules_json"></textarea></label>`)
		body.WriteString(`<label style="display:block;">Failure Matcher JSON <textarea name="failure_rules_json"></textarea></label>`)
		body.WriteString(`<label style="display:block;">ActionsJSON <textarea name="actions_json"></textarea></label>`)
		body.WriteString(`<button type="submit">Add Version</button></form></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h3>Add / Upsert Template</h3>`)
	body.WriteString(`<form method="post" action="/operator/templates">`)
	body.WriteString(`<label>Template ID <input type="text" name="id"></label>`)
	body.WriteString(`<label>Name <input type="text" name="name"></label>`)
	body.WriteString(`<label>Vendor <input type="text" name="vendor"></label>`)
	body.WriteString(`<label>Cabinet Family <input type="text" name="cabinet_family"></label>`)
	body.WriteString(`<label>Software Match <input type="text" name="software_version_match"></label>`)
	body.WriteString(`<label>Status <select name="status">` + templateStatusOptions("") + `</select></label>`)
	body.WriteString(`<label>Notes <input type="text" name="notes"></label>`)
	body.WriteString(`<button type="submit">Upsert Template</button></form></div>`)

	body.WriteString(`<div class="panel"><h3>Render Preview (No Send)</h3>`)
	body.WriteString(`<p>Supported variables: ` + esc(strings.Join(renderPreviewSupportedVariables, ", ")) + `.</p>`)
	body.WriteString(`<p>Use Action Keys to map trigger and return behavior for each action.</p>`)
	body.WriteString(`<form method="post" action="/operator/templates/render-preview">`)
	body.WriteString(`<label>Template ID <input type="text" name="template_id"></label>`)
	body.WriteString(`<label>Version (optional) <input type="number" name="version" style="width:70px"></label>`)
	body.WriteString(`<label>Template Action Key <input type="text" name="template_action_key"></label><br>`)
	body.WriteString(`<label>Action ID <input type="text" name="action_id"></label>`)
	body.WriteString(`<label>Action Run ID <input type="text" name="action_run_id"></label>`)
	body.WriteString(`<label>Action Step ID <input type="text" name="action_step_id"></label><br>`)
	body.WriteString(`<label>EGM ID <input type="text" name="egm_id"></label>`)
	body.WriteString(`<label>Host ID <input type="text" name="host_id"></label>`)
	body.WriteString(`<label>IP Address <input type="text" name="ip_address"></label>`)
	body.WriteString(`<label>Endpoint Path <input type="text" name="endpoint_path"></label>`)
	body.WriteString(`<button type="submit">Render Preview</button></form>`)
	if preview != nil && preview.Render != nil {
		renderPreview := preview.Render
		body.WriteString(`<h4>Preview Result</h4>`)
		body.WriteString(`<p>Message Type: <span class="mono">` + esc(renderPreview.MessageType) + `</span></p>`)
		body.WriteString(`<p>Content Type: <span class="mono">` + esc(renderPreview.ContentType) + `</span></p>`)
		if len(renderPreview.Headers) > 0 {
			headersJSON, _ := json.Marshal(renderPreview.Headers)
			body.WriteString(`<details><summary>Headers</summary><pre>` + esc(string(headersJSON)) + `</pre></details>`)
		}
		body.WriteString(`<pre>` + esc(renderPreview.RawPayload) + `</pre>`)
		if renderPreview.SummaryJSON != "" {
			body.WriteString(`<details><summary>Summary JSON</summary><pre>` + esc(renderPreview.SummaryJSON) + `</pre></details>`)
		}
		if len(renderPreview.Warnings) > 0 {
			body.WriteString(`<details><summary>Warnings</summary><pre>` + esc(strings.Join(renderPreview.Warnings, "\n")) + `</pre></details>`)
		}
	}
	body.WriteString(`</div>`)

	body.WriteString(`<div class="panel"><h3>Match Preview</h3>`)
	body.WriteString(`<form method="post" action="/operator/templates/match-preview">`)
	body.WriteString(`<label>Template ID <input type="text" name="template_id"></label>`)
	body.WriteString(`<label>Version (optional) <input type="number" name="version" style="width:70px"></label>`)
	body.WriteString(`<label>Message Type <input type="text" name="message_type"></label><br>`)
	body.WriteString(`<label style="display:block;">Message Payload <textarea name="raw_payload"></textarea></label>`)
	body.WriteString(`<label style="display:block;">Summary JSON <textarea name="parsed_summary_json"></textarea></label>`)
	body.WriteString(`<button type="submit">Preview Match</button></form>`)
	if preview != nil && preview.Match != nil {
		match := preview.Match
		body.WriteString(`<h4>Match Result</h4>`)
		body.WriteString(`<p>Outcome: <span class="mono">` + esc(defaultString(match.Outcome, string(g2sengine.MatchOutcomeNoMatch))) + `</span></p>`)
		body.WriteString(`<p>Rule: <span class="mono">` + esc(defaultString(strings.TrimSpace(match.RuleID+" "+match.RuleLabel), "-")) + `</span></p>`)
		body.WriteString(`<p>Reason: ` + esc(defaultString(match.Reason, "-")) + `</p>`)
		if len(match.Warnings) > 0 {
			body.WriteString(`<details><summary>Warnings</summary><pre>` + esc(strings.Join(match.Warnings, "\n")) + `</pre></details>`)
		}
	}
	body.WriteString(`</div>`)
	s.renderPage(w, operatorRoute("/templates"), "Operator Templates", body.String(), message, errText)
}

func (s *Server) handleComms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := s.Store.ListMessageJournalEntries(r.Context(), store.MessageJournalListQuery{Limit: queryLimit(r, 120)})
	if err != nil {
		s.renderError(w, "/operator/comms", "Operator Console Comms Journal", err)
		return
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("export")), "json") {
		s.writeCommsExport(w, rows)
		return
	}

	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Message Journal</h2><p><a href="/operator/comms/export">Export JSON</a></p><table>`)
	body.WriteString(`<tr><th>Timestamp</th><th>Direction</th><th>From</th><th>To</th><th>EGM ID</th><th>Action Run</th><th>Input Transition</th><th>Template</th><th>Message</th><th>Result</th><th>Match</th><th>Delivery</th><th>Error</th><th>Payload</th><th>Summary</th></tr>`)
	versionCache := map[string]*templates.G2STemplateVersion{}
	for _, row := range rows {
		templateRef := row.TemplateID
		if row.TemplateVersion != "" {
			templateRef = templateRef + "@" + row.TemplateVersion
		}
		payload := row.RawPayload
		if payload == "" {
			payload = "-"
		}
		body.WriteString(`<tr>`)
		body.WriteString(`<td>` + esc(fmtTime(row.Timestamp)) + `</td>`)
		body.WriteString(`<td>` + esc(string(row.Direction)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(defaultString(row.FromEndpoint, "-")) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(defaultString(row.ToEndpoint, "-")) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.EGMID) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.ActionRunID) + `</td>`)
		body.WriteString(`<td>` + esc(zeroDash64(row.InputTransitionID)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(templateRef) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.MessageType) + `</td>`)
		body.WriteString(`<td>` + esc(string(row.Result)) + `</td>`)
		body.WriteString(`<td>` + esc(s.commsMatcherOutcome(r.Context(), row, versionCache)) + `</td>`)
		body.WriteString(`<td>` + esc(deliverySummary(row)) + `</td>`)
		body.WriteString(`<td><details><summary>view</summary><pre>` + esc(defaultString(row.Error, "-")) + `</pre></details></td>`)
		body.WriteString(`<td><details><summary>view</summary><pre>` + esc(payload) + `</pre></details></td>`)
		body.WriteString(`<td><details><summary>view</summary><pre>` + esc(defaultString(row.ParsedSummaryJSON, "-")) + `</pre></details></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)
	s.renderPage(w, operatorRoute("/comms"), "Operator Message Journal", body.String(), "", "")
}

func (s *Server) handleCommsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := s.Store.ListMessageJournalEntries(r.Context(), store.MessageJournalListQuery{Limit: queryLimit(r, 500)})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.writeCommsExport(w, rows)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := s.Store.ListAuditTimelineEntries(r.Context(), store.AuditTimelineListQuery{Limit: queryLimit(r, 120)})
	if err != nil {
		s.renderError(w, "/operator/audit", "Operator Console Audit Timeline", err)
		return
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("export")), "json") {
		s.writeAuditExport(w, rows)
		return
	}
	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Audit Timeline</h2><p><a href="/operator/audit/export">Export JSON</a></p><table>`)
	body.WriteString(`<tr><th>Timestamp</th><th>Severity</th><th>Event</th><th>Summary</th><th>Related Input</th><th>Related Action</th><th>Related Message</th><th>Related EGM</th><th>Operator</th><th>Details</th></tr>`)
	for _, row := range rows {
		body.WriteString(`<tr>`)
		body.WriteString(`<td>` + esc(fmtTime(row.OccurredAt)) + `</td>`)
		body.WriteString(`<td>` + esc(string(row.Severity)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.EventType) + `</td>`)
		body.WriteString(`<td>` + esc(row.Summary) + `</td>`)
		body.WriteString(`<td>` + esc(zeroDash64(row.InputTransitionID)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.ActionRunID) + `</td>`)
		body.WriteString(`<td>` + esc(zeroDash64(row.MessageJournalID)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(auditRelatedEGMID(row)) + `</td>`)
		body.WriteString(`<td>` + esc(row.Operator) + `</td>`)
		body.WriteString(`<td><details><summary>view</summary><pre>` + esc(row.DetailJSON) + `</pre></details></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)
	s.renderPage(w, operatorRoute("/audit"), "Operator Audit Timeline", body.String(), "", "")
}

func (s *Server) handleAuditExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := s.Store.ListAuditTimelineEntries(r.Context(), store.AuditTimelineListQuery{Limit: queryLimit(r, 500)})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.writeAuditExport(w, rows)
}

type commsExportPayload struct {
	GeneratedAt time.Time                       `json:"generated_at"`
	Count       int                             `json:"count"`
	Rows        []g2sengine.MessageJournalEntry `json:"rows"`
}

func (s *Server) writeCommsExport(w http.ResponseWriter, rows []g2sengine.MessageJournalEntry) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="operator-comms.json"`)
	_ = json.NewEncoder(w).Encode(commsExportPayload{
		GeneratedAt: time.Now().UTC(),
		Count:       len(rows),
		Rows:        rows,
	})
}

type auditExportPayload struct {
	GeneratedAt time.Time                  `json:"generated_at"`
	Count       int                        `json:"count"`
	Rows        []audit.AuditTimelineEntry `json:"rows"`
}

func (s *Server) writeAuditExport(w http.ResponseWriter, rows []audit.AuditTimelineEntry) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="operator-audit.json"`)
	_ = json.NewEncoder(w).Encode(auditExportPayload{
		GeneratedAt: time.Now().UTC(),
		Count:       len(rows),
		Rows:        rows,
	})
}

func deliverySummary(row g2sengine.MessageJournalEntry) string {
	parts := []string{}
	if strings.TrimSpace(row.TransportMode) != "" {
		parts = append(parts, "transport="+row.TransportMode)
	}
	if row.HTTPStatusCode > 0 {
		parts = append(parts, "http="+strconv.Itoa(row.HTTPStatusCode))
	}
	if row.LatencyMS > 0 {
		parts = append(parts, "latency_ms="+strconv.Itoa(row.LatencyMS))
	}
	if row.SentAt != nil {
		parts = append(parts, "sent_at="+row.SentAt.UTC().Format(time.RFC3339))
	}
	if row.CompletedAt != nil {
		parts = append(parts, "completed_at="+row.CompletedAt.UTC().Format(time.RFC3339))
	}
	if strings.TrimSpace(row.ResponseExcerpt) != "" {
		parts = append(parts, "response_excerpt="+row.ResponseExcerpt)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "; ")
}

func auditRelatedEGMID(row audit.AuditTimelineEntry) string {
	return defaultString(extractJSONValue(row.DetailJSON, "egm_id", "egm", "target_egm_id"), "-")
}

func actionQueuedFromTransition(transition inputs.InputTransition, auditRow audit.AuditTimelineEntry) string {
	if strings.TrimSpace(transition.ActionRunID) != "" {
		return strings.TrimSpace(transition.ActionRunID)
	}
	return strings.TrimSpace(extractJSONValue(auditRow.DetailJSON, "action_queued_id"))
}

func (s *Server) commsMatcherOutcome(ctx context.Context, row g2sengine.MessageJournalEntry, cache map[string]*templates.G2STemplateVersion) string {
	templateID := strings.TrimSpace(row.TemplateID)
	if templateID == "" {
		return "-"
	}
	key := templateID + "@" + strings.TrimSpace(row.TemplateVersion)
	versionRow, ok := cache[key]
	if !ok {
		var err error
		if strings.TrimSpace(row.TemplateVersion) != "" {
			versionInt, parseErr := strconv.Atoi(strings.TrimSpace(row.TemplateVersion))
			if parseErr != nil {
				cache[key] = nil
				return "-"
			}
			versionRow, err = s.Store.GetG2STemplateVersion(ctx, templateID, versionInt)
		} else {
			versionRow, err = s.Store.GetActiveG2STemplateVersion(ctx, templateID)
		}
		if err != nil {
			cache[key] = nil
			return "-"
		}
		cache[key] = versionRow
	}
	if versionRow == nil {
		return "-"
	}
	result, err := g2sengine.MatchMessage(
		row.RawPayload,
		row.ParsedSummaryJSON,
		row.MessageType,
		versionRow.ConfirmationRulesJSON,
		versionRow.FailureRulesJSON,
	)
	if err != nil {
		return "-"
	}
	if strings.TrimSpace(result.RuleID) != "" {
		return result.Outcome + ":" + strings.TrimSpace(result.RuleID)
	}
	return result.Outcome
}

func extractJSONValue(detail string, keys ...string) string {
	trimmed := strings.TrimSpace(detail)
	if trimmed == "" {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return ""
	}
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	certs, err := s.Store.ListCertificateInventory(r.Context())
	if err != nil {
		s.renderError(w, "/operator/settings", "Operator Console Settings", err)
		return
	}
	body := strings.Builder{}

	operatorURL := "-"
	bindAddress := strings.TrimSpace(s.Options.BindAddress)
	if bindAddress != "" {
		scheme := "http"
		if s.Options.TLSRequired {
			scheme = "https"
		}
		operatorURL = fmt.Sprintf("%s://%s%s", scheme, bindAddress, operatorRouteBase)
	}
	listenerStatus := "not configured"
	if bindAddress != "" {
		listenerStatus = "configured"
	}
	g2sListener := "-"
	if strings.TrimSpace(s.Options.G2SHostURL) != "" {
		g2sListener = strings.TrimSpace(s.Options.G2SHostURL)
	}
	if strings.TrimSpace(s.Options.G2SEndpointPath) != "" && g2sListener == "-" {
		g2sListener = strings.TrimSpace(s.Options.G2SEndpointPath)
	}

	body.WriteString(`<div class="panel"><h2>Appliance</h2><table>`)
	body.WriteString(`<tr><th>Field</th><th>Value</th></tr>`)
	body.WriteString(`<tr><td>Controller</td><td class="mono">` + esc(defaultString(s.Options.ControllerID, "unknown")) + `</td></tr>`)
	body.WriteString(`<tr><td>Site</td><td>` + esc(defaultString(s.Options.SiteName, "unknown")) + `</td></tr>`)
	body.WriteString(`<tr><td>Application Version</td><td>` + esc(defaultString(s.Options.AppVersion, "unknown")) + `</td></tr>`)
	body.WriteString(`<tr><td>Config Path</td><td class="mono">` + esc(defaultString(s.Options.ConfigPath, "unknown")) + `</td></tr>`)
	body.WriteString(`<tr><td>Started At</td><td class="mono">` + esc(fmtTime(s.Options.StartedAt)) + `</td></tr>`)
	body.WriteString(`<tr><td>Current Time</td><td class="mono">` + esc(fmtTime(time.Now().UTC())) + `</td></tr>`)
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h2>Network Listener</h2><table>`)
	body.WriteString(`<tr><th>Field</th><th>Value</th></tr>`)
	body.WriteString(`<tr><td>HTTP Bind Address</td><td class="mono">` + esc(defaultString(bindAddress, "unknown")) + `</td></tr>`)
	body.WriteString(`<tr><td>Listener Status</td><td>` + esc(listenerStatus) + `</td></tr>`)
	body.WriteString(`<tr><td>Operator Console URL</td><td class="mono">` + esc(operatorURL) + `</td></tr>`)
	body.WriteString(`<tr><td>G2S Listener URL</td><td class="mono">` + esc(g2sListener) + `</td></tr>`)
	body.WriteString(`<tr><td>G2S Endpoint Path</td><td class="mono">` + esc(defaultString(s.Options.G2SEndpointPath, "unknown")) + `</td></tr>`)
	body.WriteString(`<tr><td>Host ID</td><td class="mono">` + esc(defaultString(s.Options.G2SHostID, "unknown")) + `</td></tr>`)
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h2>Certificates</h2>`)
	if len(certs) == 0 {
		body.WriteString(`<p>No certificate inventory recorded.</p>`)
	} else {
		body.WriteString(`<table><tr><th>Role</th><th>Path</th><th>Status</th><th>Subject</th><th>Issuer</th><th>Fingerprint</th><th>Not Before</th><th>Not After</th><th>Days Until Expiry</th><th>Last Checked</th><th>Runtime Note</th></tr>`)
		for _, row := range certs {
			body.WriteString(`<tr>`)
			body.WriteString(`<td class="mono">` + esc(row.Role) + `</td>`)
			body.WriteString(`<td class="mono">` + esc(defaultString(row.Path, "-")) + `</td>`)
			body.WriteString(`<td>` + esc(defaultString(row.Status, "-")) + `</td>`)
			body.WriteString(`<td class="mono">` + esc(defaultString(row.Subject, "-")) + `</td>`)
			body.WriteString(`<td class="mono">` + esc(defaultString(row.Issuer, "-")) + `</td>`)
			body.WriteString(`<td class="mono">` + esc(defaultString(row.SHA256Fingerprint, "-")) + `</td>`)
			body.WriteString(`<td class="mono">` + esc(fmtMaybeTime(row.NotBefore)) + `</td>`)
			body.WriteString(`<td class="mono">` + esc(fmtMaybeTime(row.NotAfter)) + `</td>`)
			body.WriteString(`<td>` + esc(daysUntilExpiryText(row.NotAfter)) + `</td>`)
			body.WriteString(`<td class="mono">` + esc(fmtTime(row.LastCheckedAt)) + `</td>`)
			body.WriteString(`<td>` + esc(defaultString(row.Error, "-")) + `</td>`)
			body.WriteString(`</tr>`)
		}
		body.WriteString(`</table>`)
	}
	body.WriteString(`</div>`)

	body.WriteString(`<div class="panel"><h2>Trust Material</h2><table>`)
	body.WriteString(`<tr><th>Field</th><th>Value</th></tr>`)
	body.WriteString(`<tr><td>Certificate Trust</td><td>` + esc(configuredText(s.Options.CAConfigured)) + `</td></tr>`)
	body.WriteString(`<tr><td>Client Certificate</td><td>` + esc(configuredText(s.Options.ClientCertConfigured)) + `</td></tr>`)
	body.WriteString(`<tr><td>Server Certificate</td><td>` + esc(configuredText(s.Options.ServerCertConfigured)) + `</td></tr>`)
	body.WriteString(`<tr><td>TLS</td><td>` + esc(enabledText(s.Options.TLSRequired)) + `</td></tr>`)
	body.WriteString(`<tr><td>Client Authentication</td><td>` + esc(enabledText(s.Options.ClientCertRequired)) + `</td></tr>`)
	body.WriteString(`<tr><td>Web Login</td><td>` + esc(enabledText(s.Options.WebLoginRequired)) + `</td></tr>`)
	body.WriteString(`<tr><td>Admin Client Certificate</td><td>` + esc(enabledText(s.Options.AdminClientCertRequired)) + `</td></tr>`)
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h2>Storage</h2><table>`)
	body.WriteString(`<tr><th>Field</th><th>Value</th></tr>`)
	body.WriteString(`<tr><td>Database</td><td class="mono">` + esc(defaultString(s.Options.DatabasePath, "unknown")) + `</td></tr>`)
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h2>Current Runtime Notes</h2><ul>`)
	body.WriteString(`<li>Settings view is read-only.</li>`)
	body.WriteString(`<li>Certificate material details are metadata-only; private keys are not displayed.</li>`)
	body.WriteString(`<li>Message delivery behavior is unchanged in this view.</li>`)
	body.WriteString(`</ul></div>`)

	s.renderPage(w, operatorRoute("/settings"), "Operator Settings", body.String(), "", "")
}

func (s *Server) renderError(w http.ResponseWriter, active string, title string, err error) {
	s.renderPage(w, active, title, "", "", err.Error())
}

func (s *Server) renderPage(w http.ResponseWriter, active string, title string, body string, message string, errText string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	htmlText := strings.Builder{}
	htmlText.WriteString(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>`)
	htmlText.WriteString(esc(title))
	htmlText.WriteString(`</title><link rel="stylesheet" href="` + operatorCSSRoute + `"></head><body>`)
	htmlText.WriteString(`<header><h1>Operator Console</h1><nav>`)
	htmlText.WriteString(navLink(operatorRoute(""), "Live", active))
	htmlText.WriteString(navLink(operatorRoute("/inputs"), "Inputs", active))
	htmlText.WriteString(navLink(operatorRoute("/actions"), "Actions", active))
	htmlText.WriteString(navLink(operatorRoute("/comms"), "Comms", active))
	htmlText.WriteString(navLink(operatorRoute("/egms"), "EGMs", active))
	htmlText.WriteString(navLink(operatorRoute("/templates"), "Templates", active))
	htmlText.WriteString(navLink(operatorRoute("/audit"), "Audit", active))
	htmlText.WriteString(navLink(operatorRoute("/settings"), "Settings", active))
	htmlText.WriteString(`</nav></header><main>`)
	if message != "" {
		htmlText.WriteString(`<div class="message">` + esc(message) + `</div>`)
	}
	if errText != "" {
		htmlText.WriteString(`<div class="error">` + esc(errText) + `</div>`)
	}
	htmlText.WriteString(body)
	htmlText.WriteString(`</main></body></html>`)
	_, _ = w.Write([]byte(htmlText.String()))
}

func (s *Server) authorizeMutation(w http.ResponseWriter, r *http.Request) bool {
	if s.AuthorizeMutation == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	tw := &trackingResponseWriter{ResponseWriter: w}
	if s.AuthorizeMutation(tw, r) {
		return true
	}
	if !tw.wroteHeader {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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

func navLink(path string, label string, active string) string {
	className := ""
	if path == active {
		className = ` class="active"`
	}
	return `<a href="` + esc(path) + `"` + className + `>` + esc(label) + `</a>`
}

func fmtTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}

func fmtMaybeTime(value *time.Time) string {
	if value == nil {
		return "-"
	}
	return fmtTime(*value)
}

func esc(value string) string {
	return html.EscapeString(value)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func checked(value bool) string {
	if value {
		return " checked"
	}
	return ""
}

func selected(current string, candidate string) string {
	if strings.EqualFold(strings.TrimSpace(current), strings.TrimSpace(candidate)) {
		return " selected"
	}
	return ""
}

func severityOptions(selectedSeverity actions.ActionSeverity) string {
	values := []actions.ActionSeverity{
		actions.SeverityNotice,
		actions.SeverityBroadcast,
		actions.SeverityEmergency,
		actions.SeverityRestore,
		actions.SeverityMaintenance,
	}
	builder := strings.Builder{}
	for _, value := range values {
		builder.WriteString(`<option value="` + esc(string(value)) + `"` + selected(string(selectedSeverity), string(value)) + `>` + esc(string(value)) + `</option>`)
	}
	return builder.String()
}

func templateStatusOptions(current templates.TemplateStatus) string {
	values := []templates.TemplateStatus{
		templates.TemplateStatusDraft,
		templates.TemplateStatusActive,
		templates.TemplateStatusArchived,
	}
	builder := strings.Builder{}
	for _, value := range values {
		builder.WriteString(`<option value="` + esc(string(value)) + `"` + selected(string(current), string(value)) + `>` + esc(string(value)) + `</option>`)
	}
	return builder.String()
}

const (
	targetSelectorTypeAllEmergencyEnabled = "all_emergency_enabled"
	targetSelectorTypeEGMIDs              = "egm_ids"
	targetSelectorTypeGroup               = "group"
	targetSelectorTypeTemplate            = "template"
	targetSelectorTypeZone                = "zone"
)

type retryPolicyConfig struct {
	Count   int `json:"count"`
	DelayMS int `json:"delay_ms"`
}

func (p retryPolicyConfig) toJSON() string {
	payload, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(payload)
}

type escalationPolicyConfig struct {
	ActionID      string `json:"escalation_action_id"`
	AfterAttempts int    `json:"after_attempts"`
}

func (p escalationPolicyConfig) toJSON() string {
	if strings.TrimSpace(p.ActionID) == "" && p.AfterAttempts == 0 {
		return ""
	}
	payload, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(payload)
}

func targetSelectorTypeOptions(current string) string {
	type option struct {
		Value string
		Label string
	}
	options := []option{
		{Value: targetSelectorTypeAllEmergencyEnabled, Label: "All emergency-enabled EGMs"},
		{Value: targetSelectorTypeEGMIDs, Label: "Explicit EGM IDs"},
		{Value: targetSelectorTypeGroup, Label: "Group"},
		{Value: targetSelectorTypeTemplate, Label: "Template"},
		{Value: targetSelectorTypeZone, Label: "Zone"},
	}
	builder := strings.Builder{}
	for _, option := range options {
		builder.WriteString(`<option value="` + esc(option.Value) + `"` + selected(current, option.Value) + `>` + esc(option.Label) + `</option>`)
	}
	return builder.String()
}

func parseTargetSelectorForm(selectorType string, selectorValue string) (string, error) {
	value := strings.TrimSpace(selectorValue)
	switch strings.ToLower(strings.TrimSpace(selectorType)) {
	case targetSelectorTypeAllEmergencyEnabled:
		return actionplanner.SelectorAllEmergencyEnabled, nil
	case targetSelectorTypeEGMIDs:
		if value == "" {
			return "", fmt.Errorf("target selector value is required for explicit EGM IDs")
		}
		return actionplanner.SelectorEGMIDsPrefix + value, nil
	case targetSelectorTypeGroup:
		if value == "" {
			return "", fmt.Errorf("target selector value is required for group selector")
		}
		return actionplanner.SelectorGroupPrefix + value, nil
	case targetSelectorTypeTemplate:
		if value == "" {
			return "", fmt.Errorf("target selector value is required for template selector")
		}
		return actionplanner.SelectorTemplatePrefix + value, nil
	case targetSelectorTypeZone:
		if value == "" {
			return "", fmt.Errorf("target selector value is required for zone selector")
		}
		return actionplanner.SelectorZonePrefix + value, nil
	default:
		return "", fmt.Errorf("invalid target selector type")
	}
}

func validateActionTargetSelector(selector string) error {
	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return fmt.Errorf("target selector is required")
	}
	if trimmed == actionplanner.SelectorAllEmergencyEnabled {
		return nil
	}
	switch {
	case strings.HasPrefix(trimmed, actionplanner.SelectorEGMIDsPrefix):
		if strings.TrimSpace(strings.TrimPrefix(trimmed, actionplanner.SelectorEGMIDsPrefix)) == "" {
			return fmt.Errorf("target selector EGM IDs value is required")
		}
	case strings.HasPrefix(trimmed, actionplanner.SelectorGroupPrefix):
		if strings.TrimSpace(strings.TrimPrefix(trimmed, actionplanner.SelectorGroupPrefix)) == "" {
			return fmt.Errorf("target selector group value is required")
		}
	case strings.HasPrefix(trimmed, actionplanner.SelectorTemplatePrefix):
		if strings.TrimSpace(strings.TrimPrefix(trimmed, actionplanner.SelectorTemplatePrefix)) == "" {
			return fmt.Errorf("target selector template value is required")
		}
	case strings.HasPrefix(trimmed, actionplanner.SelectorZonePrefix):
		if strings.TrimSpace(strings.TrimPrefix(trimmed, actionplanner.SelectorZonePrefix)) == "" {
			return fmt.Errorf("target selector zone value is required")
		}
	default:
		return fmt.Errorf("invalid target selector")
	}
	return nil
}

func selectorTypeValueAndReadable(selector string) (string, string, string) {
	trimmed := strings.TrimSpace(selector)
	if trimmed == actionplanner.SelectorAllEmergencyEnabled {
		return targetSelectorTypeAllEmergencyEnabled, "", "All emergency-enabled EGMs"
	}
	if strings.HasPrefix(trimmed, actionplanner.SelectorEGMIDsPrefix) {
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, actionplanner.SelectorEGMIDsPrefix))
		return targetSelectorTypeEGMIDs, value, "Explicit EGM IDs: " + value
	}
	if strings.HasPrefix(trimmed, actionplanner.SelectorGroupPrefix) {
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, actionplanner.SelectorGroupPrefix))
		return targetSelectorTypeGroup, value, "Group: " + value
	}
	if strings.HasPrefix(trimmed, actionplanner.SelectorTemplatePrefix) {
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, actionplanner.SelectorTemplatePrefix))
		return targetSelectorTypeTemplate, value, "Template: " + value
	}
	if strings.HasPrefix(trimmed, actionplanner.SelectorZonePrefix) {
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, actionplanner.SelectorZonePrefix))
		return targetSelectorTypeZone, value, "Zone: " + value
	}
	return targetSelectorTypeAllEmergencyEnabled, "", "Unrecognized selector: " + trimmed
}

func parseRetryPolicyForm(countRaw string, delayRaw string) (retryPolicyConfig, error) {
	count, err := parseNonNegativeIntField(countRaw, "retry count")
	if err != nil {
		return retryPolicyConfig{}, err
	}
	delay, err := parseNonNegativeIntField(delayRaw, "retry delay milliseconds")
	if err != nil {
		return retryPolicyConfig{}, err
	}
	return retryPolicyConfig{Count: count, DelayMS: delay}, nil
}

func parseEscalationPolicyForm(actionIDRaw string, attemptsRaw string) (escalationPolicyConfig, error) {
	actionID := strings.TrimSpace(actionIDRaw)
	attempts, err := parseNonNegativeIntField(attemptsRaw, "escalation after attempts")
	if err != nil {
		return escalationPolicyConfig{}, err
	}
	if actionID == "" && attempts > 0 {
		return escalationPolicyConfig{}, fmt.Errorf("escalation action ID is required when escalation attempts are set")
	}
	if actionID != "" && attempts <= 0 {
		return escalationPolicyConfig{}, fmt.Errorf("escalation after attempts must be greater than zero when escalation action ID is set")
	}
	return escalationPolicyConfig{
		ActionID:      actionID,
		AfterAttempts: attempts,
	}, nil
}

func parseNonNegativeIntField(raw string, fieldLabel string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", fieldLabel)
	}
	if value < 0 {
		return 0, fmt.Errorf("invalid %s", fieldLabel)
	}
	return value, nil
}

func parseRetryPolicyJSON(raw string) retryPolicyConfig {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return retryPolicyConfig{}
	}
	var policy retryPolicyConfig
	if err := json.Unmarshal([]byte(trimmed), &policy); err != nil {
		return retryPolicyConfig{}
	}
	if policy.Count < 0 || policy.DelayMS < 0 {
		return retryPolicyConfig{}
	}
	return policy
}

func parseEscalationPolicyJSON(raw string) escalationPolicyConfig {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return escalationPolicyConfig{}
	}
	var policy escalationPolicyConfig
	if err := json.Unmarshal([]byte(trimmed), &policy); err != nil {
		return escalationPolicyConfig{}
	}
	if policy.AfterAttempts < 0 {
		return escalationPolicyConfig{}
	}
	policy.ActionID = strings.TrimSpace(policy.ActionID)
	return policy
}

func parseInputStateField(raw string) (inputs.InputElectricalState, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	switch value {
	case string(inputs.InputStateHigh):
		return inputs.InputStateHigh, nil
	case string(inputs.InputStateLow):
		return inputs.InputStateLow, nil
	default:
		return "", fmt.Errorf("normal state must be HIGH or LOW")
	}
}

func parseLatchingModeField(raw string) (inputs.InputLatchingMode, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	switch value {
	case string(inputs.LatchingAutoClear):
		return inputs.LatchingAutoClear, nil
	case string(inputs.LatchingManualClear):
		return inputs.LatchingManualClear, nil
	default:
		return "", fmt.Errorf("latch mode must be AUTO_CLEAR or MANUAL_CLEAR")
	}
}

func validateTemplateVersionPayload(row templates.G2STemplateVersion) error {
	if err := validateOptionalJSONField("actions_json", row.ActionsJSON, true); err != nil {
		return err
	}
	if _, err := g2sengine.ParseActionTemplateDocument(strings.TrimSpace(row.ActionsJSON)); err != nil {
		return err
	}
	if err := validateOptionalJSONField("expected response matcher JSON", row.ConfirmationRulesJSON, false); err != nil {
		return err
	}
	if err := validateOptionalJSONField("failure matcher JSON", row.FailureRulesJSON, false); err != nil {
		return err
	}
	if err := validateOptionalJSONField("endpoint quirks JSON", row.EndpointQuirksJSON, false); err != nil {
		return err
	}
	if err := validateOptionalJSONField("heartbeat profile JSON", row.HeartbeatProfileJSON, false); err != nil {
		return err
	}
	return nil
}

func validateOptionalJSONField(label string, raw string, required bool) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		if required {
			return fmt.Errorf("%s is required", label)
		}
		return nil
	}
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return fmt.Errorf("invalid %s", label)
	}
	return nil
}

func findActiveTemplateVersion(tpl templates.G2STemplate, rows []templates.G2STemplateVersion) *templates.G2STemplateVersion {
	active := strings.TrimSpace(tpl.CurrentVersionID)
	if active == "" {
		return nil
	}
	for i := range rows {
		if strings.EqualFold(strings.TrimSpace(rows[i].VersionLabel), active) || strings.EqualFold(strings.TrimSpace(rows[i].ID), active) {
			return &rows[i]
		}
	}
	return nil
}

func actionKeysFromActionsJSON(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}, nil
	}
	doc, err := g2sengine.ParseActionTemplateDocument(trimmed)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(doc.Actions))
	for key := range doc.Actions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

func parseFormBool(raw string) bool {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

func parseInputState(raw string, fallback inputs.InputElectricalState) inputs.InputElectricalState {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == string(inputs.InputStateLow) {
		return inputs.InputStateLow
	}
	if value == string(inputs.InputStateHigh) {
		return inputs.InputStateHigh
	}
	return fallback
}

func parseLatchingMode(raw string, fallback inputs.InputLatchingMode) inputs.InputLatchingMode {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == string(inputs.LatchingAutoClear) {
		return inputs.LatchingAutoClear
	}
	if value == string(inputs.LatchingManualClear) {
		return inputs.LatchingManualClear
	}
	return fallback
}

func parseIntOrDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

func queryLimit(r *http.Request, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func parseVersionLabelInt(label string) int {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		return 0
	}
	value, err := strconv.Atoi(trimmed)
	if err == nil {
		return value
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "v") {
		value, err = strconv.Atoi(strings.TrimPrefix(strings.ToLower(trimmed), "v"))
		if err == nil {
			return value
		}
	}
	return 0
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func enabledText(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

func configuredText(value bool) string {
	if value {
		return "configured"
	}
	return "not configured"
}

func daysUntilExpiryText(notAfter *time.Time) string {
	if notAfter == nil {
		return "-"
	}
	now := time.Now().UTC()
	if notAfter.Before(now) {
		return "expired"
	}
	days := int(notAfter.Sub(now).Hours() / 24)
	return strconv.Itoa(days)
}

func zeroDash(value int) string {
	if value <= 0 {
		return "-"
	}
	return strconv.Itoa(value)
}

func zeroDash64(value int64) string {
	if value <= 0 {
		return "-"
	}
	return strconv.FormatInt(value, 10)
}
