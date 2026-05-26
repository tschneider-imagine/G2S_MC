package pendingdeliveryruntime

import (
	"context"
	"log"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/pendingdelivery"
)

const DefaultSweepInterval = 5 * time.Second

type Options struct {
	Enabled  bool
	Interval time.Duration
}

type Sweeper interface {
	SweepWaitingConfirmations(ctx context.Context, now time.Time) (pendingdelivery.SweepResult, error)
}

type Logger interface {
	Printf(format string, v ...any)
}

type Runtime struct {
	Sweeper Sweeper
	Logger  Logger
	Clock   func() time.Time
	Options Options
}

func (r *Runtime) Run(ctx context.Context) error {
	options := r.Options
	if !options.Enabled {
		return nil
	}
	if options.Interval <= 0 {
		options.Interval = DefaultSweepInterval
	}
	if r.Sweeper == nil {
		return nil
	}

	r.logf("pending_delivery_sweep started interval=%s", options.Interval)
	ticker := time.NewTicker(options.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logf("pending_delivery_sweep stopped")
			return nil
		case <-ticker.C:
			if err := r.runOnce(ctx); err != nil {
				r.logf("pending_delivery_sweep error=%v", err)
			}
		}
	}
}

func (r *Runtime) runOnce(ctx context.Context) error {
	now := time.Now().UTC()
	if r.Clock != nil {
		now = r.Clock().UTC()
	}
	result, err := r.Sweeper.SweepWaitingConfirmations(ctx, now)
	if err != nil {
		return err
	}
	if result.CheckedRuns == 0 &&
		result.MessagesExpired == 0 &&
		result.MessagesReprepared == 0 &&
		result.MessagesSuperseded == 0 &&
		result.TargetsFailed == 0 &&
		result.RunsFailed == 0 &&
		result.RunsEscalated == 0 &&
		len(result.Warnings) == 0 {
		return nil
	}
	r.logf(
		"pending_delivery_sweep checked_runs=%d messages_expired=%d messages_reprepared=%d messages_superseded=%d targets_failed=%d runs_failed=%d runs_escalated=%d warnings=%d",
		result.CheckedRuns,
		result.MessagesExpired,
		result.MessagesReprepared,
		result.MessagesSuperseded,
		result.TargetsFailed,
		result.RunsFailed,
		result.RunsEscalated,
		len(result.Warnings),
	)
	return nil
}

func (r *Runtime) logf(format string, values ...any) {
	logger := r.Logger
	if logger == nil {
		logger = log.Default()
	}
	logger.Printf(format, values...)
}
