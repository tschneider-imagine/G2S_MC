package inputruntime

import (
	"fmt"
	"sort"

	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

func ResolveActiveTriggered(channels []inputs.InputChannel, states []InputRuntimeState) (*ActiveInput, error) {
	channelByID := map[string]inputs.InputChannel{}
	for _, channel := range channels {
		channelByID[channel.ID] = channel
	}

	candidates := []ActiveInput{}
	for _, state := range states {
		channel, ok := channelByID[state.InputID]
		if !ok {
			continue
		}
		if !channel.Enabled {
			continue
		}
		if state.DerivedState != inputs.DerivedStateTriggered {
			continue
		}
		actionID := channel.OnTriggerActionID
		candidates = append(candidates, ActiveInput{
			InputID:      channel.ID,
			Name:         channel.Name,
			Priority:     channel.Priority,
			DerivedState: state.DerivedState,
			ActionID:     actionID,
		})
	}

	if len(candidates) == 0 {
		return nil, nil
	}
	for _, candidate := range candidates {
		if candidate.Priority < 0 {
			return nil, fmt.Errorf("priority must be >= 0")
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority == candidates[j].Priority {
			return candidates[i].InputID < candidates[j].InputID
		}
		return candidates[i].Priority > candidates[j].Priority
	})
	selected := candidates[0]
	return &selected, nil
}
