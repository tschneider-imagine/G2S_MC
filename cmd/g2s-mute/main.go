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
	"strconv"
	"syscall"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/certs"
	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2s"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/store"
	"github.com/tschneider-imagine/G2S_MC/internal/ui"
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

	certInventory := certs.InspectAll(certs.SourcesFromConfig(cfg.Crypto), time.Now())
	if err := auditStore.ReplaceCertificateInventory(ctx, certInventory); err != nil {
		log.Fatalf("record certificate inventory: %v", err)
	}

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
	uiServer, err := ui.NewServer()
	if err != nil {
		log.Fatalf("create dashboard: %v", err)
	}
	uiServer.RegisterRoutes(mux)

	g2sServer := g2s.NewServer(cfg.G2S.HostID, eng)
	g2sServer.RegisterRoutes(mux, cfg.G2S.EndpointPath)
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/api/status", statusHandler(eng))
	mux.HandleFunc("/api/incidents", incidentsHandler(auditStore))
	mux.HandleFunc("/api/egms/history", egmHistoryHandler(auditStore))
	mux.HandleFunc("/api/compliance", complianceHandler(auditStore))
	mux.HandleFunc("/api/state-history", stateHistoryHandler(auditStore))
	mux.HandleFunc("/api/certificates", certificatesHandler(auditStore))

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

func incidentsHandler(store *store.SQLiteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		incidents, err := store.ListIncidents(r.Context(), queryLimit(r, 50))
		writeJSON(w, incidents, err)
	}
}

func egmHistoryHandler(store *store.SQLiteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		history, err := store.ListEGMStatus(r.Context(), model.HistoryLimits{
			Limit: queryLimit(r, 50),
			EGMID: r.URL.Query().Get("egm_id"),
		})
		writeJSON(w, history, err)
	}
}

func complianceHandler(store *store.SQLiteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		logs, err := store.ListEGMComplianceLogs(r.Context(), queryLimit(r, 50))
		writeJSON(w, logs, err)
	}
}

func stateHistoryHandler(store *store.SQLiteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		changes, err := store.ListStateChanges(r.Context(), queryLimit(r, 50))
		writeJSON(w, changes, err)
	}
}

func certificatesHandler(store *store.SQLiteStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		records, err := store.ListCertificateInventory(r.Context())
		writeJSON(w, records, err)
	}
}

func writeJSON(w http.ResponseWriter, value any, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func queryLimit(r *http.Request, fallback int) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return fallback
	}
	limit, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return limit
}
