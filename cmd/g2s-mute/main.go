package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2s"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
)

func main() {
	configPath := flag.String("config", "configs/config.example.json", "path to controller config")
	simulateTrigger := flag.Bool("simulate-trigger", false, "submit a simulated security line event after boot")
	flag.Parse()

	cfg, err := config.LoadFile(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	checksum, err := config.ChecksumFile(*configPath)
	if err != nil {
		log.Fatalf("checksum config: %v", err)
	}
	log.Printf("loaded config controller_id=%s checksum=%s", cfg.ControllerID, checksum)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	auditStore, err := store.Open(ctx, cfg.Database.Path)
	if err != nil {
		log.Fatalf("open audit store: %v", err)
	}
	defer auditStore.Close()

	eng := engine.NewWithAuditSink(cfg.ControllerID, cfg.EGMRoster, auditStore)
	eng.Start(ctx)
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now(), Detail: "startup complete"})

	if *simulateTrigger {
		go func() {
			time.Sleep(500 * time.Millisecond)
			eng.Submit(engine.Event{Type: engine.EventSecurityLineDrop, At: time.Now(), Detail: "manual dev simulation"})
		}()
	}

	mux := http.NewServeMux()
	g2sServer := g2s.NewServer(cfg.G2S.HostID, eng)
	g2sServer.RegisterRoutes(mux, cfg.G2S.EndpointPath)
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/api/status", statusHandler(eng))

	server := &http.Server{
		Addr:              cfg.WebUI.BindAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on http://%s", cfg.WebUI.BindAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Print("shutdown requested")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown http server: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintln(w, "ok")
}

func statusHandler(eng *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(eng.Snapshot()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
