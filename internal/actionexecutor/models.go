package actionexecutor

import (
	"context"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type ExecuteRequest struct {
	ActionRunID string
	Actor       string
	RequestedAt time.Time
	MaxTargets  int
	Delivery    g2stransport.DeliverySettings
	Topology    string
}

type ExecutionAttemptSummary struct {
	EGMID           string `json:"egm_id"`
	ActionStepID    string `json:"action_step_id"`
	TemplateID      string `json:"template_id"`
	TemplateVersion string `json:"template_version"`
	MessageID       int64  `json:"message_id"`
	Attempt         int    `json:"attempt"`
	DeliveryResult  string `json:"delivery_result"`
	MatchOutcome    string `json:"match_outcome"`
	Error           string `json:"error,omitempty"`
}

type ExecuteResult struct {
	ActionRun     actions.ActionRun            `json:"action_run"`
	TargetResults []actions.ActionTargetResult `json:"target_results"`
	Attempts      []ExecutionAttemptSummary    `json:"attempts"`
	AuditEntryIDs []int64                      `json:"audit_entry_ids"`
	Status        string                       `json:"status"`
	Warnings      []string                     `json:"warnings,omitempty"`
	EscalationRun *actions.ActionRun           `json:"escalation_run,omitempty"`
}

type Store interface {
	GetActionRun(ctx context.Context, id string) (*actions.ActionRun, error)
	UpdateActionRun(ctx context.Context, run actions.ActionRun) error
	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)

	ListActionTargetResults(ctx context.Context, actionRunID string) ([]actions.ActionTargetResult, error)
	UpdateActionTargetResult(ctx context.Context, row actions.ActionTargetResult) error

	GetEGMRecord(ctx context.Context, egmID string) (*egms.EGMRecord, error)
	GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error)
	GetActiveG2STemplateVersion(ctx context.Context, templateID string) (*templates.G2STemplateVersion, error)

	RecordMessageJournalEntry(ctx context.Context, entry g2sengine.MessageJournalEntry) (int64, error)
	UpdateMessageJournalResult(ctx context.Context, id int64, result g2sengine.MessageResult, errText string, responseExcerpt string, httpStatusCode int, latencyMS int, transportMode string, sentAt *time.Time, completedAt *time.Time) error
	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)

	// Needed to queue escalation runs through the existing queue path.
	ListEGMRecords(ctx context.Context) ([]egms.EGMRecord, error)
	GetEGMGroup(ctx context.Context, id string) (*egms.EGMGroup, error)
	ListEGMGroups(ctx context.Context) ([]egms.EGMGroup, error)
	CreateActionRun(ctx context.Context, run actions.ActionRun) (actions.ActionRun, error)
	CreateActionTargetResult(ctx context.Context, result actions.ActionTargetResult) (actions.ActionTargetResult, error)
}
