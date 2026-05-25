package actiondispatch

import (
	"context"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2sengine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type DispatchMode string

const (
	DispatchModeDryRun DispatchMode = "DRY_RUN"
)

type DispatchRequest struct {
	ActionRunID string
	Mode        DispatchMode
	Actor       string
	RequestedAt time.Time
}

type DispatchResult struct {
	ActionRunID      string
	Mode             DispatchMode
	PreparedMessages []g2sengine.MessageJournalEntry
	TargetCount      int
	WarningCount     int
	AuditEntryID     int64
}

type SendPreparedMessagesRequest struct {
	ActionRunID     string
	TransportMode   g2stransport.Mode
	AllowRealSend   bool
	CaptureOnlySend bool
	CaptureEndpoint string
	Actor           string
	RequestedAt     time.Time
	DefaultTimeout  int
}

type SendPreparedMessagesResult struct {
	ActionRunID   string
	TransportMode g2stransport.Mode
	SentCount     int
	FailedCount   int
	BlockedCount  int
	AuditEntryID  int64
}

type Store interface {
	GetActionRun(ctx context.Context, id string) (*actions.ActionRun, error)
	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	ListActionTargetResults(ctx context.Context, actionRunID string) ([]actions.ActionTargetResult, error)
	GetEGMRecord(ctx context.Context, egmID string) (*egms.EGMRecord, error)
	GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error)
	GetActiveG2STemplateVersion(ctx context.Context, templateID string) (*templates.G2STemplateVersion, error)
	UpdateActionRun(ctx context.Context, run actions.ActionRun) error
	RecordMessageJournalEntry(ctx context.Context, entry g2sengine.MessageJournalEntry) (int64, error)
	ListMessageJournalEntries(ctx context.Context, query store.MessageJournalListQuery) ([]g2sengine.MessageJournalEntry, error)
	UpdateMessageJournalResult(ctx context.Context, id int64, result g2sengine.MessageResult, errText string, responseExcerpt string, httpStatusCode int, latencyMS int, transportMode string, sentAt *time.Time, completedAt *time.Time) error
	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)
}
