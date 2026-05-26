package pendingdeliveryruntime

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/pendingdelivery"
)

type fakeSweeper struct {
	mu       sync.Mutex
	calls    int
	result   pendingdelivery.SweepResult
	err      error
	callDone chan struct{}
}

func (f *fakeSweeper) SweepWaitingConfirmations(_ context.Context, _ time.Time) (pendingdelivery.SweepResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.callDone != nil {
		select {
		case f.callDone <- struct{}{}:
		default:
		}
	}
	return f.result, f.err
}

func (f *fakeSweeper) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type fakeLogger struct {
	mu   sync.Mutex
	rows []string
}

func (f *fakeLogger) Printf(format string, v ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, strings.TrimSpace(format))
}

func (f *fakeLogger) hasContains(needle string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, row := range f.rows {
		if strings.Contains(row, needle) {
			return true
		}
	}
	return false
}

func TestRunCallsSweepOnTick(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	sweeper := &fakeSweeper{
		result:   pendingdelivery.SweepResult{CheckedRuns: 1},
		callDone: done,
	}
	runtime := &Runtime{
		Sweeper: sweeper,
		Options: Options{
			Enabled:  true,
			Interval: 5 * time.Millisecond,
		},
	}

	go func() {
		_ = runtime.Run(ctx)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("sweep was not called on tick")
	}
	cancel()
}

func TestRunStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sweeper := &fakeSweeper{}
	runtime := &Runtime{
		Sweeper: sweeper,
		Options: Options{
			Enabled:  true,
			Interval: 5 * time.Millisecond,
		},
	}

	done := make(chan struct{})
	go func() {
		_ = runtime.Run(ctx)
		close(done)
	}()

	time.Sleep(15 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("runtime did not stop after context cancellation")
	}
}

func TestRunLogsSweepErrorsAndContinues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sweeper := &fakeSweeper{
		err:      errors.New("boom"),
		callDone: make(chan struct{}, 2),
	}
	logger := &fakeLogger{}
	runtime := &Runtime{
		Sweeper: sweeper,
		Logger:  logger,
		Options: Options{
			Enabled:  true,
			Interval: 5 * time.Millisecond,
		},
	}

	go func() {
		_ = runtime.Run(ctx)
	}()

	select {
	case <-sweeper.callDone:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected sweep call")
	}
	time.Sleep(15 * time.Millisecond)
	cancel()

	if sweeper.callCount() < 1 {
		t.Fatalf("sweep call count=%d want >=1", sweeper.callCount())
	}
	if !logger.hasContains("pending_delivery_sweep error=") {
		t.Fatalf("expected error log entry, got %+v", logger.rows)
	}
}
