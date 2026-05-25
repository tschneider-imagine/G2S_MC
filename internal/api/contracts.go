package api

import (
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type InputsListResponse struct {
	Channels []inputs.InputChannel `json:"channels"`
}

type InputUpsertRequest struct {
	Channel inputs.InputChannel `json:"channel"`
}

type InputClearLatchRequest struct {
	Actor  string `json:"actor,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type ActionsListResponse struct {
	Definitions []actions.ActionDefinition `json:"definitions"`
}

type ActionUpsertRequest struct {
	Definition actions.ActionDefinition `json:"definition"`
}

type ActionRunsListResponse struct {
	Runs []actions.ActionRun `json:"runs"`
}

type ActionRunTargetsListResponse struct {
	Targets []actions.ActionTargetResult `json:"targets"`
}

type ActionRunDispatchDryRunRequest struct {
	Actor string `json:"actor,omitempty"`
}

type ActionRunSendPreparedRequest struct {
	TransportMode   string `json:"transport_mode"`
	AllowRealSend   bool   `json:"allow_real_send"`
	CaptureOnlySend bool   `json:"capture_only_send"`
	CaptureEndpoint string `json:"capture_endpoint,omitempty"`
	Actor           string `json:"actor,omitempty"`
}

type ActionRunExecuteRequest struct {
	Actor      string `json:"actor,omitempty"`
	MaxTargets int    `json:"max_targets,omitempty"`
}

type TemplatesListResponse struct {
	Templates []templates.G2STemplate `json:"templates"`
}

type TemplateUpsertRequest struct {
	Template templates.G2STemplate `json:"template"`
}

type TemplateRenderPreviewRequest struct {
	TemplateID        string            `json:"template_id"`
	Version           int               `json:"version,omitempty"`
	TemplateActionKey string            `json:"template_action_key"`
	EGMID             string            `json:"egm_id,omitempty"`
	ActionID          string            `json:"action_id,omitempty"`
	ActionRunID       string            `json:"action_run_id,omitempty"`
	ActionStepID      string            `json:"action_step_id,omitempty"`
	HostID            string            `json:"host_id,omitempty"`
	IPAddress         string            `json:"ip_address,omitempty"`
	EndpointPath      string            `json:"endpoint_path,omitempty"`
	Variables         map[string]string `json:"variables,omitempty"`
}

type TemplateRenderPreviewResponse struct {
	Rendered g2sengine.RenderedMessage `json:"rendered"`
	Warnings []string                  `json:"warnings,omitempty"`
}

type EGMListResponse struct {
	Records []egms.EGMRecord `json:"records"`
}

type MessageJournalListResponse struct {
	Entries []g2sengine.MessageJournalEntry `json:"entries"`
}

type MessageJournalRecordRequest struct {
	Entry g2sengine.MessageJournalEntry `json:"entry"`
}

type AuditTimelineListResponse struct {
	Entries []audit.AuditTimelineEntry `json:"entries"`
}

type AuditTimelineRecordRequest struct {
	Entry audit.AuditTimelineEntry `json:"entry"`
}

type InputStateEnvelope struct {
	Channel      inputs.InputChannel             `json:"channel"`
	RuntimeState *inputruntime.InputRuntimeState `json:"runtime_state,omitempty"`
}
