package appliance

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actionexecutor"
	"github.com/tschneider-imagine/G2S_MC/internal/actionruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/inputpoller"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

const defaultPollInterval = 100 * time.Millisecond

type RuntimeOptions struct {
	Enabled           bool
	PollInterval      time.Duration
	SeedDefaultInputs bool
	ExecuteActions    bool
	DeliverySettings  g2stransport.DeliverySettings
	Actor             string
}

type Poller interface {
	PollOnce(ctx context.Context) (inputpoller.PollResult, error)
}

type Queuer interface {
	QueueActionRun(ctx context.Context, request actionruntime.QueueRequest) (actionruntime.QueueResult, error)
}

type Executor interface {
	Execute(ctx context.Context, request actionexecutor.ExecuteRequest) (actionexecutor.ExecuteResult, error)
}

type Logger interface {
	Printf(format string, v ...any)
}

type Runtime struct {
	Poller              Poller
	Queuer              Queuer
	Executor            Executor
	SeedDefaultInputsFn func(context.Context) error
	Logger              Logger
	Clock               func() time.Time
	Options             RuntimeOptions
}

func (r *Runtime) Run(ctx context.Context) error {
	options := r.Options
	if !options.Enabled {
		return nil
	}
	if options.PollInterval <= 0 {
		options.PollInterval = defaultPollInterval
	}
	options.DeliverySettings = options.DeliverySettings.Normalize()
	options.Actor = strings.TrimSpace(options.Actor)
	if options.Actor == "" {
		options.Actor = "g2s-mute"
	}

	if r.Poller == nil {
		return fmt.Errorf("poller is required")
	}
	if r.Queuer == nil {
		return fmt.Errorf("queuer is required")
	}
	if options.ExecuteActions && r.Executor == nil {
		return fmt.Errorf("executor is required when execute_actions is enabled")
	}

	if options.SeedDefaultInputs && r.SeedDefaultInputsFn != nil {
		if err := r.SeedDefaultInputsFn(ctx); err != nil {
			return fmt.Errorf("seed default input channels: %w", err)
		}
		r.logf("input_runtime defaults_seeded=true")
	}

	r.logf(
		"input_runtime started interval=%s execute_actions=%t delivery_mode=%s allow_delivery=%t capture_only=%t timeout_ms=%d",
		options.PollInterval,
		options.ExecuteActions,
		options.DeliverySettings.Mode,
		options.DeliverySettings.AllowDelivery,
		options.DeliverySettings.CaptureOnly,
		options.DeliverySettings.TimeoutMS,
	)

	ticker := time.NewTicker(options.PollInterval)
	defer ticker.Stop()

	lastPollError := ""
	for {
		if ctx.Err() != nil {
			r.logf("input_runtime stopped")
			return nil
		}

		if err := r.runOnce(ctx, options); err != nil {
			if !errors.Is(err, context.Canceled) {
				errText := err.Error()
				if errText != lastPollError {
					r.logf("input_runtime poll_error=%s", errText)
					lastPollError = errText
				}
			}
		} else {
			lastPollError = ""
		}

		select {
		case <-ctx.Done():
			r.logf("input_runtime stopped")
			return nil
		case <-ticker.C:
		}
	}
}

func (r *Runtime) runOnce(ctx context.Context, options RuntimeOptions) error {
	result, err := r.Poller.PollOnce(ctx)
	if err != nil {
		return err
	}

	samples := make([]inputpoller.PollSampleResult, len(result.Samples))
	copy(samples, result.Samples)
	sort.Slice(samples, func(i, j int) bool {
		if samples[i].GPIOChannel == samples[j].GPIOChannel {
			return samples[i].InputID < samples[j].InputID
		}
		return samples[i].GPIOChannel < samples[j].GPIOChannel
	})

	for _, sample := range samples {
		if strings.TrimSpace(sample.Error) != "" {
			r.logf("input_runtime sample_error input=%s gpio=%s error=%s", sample.InputID, sample.GPIOChannel, sample.Error)
			continue
		}
		if !sample.Transitioned {
			continue
		}

		r.logf(
			"input_transition input=%s gpio=%s raw=%s derived=%s transition_id=%d action_id=%s",
			sample.InputID,
			sample.GPIOChannel,
			sample.RawState,
			sample.DerivedState,
			sample.TransitionID,
			sample.ActionQueuedID,
		)

		actionID := strings.TrimSpace(sample.ActionQueuedID)
		if actionID == "" {
			continue
		}

		queueResult, queueErr := r.Queuer.QueueActionRun(ctx, actionruntime.QueueRequest{
			InputTransition: inputs.InputTransition{
				ID:             sample.TransitionID,
				InputChannelID: sample.InputID,
				TransitionAt:   result.ObservedAt,
			},
			ActionID:      actionID,
			TriggerReason: fmt.Sprintf("input transition %d", sample.TransitionID),
			Actor:         options.Actor,
			QueuedAt:      result.ObservedAt,
		})
		if queueErr != nil {
			r.logf("action_queue_error input=%s transition_id=%d action_id=%s error=%s", sample.InputID, sample.TransitionID, actionID, queueErr.Error())
			continue
		}
		if !queueResult.Queued || queueResult.ActionRun == nil {
			r.logf("action_queue_skipped input=%s transition_id=%d action_id=%s reason=%s", sample.InputID, sample.TransitionID, actionID, queueResult.Reason)
			continue
		}

		runID := strings.TrimSpace(queueResult.ActionRun.ID)
		r.logf(
			"action_queued run_id=%s action_id=%s targets=%d warnings=%d",
			runID,
			actionID,
			len(queueResult.TargetResults),
			len(queueResult.PlanWarnings),
		)

		if !options.ExecuteActions {
			continue
		}
		if r.Executor == nil {
			r.logf("action_execution_failed run_id=%s error=executor_not_configured", runID)
			continue
		}

		executeResult, executeErr := r.Executor.Execute(ctx, actionexecutor.ExecuteRequest{
			ActionRunID: runID,
			Actor:       options.Actor,
			RequestedAt: result.ObservedAt,
			Delivery:    options.DeliverySettings,
		})
		if executeErr != nil {
			r.logf("action_execution_failed run_id=%s error=%s", runID, executeErr.Error())
			continue
		}

		r.logf(
			"action_executed run_id=%s status=%s confirmed=%d failed=%d attempts=%d",
			executeResult.ActionRun.ID,
			executeResult.ActionRun.Status,
			executeResult.ActionRun.ConfirmedCount,
			executeResult.ActionRun.FailedCount,
			len(executeResult.Attempts),
		)
		if executeResult.EscalationRun != nil {
			r.logf("escalation_queued run_id=%s action_id=%s", executeResult.EscalationRun.ID, executeResult.EscalationRun.ActionDefinitionID)
		}
	}

	for _, msg := range result.Errors {
		if strings.TrimSpace(msg) == "" {
			continue
		}
		r.logf("input_runtime poll_detail=%s", msg)
	}
	return nil
}

func (r *Runtime) logf(format string, values ...any) {
	logger := r.Logger
	if logger == nil {
		logger = log.Default()
	}
	logger.Printf(format, values...)
}
