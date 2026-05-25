package inbound

import (
	"context"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type InboundMessage struct {
	ReceivedAt    time.Time
	FromEndpoint  string
	ToEndpoint    string
	RemoteAddr    string
	EGMID         string
	ActionRunID   string
	MessageType   string
	RawPayload    string
	Headers       map[string]string
	QueryParams   map[string]string
	HandlerRuleID string
}

type ProcessResult struct {
	MessageID      int64
	EGMID          string
	ActionRunID    string
	MatchOutcome   string
	TargetUpdated  bool
	TargetStatus   string
	AuditEntryIDs  []int64
	Warnings       []string
	Correlated     bool
	CorrelationRef string
}

type Store interface {
	RecordMessageJournalEntry(ctx context.Context, entry g2sengine.MessageJournalEntry) (int64, error)
	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)
	GetActionRun(ctx context.Context, id string) (*actions.ActionRun, error)
	UpdateActionRun(ctx context.Context, run actions.ActionRun) error
	ListActionRuns(ctx context.Context, query store.ActionRunListQuery) ([]actions.ActionRun, error)
	ListActionTargetResults(ctx context.Context, actionRunID string) ([]actions.ActionTargetResult, error)
	UpdateActionTargetResult(ctx context.Context, row actions.ActionTargetResult) error
	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	GetEGMRecord(ctx context.Context, egmID string) (*egms.EGMRecord, error)
	GetActiveG2STemplateVersion(ctx context.Context, templateID string) (*templates.G2STemplateVersion, error)
}

type Service struct {
	Store Store
	Clock func() time.Time
}
