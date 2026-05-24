package inputpoller

import (
	"context"
	"fmt"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

func (p *Poller) PollOnce(ctx context.Context) (PollResult, error) {
	if p.Store == nil {
		return PollResult{}, fmt.Errorf("store is required")
	}
	if p.Reader == nil {
		return PollResult{}, fmt.Errorf("reader is required")
	}

	clock := p.Clock
	if clock == nil {
		clock = time.Now
	}

	evaluator := p.Evaluator
	if evaluator == nil {
		evaluator = &inputruntime.Evaluator{Store: p.Store, Clock: clock}
		p.Evaluator = evaluator
	}
	if evaluator.Store == nil {
		evaluator.Store = p.Store
	}

	channels, err := p.Store.ListInputChannels(ctx)
	if err != nil {
		return PollResult{}, fmt.Errorf("list input channels: %w", err)
	}

	result := PollResult{
		ObservedAt: clock().UTC(),
		Samples:    make([]PollSampleResult, 0, len(channels)),
	}

	enabled := make([]inputs.InputChannel, 0, len(channels))
	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}
		enabled = append(enabled, channel)

		sample := PollSampleResult{
			InputID:     channel.ID,
			GPIOChannel: channel.GPIOChannel,
		}

		raw, readErr := p.Reader.Read(ctx, channel.GPIOChannel)
		if readErr != nil {
			sample.Error = readErr.Error()
			result.Errors = append(result.Errors, fmt.Sprintf("%s read failed: %v", channel.ID, readErr))
			result.Samples = append(result.Samples, sample)
			continue
		}
		sample.RawState = raw

		observedAt := clock().UTC()
		evalResult, evalErr := evaluator.ApplySample(ctx, inputruntime.InputSample{
			InputID:    channel.ID,
			RawState:   raw,
			ObservedAt: observedAt,
		})
		if evalErr != nil {
			sample.Error = evalErr.Error()
			result.Errors = append(result.Errors, fmt.Sprintf("%s evaluate failed: %v", channel.ID, evalErr))
			result.Samples = append(result.Samples, sample)
			continue
		}

		sample.DerivedState = evalResult.State.DerivedState
		sample.ActionQueuedID = evalResult.ActionQueuedID
		if evalResult.Transition != nil {
			sample.Transitioned = true
			sample.TransitionID = evalResult.Transition.ID
		}

		result.Samples = append(result.Samples, sample)
	}

	states := make([]inputruntime.InputRuntimeState, 0, len(enabled))
	for _, channel := range enabled {
		state, stateErr := p.Store.GetInputRuntimeState(ctx, channel.ID)
		if stateErr != nil {
			return result, fmt.Errorf("get runtime state for %s: %w", channel.ID, stateErr)
		}
		if state == nil {
			continue
		}
		states = append(states, *state)
	}

	active, err := inputruntime.ResolveActiveTriggered(enabled, states)
	if err != nil {
		return result, fmt.Errorf("resolve active triggered input: %w", err)
	}
	result.Active = active
	return result, nil
}
