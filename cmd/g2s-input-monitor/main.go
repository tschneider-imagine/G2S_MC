package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actiondispatch"
	"github.com/tschneider-imagine/G2S_MC/internal/actionruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/gpioinput"
	"github.com/tschneider-imagine/G2S_MC/internal/inputpoller"
	"github.com/tschneider-imagine/G2S_MC/internal/inputruntime"
	"github.com/tschneider-imagine/G2S_MC/internal/inputs"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

func main() {
	dbPath := flag.String("db", "./data/g2s-input-monitor.db", "sqlite database path")
	initDefaults := flag.Bool("init-defaults", false, "seed default Pi4 input channels")
	overwriteDefaults := flag.Bool("overwrite-defaults", false, "overwrite existing default Pi4 channels")
	once := flag.Bool("once", false, "poll once and exit")
	interval := flag.Duration("interval", 100*time.Millisecond, "poll interval")
	duration := flag.Duration("duration", 0, "poll duration (default 30s when not -once)")
	queueActions := flag.Bool("queue-actions", false, "queue pending action runs when transitions include action IDs")
	dispatchDryRun := flag.Bool("dispatch-dry-run", false, "dry-run dispatch queued runs from this monitor process")
	sendPrepared := flag.Bool("send-prepared", false, "send prepared outbound messages for newly queued runs")
	transportModeRaw := flag.String("transport", "disabled", "transport mode: disabled|dry-run|http")
	allowRealSend := flag.Bool("allow-real-send", false, "allow real network sends (requires -transport http)")
	clearLatchInputID := flag.String("clear-latch", "", "manually clear a MANUAL_CLEAR latched input by input ID")
	seedDemoActions := flag.Bool("seed-demo-actions", false, "seed queue-only demo action definitions and bind default channels")
	seedDemoEGMs := flag.Bool("seed-demo-egms", false, "seed no-send smoke EGM registry records and template")
	flag.Parse()

	if *interval <= 0 {
		fmt.Fprintln(os.Stderr, "-interval must be > 0")
		os.Exit(2)
	}
	if *duration < 0 {
		fmt.Fprintln(os.Stderr, "-duration must be >= 0")
		os.Exit(2)
	}
	if *dispatchDryRun && !*queueActions {
		fmt.Fprintln(os.Stderr, "-dispatch-dry-run requires -queue-actions")
		os.Exit(2)
	}
	if *sendPrepared && !*dispatchDryRun {
		fmt.Fprintln(os.Stderr, "-send-prepared requires -dispatch-dry-run")
		os.Exit(2)
	}
	transportMode, modeErr := parseTransportMode(*transportModeRaw)
	if modeErr != nil {
		fmt.Fprintf(os.Stderr, "%v\n", modeErr)
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
	if *seedDemoActions {
		if err := seedDemoActionDefinitionsAndBindings(ctx, st); err != nil {
			fmt.Fprintf(os.Stderr, "seed demo actions: %v\n", err)
			os.Exit(1)
		}
	}
	if *seedDemoEGMs {
		if err := seedDemoEGMRegistry(ctx, st); err != nil {
			fmt.Fprintf(os.Stderr, "seed demo egms: %v\n", err)
			os.Exit(1)
		}
	}

	reader := gpioinput.NewReader()
	reader.Consumer = "g2s_input_monitor"
	queuer := &actionruntime.Queuer{Store: st, Clock: time.Now}
	dispatcher := &actiondispatch.Dispatcher{Store: st, Clock: time.Now}
	evaluator := &inputruntime.Evaluator{Store: st, Clock: time.Now}

	poller := &inputpoller.Poller{
		Store:     st,
		Reader:    reader,
		Evaluator: evaluator,
		Clock:     time.Now,
	}

	if strings.TrimSpace(*clearLatchInputID) != "" {
		if err := runClearLatch(ctx, evaluator, queuer, dispatcher, strings.TrimSpace(*clearLatchInputID), *queueActions, *dispatchDryRun, *sendPrepared, transportMode, *allowRealSend); err != nil {
			fmt.Fprintf(os.Stderr, "clear latch: %v\n", err)
			os.Exit(1)
		}
	}

	if *once {
		if err := runPoll(ctx, poller, queuer, dispatcher, *queueActions, *dispatchDryRun, *sendPrepared, transportMode, *allowRealSend, 1); err != nil {
			fmt.Fprintf(os.Stderr, "poll once: %v\n", err)
			os.Exit(1)
		}
		return
	}

	deadline := time.Now().Add(runDuration)
	count := 0
	for {
		count++
		if err := runPoll(ctx, poller, queuer, dispatcher, *queueActions, *dispatchDryRun, *sendPrepared, transportMode, *allowRealSend, count); err != nil {
			fmt.Fprintf(os.Stderr, "poll #%d: %v\n", count, err)
			os.Exit(1)
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(*interval)
	}
}

func runPoll(ctx context.Context, poller *inputpoller.Poller, queuer *actionruntime.Queuer, dispatcher *actiondispatch.Dispatcher, queueActions bool, dispatchDryRun bool, sendPrepared bool, transportMode g2stransport.Mode, allowRealSend bool, iteration int) error {
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
		if queueActions && sample.Transitioned && strings.TrimSpace(sample.ActionQueuedID) != "" {
			queueResult, queueErr := queuer.QueueActionRun(ctx, actionruntime.QueueRequest{
				InputTransition: inputs.InputTransition{
					ID:             sample.TransitionID,
					InputChannelID: sample.InputID,
					TransitionAt:   result.ObservedAt,
				},
				ActionID:      sample.ActionQueuedID,
				TriggerReason: fmt.Sprintf("input transition %d", sample.TransitionID),
				Actor:         "g2s-input-monitor",
				QueuedAt:      result.ObservedAt,
			})
			if queueErr != nil {
				fmt.Printf("queue_error input=%s transition_id=%d action_id=%s err=%v\n", sample.InputID, sample.TransitionID, sample.ActionQueuedID, queueErr)
				continue
			}
			if queueResult.Queued && queueResult.ActionRun != nil {
				fmt.Printf("queued_run run_id=%s action_id=%s targets=%d warnings=%d\n",
					queueResult.ActionRun.ID,
					sample.ActionQueuedID,
					len(queueResult.TargetResults),
					len(queueResult.PlanWarnings),
				)
				if dispatchDryRun {
					dispatchResult, dispatchErr := dispatcher.Dispatch(ctx, actiondispatch.DispatchRequest{
						ActionRunID: queueResult.ActionRun.ID,
						Mode:        actiondispatch.DispatchModeDryRun,
						Actor:       "g2s-input-monitor",
						RequestedAt: result.ObservedAt,
					})
					if dispatchErr != nil {
						fmt.Printf("dry_run_dispatch_error run_id=%s err=%v\n", queueResult.ActionRun.ID, dispatchErr)
					} else {
						fmt.Printf("dry_run_dispatch run_id=%s messages=%d warnings=%d\n",
							dispatchResult.ActionRunID,
							len(dispatchResult.PreparedMessages),
							dispatchResult.WarningCount,
						)
						if sendPrepared {
							sendResult, sendErr := dispatcher.SendPreparedMessages(ctx, actiondispatch.SendPreparedMessagesRequest{
								ActionRunID:   dispatchResult.ActionRunID,
								TransportMode: transportMode,
								AllowRealSend: allowRealSend,
								Actor:         "g2s-input-monitor",
								RequestedAt:   result.ObservedAt,
							})
							if sendErr != nil {
								fmt.Printf("send_prepared_error run_id=%s err=%v\n", dispatchResult.ActionRunID, sendErr)
							} else if sendResult.BlockedCount > 0 && sendResult.SentCount == 0 && sendResult.FailedCount == 0 {
								fmt.Printf("send_blocked run_id=%s messages=%d reason=transport_gate\n", sendResult.ActionRunID, sendResult.BlockedCount)
							} else {
								fmt.Printf("send_result run_id=%s sent=%d failed=%d blocked=%d\n",
									sendResult.ActionRunID,
									sendResult.SentCount,
									sendResult.FailedCount,
									sendResult.BlockedCount,
								)
							}
						}
					}
				}
			} else {
				fmt.Printf("queue_skipped input=%s action_id=%s reason=%s\n", sample.InputID, sample.ActionQueuedID, queueResult.Reason)
			}
		}
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

func parseTransportMode(raw string) (g2stransport.Mode, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "DISABLED":
		return g2stransport.ModeDisabled, nil
	case "DRY_RUN", "DRY-RUN":
		return g2stransport.ModeDryRun, nil
	case "HTTP":
		return g2stransport.ModeHTTP, nil
	default:
		return "", fmt.Errorf("invalid -transport value %q (use disabled|dry-run|http)", raw)
	}
}

func runClearLatch(ctx context.Context, evaluator *inputruntime.Evaluator, queuer *actionruntime.Queuer, dispatcher *actiondispatch.Dispatcher, inputID string, queueActions bool, dispatchDryRun bool, sendPrepared bool, transportMode g2stransport.Mode, allowRealSend bool) error {
	clearedAt := time.Now().UTC()
	clearResult, err := evaluator.ClearLatchedInput(ctx, inputID, "g2s-input-monitor", "operator requested clear-latch")
	if err != nil {
		fmt.Printf("clear_latch_failed input=%s err=%v\n", inputID, err)
		return err
	}
	fmt.Printf("clear_latch_succeeded input=%s transition_id=%d action_queued=%s\n",
		clearResult.InputID,
		clearResult.Transition.ID,
		clearResult.ActionQueuedID,
	)
	if !queueActions || strings.TrimSpace(clearResult.ActionQueuedID) == "" || clearResult.Transition == nil {
		return nil
	}
	queueResult, queueErr := queuer.QueueActionRun(ctx, actionruntime.QueueRequest{
		InputTransition: *clearResult.Transition,
		ActionID:        clearResult.ActionQueuedID,
		TriggerReason:   fmt.Sprintf("manual clear transition %d", clearResult.Transition.ID),
		Actor:           "g2s-input-monitor",
		QueuedAt:        clearedAt,
	})
	if queueErr != nil {
		return queueErr
	}
	if !queueResult.Queued || queueResult.ActionRun == nil {
		fmt.Printf("queue_skipped input=%s action_id=%s reason=%s\n", clearResult.InputID, clearResult.ActionQueuedID, queueResult.Reason)
		return nil
	}
	fmt.Printf("queued_run run_id=%s action_id=%s targets=%d warnings=%d\n",
		queueResult.ActionRun.ID,
		clearResult.ActionQueuedID,
		len(queueResult.TargetResults),
		len(queueResult.PlanWarnings),
	)
	if !dispatchDryRun {
		return nil
	}
	dispatchResult, dispatchErr := dispatcher.Dispatch(ctx, actiondispatch.DispatchRequest{
		ActionRunID: queueResult.ActionRun.ID,
		Mode:        actiondispatch.DispatchModeDryRun,
		Actor:       "g2s-input-monitor",
		RequestedAt: clearedAt,
	})
	if dispatchErr != nil {
		return dispatchErr
	}
	fmt.Printf("dry_run_dispatch run_id=%s messages=%d warnings=%d\n",
		dispatchResult.ActionRunID,
		len(dispatchResult.PreparedMessages),
		dispatchResult.WarningCount,
	)
	if !sendPrepared {
		return nil
	}
	sendResult, sendErr := dispatcher.SendPreparedMessages(ctx, actiondispatch.SendPreparedMessagesRequest{
		ActionRunID:   dispatchResult.ActionRunID,
		TransportMode: transportMode,
		AllowRealSend: allowRealSend,
		Actor:         "g2s-input-monitor",
		RequestedAt:   clearedAt,
	})
	if sendErr != nil {
		return sendErr
	}
	if sendResult.BlockedCount > 0 && sendResult.SentCount == 0 && sendResult.FailedCount == 0 {
		fmt.Printf("send_blocked run_id=%s messages=%d reason=transport_gate\n", sendResult.ActionRunID, sendResult.BlockedCount)
	} else {
		fmt.Printf("send_result run_id=%s sent=%d failed=%d blocked=%d\n",
			sendResult.ActionRunID,
			sendResult.SentCount,
			sendResult.FailedCount,
			sendResult.BlockedCount,
		)
	}
	return nil
}

func seedDemoEGMRegistry(ctx context.Context, st *store.SQLiteStore) error {
	template := templates.G2STemplate{
		ID:     "template-smoke-no-send",
		Name:   "Template Smoke No Send",
		Vendor: "SMOKE",
		Status: templates.TemplateStatusActive,
		Notes:  "No-send dry-run smoke template seed",
	}
	if err := st.UpsertG2STemplate(ctx, template); err != nil {
		return fmt.Errorf("upsert demo template: %w", err)
	}
	templateVersion := templates.G2STemplateVersion{
		ID:           "template-smoke-no-send-v1",
		TemplateID:   template.ID,
		VersionLabel: "1",
		ActionsJSON:  `{"actions":{"queue_only_no_send":{"message_type":"DRY_RUN_NO_SEND","content_type":"application/xml","payload_template":"<dryRunG2SMessage noSend=\"true\" action=\"{{.ActionID}}\" run=\"{{.ActionRunID}}\" egm=\"{{.EGMID}}\" step=\"{{.TemplateActionKey}}\" timestamp=\"{{.TimestampRFC3339}}\"/>","notes":"No-send dry-run smoke template action"}}}`,
		Notes:        "No-send dry-run template version seed",
	}
	if err := st.UpsertG2STemplateVersion(ctx, templateVersion); err != nil {
		return fmt.Errorf("upsert demo template version: %w", err)
	}
	if err := st.SetActiveG2STemplateVersion(ctx, template.ID, 1); err != nil {
		return fmt.Errorf("set active demo template version: %w", err)
	}

	egmRows := []egms.EGMRecord{
		{
			EGMID:              "EGM-SMOKE-001",
			DisplayName:        "Smoke EGM 001",
			Enabled:            true,
			EmergencyEnabled:   true,
			TemplateID:         template.ID,
			CurrentActionState: egms.EGMActionStateNormal,
			Notes:              "No-send dry-run registry seed",
		},
		{
			EGMID:              "EGM-SMOKE-002",
			DisplayName:        "Smoke EGM 002",
			Enabled:            true,
			EmergencyEnabled:   true,
			TemplateID:         template.ID,
			CurrentActionState: egms.EGMActionStateNormal,
			Notes:              "No-send dry-run registry seed",
		},
	}
	for _, row := range egmRows {
		if err := st.UpsertEGMRecord(ctx, row); err != nil {
			return fmt.Errorf("upsert demo egm %s: %w", row.EGMID, err)
		}
	}
	return nil
}

func seedDemoActionDefinitionsAndBindings(ctx context.Context, st *store.SQLiteStore) error {
	for _, row := range demoActionDefinitions() {
		if err := st.UpsertActionDefinition(ctx, row); err != nil {
			return fmt.Errorf("upsert demo action %s: %w", row.ID, err)
		}
	}

	channels, err := st.ListInputChannels(ctx)
	if err != nil {
		return fmt.Errorf("list input channels: %w", err)
	}
	channelByID := map[string]inputs.InputChannel{}
	for _, channel := range channels {
		channelByID[channel.ID] = channel
	}

	bindings := map[string]struct {
		trigger string
		normal  string
	}{
		"regular-operation":   {trigger: "regular-operation-trigger", normal: ""},
		"general-broadcast":   {trigger: "general-broadcast-trigger", normal: "general-broadcast-normal"},
		"emergency-broadcast": {trigger: "emergency-broadcast-trigger", normal: "emergency-broadcast-normal"},
		"local-notice":        {trigger: "local-notice-trigger", normal: "local-notice-normal"},
	}

	for id, pair := range bindings {
		channel, ok := channelByID[id]
		if !ok {
			continue
		}
		channel.OnTriggerActionID = pair.trigger
		channel.OnNormalActionID = pair.normal
		if err := st.UpsertInputChannel(ctx, channel); err != nil {
			return fmt.Errorf("bind action ids to input channel %s: %w", id, err)
		}
	}
	return nil
}

func demoActionDefinitions() []actions.ActionDefinition {
	def := func(id string, name string, severity actions.ActionSeverity) actions.ActionDefinition {
		return actions.ActionDefinition{
			ID:               id,
			Name:             name,
			Severity:         severity,
			Enabled:          true,
			TargetSelector:   "ALL_EMERGENCY_ENABLED",
			TemplateSelector: "template-by-egm",
			Steps: []actions.ActionStep{{
				ID:                "step-1",
				Name:              "Queue only no send",
				Sequence:          0,
				TemplateActionKey: "queue_only_no_send",
			}},
			Version: 1,
		}
	}

	return []actions.ActionDefinition{
		def("regular-operation-trigger", "Regular Operation Trigger (Queue Only)", actions.SeverityNotice),
		def("general-broadcast-trigger", "General Broadcast Trigger (Queue Only)", actions.SeverityBroadcast),
		def("emergency-broadcast-trigger", "Emergency Broadcast Trigger (Queue Only)", actions.SeverityEmergency),
		def("local-notice-trigger", "Local Notice Trigger (Queue Only)", actions.SeverityNotice),
		def("emergency-broadcast-normal", "Emergency Broadcast Normal (Queue Only)", actions.SeverityRestore),
		def("general-broadcast-normal", "General Broadcast Normal (Queue Only)", actions.SeverityRestore),
		def("local-notice-normal", "Local Notice Normal (Queue Only)", actions.SeverityRestore),
	}
}
