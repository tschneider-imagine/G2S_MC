package inputpoller

import (
	"context"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/audit"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

type DigitalReader interface {
	Read(ctx context.Context, gpioChannel string) (inputs.DigitalState, error)
}

type Store interface {
	ListInputChannels(ctx context.Context) ([]inputs.InputChannel, error)
	UpsertInputChannel(ctx context.Context, channel inputs.InputChannel) error

	GetInputChannel(ctx context.Context, id string) (*inputs.InputChannel, error)
	GetInputRuntimeState(ctx context.Context, inputID string) (*inputruntime.InputRuntimeState, error)
	UpsertInputRuntimeState(ctx context.Context, state inputruntime.InputRuntimeState) error
	RecordInputTransition(ctx context.Context, transition inputs.InputTransition) (int64, error)
	RecordAuditTimelineEntry(ctx context.Context, entry audit.AuditTimelineEntry) (int64, error)
}

type Poller struct {
	Store     Store
	Reader    DigitalReader
	Evaluator *inputruntime.Evaluator
	Clock     func() time.Time
}

type PollResult struct {
	ObservedAt time.Time
	Samples    []PollSampleResult
	Active     *inputruntime.ActiveInput
	Errors     []string
}

type PollSampleResult struct {
	InputID        string
	GPIOChannel    string
	RawState       inputs.DigitalState
	DerivedState   inputs.DerivedInputState
	Transitioned   bool
	TransitionID   int64
	ActionQueuedID string
	Error          string
}
