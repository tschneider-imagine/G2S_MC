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

func TestDeriveState(t *testing.T) {
	state, err := DeriveState(InputStateHigh, InputStateHigh)
	if err != nil || state != DerivedStateNormal {
		t.Fatalf("derive high/high = %v, %v", state, err)
	}
	state, err = DeriveState(InputStateLow, InputStateHigh)
	if err != nil || state != DerivedStateTriggered {
		t.Fatalf("derive low/high = %v, %v", state, err)
	}
	state, err = DeriveState(InputStateLow, InputStateLow)
	if err != nil || state != DerivedStateNormal {
		t.Fatalf("derive low/low = %v, %v", state, err)
	}
	if _, err := DeriveState("NOPE", InputStateLow); err == nil {
		t.Fatal("expected invalid raw state error")
	}
}
