package engine

import (
	"context"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
)

func TestSecurityLineDropCreatesIncident(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)
	eng.Submit(Event{Type: EventBootComplete})
	eng.Submit(Event{Type: EventSecurityLineDrop, Detail: "test"})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := eng.Snapshot()
		if snapshot.State == model.StateEmergencyActive && snapshot.Incident != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected emergency incident")
}

func TestSessionOnlineMarksEGMGreen(t *testing.T) {
	eng := New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.handle(Event{Type: EventG2SSessionOnline, EGMID: "EGM-1", At: time.Now()})

	snapshot := eng.Snapshot()
	if got := snapshot.EGMs[0].Status; got != model.EGMGreen {
		t.Fatalf("expected EGM green, got %s", got)
	}
}
