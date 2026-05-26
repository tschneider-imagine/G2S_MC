package inbound

import (
	"context"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/pendingdelivery"
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
	OfferedMessage *OfferedMessage
}

type OfferedMessage struct {
	MessageID       int64
	ActionRunID     string
	ActionStepID    string
	TemplateID      string
	TemplateVersion string
	MessageType     string
	RawPayload      string
	OfferedAt       time.Time
	OfferCount      int
}

type Store interface {
	RecordMessageJournalEntry(ctx context.Context, entry g2sengine.MessageJournalEntry) (int64, error)
	UpdateMessageJournalHandlerRule(ctx context.Context, id int64, handlerRuleID string) error
	UpdateMessageJournalResult(ctx context.Context, id int64, result g2sengine.MessageResult, errText string, responseExcerpt string, httpStatusCode int, latencyMS int, transportMode string, sentAt *time.Time, completedAt *time.Time) error
	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)
	ListMessageJournalEntries(ctx context.Context, query store.MessageJournalListQuery) ([]g2sengine.MessageJournalEntry, error)
	GetActionRun(ctx context.Context, id string) (*actions.ActionRun, error)
	UpdateActionRun(ctx context.Context, run actions.ActionRun) error
	ListActionRuns(ctx context.Context, query store.ActionRunListQuery) ([]actions.ActionRun, error)
	ListActionTargetResults(ctx context.Context, actionRunID string) ([]actions.ActionTargetResult, error)
	UpdateActionTargetResult(ctx context.Context, row actions.ActionTargetResult) error
	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	GetEGMRecord(ctx context.Context, egmID string) (*egms.EGMRecord, error)
	UpsertEGMRecord(ctx context.Context, record egms.EGMRecord) error
	GetActiveG2STemplateVersion(ctx context.Context, templateID string) (*templates.G2STemplateVersion, error)
	ListEnabledHandlerRules(ctx context.Context, limit int) ([]g2sengine.HandlerRule, error)
}

type PendingDeliveryService interface {
	HandleClientContact(ctx context.Context, req pendingdelivery.ContactRequest) (pendingdelivery.ContactResult, error)
}

type Service struct {
	Store           Store
	Clock           func() time.Time
	PendingDelivery PendingDeliveryService
}
