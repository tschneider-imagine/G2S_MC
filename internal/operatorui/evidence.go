package operatorui

import (
	"context"
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

type OperatorInputSnapshot struct {
	Channel      inputs.InputChannel             `json:"channel"`
	RuntimeState *inputruntime.InputRuntimeState `json:"runtime_state,omitempty"`
}

type OperatorTemplateSnapshot struct {
	Template      templates.G2STemplate          `json:"template"`
	ActiveVersion *templates.G2STemplateVersion  `json:"active_version,omitempty"`
	Versions      []templates.G2STemplateVersion `json:"versions"`
}

type OperatorActionPreview struct {
	ActionID string                    `json:"action_id"`
	Plan     *actionplanner.ActionPlan `json:"plan,omitempty"`
	Error    string                    `json:"error,omitempty"`
}

type OperatorEvidencePackage struct {
	GeneratedAt      time.Time                       `json:"generated_at"`
	TransportStatus  map[string]any                  `json:"transport_status"`
	Inputs           []OperatorInputSnapshot         `json:"inputs"`
	Actions          []actions.ActionDefinition      `json:"actions"`
	ActionPreviews   []OperatorActionPreview         `json:"action_previews"`
	EGMs             []egms.EGMRecord                `json:"egms"`
	Templates        []OperatorTemplateSnapshot      `json:"templates"`
	MessageJournal   []g2sengine.MessageJournalEntry `json:"message_journal"`
	AuditTimeline    []audit.AuditTimelineEntry      `json:"audit_timeline"`
	CertificateMeta  []model.CertificateInventory    `json:"certificate_inventory"`
	ListenerBindAddr string                          `json:"listener_bind_address"`
	DatabasePath     string                          `json:"database_path"`
}

func (s *Server) buildEvidencePackage(ctx context.Context) (OperatorEvidencePackage, error) {
	channels, err := s.Store.ListInputChannels(ctx)
	if err != nil {
		return OperatorEvidencePackage{}, err
	}
	inputSnapshots := make([]OperatorInputSnapshot, 0, len(channels))
	for _, channel := range channels {
		runtimeState, runtimeErr := s.Store.GetInputRuntimeState(ctx, channel.ID)
		if runtimeErr != nil {
			return OperatorEvidencePackage{}, runtimeErr
		}
		inputSnapshots = append(inputSnapshots, OperatorInputSnapshot{
			Channel:      channel,
			RuntimeState: runtimeState,
		})
	}

	definitions, err := s.Store.ListActionDefinitions(ctx)
	if err != nil {
		return OperatorEvidencePackage{}, err
	}
	previews := make([]OperatorActionPreview, 0, len(definitions))
	planner := actionplanner.Planner{Store: s.Store}
	for _, definition := range definitions {
		plan, planErr := planner.BuildPlanForDefinition(ctx, definition)
		preview := OperatorActionPreview{ActionID: definition.ID}
		if planErr != nil {
			preview.Error = planErr.Error()
		} else {
			preview.Plan = plan
		}
		previews = append(previews, preview)
	}

	records, err := s.Store.ListEGMRecords(ctx)
	if err != nil {
		return OperatorEvidencePackage{}, err
	}

	templateRows, err := s.Store.ListG2STemplates(ctx)
	if err != nil {
		return OperatorEvidencePackage{}, err
	}
	templateSnapshots := make([]OperatorTemplateSnapshot, 0, len(templateRows))
	for _, tpl := range templateRows {
		activeVersion, activeErr := s.Store.GetActiveG2STemplateVersion(ctx, tpl.ID)
		if activeErr != nil {
			return OperatorEvidencePackage{}, activeErr
		}
		versions, versionsErr := s.Store.ListG2STemplateVersions(ctx, tpl.ID)
		if versionsErr != nil {
			return OperatorEvidencePackage{}, versionsErr
		}
		templateSnapshots = append(templateSnapshots, OperatorTemplateSnapshot{
			Template:      tpl,
			ActiveVersion: activeVersion,
			Versions:      versions,
		})
	}

	messageRows, err := s.Store.ListMessageJournalEntries(ctx, store.MessageJournalListQuery{Limit: 500})
	if err != nil {
		return OperatorEvidencePackage{}, err
	}
	auditRows, err := s.Store.ListAuditTimelineEntries(ctx, store.AuditTimelineListQuery{Limit: 500})
	if err != nil {
		return OperatorEvidencePackage{}, err
	}
	certRows, err := s.Store.ListCertificateInventory(ctx)
	if err != nil {
		return OperatorEvidencePackage{}, err
	}

	evidence := OperatorEvidencePackage{
		GeneratedAt:      time.Now().UTC(),
		TransportStatus:  map[string]any{},
		Inputs:           inputSnapshots,
		Actions:          definitions,
		ActionPreviews:   previews,
		EGMs:             records,
		Templates:        templateSnapshots,
		MessageJournal:   messageRows,
		AuditTimeline:    auditRows,
		CertificateMeta:  certRows,
		ListenerBindAddr: s.Options.BindAddress,
		DatabasePath:     s.Options.DatabasePath,
	}
	evidence.TransportStatus["sending_status"] = "disabled"
	evidence.TransportStatus["transport_status"] = s.Options.TransportStatusSummary
	evidence.TransportStatus["capture_endpoint_status"] = s.Options.CaptureStatusSummary
	return evidence, nil
}
