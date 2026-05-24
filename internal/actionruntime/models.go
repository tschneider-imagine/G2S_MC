package actionruntime

import (
	"context"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actionplanner"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type QueueRequest struct {
	InputTransition inputs.InputTransition
	ActionID        string
	TriggerReason   string
	Actor           string
	QueuedAt        time.Time
}

type QueueResult struct {
	Queued        bool
	ActionRun     *actions.ActionRun
	TargetResults []actions.ActionTargetResult
	PlanWarnings  []actionplanner.PlanningWarning
	AuditEntryID  int64
	Reason        string
}

type Store interface {
	GetActionDefinition(ctx context.Context, id string) (*actions.ActionDefinition, error)
	ListEGMRecords(ctx context.Context) ([]egms.EGMRecord, error)
	GetG2STemplate(ctx context.Context, id string) (*templates.G2STemplate, error)
	GetEGMGroup(ctx context.Context, id string) (*egms.EGMGroup, error)
	ListEGMGroups(ctx context.Context) ([]egms.EGMGroup, error)

	CreateActionRun(ctx context.Context, run actions.ActionRun) (actions.ActionRun, error)
	CreateActionTargetResult(ctx context.Context, result actions.ActionTargetResult) (actions.ActionTargetResult, error)

	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)
}

