package inputpoller

import (
	"context"
	"fmt"

	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

func DefaultPi4InputChannels() []inputs.InputChannel {
	return []inputs.InputChannel{
		{
			ID:                "regular-operation",
			Name:              "Regular Operation",
			GPIOChannel:       "GPIO16",
			Enabled:           true,
			NormalState:       inputs.InputStateHigh,
			CurrentState:      inputs.InputStateHigh,
			DerivedState:      inputs.DerivedStateNormal,
			DebounceMS:        30,
			Priority:          100,
			OnTriggerActionID: "",
			OnNormalActionID:  "",
			LatchingMode:      inputs.LatchingAutoClear,
		},
		{
			ID:                "general-broadcast",
			Name:              "General Broadcast",
			GPIOChannel:       "GPIO20",
			Enabled:           true,
			NormalState:       inputs.InputStateHigh,
			CurrentState:      inputs.InputStateHigh,
			DerivedState:      inputs.DerivedStateNormal,
			DebounceMS:        30,
			Priority:          300,
			OnTriggerActionID: "",
			OnNormalActionID:  "",
			LatchingMode:      inputs.LatchingAutoClear,
		},
		{
			ID:                "emergency-broadcast",
			Name:              "Emergency Broadcast",
			GPIOChannel:       "GPIO21",
			Enabled:           true,
			NormalState:       inputs.InputStateHigh,
			CurrentState:      inputs.InputStateHigh,
			DerivedState:      inputs.DerivedStateNormal,
			DebounceMS:        30,
			Priority:          400,
			OnTriggerActionID: "",
			OnNormalActionID:  "",
			LatchingMode:      inputs.LatchingManualClear,
		},
		{
			ID:                "local-notice",
			Name:              "Local Notice",
			GPIOChannel:       "GPIO26",
			Enabled:           true,
			NormalState:       inputs.InputStateHigh,
			CurrentState:      inputs.InputStateHigh,
			DerivedState:      inputs.DerivedStateNormal,
			DebounceMS:        30,
			Priority:          200,
			OnTriggerActionID: "",
			OnNormalActionID:  "",
			LatchingMode:      inputs.LatchingAutoClear,
		},
	}
}

func EnsureDefaultPi4InputChannels(ctx context.Context, store Store, overwrite bool) error {
	if store == nil {
		return fmt.Errorf("store is required")
	}
	for _, channel := range DefaultPi4InputChannels() {
		if !overwrite {
			existing, err := store.GetInputChannel(ctx, channel.ID)
			if err != nil {
				return fmt.Errorf("get input channel %s: %w", channel.ID, err)
			}
			if existing != nil {
				continue
			}
		}
		if err := store.UpsertInputChannel(ctx, channel); err != nil {
			return fmt.Errorf("upsert input channel %s: %w", channel.ID, err)
		}
	}
	return nil
}
