package appliance

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actionexecutor"
	"github.com/tschneider-imagine/G2S_MC/internal/actionruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/inputpoller"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
)

type runtimeTestPoller struct {
	mu       sync.Mutex
	calls    int
	steps    []runtimePollStep
	onPoll   func(call int)
	fallback inputpoller.PollResult
}

type runtimePollStep struct {
	result inputpoller.PollResult
	err    error
}

func (p *runtimeTestPoller) PollOnce(_ context.Context) (inputpoller.PollResult, error) {
	p.mu.Lock()
	p.calls++
	call := p.calls
	stepIndex := call - 1
	step := runtimePollStep{result: p.fallback}
	if stepIndex >= 0 && stepIndex < len(p.steps) {
		step = p.steps[stepIndex]
	} else if len(p.steps) > 0 {
		step = p.steps[len(p.steps)-1]
	}
	onPoll := p.onPoll
	p.mu.Unlock()

	if onPoll != nil {
		onPoll(call)
	}
	if step.err != nil {
		return inputpoller.PollResult{}, step.err
	}
	return step.result, nil
}

func (p *runtimeTestPoller) CallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

type runtimeTestQueuer struct {
	mu       sync.Mutex
	requests []actionruntime.QueueRequest
	result   actionruntime.QueueResult
	err      error
}

func (q *runtimeTestQueuer) QueueActionRun(_ context.Context, request actionruntime.QueueRequest) (actionruntime.QueueResult, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.requests = append(q.requests, request)
	if q.err != nil {
		return actionruntime.QueueResult{}, q.err
	}
	return q.result, nil
}

func (q *runtimeTestQueuer) Requests() []actionruntime.QueueRequest {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]actionruntime.QueueRequest, len(q.requests))
	copy(out, q.requests)
	return out
}

type runtimeTestExecutor struct {
	mu       sync.Mutex
	requests []actionexecutor.ExecuteRequest
	result   actionexecutor.ExecuteResult
	err      error
}

func (e *runtimeTestExecutor) Execute(_ context.Context, request actionexecutor.ExecuteRequest) (actionexecutor.ExecuteResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.requests = append(e.requests, request)
	if e.err != nil {
		return actionexecutor.ExecuteResult{}, e.err
	}
	return e.result, nil
}

func (e *runtimeTestExecutor) Requests() []actionexecutor.ExecuteRequest {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]actionexecutor.ExecuteRequest, len(e.requests))
	copy(out, e.requests)
	return out
}

type runtimeTestLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *runtimeTestLogger) Printf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, fmt.Sprintf(format, v...))
}

func (l *runtimeTestLogger) Lines() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.lines))
	copy(out, l.lines)
	return out
}

func sampleTransitionResult(actionID string) inputpoller.PollResult {
	return inputpoller.PollResult{
		ObservedAt: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC),
		Samples: []inputpoller.PollSampleResult{
			{
				InputID:        "emergency-broadcast",
				GPIOChannel:    "GPIO21",
				RawState:       inputs.InputStateLow,
				DerivedState:   inputs.DerivedStateTriggered,
				Transitioned:   true,
				TransitionID:   91,
				ActionQueuedID: actionID,
			},
		},
	}
}

func TestRuntimeDisabledDoesNothing(t *testing.T) {
	poller := &runtimeTestPoller{}
	queuer := &runtimeTestQueuer{}
	logger := &runtimeTestLogger{}
	runtime := Runtime{
		Poller:  poller,
		Queuer:  queuer,
		Logger:  logger,
		Options: RuntimeOptions{Enabled: false},
	}
	if err := runtime.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if poller.CallCount() != 0 {
		t.Fatalf("poll calls=%d, want 0", poller.CallCount())
	}
}

func TestRuntimeEnabledPollsInputsAndQueuesTransitions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	poller := &runtimeTestPoller{
		steps: []runtimePollStep{{result: sampleTransitionResult("emergency_broadcast_silence")}},
		onPoll: func(call int) {
			if call == 1 {
				cancel()
			}
		},
	}
	queuedRun := actions.ActionRun{ID: "run-new", ActionDefinitionID: "action-1", Status: actions.RunStatusPending}
	queuer := &runtimeTestQueuer{
		result: actionruntime.QueueResult{
			Queued:    true,
			ActionRun: &queuedRun,
		},
	}
	executor := &runtimeTestExecutor{}
	logger := &runtimeTestLogger{}

	runtime := Runtime{
		Poller:   poller,
		Queuer:   queuer,
		Executor: executor,
		Logger:   logger,
		Options: RuntimeOptions{
			Enabled:        true,
			PollInterval:   time.Millisecond,
			ExecuteActions: false,
		},
	}
	if err := runtime.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}

	requests := queuer.Requests()
	if len(requests) != 1 {
		t.Fatalf("queue requests=%d, want 1", len(requests))
	}
	if requests[0].ActionID != "emergency_broadcast_silence" {
		t.Fatalf("queued action id=%q", requests[0].ActionID)
	}
	if len(executor.Requests()) != 0 {
		t.Fatalf("executor calls=%d, want 0", len(executor.Requests()))
	}
}

func TestRuntimeExecuteEnabledExecutesOnlyNewlyQueuedRuns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	poller := &runtimeTestPoller{
		steps: []runtimePollStep{{result: sampleTransitionResult("emergency_broadcast_silence")}},
		onPoll: func(call int) {
			if call == 1 {
				cancel()
			}
		},
	}
	queuedRun := actions.ActionRun{ID: "run-new", ActionDefinitionID: "action-1", Status: actions.RunStatusPending}
	queuer := &runtimeTestQueuer{
		result: actionruntime.QueueResult{
			Queued:    true,
			ActionRun: &queuedRun,
		},
	}
	executor := &runtimeTestExecutor{
		result: actionexecutor.ExecuteResult{
			ActionRun: actions.ActionRun{
				ID:             "run-new",
				Status:         actions.RunStatusSucceeded,
				ConfirmedCount: 1,
				FailedCount:    0,
			},
		},
	}
	runtime := Runtime{
		Poller:   poller,
		Queuer:   queuer,
		Executor: executor,
		Logger:   &runtimeTestLogger{},
		Options: RuntimeOptions{
			Enabled:        true,
			PollInterval:   time.Millisecond,
			ExecuteActions: true,
			DeliverySettings: g2stransport.DeliverySettings{
				Mode:          g2stransport.DeliveryModeHTTP,
				AllowDelivery: true,
				CaptureOnly:   false,
				TimeoutMS:     4200,
			},
		},
	}
	if err := runtime.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	calls := executor.Requests()
	if len(calls) != 1 {
		t.Fatalf("executor calls=%d, want 1", len(calls))
	}
	if calls[0].ActionRunID != "run-new" {
		t.Fatalf("executed run id=%q", calls[0].ActionRunID)
	}
	if calls[0].Delivery.Mode != g2stransport.DeliveryModeHTTP || !calls[0].Delivery.AllowDelivery || calls[0].Delivery.TimeoutMS != 4200 {
		t.Fatalf("unexpected delivery settings: %+v", calls[0].Delivery)
	}
}

func TestRuntimePollErrorsAreLoggedAndLoopContinues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	poller := &runtimeTestPoller{
		steps: []runtimePollStep{
			{err: fmt.Errorf("read failed")},
			{result: inputpoller.PollResult{ObservedAt: time.Now().UTC()}},
		},
		onPoll: func(call int) {
			if call == 2 {
				cancel()
			}
		},
	}
	logger := &runtimeTestLogger{}
	runtime := Runtime{
		Poller: poller,
		Queuer: &runtimeTestQueuer{},
		Logger: logger,
		Options: RuntimeOptions{
			Enabled:      true,
			PollInterval: time.Millisecond,
		},
	}
	if err := runtime.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if poller.CallCount() < 2 {
		t.Fatalf("poll calls=%d, want at least 2", poller.CallCount())
	}
	lines := strings.Join(logger.Lines(), "\n")
	if !strings.Contains(lines, "poll_error=read failed") {
		t.Fatalf("expected poll error in logs, got: %s", lines)
	}
}

func TestRuntimeSeedsDefaultsOnlyWhenEnabled(t *testing.T) {
	seedCalls := 0
	seedFn := func(context.Context) error {
		seedCalls++
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	poller := &runtimeTestPoller{onPoll: func(call int) {
		if call == 1 {
			cancel()
		}
	}}
	runtime := Runtime{
		Poller:              poller,
		Queuer:              &runtimeTestQueuer{},
		SeedDefaultInputsFn: seedFn,
		Logger:              &runtimeTestLogger{},
		Options: RuntimeOptions{
			Enabled:           true,
			SeedDefaultInputs: true,
			PollInterval:      time.Millisecond,
		},
	}
	if err := runtime.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if seedCalls != 1 {
		t.Fatalf("seed calls=%d, want 1", seedCalls)
	}

	seedCalls = 0
	disabledRuntime := Runtime{
		Poller:              poller,
		Queuer:              &runtimeTestQueuer{},
		SeedDefaultInputsFn: seedFn,
		Logger:              &runtimeTestLogger{},
		Options: RuntimeOptions{
			Enabled:           false,
			SeedDefaultInputs: true,
		},
	}
	if err := disabledRuntime.Run(context.Background()); err != nil {
		t.Fatalf("run disabled: %v", err)
	}
	if seedCalls != 0 {
		t.Fatalf("seed calls when disabled=%d, want 0", seedCalls)
	}
}
