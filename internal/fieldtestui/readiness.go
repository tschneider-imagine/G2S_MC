package fieldtestui

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

type ReadinessStatus string

const (
	ReadinessPass ReadinessStatus = "PASS"
	ReadinessWarn ReadinessStatus = "WARN"
	ReadinessFail ReadinessStatus = "FAIL"
	ReadinessInfo ReadinessStatus = "INFO"
)

type ReadinessCheck struct {
	Status  ReadinessStatus `json:"status"`
	Code    string          `json:"code"`
	Summary string          `json:"summary"`
	Detail  string          `json:"detail,omitempty"`
}

type ReadinessSection struct {
	Name   string           `json:"name"`
	Checks []ReadinessCheck `json:"checks"`
}

type ReadinessReport struct {
	GeneratedAt time.Time          `json:"generated_at"`
	Sections    []ReadinessSection `json:"sections"`
}

type FieldTestInputSnapshot struct {
	Channel      inputs.InputChannel             `json:"channel"`
	RuntimeState *inputruntime.InputRuntimeState `json:"runtime_state,omitempty"`
}

type FieldTestTemplateSnapshot struct {
	Template      templates.G2STemplate          `json:"template"`
	ActiveVersion *templates.G2STemplateVersion  `json:"active_version,omitempty"`
	Versions      []templates.G2STemplateVersion `json:"versions"`
}

type FieldTestActionPreview struct {
	ActionID string                    `json:"action_id"`
	Plan     *actionplanner.ActionPlan `json:"plan,omitempty"`
	Error    string                    `json:"error,omitempty"`
}

type FieldTestExportPackage struct {
	GeneratedAt     time.Time                       `json:"generated_at"`
	SafetyGate      map[string]any                  `json:"safety_gate"`
	Inputs          []FieldTestInputSnapshot        `json:"inputs"`
	Actions         []actions.ActionDefinition      `json:"actions"`
	ActionPreviews  []FieldTestActionPreview        `json:"action_previews"`
	EGMs            []egms.EGMRecord                `json:"egms"`
	Templates       []FieldTestTemplateSnapshot     `json:"templates"`
	MessageJournal  []g2sengine.MessageJournalEntry `json:"message_journal"`
	AuditTimeline   []audit.AuditTimelineEntry      `json:"audit_timeline"`
	CertificateMeta []model.CertificateInventory    `json:"certificate_inventory"`
	Readiness       ReadinessReport                 `json:"readiness"`
}

func (s *Server) buildReadinessReport(ctx context.Context) (ReadinessReport, error) {
	now := time.Now().UTC()
	report := ReadinessReport{
		GeneratedAt: now,
		Sections:    []ReadinessSection{},
	}

	channels, err := s.Store.ListInputChannels(ctx)
	if err != nil {
		return report, err
	}
	channelByID := map[string]inputs.InputChannel{}
	inputSnapshots := []FieldTestInputSnapshot{}
	for _, channel := range channels {
		channelByID[channel.ID] = channel
		runtimeState, runtimeErr := s.Store.GetInputRuntimeState(ctx, channel.ID)
		if runtimeErr != nil {
			return report, runtimeErr
		}
		inputSnapshots = append(inputSnapshots, FieldTestInputSnapshot{Channel: channel, RuntimeState: runtimeState})
	}

	definitions, err := s.Store.ListActionDefinitions(ctx)
	if err != nil {
		return report, err
	}
	definitionByID := map[string]actions.ActionDefinition{}
	for _, definition := range definitions {
		definitionByID[definition.ID] = definition
	}

	records, err := s.Store.ListEGMRecords(ctx)
	if err != nil {
		return report, err
	}

	templateRows, err := s.Store.ListG2STemplates(ctx)
	if err != nil {
		return report, err
	}
	templateByID := map[string]templates.G2STemplate{}
	activeVersionByTemplate := map[string]*templates.G2STemplateVersion{}
	for _, tpl := range templateRows {
		templateByID[tpl.ID] = tpl
		activeVersion, activeErr := s.Store.GetActiveG2STemplateVersion(ctx, tpl.ID)
		if activeErr != nil {
			return report, activeErr
		}
		activeVersionByTemplate[tpl.ID] = activeVersion
	}

	messageRows, err := s.Store.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 250})
	if err != nil {
		return report, err
	}
	auditRows, err := s.Store.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{Limit: 250})
	if err != nil {
		return report, err
	}
	certRows, err := s.Store.ListCertificateInventory(ctx)
	if err != nil {
		return report, err
	}

	report.Sections = append(report.Sections, s.buildInputsReadinessSection(channelByID, inputSnapshots))
	report.Sections = append(report.Sections, s.buildActionsReadinessSection(channels, definitionByID, definitions))
	report.Sections = append(report.Sections, s.buildEGMsReadinessSection(records, templateByID))
	report.Sections = append(report.Sections, s.buildTemplatesReadinessSection(definitions, templateRows, activeVersionByTemplate))
	report.Sections = append(report.Sections, s.buildCommsReadinessSection(messageRows))
	report.Sections = append(report.Sections, s.buildAuditReadinessSection(auditRows))
	report.Sections = append(report.Sections, s.buildSettingsReadinessSection(messageRows, certRows))
	return report, nil
}

func (s *Server) buildInputsReadinessSection(channelByID map[string]inputs.InputChannel, snapshots []FieldTestInputSnapshot) ReadinessSection {
	checks := []ReadinessCheck{}
	required := []string{"regular-operation", "general-broadcast", "emergency-broadcast", "local-notice"}
	missing := []string{}
	for _, id := range required {
		if _, ok := channelByID[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessFail,
			Code:    "INPUT_REQUIRED_CHANNELS",
			Summary: "Required field-test input channels are missing",
			Detail:  strings.Join(missing, ", "),
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "INPUT_REQUIRED_CHANNELS",
			Summary: "All required field-test input channels are present",
			Detail:  strings.Join(required, ", "),
		})
	}

	if len(channelByID) != 4 {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessWarn,
			Code:    "INPUT_CHANNEL_COUNT",
			Summary: "Input channel count differs from expected field-test set of 4",
			Detail:  fmt.Sprintf("configured=%d", len(channelByID)),
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "INPUT_CHANNEL_COUNT",
			Summary: "Input channel count matches expected field-test set",
			Detail:  "configured=4",
		})
	}

	emergency, hasEmergency := channelByID["emergency-broadcast"]
	if !hasEmergency {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessFail,
			Code:    "INPUT_EMERGENCY_LATCH_MODE",
			Summary: "Emergency broadcast input is missing",
		})
	} else if emergency.LatchingMode != inputs.LatchingManualClear {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessFail,
			Code:    "INPUT_EMERGENCY_LATCH_MODE",
			Summary: "Emergency broadcast input must be MANUAL_CLEAR",
			Detail:  "configured=" + string(emergency.LatchingMode),
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "INPUT_EMERGENCY_LATCH_MODE",
			Summary: "Emergency broadcast input uses MANUAL_CLEAR",
		})
	}

	misconfigured := []string{}
	for _, snapshot := range snapshots {
		channel := snapshot.Channel
		reasons := []string{}
		if strings.TrimSpace(channel.GPIOChannel) == "" {
			reasons = append(reasons, "missing GPIO channel")
		}
		if channel.DebounceMS <= 0 {
			reasons = append(reasons, "missing/zero debounce")
		}
		if strings.TrimSpace(channel.OnTriggerActionID) == "" {
			reasons = append(reasons, "missing on-trigger action")
		}
		if channel.ID != "regular-operation" && strings.TrimSpace(channel.OnNormalActionID) == "" {
			reasons = append(reasons, "missing on-normal action")
		}
		if len(reasons) > 0 {
			misconfigured = append(misconfigured, channel.ID+": "+strings.Join(reasons, "; "))
		}
	}
	if len(misconfigured) > 0 {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessWarn,
			Code:    "INPUT_BINDING_COMPLETENESS",
			Summary: "One or more input channels have incomplete field-test configuration",
			Detail:  strings.Join(misconfigured, " | "),
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "INPUT_BINDING_COMPLETENESS",
			Summary: "All configured inputs have GPIO, debounce, and action bindings",
		})
	}
	return ReadinessSection{Name: "Inputs", Checks: checks}
}

func (s *Server) buildActionsReadinessSection(channels []inputs.InputChannel, definitionByID map[string]actions.ActionDefinition, definitions []actions.ActionDefinition) ReadinessSection {
	checks := []ReadinessCheck{}
	if len(definitions) == 0 {
		checks = append(checks, ReadinessCheck{Status: ReadinessFail, Code: "ACTION_DEFINITIONS_PRESENT", Summary: "No action definitions configured"})
	} else {
		checks = append(checks, ReadinessCheck{Status: ReadinessPass, Code: "ACTION_DEFINITIONS_PRESENT", Summary: "Action definitions are configured", Detail: fmt.Sprintf("count=%d", len(definitions))})
	}

	boundIDs := map[string]struct{}{}
	for _, channel := range channels {
		if strings.TrimSpace(channel.OnTriggerActionID) != "" {
			boundIDs[channel.OnTriggerActionID] = struct{}{}
		}
		if strings.TrimSpace(channel.OnNormalActionID) != "" {
			boundIDs[channel.OnNormalActionID] = struct{}{}
		}
	}
	missing := []string{}
	for id := range boundIDs {
		if _, ok := definitionByID[id]; !ok {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessFail,
			Code:    "ACTION_BOUND_REFERENCES",
			Summary: "Input action bindings reference undefined actions",
			Detail:  strings.Join(missing, ", "),
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "ACTION_BOUND_REFERENCES",
			Summary: "All input action bindings resolve to configured action definitions",
		})
	}

	incomplete := []string{}
	for _, definition := range definitions {
		reasons := []string{}
		if strings.TrimSpace(string(definition.Severity)) == "" {
			reasons = append(reasons, "missing severity")
		}
		if strings.TrimSpace(definition.TargetSelector) == "" {
			reasons = append(reasons, "missing target selector")
		}
		if strings.TrimSpace(definition.TemplateSelector) == "" {
			reasons = append(reasons, "missing template selector")
		}
		if len(definition.Steps) == 0 || strings.TrimSpace(definition.Steps[0].TemplateActionKey) == "" {
			reasons = append(reasons, "missing template action key step")
		}
		if len(reasons) > 0 {
			incomplete = append(incomplete, definition.ID+": "+strings.Join(reasons, "; "))
		}
	}
	if len(incomplete) > 0 {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessWarn,
			Code:    "ACTION_FIELD_COMPLETENESS",
			Summary: "One or more action definitions are incomplete for field-test review",
			Detail:  strings.Join(incomplete, " | "),
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "ACTION_FIELD_COMPLETENESS",
			Summary: "Action definitions include severity/selectors/steps",
		})
	}

	checks = append(checks, ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    "ACTION_RETRY_ESCALATION_STORAGE",
		Summary: "Retry/escalation fields are stored for configuration review",
		Detail:  "execution behavior is intentionally not implemented in this phase",
	})
	return ReadinessSection{Name: "Actions", Checks: checks}
}

func (s *Server) buildEGMsReadinessSection(records []egms.EGMRecord, templateByID map[string]templates.G2STemplate) ReadinessSection {
	checks := []ReadinessCheck{}
	if len(records) == 0 {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessWarn,
			Code:    "EGM_RECORDS_PRESENT",
			Summary: "No EGM records are configured",
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "EGM_RECORDS_PRESENT",
			Summary: "EGM records are configured",
			Detail:  fmt.Sprintf("count=%d", len(records)),
		})
	}

	emergencyEnabled := 0
	disabled := 0
	missingTemplates := []string{}
	for _, record := range records {
		if !record.Enabled {
			disabled++
		}
		if record.EmergencyEnabled {
			emergencyEnabled++
		}
		if strings.TrimSpace(record.TemplateID) == "" {
			missingTemplates = append(missingTemplates, record.EGMID+"(none)")
			continue
		}
		if _, ok := templateByID[record.TemplateID]; !ok {
			missingTemplates = append(missingTemplates, record.EGMID+"("+record.TemplateID+")")
		}
	}
	checks = append(checks, ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    "EGM_COUNTS",
		Summary: "EGM enabled/disabled counts",
		Detail:  fmt.Sprintf("emergency_enabled=%d disabled=%d", emergencyEnabled, disabled),
	})
	if len(missingTemplates) > 0 {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessWarn,
			Code:    "EGM_TEMPLATE_ASSIGNMENT",
			Summary: "Some EGMs are missing template assignments",
			Detail:  strings.Join(missingTemplates, ", "),
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "EGM_TEMPLATE_ASSIGNMENT",
			Summary: "All configured EGMs have template assignments",
		})
	}
	return ReadinessSection{Name: "EGM Registry", Checks: checks}
}

func (s *Server) buildTemplatesReadinessSection(definitions []actions.ActionDefinition, templateRows []templates.G2STemplate, activeVersionByTemplate map[string]*templates.G2STemplateVersion) ReadinessSection {
	checks := []ReadinessCheck{}
	if len(templateRows) == 0 {
		checks = append(checks, ReadinessCheck{Status: ReadinessWarn, Code: "TEMPLATE_PRESENT", Summary: "No templates configured"})
		return ReadinessSection{Name: "Templates", Checks: checks}
	}
	checks = append(checks, ReadinessCheck{Status: ReadinessPass, Code: "TEMPLATE_PRESENT", Summary: "Templates are configured", Detail: fmt.Sprintf("count=%d", len(templateRows))})

	missingActive := []string{}
	stepKeys := map[string]struct{}{}
	for _, definition := range definitions {
		for _, step := range definition.Steps {
			key := strings.TrimSpace(step.TemplateActionKey)
			if key != "" {
				stepKeys[key] = struct{}{}
			}
		}
	}

	keysPresent := map[string]bool{}
	for _, tpl := range templateRows {
		activeVersion := activeVersionByTemplate[tpl.ID]
		if activeVersion == nil {
			missingActive = append(missingActive, tpl.ID)
			continue
		}
		if strings.TrimSpace(activeVersion.ActionsJSON) == "" {
			missingActive = append(missingActive, tpl.ID+"(empty actions_json)")
			continue
		}
		doc, err := g2sengine.ParseActionTemplateDocument(activeVersion.ActionsJSON)
		if err != nil {
			missingActive = append(missingActive, tpl.ID+"(invalid actions_json)")
			continue
		}
		for key := range stepKeys {
			if _, ok := doc.Actions[key]; ok {
				keysPresent[key] = true
			}
		}
	}

	if len(missingActive) > 0 {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessWarn,
			Code:    "TEMPLATE_ACTIVE_VERSION",
			Summary: "Some templates are missing active versions or valid actions JSON",
			Detail:  strings.Join(missingActive, ", "),
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "TEMPLATE_ACTIVE_VERSION",
			Summary: "Active template versions are available",
		})
	}

	missingKeys := []string{}
	for key := range stepKeys {
		if !keysPresent[key] {
			missingKeys = append(missingKeys, key)
		}
	}
	sort.Strings(missingKeys)
	if len(missingKeys) > 0 {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessWarn,
			Code:    "TEMPLATE_ACTION_KEYS_RENDERABLE",
			Summary: "Configured action step keys are not renderable from active templates",
			Detail:  strings.Join(missingKeys, ", "),
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "TEMPLATE_ACTION_KEYS_RENDERABLE",
			Summary: "Configured action step keys are renderable from active templates",
		})
	}

	checks = append(checks, ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    "TEMPLATE_MATCHER_PLACEHOLDERS",
		Summary: "Expected/failure matcher fields are configuration placeholders in this phase",
		Detail:  "review/export only; no confirmation execution in this phase",
	})

	return ReadinessSection{Name: "Templates", Checks: checks}
}

func (s *Server) buildCommsReadinessSection(rows []g2sengine.MessageJournalEntry) ReadinessSection {
	checks := []ReadinessCheck{}
	if len(rows) == 0 {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessInfo,
			Code:    "COMMS_HISTORY_PRESENT",
			Summary: "No message journal history yet",
		})
		return ReadinessSection{Name: "Comms Journal", Checks: checks}
	}
	checks = append(checks, ReadinessCheck{
		Status:  ReadinessPass,
		Code:    "COMMS_HISTORY_PRESENT",
		Summary: "Message journal history exists",
		Detail:  fmt.Sprintf("rows=%d", len(rows)),
	})

	latestOutbound := rows[0]
	for _, row := range rows {
		if row.Direction == g2sengine.DirectionOutbound {
			latestOutbound = row
			break
		}
	}
	checks = append(checks, ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    "COMMS_LATEST_OUTBOUND",
		Summary: "Latest outbound message status",
		Detail:  string(latestOutbound.Result),
	})

	lastGate := "not_observed"
	for _, row := range rows {
		if row.Result == g2sengine.MessageResultSendBlocked || row.Result == g2sengine.MessageResultSendSucceeded || row.Result == g2sengine.MessageResultSendFailed {
			lastGate = string(row.Result)
			break
		}
	}
	checks = append(checks, ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    "COMMS_LAST_SEND_GATE_STATUS",
		Summary: "Latest send gate result observed in comms journal",
		Detail:  lastGate,
	})
	return ReadinessSection{Name: "Comms Journal", Checks: checks}
}

func (s *Server) buildAuditReadinessSection(rows []audit.AuditTimelineEntry) ReadinessSection {
	checks := []ReadinessCheck{}
	if len(rows) == 0 {
		checks = append(checks, ReadinessCheck{Status: ReadinessInfo, Code: "AUDIT_HISTORY_PRESENT", Summary: "No audit timeline history yet"})
		return ReadinessSection{Name: "Audit Timeline", Checks: checks}
	}
	checks = append(checks, ReadinessCheck{Status: ReadinessPass, Code: "AUDIT_HISTORY_PRESENT", Summary: "Audit timeline entries exist", Detail: fmt.Sprintf("rows=%d", len(rows))})

	checks = append(checks, latestAuditCheck(rows, "AUDIT_LATEST_INPUT_TRANSITION", "Latest input transition", audit.EventTypeInputTransition))
	checks = append(checks, latestAuditCheck(rows, "AUDIT_LATEST_ACTION_QUEUED", "Latest action queued", audit.EventTypeActionQueued))
	checks = append(checks, latestAuditCheck(rows, "AUDIT_LATEST_MANUAL_CLEAR", "Latest manual clear", audit.EventTypeInputLatchClearSucceeded))

	sendProof := "not observed"
	for _, row := range rows {
		if row.EventType == audit.EventTypeMessageSendBlocked || row.EventType == audit.EventTypeMessageSendSucceeded || row.EventType == audit.EventTypeMessageSendFailed {
			sendProof = row.EventType + " @ " + fmtTime(row.OccurredAt)
			break
		}
	}
	checks = append(checks, ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    "AUDIT_LATEST_SEND_PROOF",
		Summary: "Latest send blocked/succeeded proof",
		Detail:  sendProof,
	})
	return ReadinessSection{Name: "Audit Timeline", Checks: checks}
}

func (s *Server) buildSettingsReadinessSection(messages []g2sengine.MessageJournalEntry, certs []model.CertificateInventory) ReadinessSection {
	checks := []ReadinessCheck{}
	if s.Options.RealSendDefaultDisabled {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessPass,
			Code:    "SETTINGS_REAL_SEND_GATED",
			Summary: "Real send default is gated/disabled",
			Detail:  s.Options.TransportGateSummary,
		})
	} else {
		checks = append(checks, ReadinessCheck{
			Status:  ReadinessFail,
			Code:    "SETTINGS_REAL_SEND_GATED",
			Summary: "Real send default is not marked gated",
		})
	}
	checks = append(checks, ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    "SETTINGS_CAPTURE_POLICY",
		Summary: "Current capture behavior policy",
		Detail:  s.Options.CapturePolicySummary,
	})
	checks = append(checks, ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    "SETTINGS_DB_BIND",
		Summary: "Database path and bind address",
		Detail:  fmt.Sprintf("db=%s bind=%s", defaultString(s.Options.DatabasePath, "unknown"), defaultString(s.Options.BindAddress, "unknown")),
	})
	statusSummary := summarizeCertStatuses(certs)
	checks = append(checks, ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    "SETTINGS_CERT_STATUS",
		Summary: "Certificate inventory status summary",
		Detail:  statusSummary,
	})

	lastSend := "none"
	for _, row := range messages {
		if row.Result == g2sengine.MessageResultSendBlocked || row.Result == g2sengine.MessageResultSendSucceeded || row.Result == g2sengine.MessageResultSendFailed {
			lastSend = string(row.Result)
			break
		}
	}
	checks = append(checks, ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    "SETTINGS_LAST_SEND_RESULT",
		Summary: "Last send-prepared result",
		Detail:  lastSend,
	})
	return ReadinessSection{Name: "Settings / Safety", Checks: checks}
}

func latestAuditCheck(rows []audit.AuditTimelineEntry, code string, summary string, eventType string) ReadinessCheck {
	for _, row := range rows {
		if row.EventType == eventType {
			return ReadinessCheck{
				Status:  ReadinessInfo,
				Code:    code,
				Summary: summary,
				Detail:  row.EventType + " @ " + fmtTime(row.OccurredAt),
			}
		}
	}
	return ReadinessCheck{
		Status:  ReadinessInfo,
		Code:    code,
		Summary: summary,
		Detail:  "not observed",
	}
}

func summarizeCertStatuses(certs []model.CertificateInventory) string {
	if len(certs) == 0 {
		return "none"
	}
	counts := map[string]int{}
	for _, row := range certs {
		counts[row.Status]++
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}

func (s *Server) buildExportPackage(ctx context.Context, report ReadinessReport) (FieldTestExportPackage, error) {
	channels, err := s.Store.ListInputChannels(ctx)
	if err != nil {
		return FieldTestExportPackage{}, err
	}
	inputSnapshots := make([]FieldTestInputSnapshot, 0, len(channels))
	for _, channel := range channels {
		runtimeState, runtimeErr := s.Store.GetInputRuntimeState(ctx, channel.ID)
		if runtimeErr != nil {
			return FieldTestExportPackage{}, runtimeErr
		}
		inputSnapshots = append(inputSnapshots, FieldTestInputSnapshot{
			Channel:      channel,
			RuntimeState: runtimeState,
		})
	}

	definitions, err := s.Store.ListActionDefinitions(ctx)
	if err != nil {
		return FieldTestExportPackage{}, err
	}
	previews := make([]FieldTestActionPreview, 0, len(definitions))
	planner := actionplanner.Planner{Store: s.Store}
	for _, definition := range definitions {
		plan, planErr := planner.BuildPlanForDefinition(ctx, definition)
		preview := FieldTestActionPreview{ActionID: definition.ID}
		if planErr != nil {
			preview.Error = planErr.Error()
		} else {
			preview.Plan = plan
		}
		previews = append(previews, preview)
	}

	records, err := s.Store.ListEGMRecords(ctx)
	if err != nil {
		return FieldTestExportPackage{}, err
	}

	templateRows, err := s.Store.ListG2STemplates(ctx)
	if err != nil {
		return FieldTestExportPackage{}, err
	}
	templateSnapshots := make([]FieldTestTemplateSnapshot, 0, len(templateRows))
	for _, tpl := range templateRows {
		activeVersion, activeErr := s.Store.GetActiveG2STemplateVersion(ctx, tpl.ID)
		if activeErr != nil {
			return FieldTestExportPackage{}, activeErr
		}
		versions, versionsErr := s.Store.ListG2STemplateVersions(ctx, tpl.ID)
		if versionsErr != nil {
			return FieldTestExportPackage{}, versionsErr
		}
		templateSnapshots = append(templateSnapshots, FieldTestTemplateSnapshot{
			Template:      tpl,
			ActiveVersion: activeVersion,
			Versions:      versions,
		})
	}

	messageRows, err := s.Store.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 500})
	if err != nil {
		return FieldTestExportPackage{}, err
	}
	auditRows, err := s.Store.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{Limit: 500})
	if err != nil {
		return FieldTestExportPackage{}, err
	}
	certRows, err := s.Store.ListCertificateInventory(ctx)
	if err != nil {
		return FieldTestExportPackage{}, err
	}

	exportPkg := FieldTestExportPackage{
		GeneratedAt:     time.Now().UTC(),
		SafetyGate:      map[string]any{},
		Inputs:          inputSnapshots,
		Actions:         definitions,
		ActionPreviews:  previews,
		EGMs:            records,
		Templates:       templateSnapshots,
		MessageJournal:  messageRows,
		AuditTimeline:   auditRows,
		CertificateMeta: certRows,
		Readiness:       report,
	}
	exportPkg.SafetyGate["real_send_default"] = "gated"
	exportPkg.SafetyGate["transport_gate"] = s.Options.TransportGateSummary
	exportPkg.SafetyGate["capture_policy"] = s.Options.CapturePolicySummary
	exportPkg.SafetyGate["phase_send_policy"] = "no real EGM send approved from field-test UI"
	exportPkg.SafetyGate["bind_address"] = s.Options.BindAddress
	exportPkg.SafetyGate["database_path"] = s.Options.DatabasePath
	return exportPkg, nil
}

func (r ReadinessReport) MarshalJSON() ([]byte, error) {
	type alias ReadinessReport
	return json.Marshal(alias(r))
}
