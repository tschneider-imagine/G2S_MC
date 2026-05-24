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

type TemplatesListResponse struct {
	Templates []templates.G2STemplate `json:"templates"`
}

type TemplateUpsertRequest struct {
	Template templates.G2STemplate `json:"template"`
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
