package g2s

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
)

func TestCommsOnlineUpdatesEngine(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	NewServer("HOST-1", eng).RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(`<g2sBody egmId="EGM-1"><commsOnLine/></g2sBody>`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "commsOnLineAck") {
		t.Fatalf("expected commsOnLineAck, got %s", rr.Body.String())
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := eng.Snapshot()
		if len(snapshot.EGMs) == 1 && snapshot.EGMs[0].Status == model.EGMGreen && !snapshot.EGMs[0].LastSeen.IsZero() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected EGM last seen to update")
}
