package fieldtestui

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

type Store interface {
	GetInputChannel(ctx context.Context, id string) (*inputs.InputChannel, error)
	UpsertInputChannel(ctx context.Context, channel inputs.InputChannel) error
	ListInputChannels(ctx context.Context) ([]inputs.InputChannel, error)
	GetInputRuntimeState(ctx context.Context, inputID string) (*inputruntime.InputRuntimeState, error)
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
	DatabasePath            string
	BindAddress             string
	TransportGateSummary    string
	CapturePolicySummary    string
	RealSendDefaultDisabled bool
}

type Server struct {
	Store             Store
	Options           Options
	AuthorizeMutation func(http.ResponseWriter, *http.Request) bool
}

func NewServer(store Store, options Options, authorizeMutation func(http.ResponseWriter, *http.Request) bool) *Server {
	if strings.TrimSpace(options.TransportGateSummary) == "" {
		options.TransportGateSummary = "Real send is blocked unless transport=http, allow_real_send=true, capture_only_send=true, and endpoint is localhost/loopback."
	}
	if strings.TrimSpace(options.CapturePolicySummary) == "" {
		options.CapturePolicySummary = "Phase 2G capture proof only: localhost/127.0.0.1/::1 capture endpoints."
	}
	if !options.RealSendDefaultDisabled {
		options.RealSendDefaultDisabled = true
	}
	return &Server{
		Store:             store,
		Options:           options,
		AuthorizeMutation: authorizeMutation,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/field-test", s.handleHome)
	mux.HandleFunc("/field-test/inputs", s.handleInputs)
	mux.HandleFunc("/field-test/inputs/", s.handleInputByID)
	mux.HandleFunc("/field-test/actions", s.handleActions)
	mux.HandleFunc("/field-test/actions/", s.handleActionByID)
	mux.HandleFunc("/field-test/egms", s.handleEGMs)
	mux.HandleFunc("/field-test/egms/", s.handleEGMByID)
	mux.HandleFunc("/field-test/templates", s.handleTemplates)
	mux.HandleFunc("/field-test/templates/", s.handleTemplateByID)
	mux.HandleFunc("/field-test/comms", s.handleComms)
	mux.HandleFunc("/field-test/audit", s.handleAudit)
	mux.HandleFunc("/field-test/settings", s.handleSettings)
	mux.HandleFunc("/field-test/static/field-test.css", s.handleStyles)
}

func (s *Server) handleStyles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write([]byte(fieldTestCSS))
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	channels, err := s.Store.ListInputChannels(r.Context())
	if err != nil {
		s.renderError(w, "/field-test", "Field-Test Shell", err)
		return
	}
	triggered := 0
	latches := []string{}
	for _, channel := range channels {
		runtimeState, err := s.Store.GetInputRuntimeState(r.Context(), channel.ID)
		if err != nil {
			s.renderError(w, "/field-test", "Field-Test Shell", err)
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
		s.renderError(w, "/field-test", "Field-Test Shell", err)
		return
	}
	messages, err := s.Store.ListMessageJournalEntries(r.Context(), store.MessageJournalListQuery{Limit: 8})
	if err != nil {
		s.renderError(w, "/field-test", "Field-Test Shell", err)
		return
	}

	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Safety Gate Summary</h2>`)
	body.WriteString(`<p><span class="badge">REAL SEND IS GATED / DISABLED</span></p>`)
	body.WriteString(`<p>` + esc(s.Options.TransportGateSummary) + `</p>`)
	body.WriteString(`<p>` + esc(s.Options.CapturePolicySummary) + `</p>`)
	body.WriteString(`</div>`)

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

	s.renderPage(w, "/field-test", "Field-Test Configuration Shell", body.String(), "", "")
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
	inputID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/field-test/inputs/"))
	if inputID == "" || strings.Contains(inputID, "/") {
		http.NotFound(w, r)
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
	updated := *existing
	updated.NormalState = parseInputState(r.FormValue("normal_state"), updated.NormalState)
	updated.DebounceMS = parseIntOrDefault(r.FormValue("debounce_ms"), updated.DebounceMS)
	updated.LatchingMode = parseLatchingMode(r.FormValue("latching_mode"), updated.LatchingMode)
	updated.Priority = parseIntOrDefault(r.FormValue("priority"), updated.Priority)
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
		s.renderError(w, "/field-test/inputs", "Field-Test Inputs", err)
		return
	}
	transitions, err := s.Store.ListInputTransitions(r.Context(), 200)
	if err != nil {
		s.renderError(w, "/field-test/inputs", "Field-Test Inputs", err)
		return
	}
	lastTransitionByInput := map[string]inputs.InputTransition{}
	for _, transition := range transitions {
		if _, exists := lastTransitionByInput[transition.InputChannelID]; !exists {
			lastTransitionByInput[transition.InputChannelID] = transition
		}
	}

	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Configured Inputs</h2><table>`)
	body.WriteString(`<tr><th>ID</th><th>Name</th><th>GPIO</th><th>Raw</th><th>Derived</th><th>Latch Mode</th><th>Latch Active</th><th>Normal</th><th>Debounce</th><th>Priority</th><th>On Trigger</th><th>On Normal</th><th>Last Transition</th><th>Edit</th></tr>`)
	for _, channel := range channels {
		runtimeState, err := s.Store.GetInputRuntimeState(r.Context(), channel.ID)
		if err != nil {
			s.renderError(w, "/field-test/inputs", "Field-Test Inputs", err)
			return
		}
		lastTransition := "-"
		if transition, ok := lastTransitionByInput[channel.ID]; ok {
			lastTransition = fmt.Sprintf("%s %s→%s", fmtTime(transition.TransitionAt), transition.PreviousDerived, transition.NewDerived)
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
		body.WriteString(`<td><form class="inline-form" method="post" action="/field-test/inputs/` + esc(channel.ID) + `">`)
		body.WriteString(`normal <select name="normal_state"><option value="HIGH"` + selected(string(channel.NormalState), string(inputs.InputStateHigh)) + `>HIGH</option><option value="LOW"` + selected(string(channel.NormalState), string(inputs.InputStateLow)) + `>LOW</option></select> `)
		body.WriteString(`debounce <input type="number" name="debounce_ms" value="` + esc(strconv.Itoa(channel.DebounceMS)) + `" style="width:76px"> `)
		body.WriteString(`latch <select name="latching_mode"><option value="AUTO_CLEAR"` + selected(string(channel.LatchingMode), string(inputs.LatchingAutoClear)) + `>AUTO_CLEAR</option><option value="MANUAL_CLEAR"` + selected(string(channel.LatchingMode), string(inputs.LatchingManualClear)) + `>MANUAL_CLEAR</option></select> `)
		body.WriteString(`priority <input type="number" name="priority" value="` + esc(strconv.Itoa(channel.Priority)) + `" style="width:66px"> `)
		body.WriteString(`<br>on-trigger <input type="text" name="on_trigger_action_id" value="` + esc(channel.OnTriggerActionID) + `" style="width:180px"> `)
		body.WriteString(`on-normal <input type="text" name="on_normal_action_id" value="` + esc(channel.OnNormalActionID) + `" style="width:180px"> `)
		body.WriteString(`<button type="submit">Save</button></form></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)
	s.renderPage(w, "/field-test/inputs", "Field-Test Inputs", body.String(), message, errText)
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
	actionID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/field-test/actions/"))
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
	stepKey := strings.TrimSpace(r.FormValue("step_template_action_key"))
	if stepKey == "" && existing != nil && len(existing.Steps) > 0 {
		stepKey = existing.Steps[0].TemplateActionKey
	}
	if stepKey == "" {
		stepKey = "queue_only_no_send"
	}

	definition := actions.ActionDefinition{
		ID:               actionID,
		Name:             strings.TrimSpace(r.FormValue("name")),
		Severity:         actions.ActionSeverity(strings.ToUpper(strings.TrimSpace(r.FormValue("severity")))),
		Enabled:          enabled,
		TargetSelector:   strings.TrimSpace(r.FormValue("target_selector")),
		TemplateSelector: strings.TrimSpace(r.FormValue("template_selector")),
		Steps: []actions.ActionStep{{
			ID:                "step-1",
			Name:              "Step 1",
			Sequence:          0,
			TemplateActionKey: stepKey,
		}},
		RetryPolicyJSON: strings.TrimSpace(r.FormValue("retry_policy_json")),
		EscalationJSON:  strings.TrimSpace(r.FormValue("escalation_policy_json")),
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
		if definition.RetryPolicyJSON == "" {
			definition.RetryPolicyJSON = existing.RetryPolicyJSON
		}
		if definition.EscalationJSON == "" {
			definition.EscalationJSON = existing.EscalationJSON
		}
		if definition.ReturnActionID == "" {
			definition.ReturnActionID = existing.ReturnActionID
		}
	}
	return s.Store.UpsertActionDefinition(ctx, definition)
}

func (s *Server) renderActionsPage(w http.ResponseWriter, r *http.Request, message string, errText string) {
	definitions, err := s.Store.ListActionDefinitions(r.Context())
	if err != nil {
		s.renderError(w, "/field-test/actions", "Field-Test Actions", err)
		return
	}
	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Action Definitions</h2><table>`)
	body.WriteString(`<tr><th>ID</th><th>Name</th><th>Severity</th><th>Enabled</th><th>Target Selector</th><th>Template Selector</th><th>Step Keys</th><th>Return Action</th><th>Retry</th><th>Escalation</th><th>Preview</th><th>Edit</th></tr>`)
	for _, definition := range definitions {
		stepKeys := []string{}
		for _, step := range definition.Steps {
			stepKeys = append(stepKeys, step.TemplateActionKey)
		}
		firstStep := ""
		if len(stepKeys) > 0 {
			firstStep = stepKeys[0]
		}
		body.WriteString(`<tr>`)
		body.WriteString(`<td class="mono">` + esc(definition.ID) + `</td>`)
		body.WriteString(`<td>` + esc(definition.Name) + `</td>`)
		body.WriteString(`<td>` + esc(string(definition.Severity)) + `</td>`)
		body.WriteString(`<td>` + yesNo(definition.Enabled) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(definition.TargetSelector) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(definition.TemplateSelector) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(strings.Join(stepKeys, ", ")) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(definition.ReturnActionID) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(definition.RetryPolicyJSON) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(definition.EscalationJSON) + `</td>`)
		body.WriteString(`<td><a href="/api/v2/actions/` + esc(definition.ID) + `/preview" target="_blank" rel="noreferrer">Preview Targets</a></td>`)
		body.WriteString(`<td><form class="inline-form" method="post" action="/field-test/actions/` + esc(definition.ID) + `">`)
		body.WriteString(`<input type="hidden" name="id" value="` + esc(definition.ID) + `">`)
		body.WriteString(`name <input type="text" name="name" value="` + esc(definition.Name) + `" style="width:140px"> `)
		body.WriteString(`severity <select name="severity">` + severityOptions(definition.Severity) + `</select> `)
		body.WriteString(`enabled <input type="checkbox" name="enabled" value="true"` + checked(definition.Enabled) + `> `)
		body.WriteString(`<br>target <input type="text" name="target_selector" value="` + esc(definition.TargetSelector) + `" style="width:220px"> `)
		body.WriteString(`template <input type="text" name="template_selector" value="` + esc(definition.TemplateSelector) + `" style="width:150px"> `)
		body.WriteString(`<br>step key <input type="text" name="step_template_action_key" value="` + esc(firstStep) + `" style="width:180px"> `)
		body.WriteString(`return <input type="text" name="return_action_id" value="` + esc(definition.ReturnActionID) + `" style="width:180px"> `)
		body.WriteString(`<button type="submit">Save</button></form></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h3>Add / Upsert Action</h3>`)
	body.WriteString(`<form method="post" action="/field-test/actions">`)
	body.WriteString(`<label>ID <input type="text" name="id"></label>`)
	body.WriteString(`<label>Name <input type="text" name="name"></label>`)
	body.WriteString(`<label>Severity <select name="severity">` + severityOptions("") + `</select></label>`)
	body.WriteString(`<label>Enabled <input type="checkbox" name="enabled" value="true" checked></label><br>`)
	body.WriteString(`<label>Target Selector <input type="text" name="target_selector" value="ALL_EMERGENCY_ENABLED" style="width:260px"></label>`)
	body.WriteString(`<label>Template Selector <input type="text" name="template_selector" value="template-by-egm" style="width:200px"></label><br>`)
	body.WriteString(`<label>Step Template Action Key <input type="text" name="step_template_action_key" value="queue_only_no_send" style="width:220px"></label>`)
	body.WriteString(`<label>Return Action ID <input type="text" name="return_action_id" style="width:220px"></label><br>`)
	body.WriteString(`<button type="submit">Upsert Action</button></form></div>`)

	s.renderPage(w, "/field-test/actions", "Field-Test Actions", body.String(), message, errText)
}

func (s *Server) handleEGMs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderEGMsPage(w, r, "", "")
	case http.MethodPost:
		if !s.authorizeMutation(w, r) {
			return
		}
		if err := r.ParseForm(); err != nil {
			s.renderEGMsPage(w, r, "", "invalid form payload")
			return
		}
		egmID := strings.TrimSpace(r.FormValue("egm_id"))
		if egmID == "" {
			s.renderEGMsPage(w, r, "", "egm_id is required")
			return
		}
		if err := s.upsertEGMFromForm(r.Context(), egmID, r); err != nil {
			s.renderEGMsPage(w, r, "", err.Error())
			return
		}
		s.renderEGMsPage(w, r, "EGM upserted: "+egmID, "")
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
	egmID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/field-test/egms/"))
	if egmID == "" || strings.Contains(egmID, "/") {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderEGMsPage(w, r, "", "invalid form payload")
		return
	}
	if err := s.upsertEGMFromForm(r.Context(), egmID, r); err != nil {
		s.renderEGMsPage(w, r, "", err.Error())
		return
	}
	s.renderEGMsPage(w, r, "EGM updated: "+egmID, "")
}

func (s *Server) upsertEGMFromForm(ctx context.Context, egmID string, r *http.Request) error {
	existing, err := s.Store.GetEGMRecord(ctx, egmID)
	if err != nil {
		return err
	}
	state := egms.EGMActionStateNormal
	if existing != nil && existing.CurrentActionState != "" {
		state = existing.CurrentActionState
	}
	record := egms.EGMRecord{
		EGMID:              egmID,
		DisplayName:        strings.TrimSpace(r.FormValue("display_name")),
		IPAddress:          strings.TrimSpace(r.FormValue("ip_address")),
		EndpointPath:       strings.TrimSpace(r.FormValue("endpoint_path")),
		Vendor:             strings.TrimSpace(r.FormValue("vendor")),
		Zone:               strings.TrimSpace(r.FormValue("zone")),
		Enabled:            parseFormBool(r.FormValue("enabled")),
		EmergencyEnabled:   parseFormBool(r.FormValue("emergency_enabled")),
		TemplateID:         strings.TrimSpace(r.FormValue("template_id")),
		CurrentActionState: state,
	}
	if existing != nil {
		record.CabinetFamily = existing.CabinetFamily
		record.GameTitle = existing.GameTitle
		record.SoftwareVersion = existing.SoftwareVersion
		record.HeartbeatOverrideJSON = existing.HeartbeatOverrideJSON
		record.LastSeenAt = existing.LastSeenAt
		record.Notes = existing.Notes
	}
	return s.Store.UpsertEGMRecord(ctx, record)
}

func (s *Server) renderEGMsPage(w http.ResponseWriter, r *http.Request, message string, errText string) {
	records, err := s.Store.ListEGMRecords(r.Context())
	if err != nil {
		s.renderError(w, "/field-test/egms", "Field-Test EGMs", err)
		return
	}
	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>EGM Registry</h2><table>`)
	body.WriteString(`<tr><th>EGM ID</th><th>Name</th><th>IP</th><th>Endpoint</th><th>Vendor</th><th>Zone</th><th>Enabled</th><th>Emergency Enabled</th><th>Template ID</th><th>Current State</th><th>Edit</th></tr>`)
	for _, record := range records {
		body.WriteString(`<tr>`)
		body.WriteString(`<td class="mono">` + esc(record.EGMID) + `</td>`)
		body.WriteString(`<td>` + esc(record.DisplayName) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(record.IPAddress) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(record.EndpointPath) + `</td>`)
		body.WriteString(`<td>` + esc(record.Vendor) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(record.Zone) + `</td>`)
		body.WriteString(`<td>` + yesNo(record.Enabled) + `</td>`)
		body.WriteString(`<td>` + yesNo(record.EmergencyEnabled) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(record.TemplateID) + `</td>`)
		body.WriteString(`<td>` + esc(string(record.CurrentActionState)) + `</td>`)
		body.WriteString(`<td><form class="inline-form" method="post" action="/field-test/egms/` + esc(record.EGMID) + `">`)
		body.WriteString(`<label>Name <input type="text" name="display_name" value="` + esc(record.DisplayName) + `" style="width:130px"></label>`)
		body.WriteString(`<label>IP <input type="text" name="ip_address" value="` + esc(record.IPAddress) + `" style="width:110px"></label>`)
		body.WriteString(`<label>Endpoint <input type="text" name="endpoint_path" value="` + esc(record.EndpointPath) + `" style="width:110px"></label><br>`)
		body.WriteString(`<label>Vendor <input type="text" name="vendor" value="` + esc(record.Vendor) + `" style="width:120px"></label>`)
		body.WriteString(`<label>Zone <input type="text" name="zone" value="` + esc(record.Zone) + `" style="width:90px"></label>`)
		body.WriteString(`<label>Enabled <input type="checkbox" name="enabled" value="true"` + checked(record.Enabled) + `></label>`)
		body.WriteString(`<label>Emergency <input type="checkbox" name="emergency_enabled" value="true"` + checked(record.EmergencyEnabled) + `></label>`)
		body.WriteString(`<label>Template <input type="text" name="template_id" value="` + esc(record.TemplateID) + `" style="width:140px"></label>`)
		body.WriteString(`<button type="submit">Save</button></form></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h3>Add / Upsert EGM</h3>`)
	body.WriteString(`<form method="post" action="/field-test/egms">`)
	body.WriteString(`<label>EGM ID <input type="text" name="egm_id"></label>`)
	body.WriteString(`<label>Name <input type="text" name="display_name"></label>`)
	body.WriteString(`<label>IP <input type="text" name="ip_address"></label>`)
	body.WriteString(`<label>Endpoint <input type="text" name="endpoint_path"></label><br>`)
	body.WriteString(`<label>Vendor <input type="text" name="vendor"></label>`)
	body.WriteString(`<label>Zone <input type="text" name="zone"></label>`)
	body.WriteString(`<label>Enabled <input type="checkbox" name="enabled" value="true" checked></label>`)
	body.WriteString(`<label>Emergency Enabled <input type="checkbox" name="emergency_enabled" value="true" checked></label>`)
	body.WriteString(`<label>Template ID <input type="text" name="template_id"></label>`)
	body.WriteString(`<button type="submit">Upsert EGM</button></form></div>`)

	s.renderPage(w, "/field-test/egms", "Field-Test EGM Registry", body.String(), message, errText)
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
			ID:     templateID,
			Name:   strings.TrimSpace(r.FormValue("name")),
			Vendor: strings.TrimSpace(r.FormValue("vendor")),
			Status: templates.TemplateStatus(strings.ToUpper(strings.TrimSpace(r.FormValue("status")))),
			Notes:  strings.TrimSpace(r.FormValue("notes")),
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
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/field-test/templates/"))
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
		s.renderTemplatesPage(w, r, "Render preview complete", "", preview)
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
		versionID := strings.TrimSpace(r.FormValue("version_id"))
		if versionID == "" {
			versionID = fmt.Sprintf("%s-v%s", templateID, sanitizeVersionLabel(versionLabel))
		}
		row := templates.G2STemplateVersion{
			ID:           versionID,
			TemplateID:   templateID,
			VersionLabel: versionLabel,
			ActionsJSON:  strings.TrimSpace(r.FormValue("actions_json")),
			Notes:        strings.TrimSpace(r.FormValue("notes")),
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
		ID:     templateID,
		Name:   strings.TrimSpace(r.FormValue("name")),
		Vendor: strings.TrimSpace(r.FormValue("vendor")),
		Status: templates.TemplateStatus(strings.ToUpper(strings.TrimSpace(r.FormValue("status")))),
		Notes:  strings.TrimSpace(r.FormValue("notes")),
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
	RawPayload  string
	SummaryJSON string
	Warnings    []string
}

func (s *Server) renderTemplatePreview(ctx context.Context, r *http.Request) (*renderPreviewResult, error) {
	templateID := strings.TrimSpace(r.FormValue("template_id"))
	actionKey := strings.TrimSpace(r.FormValue("template_action_key"))
	if templateID == "" || actionKey == "" {
		return nil, fmt.Errorf("template_id and template_action_key are required")
	}
	versionRaw := strings.TrimSpace(r.FormValue("version"))
	var versionRow *templates.G2STemplateVersion
	var err error
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
		RawPayload:  rendered.RawPayload,
		SummaryJSON: rendered.SummaryJSON,
		Warnings:    rendered.Warnings,
	}, nil
}

func (s *Server) renderTemplatesPage(w http.ResponseWriter, r *http.Request, message string, errText string, preview *renderPreviewResult) {
	templateRows, err := s.Store.ListG2STemplates(r.Context())
	if err != nil {
		s.renderError(w, "/field-test/templates", "Field-Test Templates", err)
		return
	}
	versionsByTemplate := map[string][]templates.G2STemplateVersion{}
	for _, tpl := range templateRows {
		rows, err := s.Store.ListG2STemplateVersions(r.Context(), tpl.ID)
		if err != nil {
			s.renderError(w, "/field-test/templates", "Field-Test Templates", err)
			return
		}
		versionsByTemplate[tpl.ID] = rows
	}

	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Templates</h2><table>`)
	body.WriteString(`<tr><th>ID</th><th>Name</th><th>Vendor</th><th>Status</th><th>Active Version</th><th>Versions</th><th>Edit</th></tr>`)
	for _, tpl := range templateRows {
		versionLabels := []string{}
		for _, version := range versionsByTemplate[tpl.ID] {
			versionLabels = append(versionLabels, version.VersionLabel)
		}
		body.WriteString(`<tr>`)
		body.WriteString(`<td class="mono">` + esc(tpl.ID) + `</td>`)
		body.WriteString(`<td>` + esc(tpl.Name) + `</td>`)
		body.WriteString(`<td>` + esc(tpl.Vendor) + `</td>`)
		body.WriteString(`<td>` + esc(string(tpl.Status)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(tpl.CurrentVersionID) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(strings.Join(versionLabels, ", ")) + `</td>`)
		body.WriteString(`<td><form class="inline-form" method="post" action="/field-test/templates/` + esc(tpl.ID) + `">`)
		body.WriteString(`<label>Name <input type="text" name="name" value="` + esc(tpl.Name) + `" style="width:120px"></label>`)
		body.WriteString(`<label>Vendor <input type="text" name="vendor" value="` + esc(tpl.Vendor) + `" style="width:100px"></label>`)
		body.WriteString(`<label>Status <select name="status">` + templateStatusOptions(tpl.Status) + `</select></label>`)
		body.WriteString(`<button type="submit">Save</button></form>`)
		body.WriteString(`<form class="inline-form" method="post" action="/field-test/templates/` + esc(tpl.ID) + `/active-version">`)
		body.WriteString(`<label>Set Active Version <input type="number" name="active_version" style="width:70px"></label> <button type="submit">Set</button></form>`)
		body.WriteString(`<form class="inline-form" method="post" action="/field-test/templates/` + esc(tpl.ID) + `/versions">`)
		body.WriteString(`<label>Version Label <input type="text" name="version_label" style="width:90px"></label>`)
		body.WriteString(`<label>Version ID <input type="text" name="version_id" style="width:150px"></label>`)
		body.WriteString(`<label>Notes <input type="text" name="notes" style="width:150px"></label><br>`)
		body.WriteString(`<label style="display:block;">ActionsJSON <textarea name="actions_json"></textarea></label>`)
		body.WriteString(`<button type="submit">Add Version</button></form></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)

	body.WriteString(`<div class="panel"><h3>Add / Upsert Template</h3>`)
	body.WriteString(`<form method="post" action="/field-test/templates">`)
	body.WriteString(`<label>ID <input type="text" name="id"></label>`)
	body.WriteString(`<label>Name <input type="text" name="name"></label>`)
	body.WriteString(`<label>Vendor <input type="text" name="vendor"></label>`)
	body.WriteString(`<label>Status <select name="status">` + templateStatusOptions("") + `</select></label>`)
	body.WriteString(`<button type="submit">Upsert Template</button></form></div>`)

	body.WriteString(`<div class="panel"><h3>Render Preview (No Send)</h3>`)
	body.WriteString(`<p>Supported variables: ` + esc(strings.Join(renderPreviewSupportedVariables, ", ")) + `.</p>`)
	body.WriteString(`<form method="post" action="/field-test/templates/render-preview">`)
	body.WriteString(`<label>Template ID <input type="text" name="template_id"></label>`)
	body.WriteString(`<label>Version (optional) <input type="number" name="version" style="width:70px"></label>`)
	body.WriteString(`<label>Action Key <input type="text" name="template_action_key"></label><br>`)
	body.WriteString(`<label>Action ID <input type="text" name="action_id"></label>`)
	body.WriteString(`<label>Action Run ID <input type="text" name="action_run_id"></label>`)
	body.WriteString(`<label>Action Step ID <input type="text" name="action_step_id"></label><br>`)
	body.WriteString(`<label>EGM ID <input type="text" name="egm_id"></label>`)
	body.WriteString(`<label>Host ID <input type="text" name="host_id"></label>`)
	body.WriteString(`<label>IP Address <input type="text" name="ip_address"></label>`)
	body.WriteString(`<label>Endpoint Path <input type="text" name="endpoint_path"></label>`)
	body.WriteString(`<button type="submit">Render Preview</button></form>`)
	if preview != nil {
		body.WriteString(`<h4>Preview Result</h4>`)
		body.WriteString(`<p>Message Type: <span class="mono">` + esc(preview.MessageType) + `</span></p>`)
		body.WriteString(`<p>Content Type: <span class="mono">` + esc(preview.ContentType) + `</span></p>`)
		body.WriteString(`<pre>` + esc(preview.RawPayload) + `</pre>`)
		if preview.SummaryJSON != "" {
			body.WriteString(`<details><summary>Summary JSON</summary><pre>` + esc(preview.SummaryJSON) + `</pre></details>`)
		}
		if len(preview.Warnings) > 0 {
			body.WriteString(`<details><summary>Warnings</summary><pre>` + esc(strings.Join(preview.Warnings, "\n")) + `</pre></details>`)
		}
	}
	body.WriteString(`</div>`)
	s.renderPage(w, "/field-test/templates", "Field-Test Templates", body.String(), message, errText)
}

func (s *Server) handleComms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := s.Store.ListMessageJournalEntries(r.Context(), store.MessageJournalListQuery{Limit: queryLimit(r, 120)})
	if err != nil {
		s.renderError(w, "/field-test/comms", "Field-Test Comms Journal", err)
		return
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("export")), "json") {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="field-test-comms.json"`)
		_ = json.NewEncoder(w).Encode(rows)
		return
	}

	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Comms Journal</h2><p><a href="/field-test/comms?export=json">Export JSON</a></p><table>`)
	body.WriteString(`<tr><th>Timestamp</th><th>Direction</th><th>EGM</th><th>Action Run</th><th>Template</th><th>Message Type</th><th>Result</th><th>Transport</th><th>HTTP</th><th>Latency(ms)</th><th>Payload</th></tr>`)
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
		body.WriteString(`<td class="mono">` + esc(row.EGMID) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.ActionRunID) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(templateRef) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.MessageType) + `</td>`)
		body.WriteString(`<td>` + esc(string(row.Result)) + `</td>`)
		body.WriteString(`<td>` + esc(row.TransportMode) + `</td>`)
		body.WriteString(`<td>` + esc(zeroDash(row.HTTPStatusCode)) + `</td>`)
		body.WriteString(`<td>` + esc(zeroDash(row.LatencyMS)) + `</td>`)
		body.WriteString(`<td><details><summary>view</summary><pre>` + esc(payload) + `</pre></details></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)
	s.renderPage(w, "/field-test/comms", "Field-Test Comms Journal", body.String(), "", "")
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := s.Store.ListAuditTimelineEntries(r.Context(), store.AuditTimelineListQuery{Limit: queryLimit(r, 120)})
	if err != nil {
		s.renderError(w, "/field-test/audit", "Field-Test Audit Timeline", err)
		return
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("export")), "json") {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="field-test-audit.json"`)
		_ = json.NewEncoder(w).Encode(rows)
		return
	}
	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Emergency Audit Timeline</h2><p><a href="/field-test/audit?export=json">Export JSON</a></p><table>`)
	body.WriteString(`<tr><th>Timestamp</th><th>Severity</th><th>Event</th><th>Summary</th><th>Input Transition</th><th>Action Run</th><th>Message</th><th>Operator</th><th>Details</th></tr>`)
	for _, row := range rows {
		body.WriteString(`<tr>`)
		body.WriteString(`<td>` + esc(fmtTime(row.OccurredAt)) + `</td>`)
		body.WriteString(`<td>` + esc(string(row.Severity)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.EventType) + `</td>`)
		body.WriteString(`<td>` + esc(row.Summary) + `</td>`)
		body.WriteString(`<td>` + esc(zeroDash64(row.InputTransitionID)) + `</td>`)
		body.WriteString(`<td class="mono">` + esc(row.ActionRunID) + `</td>`)
		body.WriteString(`<td>` + esc(zeroDash64(row.MessageJournalID)) + `</td>`)
		body.WriteString(`<td>` + esc(row.Operator) + `</td>`)
		body.WriteString(`<td><details><summary>view</summary><pre>` + esc(row.DetailJSON) + `</pre></details></td>`)
		body.WriteString(`</tr>`)
	}
	body.WriteString(`</table></div>`)
	s.renderPage(w, "/field-test/audit", "Field-Test Audit Timeline", body.String(), "", "")
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	certs, err := s.Store.ListCertificateInventory(r.Context())
	if err != nil {
		s.renderError(w, "/field-test/settings", "Field-Test Settings", err)
		return
	}
	statusCounts := map[string]int{}
	for _, row := range certs {
		statusCounts[row.Status]++
	}
	statusKeys := make([]string, 0, len(statusCounts))
	for key := range statusCounts {
		statusKeys = append(statusKeys, key)
	}
	sort.Strings(statusKeys)

	certSummary := "no certificate inventory records"
	if len(statusKeys) > 0 {
		parts := make([]string, 0, len(statusKeys))
		for _, key := range statusKeys {
			parts = append(parts, fmt.Sprintf("%s=%d", key, statusCounts[key]))
		}
		certSummary = strings.Join(parts, ", ")
	}

	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Network / Cert / Safety Settings (Read-Only)</h2><table>`)
	body.WriteString(`<tr><th>Field</th><th>Value</th></tr>`)
	body.WriteString(`<tr><td>App Version</td><td>` + esc(defaultString(s.Options.AppVersion, "unknown")) + `</td></tr>`)
	body.WriteString(`<tr><td>Database Path</td><td class="mono">` + esc(defaultString(s.Options.DatabasePath, "unknown")) + `</td></tr>`)
	body.WriteString(`<tr><td>Bind Address</td><td class="mono">` + esc(defaultString(s.Options.BindAddress, "unknown")) + `</td></tr>`)
	body.WriteString(`<tr><td>Real Send Default</td><td>disabled/gated</td></tr>`)
	body.WriteString(`<tr><td>Transport Gate</td><td>` + esc(s.Options.TransportGateSummary) + `</td></tr>`)
	body.WriteString(`<tr><td>Capture Safety</td><td>` + esc(s.Options.CapturePolicySummary) + `</td></tr>`)
	body.WriteString(`<tr><td>Certificate Status Summary</td><td>` + esc(certSummary) + `</td></tr>`)
	body.WriteString(`<tr><td>Trust Material</td><td>placeholder/read-only in Phase 3A</td></tr>`)
	body.WriteString(`</table></div>`)
	s.renderPage(w, "/field-test/settings", "Field-Test Settings", body.String(), "", "")
}

func (s *Server) renderError(w http.ResponseWriter, active string, title string, err error) {
	s.renderPage(w, active, title, "", "", err.Error())
}

func (s *Server) renderPage(w http.ResponseWriter, active string, title string, body string, message string, errText string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	htmlText := strings.Builder{}
	htmlText.WriteString(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>`)
	htmlText.WriteString(esc(title))
	htmlText.WriteString(`</title><link rel="stylesheet" href="/field-test/static/field-test.css"></head><body>`)
	htmlText.WriteString(`<header><h1>Field-Test Operator Configuration Shell</h1><nav>`)
	htmlText.WriteString(navLink("/field-test", "Home", active))
	htmlText.WriteString(navLink("/field-test/inputs", "Inputs", active))
	htmlText.WriteString(navLink("/field-test/actions", "Actions", active))
	htmlText.WriteString(navLink("/field-test/egms", "EGMs", active))
	htmlText.WriteString(navLink("/field-test/templates", "Templates", active))
	htmlText.WriteString(navLink("/field-test/comms", "Comms", active))
	htmlText.WriteString(navLink("/field-test/audit", "Audit", active))
	htmlText.WriteString(navLink("/field-test/settings", "Settings", active))
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

func sanitizeVersionLabel(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "version"
	}
	builder := strings.Builder{}
	for _, ch := range trimmed {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			builder.WriteRune(ch)
		} else {
			builder.WriteRune('-')
		}
	}
	return strings.Trim(builder.String(), "-")
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
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
