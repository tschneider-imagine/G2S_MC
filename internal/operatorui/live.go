package operatorui

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/incidents"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

type LiveView struct {
	GeneratedAt          time.Time            `json:"generated_at"`
	LastUpdated          string               `json:"last_updated"`
	CurrentOperation     string               `json:"current_operation"`
	ActiveInputID        string               `json:"active_input_id,omitempty"`
	ActiveInputName      string               `json:"active_input_name,omitempty"`
	ActivePriority       int                  `json:"active_priority,omitempty"`
	ActiveInputCount     int                  `json:"active_input_count"`
	ActiveLatchCount     int                  `json:"active_latch_count"`
	LatestTransitionAt   string               `json:"latest_transition_at,omitempty"`
	ActiveIncidentID     string               `json:"active_incident_id,omitempty"`
	ActiveIncident       *LiveIncidentSummary `json:"active_incident,omitempty"`
	ActiveInputs         []LiveInputSummary   `json:"active_inputs"`
	ActiveActions        []LiveActionSummary  `json:"active_actions"`
	EGMAttention         []LiveEGMAttention   `json:"egm_attention"`
	RecentMessages       []LiveMessageSummary `json:"recent_messages"`
	RecentAuditEvents    []LiveAuditSummary   `json:"recent_audit_events"`
	PendingDeliveryCount int                  `json:"pending_delivery_count"`
	RuntimeListener      string               `json:"runtime_listener,omitempty"`
	RuntimeDatabasePath  string               `json:"runtime_database_path,omitempty"`
	InputRuntimeEnabled  bool                 `json:"input_runtime_enabled"`
	CertificateCount     int                  `json:"certificate_count"`
}

type LiveIncidentSummary struct {
	IncidentID       string `json:"incident_id"`
	OpenedAt         string `json:"opened_at"`
	Status           string `json:"status"`
	Severity         string `json:"severity"`
	PrimaryInputID   string `json:"primary_input_id"`
	PrimaryActionRun string `json:"primary_action_run_id,omitempty"`
	Summary          string `json:"summary,omitempty"`
}

type LiveInputSummary struct {
	InputID         string `json:"input_id"`
	Name            string `json:"name"`
	GPIOChannel     string `json:"gpio_channel"`
	RawState        string `json:"raw_state"`
	DerivedState    string `json:"derived_state"`
	LatchMode       string `json:"latch_mode"`
	LatchActive     bool   `json:"latch_active"`
	Priority        int    `json:"priority"`
	LastObservedAt  string `json:"last_observed_at,omitempty"`
	LastTransition  string `json:"last_transition,omitempty"`
	OnTriggerAction string `json:"on_trigger_action,omitempty"`
	OnNormalAction  string `json:"on_normal_action,omitempty"`
}

type LiveActionSummary struct {
	StartedAt      string `json:"started_at"`
	RunID          string `json:"run_id"`
	ActionID       string `json:"action_id"`
	Status         string `json:"status"`
	TargetCount    int    `json:"target_count"`
	ConfirmedCount int    `json:"confirmed_count"`
	FailedCount    int    `json:"failed_count"`
	EscalatedCount int    `json:"escalated_count"`
	TriggerReason  string `json:"trigger_reason,omitempty"`
}

type LiveEGMAttention struct {
	EGMID              string `json:"egm_id"`
	DisplayName        string `json:"display_name"`
	TemplateID         string `json:"template_id,omitempty"`
	CurrentActionState string `json:"current_action_state,omitempty"`
	Reason             string `json:"reason"`
	LastSeenAt         string `json:"last_seen_at,omitempty"`
}

type LiveMessageSummary struct {
	Timestamp   string `json:"timestamp"`
	Direction   string `json:"direction"`
	EGMID       string `json:"egm_id,omitempty"`
	ActionRunID string `json:"action_run_id,omitempty"`
	MessageType string `json:"message_type,omitempty"`
	Result      string `json:"result,omitempty"`
}

type LiveAuditSummary struct {
	Timestamp         string `json:"timestamp"`
	Severity          string `json:"severity"`
	EventType         string `json:"event_type"`
	Summary           string `json:"summary"`
	ActionRunID       string `json:"action_run_id,omitempty"`
	InputTransitionID int64  `json:"input_transition_id,omitempty"`
}

type liveActiveInput struct {
	channel inputs.InputChannel
	state   *inputs.InputTransition
}

func (s *Server) buildLiveView(ctx context.Context) (LiveView, error) {
	view := LiveView{
		GeneratedAt:         time.Now().UTC(),
		RuntimeListener:     strings.TrimSpace(s.Options.BindAddress),
		RuntimeDatabasePath: strings.TrimSpace(s.Options.DatabasePath),
		InputRuntimeEnabled: s.Options.InputRuntimeEnabled,
	}
	view.LastUpdated = view.GeneratedAt.Format(time.RFC3339)

	channels, err := s.Store.ListInputChannels(ctx)
	if err != nil {
		return LiveView{}, err
	}
	_, auditByTransitionID, lastTransitionByInput, err := s.loadInputTransitionView(ctx, 200, 300)
	if err != nil {
		return LiveView{}, err
	}

	latestTransitionAt := time.Time{}
	activeInputs := make([]LiveInputSummary, 0, len(channels))
	activeCandidates := make([]LiveInputSummary, 0, len(channels))
	latchCount := 0
	for _, channel := range channels {
		runtimeState, stateErr := s.Store.GetInputRuntimeState(ctx, channel.ID)
		if stateErr != nil {
			return LiveView{}, stateErr
		}
		rawState := defaultString(strings.TrimSpace(string(channel.CurrentState)), "unknown")
		derivedState := defaultString(strings.TrimSpace(string(channel.DerivedState)), "unknown")
		latchActive := false
		lastObserved := ""
		if runtimeState != nil {
			if strings.TrimSpace(string(runtimeState.LastObservedRawState)) != "" {
				rawState = string(runtimeState.LastObservedRawState)
			}
			if strings.TrimSpace(string(runtimeState.DerivedState)) != "" {
				derivedState = string(runtimeState.DerivedState)
			}
			latchActive = runtimeState.LatchActive
			if latchActive {
				latchCount++
			}
			if !runtimeState.LastObservedAt.IsZero() {
				lastObserved = runtimeState.LastObservedAt.UTC().Format(time.RFC3339)
			}
			if runtimeState.LastTransitionAt != nil && runtimeState.LastTransitionAt.After(latestTransitionAt) {
				latestTransitionAt = *runtimeState.LastTransitionAt
			}
		}

		lastTransition := "-"
		if transition, ok := lastTransitionByInput[channel.ID]; ok {
			lastTransition = transitionSummary(transition, auditByTransitionID[transition.ID])
			if transition.TransitionAt.After(latestTransitionAt) {
				latestTransitionAt = transition.TransitionAt
			}
		}

		row := LiveInputSummary{
			InputID:         channel.ID,
			Name:            channel.Name,
			GPIOChannel:     channel.GPIOChannel,
			RawState:        rawState,
			DerivedState:    derivedState,
			LatchMode:       string(channel.LatchingMode),
			LatchActive:     latchActive,
			Priority:        channel.Priority,
			LastObservedAt:  lastObserved,
			LastTransition:  lastTransition,
			OnTriggerAction: strings.TrimSpace(channel.OnTriggerActionID),
			OnNormalAction:  strings.TrimSpace(channel.OnNormalActionID),
		}
		activeInputs = append(activeInputs, row)
		if channel.Enabled && strings.EqualFold(derivedState, string(inputs.DerivedStateTriggered)) {
			activeCandidates = append(activeCandidates, row)
		}
	}
	sort.Slice(activeInputs, func(i, j int) bool {
		if activeInputs[i].Priority == activeInputs[j].Priority {
			return activeInputs[i].InputID < activeInputs[j].InputID
		}
		return activeInputs[i].Priority > activeInputs[j].Priority
	})
	sort.Slice(activeCandidates, func(i, j int) bool {
		if activeCandidates[i].Priority == activeCandidates[j].Priority {
			return activeCandidates[i].InputID < activeCandidates[j].InputID
		}
		return activeCandidates[i].Priority > activeCandidates[j].Priority
	})
	view.ActiveInputs = activeInputs
	view.ActiveInputCount = len(activeCandidates)
	view.ActiveLatchCount = latchCount
	if !latestTransitionAt.IsZero() {
		view.LatestTransitionAt = latestTransitionAt.UTC().Format(time.RFC3339)
	}

	switch len(activeCandidates) {
	case 0:
		view.CurrentOperation = "Normal"
	default:
		primary := activeCandidates[0]
		view.ActiveInputID = primary.InputID
		view.ActiveInputName = primary.Name
		view.ActivePriority = primary.Priority
		if len(activeCandidates) > 1 {
			view.CurrentOperation = "Multiple Active"
		} else {
			view.CurrentOperation = operationName(primary.InputID, primary.Name)
		}
	}

	openIncidents, err := s.Store.ListOpenIncidentRecords(ctx, 10)
	if err != nil {
		return LiveView{}, err
	}
	if len(openIncidents) > 0 {
		incident := choosePrimaryIncident(openIncidents, activeCandidates)
		if incident != nil {
			view.ActiveIncidentID = strconv.FormatInt(incident.ID, 10)
			view.ActiveIncident = &LiveIncidentSummary{
				IncidentID:       view.ActiveIncidentID,
				OpenedAt:         fmtTime(incident.OpenedAt),
				Status:           string(incident.Status),
				Severity:         defaultString(strings.TrimSpace(incident.Severity), "INFO"),
				PrimaryInputID:   incident.PrimaryInputID,
				PrimaryActionRun: strings.TrimSpace(incident.PrimaryActionRunID),
				Summary:          strings.TrimSpace(incident.Summary),
			}
		}
	}

	runs, err := s.Store.ListActionRuns(ctx, store.ActionRunListQuery{Limit: 80})
	if err != nil {
		return LiveView{}, err
	}
	activeStatuses := map[actions.ActionRunStatus]struct{}{
		actions.RunStatusPending:             {},
		actions.RunStatusRunning:             {},
		actions.RunStatusWaitingConfirmation: {},
		actions.RunStatusFailed:              {},
		actions.RunStatusEscalated:           {},
	}
	actionRows := make([]LiveActionSummary, 0, len(runs))
	for _, run := range runs {
		if _, ok := activeStatuses[run.Status]; !ok {
			continue
		}
		actionRows = append(actionRows, LiveActionSummary{
			StartedAt:      fmtTime(run.StartedAt),
			RunID:          run.ID,
			ActionID:       run.ActionDefinitionID,
			Status:         string(run.Status),
			TargetCount:    run.TargetCount,
			ConfirmedCount: run.ConfirmedCount,
			FailedCount:    run.FailedCount,
			EscalatedCount: run.EscalatedCount,
			TriggerReason:  strings.TrimSpace(run.TriggerReason),
		})
	}
	view.ActiveActions = actionRows

	templatesRows, err := s.Store.ListG2STemplates(ctx)
	if err != nil {
		return LiveView{}, err
	}
	templateIDs := map[string]struct{}{}
	for _, tpl := range templatesRows {
		templateIDs[strings.TrimSpace(tpl.ID)] = struct{}{}
	}
	egmRows, err := s.Store.ListEGMRecords(ctx)
	if err != nil {
		return LiveView{}, err
	}
	latestFailedByEGM := map[string]time.Time{}
	for _, run := range runs {
		targetRows, listErr := s.Store.ListActionTargetResults(ctx, run.ID)
		if listErr != nil {
			return LiveView{}, listErr
		}
		for _, target := range targetRows {
			if target.Status != actions.TargetStatusFailed || target.LastResultAt == nil {
				continue
			}
			existing, exists := latestFailedByEGM[target.TargetEGMID]
			if !exists || target.LastResultAt.After(existing) {
				latestFailedByEGM[target.TargetEGMID] = *target.LastResultAt
			}
		}
	}

	attentionRows := make([]LiveEGMAttention, 0)
	for _, row := range egmRows {
		reasons := make([]string, 0, 4)
		templateID := strings.TrimSpace(row.TemplateID)
		if !row.Enabled && row.EmergencyEnabled {
			reasons = append(reasons, "Emergency participation requires Enabled")
		}
		if templateID == "" {
			reasons = append(reasons, "Template not assigned")
		} else if _, ok := templateIDs[templateID]; !ok {
			reasons = append(reasons, "Template not found")
		}
		switch row.CurrentActionState {
		case egms.EGMActionStatePending, egms.EGMActionStateFailed, egms.EGMActionStateEscalating, egms.EGMActionStateRestoring:
			reasons = append(reasons, "Action state "+string(row.CurrentActionState))
		}
		if when, ok := latestFailedByEGM[row.EGMID]; ok {
			reasons = append(reasons, "Latest target failure "+when.UTC().Format(time.RFC3339))
		}
		if row.LastSeenAt == nil {
			reasons = append(reasons, "No recent communication recorded")
		}
		if len(reasons) == 0 {
			continue
		}
		attentionRows = append(attentionRows, LiveEGMAttention{
			EGMID:              row.EGMID,
			DisplayName:        defaultString(strings.TrimSpace(row.DisplayName), row.EGMID),
			TemplateID:         templateID,
			CurrentActionState: string(row.CurrentActionState),
			Reason:             strings.Join(reasons, "; "),
			LastSeenAt:         fmtMaybeTime(row.LastSeenAt),
		})
	}
	sort.Slice(attentionRows, func(i, j int) bool {
		return attentionRows[i].EGMID < attentionRows[j].EGMID
	})
	view.EGMAttention = attentionRows

	messageRows, err := s.Store.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 10})
	if err != nil {
		return LiveView{}, err
	}
	recentMessages := make([]LiveMessageSummary, 0, len(messageRows))
	for _, row := range messageRows {
		recentMessages = append(recentMessages, LiveMessageSummary{
			Timestamp:   fmtTime(row.Timestamp),
			Direction:   string(row.Direction),
			EGMID:       strings.TrimSpace(row.EGMID),
			ActionRunID: strings.TrimSpace(row.ActionRunID),
			MessageType: strings.TrimSpace(row.MessageType),
			Result:      string(row.Result),
		})
	}
	view.RecentMessages = recentMessages

	pendingRows, err := s.Store.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{
		Limit: 500,
		Results: []g2sengine.MessageResult{
			g2sengine.MessageResultPrepared,
			g2sengine.MessageResultPending,
			g2sengine.MessageResultOffered,
			g2sengine.MessageResultDelivered,
		},
	})
	if err != nil {
		return LiveView{}, err
	}
	view.PendingDeliveryCount = len(pendingRows)

	auditRows, err := s.Store.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{Limit: 10})
	if err != nil {
		return LiveView{}, err
	}
	recentAudit := make([]LiveAuditSummary, 0, len(auditRows))
	for _, row := range auditRows {
		recentAudit = append(recentAudit, LiveAuditSummary{
			Timestamp:         fmtTime(row.OccurredAt),
			Severity:          string(row.Severity),
			EventType:         row.EventType,
			Summary:           row.Summary,
			ActionRunID:       strings.TrimSpace(row.ActionRunID),
			InputTransitionID: row.InputTransitionID,
		})
	}
	view.RecentAuditEvents = recentAudit

	certRows, err := s.Store.ListCertificateInventory(ctx)
	if err != nil {
		return LiveView{}, err
	}
	view.CertificateCount = len(certRows)

	return view, nil
}

func operationName(inputID string, inputName string) string {
	id := strings.ToLower(strings.TrimSpace(inputID))
	switch id {
	case "regular-operation":
		return "Regular Operation"
	case "general-broadcast":
		return "General Broadcast"
	case "emergency-broadcast":
		return "Emergency Broadcast"
	case "local-notice":
		return "Local Notice"
	}
	name := strings.TrimSpace(inputName)
	if name != "" {
		return name
	}
	return "Unknown"
}

func choosePrimaryIncident(rows []incidents.IncidentRecord, activeInputs []LiveInputSummary) *incidents.IncidentRecord {
	if len(rows) == 0 {
		return nil
	}
	priorityByInput := map[string]int{}
	for _, row := range activeInputs {
		priorityByInput[row.InputID] = row.Priority
	}
	selected := rows[0]
	selectedPriority := priorityByInput[strings.TrimSpace(selected.PrimaryInputID)]
	for i := 1; i < len(rows); i++ {
		candidate := rows[i]
		candidatePriority := priorityByInput[strings.TrimSpace(candidate.PrimaryInputID)]
		if candidatePriority > selectedPriority {
			selected = candidate
			selectedPriority = candidatePriority
			continue
		}
		if candidatePriority == selectedPriority {
			if candidate.OpenedAt.After(selected.OpenedAt) {
				selected = candidate
				selectedPriority = candidatePriority
			}
		}
	}
	copy := selected
	return &copy
}

func (s *Server) renderLivePanels(view LiveView) string {
	body := strings.Builder{}
	body.WriteString(`<div class="panel"><h2>Current Operation</h2>`)
	body.WriteString(`<p><strong id="live-operation-name">` + esc(view.CurrentOperation) + `</strong></p>`)
	body.WriteString(`<p>Active Input: <span class="mono" id="live-active-input">` + esc(defaultString(view.ActiveInputID, "-")) + `</span>`)
	if strings.TrimSpace(view.ActiveInputName) != "" {
		body.WriteString(` (` + esc(view.ActiveInputName) + `)`)
	}
	body.WriteString(`</p>`)
	body.WriteString(`<p>Active Priority: <span class="mono" id="live-active-priority">` + esc(zeroDash(view.ActivePriority)) + `</span></p>`)
	body.WriteString(`<p>Active Inputs: <span class="mono" id="live-active-count">` + esc(strconv.Itoa(view.ActiveInputCount)) + `</span></p>`)
	body.WriteString(`<p>Emergency Latches: <span class="mono" id="live-latch-count">` + esc(strconv.Itoa(view.ActiveLatchCount)) + `</span></p>`)
	body.WriteString(`<p>Latest Transition: <span class="mono" id="live-latest-transition">` + esc(defaultString(view.LatestTransitionAt, "-")) + `</span></p>`)
	body.WriteString(`<p>Pending Delivery: <span class="mono" id="live-pending-delivery-count">` + esc(strconv.Itoa(view.PendingDeliveryCount)) + `</span></p>`)
	if view.ActiveIncident != nil {
		body.WriteString(`<p>Active Incident: <a class="mono" id="live-active-incident" href="/operator/audit?incident_id=` + esc(view.ActiveIncidentID) + `">` + esc(view.ActiveIncidentID) + `</a></p>`)
		body.WriteString(`<p>Opened: <span class="mono" id="live-incident-opened">` + esc(defaultString(view.ActiveIncident.OpenedAt, "-")) + `</span></p>`)
		body.WriteString(`<p>Incident Input: <span class="mono" id="live-incident-input">` + esc(defaultString(view.ActiveIncident.PrimaryInputID, "-")) + `</span></p>`)
	} else {
		body.WriteString(`<p>Active Incident: <span class="mono" id="live-active-incident">none</span></p>`)
	}
	body.WriteString(`<p>Last Updated: <span class="mono" id="live-last-updated">` + esc(view.LastUpdated) + `</span></p>`)
	body.WriteString(`</div>`)

	body.WriteString(`<div class="panel"><h2>Active Inputs</h2><p><a href="/operator/inputs">Open Inputs</a></p>`)
	body.WriteString(`<table><tr><th>Name</th><th>GPIO</th><th>Raw</th><th>Derived</th><th>Latch Mode</th><th>Latch Active</th><th>Priority</th><th>Last Observed</th><th>Last Transition</th></tr><tbody id="live-inputs-body">`)
	for _, row := range view.ActiveInputs {
		body.WriteString(renderLiveInputRow(row))
	}
	if len(view.ActiveInputs) == 0 {
		body.WriteString(`<tr><td colspan="9">No configured inputs.</td></tr>`)
	}
	body.WriteString(`</tbody></table></div>`)

	body.WriteString(`<div class="panel"><h2>Active Actions</h2><p><a href="/operator/actions">Open Actions</a></p>`)
	body.WriteString(`<table><tr><th>Started</th><th>Run ID</th><th>Action ID</th><th>Status</th><th>Targets</th><th>Confirmed</th><th>Failed</th><th>Escalated</th><th>Trigger Reason</th></tr><tbody id="live-actions-body">`)
	for _, row := range view.ActiveActions {
		body.WriteString(renderLiveActionRow(row))
	}
	if len(view.ActiveActions) == 0 {
		body.WriteString(`<tr><td colspan="9">No active action runs.</td></tr>`)
	}
	body.WriteString(`</tbody></table></div>`)

	body.WriteString(`<div class="panel"><h2>EGM Attention</h2><p><a href="/operator/egms">Open EGMs</a></p>`)
	body.WriteString(`<table><tr><th>EGM ID</th><th>Name</th><th>Template</th><th>Action State</th><th>Last Seen</th><th>Reason</th></tr><tbody id="live-egm-attention-body">`)
	for _, row := range view.EGMAttention {
		body.WriteString(renderLiveEGMAttentionRow(row))
	}
	if len(view.EGMAttention) == 0 {
		body.WriteString(`<tr><td colspan="6">No EGM attention items.</td></tr>`)
	}
	body.WriteString(`</tbody></table></div>`)

	body.WriteString(`<div class="panel"><h2>Recent Messages</h2><p><a href="/operator/comms">Open Comms</a></p>`)
	body.WriteString(`<table><tr><th>Timestamp</th><th>Direction</th><th>EGM ID</th><th>Action Run</th><th>Message</th><th>Result</th></tr><tbody id="live-messages-body">`)
	for _, row := range view.RecentMessages {
		body.WriteString(renderLiveMessageRow(row))
	}
	if len(view.RecentMessages) == 0 {
		body.WriteString(`<tr><td colspan="6">No recent messages.</td></tr>`)
	}
	body.WriteString(`</tbody></table></div>`)

	body.WriteString(`<div class="panel"><h2>Recent Audit Events</h2><p><a href="/operator/audit">Open Audit</a></p>`)
	body.WriteString(`<table><tr><th>Timestamp</th><th>Severity</th><th>Event</th><th>Summary</th><th>Action Run</th><th>Input Transition</th></tr><tbody id="live-audit-body">`)
	for _, row := range view.RecentAuditEvents {
		body.WriteString(renderLiveAuditRow(row))
	}
	if len(view.RecentAuditEvents) == 0 {
		body.WriteString(`<tr><td colspan="6">No recent audit events.</td></tr>`)
	}
	body.WriteString(`</tbody></table></div>`)

	body.WriteString(`<div class="panel"><h2>Current Runtime</h2>`)
	body.WriteString(`<p>Listener: <span class="mono">` + esc(defaultString(view.RuntimeListener, "-")) + `</span></p>`)
	body.WriteString(`<p>Database: <span class="mono">` + esc(defaultString(view.RuntimeDatabasePath, "-")) + `</span></p>`)
	body.WriteString(`<p>Input Runtime Enabled: <span id="live-input-runtime-enabled">` + esc(yesNo(view.InputRuntimeEnabled)) + `</span></p>`)
	body.WriteString(`<p>Certificate Status: <span class="mono" id="live-certificate-count">` + esc(strconv.Itoa(view.CertificateCount)) + `</span> records</p>`)
	body.WriteString(`</div>`)

	body.WriteString(liveRefreshScript())
	return body.String()
}

func renderLiveInputRow(row LiveInputSummary) string {
	return `<tr>` +
		`<td>` + esc(defaultString(row.Name, row.InputID)) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.GPIOChannel, "-")) + `</td>` +
		`<td>` + esc(defaultString(row.RawState, "unknown")) + `</td>` +
		`<td>` + esc(defaultString(row.DerivedState, "unknown")) + `</td>` +
		`<td>` + esc(defaultString(row.LatchMode, "-")) + `</td>` +
		`<td>` + esc(yesNo(row.LatchActive)) + `</td>` +
		`<td>` + esc(strconv.Itoa(row.Priority)) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.LastObservedAt, "-")) + `</td>` +
		`<td>` + esc(defaultString(row.LastTransition, "-")) + `</td>` +
		`</tr>`
}

func renderLiveActionRow(row LiveActionSummary) string {
	return `<tr>` +
		`<td class="mono">` + esc(defaultString(row.StartedAt, "-")) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.RunID, "-")) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.ActionID, "-")) + `</td>` +
		`<td>` + esc(defaultString(row.Status, "-")) + `</td>` +
		`<td>` + esc(strconv.Itoa(row.TargetCount)) + `</td>` +
		`<td>` + esc(strconv.Itoa(row.ConfirmedCount)) + `</td>` +
		`<td>` + esc(strconv.Itoa(row.FailedCount)) + `</td>` +
		`<td>` + esc(strconv.Itoa(row.EscalatedCount)) + `</td>` +
		`<td>` + esc(defaultString(row.TriggerReason, "-")) + `</td>` +
		`</tr>`
}

func renderLiveEGMAttentionRow(row LiveEGMAttention) string {
	return `<tr>` +
		`<td class="mono">` + esc(defaultString(row.EGMID, "-")) + `</td>` +
		`<td>` + esc(defaultString(row.DisplayName, "-")) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.TemplateID, "-")) + `</td>` +
		`<td>` + esc(defaultString(row.CurrentActionState, "-")) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.LastSeenAt, "-")) + `</td>` +
		`<td>` + esc(defaultString(row.Reason, "-")) + `</td>` +
		`</tr>`
}

func renderLiveMessageRow(row LiveMessageSummary) string {
	return `<tr>` +
		`<td class="mono">` + esc(defaultString(row.Timestamp, "-")) + `</td>` +
		`<td>` + esc(defaultString(row.Direction, "-")) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.EGMID, "-")) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.ActionRunID, "-")) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.MessageType, "-")) + `</td>` +
		`<td>` + esc(defaultString(row.Result, "-")) + `</td>` +
		`</tr>`
}

func renderLiveAuditRow(row LiveAuditSummary) string {
	return `<tr>` +
		`<td class="mono">` + esc(defaultString(row.Timestamp, "-")) + `</td>` +
		`<td>` + esc(defaultString(row.Severity, "-")) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.EventType, "-")) + `</td>` +
		`<td>` + esc(defaultString(row.Summary, "-")) + `</td>` +
		`<td class="mono">` + esc(defaultString(row.ActionRunID, "-")) + `</td>` +
		`<td>` + esc(zeroDash64(row.InputTransitionID)) + `</td>` +
		`</tr>`
}

func liveRefreshScript() string {
	return `<script>
(function () {
  if (!window.fetch) {
    return;
  }
  var url = '/operator/live.json';
  var lastUpdated = document.getElementById('live-last-updated');
  function text(id, value, fallback) {
    var node = document.getElementById(id);
    if (!node) {
      return;
    }
    var v = (value || '').toString().trim();
    node.textContent = v === '' ? fallback : v;
  }
  function boolText(value) {
    return value ? 'yes' : 'no';
  }
  function fillRows(bodyID, rows, fallbackHTML) {
    var body = document.getElementById(bodyID);
    if (!body) {
      return;
    }
    if (!rows || rows.length === 0) {
      body.innerHTML = fallbackHTML;
      return;
    }
    body.innerHTML = rows.join('');
  }
  function esc(value) {
    return (value || '').toString()
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }
  function rowInput(r) {
    return '<tr><td>' + esc(r.name || r.input_id) + '</td><td class="mono">' + esc(r.gpio_channel || '-') + '</td><td>' + esc(r.raw_state || 'unknown') + '</td><td>' + esc(r.derived_state || 'unknown') + '</td><td>' + esc(r.latch_mode || '-') + '</td><td>' + esc(r.latch_active ? 'yes' : 'no') + '</td><td>' + esc(String(r.priority || 0)) + '</td><td class="mono">' + esc(r.last_observed_at || '-') + '</td><td>' + esc(r.last_transition || '-') + '</td></tr>';
  }
  function rowAction(r) {
    return '<tr><td class="mono">' + esc(r.started_at || '-') + '</td><td class="mono">' + esc(r.run_id || '-') + '</td><td class="mono">' + esc(r.action_id || '-') + '</td><td>' + esc(r.status || '-') + '</td><td>' + esc(String(r.target_count || 0)) + '</td><td>' + esc(String(r.confirmed_count || 0)) + '</td><td>' + esc(String(r.failed_count || 0)) + '</td><td>' + esc(String(r.escalated_count || 0)) + '</td><td>' + esc(r.trigger_reason || '-') + '</td></tr>';
  }
  function rowEGM(r) {
    return '<tr><td class="mono">' + esc(r.egm_id || '-') + '</td><td>' + esc(r.display_name || '-') + '</td><td class="mono">' + esc(r.template_id || '-') + '</td><td>' + esc(r.current_action_state || '-') + '</td><td class="mono">' + esc(r.last_seen_at || '-') + '</td><td>' + esc(r.reason || '-') + '</td></tr>';
  }
  function rowMessage(r) {
    return '<tr><td class="mono">' + esc(r.timestamp || '-') + '</td><td>' + esc(r.direction || '-') + '</td><td class="mono">' + esc(r.egm_id || '-') + '</td><td class="mono">' + esc(r.action_run_id || '-') + '</td><td class="mono">' + esc(r.message_type || '-') + '</td><td>' + esc(r.result || '-') + '</td></tr>';
  }
  function rowAudit(r) {
    var transition = r.input_transition_id && r.input_transition_id > 0 ? String(r.input_transition_id) : '-';
    return '<tr><td class="mono">' + esc(r.timestamp || '-') + '</td><td>' + esc(r.severity || '-') + '</td><td class="mono">' + esc(r.event_type || '-') + '</td><td>' + esc(r.summary || '-') + '</td><td class="mono">' + esc(r.action_run_id || '-') + '</td><td>' + esc(transition) + '</td></tr>';
  }
  function refresh() {
    fetch(url + '?t=' + Date.now(), { cache: 'no-store' })
      .then(function (response) {
        if (!response.ok) {
          throw new Error('status');
        }
        return response.json();
      })
      .then(function (payload) {
        text('live-operation-name', payload.current_operation, 'Unknown');
        text('live-active-input', payload.active_input_id, '-');
        text('live-active-priority', payload.active_priority ? String(payload.active_priority) : '-', '-');
        text('live-active-count', String(payload.active_input_count || 0), '0');
        text('live-latch-count', String(payload.active_latch_count || 0), '0');
        text('live-latest-transition', payload.latest_transition_at, '-');
        text('live-pending-delivery-count', String(payload.pending_delivery_count || 0), '0');
        if (payload.active_incident_id) {
          var incidentNode = document.getElementById('live-active-incident');
          if (incidentNode) {
            incidentNode.textContent = payload.active_incident_id;
            if (incidentNode.tagName && incidentNode.tagName.toLowerCase() === 'a') {
              incidentNode.setAttribute('href', '/operator/audit?incident_id=' + encodeURIComponent(payload.active_incident_id));
            }
          }
          text('live-incident-opened', payload.active_incident && payload.active_incident.opened_at, '-');
          text('live-incident-input', payload.active_incident && payload.active_incident.primary_input_id, '-');
        } else {
          text('live-active-incident', 'none', 'none');
          text('live-incident-opened', '-', '-');
          text('live-incident-input', '-', '-');
        }
        text('live-last-updated', payload.last_updated, '-');
        text('live-input-runtime-enabled', boolText(!!payload.input_runtime_enabled), 'no');
        text('live-certificate-count', String(payload.certificate_count || 0), '0');
        fillRows('live-inputs-body', (payload.active_inputs || []).map(rowInput), '<tr><td colspan="9">No configured inputs.</td></tr>');
        fillRows('live-actions-body', (payload.active_actions || []).map(rowAction), '<tr><td colspan="9">No active action runs.</td></tr>');
        fillRows('live-egm-attention-body', (payload.egm_attention || []).map(rowEGM), '<tr><td colspan="6">No EGM attention items.</td></tr>');
        fillRows('live-messages-body', (payload.recent_messages || []).map(rowMessage), '<tr><td colspan="6">No recent messages.</td></tr>');
        fillRows('live-audit-body', (payload.recent_audit_events || []).map(rowAudit), '<tr><td colspan="6">No recent audit events.</td></tr>');
      })
      .catch(function () {
        if (lastUpdated) {
          lastUpdated.textContent = 'live update unavailable';
        }
      });
  }
  refresh();
  window.setInterval(refresh, 2000);
})();
</script>`
}

func failedActionState(state egms.EGMActionState) bool {
	return state == egms.EGMActionStateFailed || state == egms.EGMActionStateEscalating || state == egms.EGMActionStateRestoring || state == egms.EGMActionStatePending
}
