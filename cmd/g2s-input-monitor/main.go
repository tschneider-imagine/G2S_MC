package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/gpioinput"
	"github.com/tschneider-imagine/G2S_MC/internal/inputpoller"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func main() {
	dbPath := flag.String("db", "./data/g2s-input-monitor.db", "sqlite database path")
	initDefaults := flag.Bool("init-defaults", false, "seed default Pi4 input channels")
	overwriteDefaults := flag.Bool("overwrite-defaults", false, "overwrite existing default Pi4 channels")
	once := flag.Bool("once", false, "poll once and exit")
	interval := flag.Duration("interval", 100*time.Millisecond, "poll interval")
	duration := flag.Duration("duration", 0, "poll duration (default 30s when not -once)")
	flag.Parse()

	if *interval <= 0 {
		fmt.Fprintln(os.Stderr, "-interval must be > 0")
		os.Exit(2)
	}
	if *duration < 0 {
		fmt.Fprintln(os.Stderr, "-duration must be >= 0")
		os.Exit(2)
	}

	runDuration := *duration
	if !*once && runDuration == 0 {
		runDuration = 30 * time.Second
	}

	ctx := context.Background()
	st, err := store.Open(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	if *initDefaults {
		if err := inputpoller.EnsureDefaultPi4InputChannels(ctx, st, *overwriteDefaults); err != nil {
			fmt.Fprintf(os.Stderr, "initialize default channels: %v\n", err)
			os.Exit(1)
		}
	}

	reader := gpioinput.NewReader()
	reader.Consumer = "g2s_input_monitor"

	poller := &inputpoller.Poller{
		Store:  st,
		Reader: reader,
		Evaluator: &inputruntime.Evaluator{
			Store: st,
			Clock: time.Now,
		},
		Clock: time.Now,
	}

	if *once {
		if err := runPoll(ctx, poller, 1); err != nil {
			fmt.Fprintf(os.Stderr, "poll once: %v\n", err)
			os.Exit(1)
		}
		return
	}

	deadline := time.Now().Add(runDuration)
	count := 0
	for {
		count++
		if err := runPoll(ctx, poller, count); err != nil {
			fmt.Fprintf(os.Stderr, "poll #%d: %v\n", count, err)
			os.Exit(1)
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(*interval)
	}
}

func runPoll(ctx context.Context, poller *inputpoller.Poller, iteration int) error {
	result, err := poller.PollOnce(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("poll=%d observed_at=%s\n", iteration, result.ObservedAt.Format(time.RFC3339Nano))

	samples := make([]inputpoller.PollSampleResult, len(result.Samples))
	copy(samples, result.Samples)
	sort.Slice(samples, func(i, j int) bool {
		if samples[i].GPIOChannel == samples[j].GPIOChannel {
			return samples[i].InputID < samples[j].InputID
		}
		return samples[i].GPIOChannel < samples[j].GPIOChannel
	})

	for _, sample := range samples {
		if sample.Error != "" {
			fmt.Printf("%s ERROR input=%s err=%s\n", sample.GPIOChannel, sample.InputID, sample.Error)
			continue
		}
		fmt.Printf("%s %s %s input=%s transitioned=%t transition_id=%d action_queued=%s\n",
			sample.GPIOChannel,
			sample.RawState,
			sample.DerivedState,
			sample.InputID,
			sample.Transitioned,
			sample.TransitionID,
			sample.ActionQueuedID,
		)
	}

	if result.Active == nil {
		fmt.Println("active_input=none")
	} else {
		fmt.Printf("active_input=%s priority=%d action_id=%s\n", result.Active.InputID, result.Active.Priority, result.Active.ActionID)
	}
	if len(result.Errors) > 0 {
		for _, msg := range result.Errors {
			fmt.Printf("poll_error=%s\n", msg)
		}
	}
	fmt.Println()
	return nil
}
