package inputs

import (
	"testing"
	"time"
)

func TestInputChannelValidate(t *testing.T) {
	channel := InputChannel{
		ID:           "input-1",
		Name:         "Emergency Broadcast",
		GPIOChannel:  "GPIO17",
		Enabled:      true,
		NormalState:  InputStateHigh,
		CurrentState: InputStateLow,
		DerivedState: DerivedStateTriggered,
		DebounceMS:   50,
		Priority:     3,
		LatchingMode: LatchingManualClear,
	}
	if err := channel.Validate(); err != nil {
		t.Fatalf("validate channel: %v", err)
	}

	channel.ID = ""
	if err := channel.Validate(); err == nil {
		t.Fatal("expected validation error for missing id")
	}
}

func TestInputTransitionValidate(t *testing.T) {
	transition := InputTransition{
		InputChannelID:  "input-1",
		PreviousDerived: DerivedStateNormal,
		NewDerived:      DerivedStateTriggered,
		TransitionAt:    time.Now(),
	}
	if err := transition.Validate(); err != nil {
		t.Fatalf("validate transition: %v", err)
	}

	transition.TransitionAt = time.Time{}
	if err := transition.Validate(); err == nil {
		t.Fatal("expected validation error for zero transition_at")
	}
}
