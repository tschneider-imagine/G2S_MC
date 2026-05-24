package inputruntime

import (
	"testing"

	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

func TestResolveActiveTriggeredNoneReturnsNil(t *testing.T) {
	selected, err := ResolveActiveTriggered(
		[]inputs.InputChannel{{ID: "in-1", Name: "A", Enabled: true, Priority: 1}},
		[]InputRuntimeState{{InputID: "in-1", DerivedState: inputs.DerivedStateNormal}},
	)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if selected != nil {
		t.Fatalf("expected nil, got %+v", selected)
	}
}

func TestResolveActiveTriggeredHighestPriorityWins(t *testing.T) {
	channels := []inputs.InputChannel{
		{ID: "in-1", Name: "A", Enabled: true, Priority: 1, OnTriggerActionID: "a1"},
		{ID: "in-2", Name: "B", Enabled: true, Priority: 5, OnTriggerActionID: "a2"},
	}
	states := []InputRuntimeState{
		{InputID: "in-1", DerivedState: inputs.DerivedStateTriggered},
		{InputID: "in-2", DerivedState: inputs.DerivedStateTriggered},
	}
	selected, err := ResolveActiveTriggered(channels, states)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if selected == nil || selected.InputID != "in-2" {
		t.Fatalf("unexpected selected input: %+v", selected)
	}
}

func TestResolveActiveTriggeredPriorityTieSortedByID(t *testing.T) {
	channels := []inputs.InputChannel{
		{ID: "in-2", Name: "B", Enabled: true, Priority: 5, OnTriggerActionID: "a2"},
		{ID: "in-1", Name: "A", Enabled: true, Priority: 5, OnTriggerActionID: "a1"},
	}
	states := []InputRuntimeState{
		{InputID: "in-2", DerivedState: inputs.DerivedStateTriggered},
		{InputID: "in-1", DerivedState: inputs.DerivedStateTriggered},
	}
	selected, err := ResolveActiveTriggered(channels, states)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if selected == nil || selected.InputID != "in-1" {
		t.Fatalf("unexpected selected input: %+v", selected)
	}
}

func TestResolveActiveTriggeredDisabledIgnored(t *testing.T) {
	channels := []inputs.InputChannel{
		{ID: "in-1", Name: "A", Enabled: false, Priority: 10, OnTriggerActionID: "a1"},
		{ID: "in-2", Name: "B", Enabled: true, Priority: 1, OnTriggerActionID: "a2"},
	}
	states := []InputRuntimeState{
		{InputID: "in-1", DerivedState: inputs.DerivedStateTriggered},
		{InputID: "in-2", DerivedState: inputs.DerivedStateTriggered},
	}
	selected, err := ResolveActiveTriggered(channels, states)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if selected == nil || selected.InputID != "in-2" {
		t.Fatalf("unexpected selected input: %+v", selected)
	}
}
