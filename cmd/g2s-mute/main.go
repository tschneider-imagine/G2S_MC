package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
	startedAt := time.Now()
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
	log.Printf("loaded config controller_id=%s site=%q checksum=%s", cfg.ControllerID, cfg.SiteName, checksum)
	log.Printf(
		"runtime config_path=%s database_path=%s bind_address=%s dashboard_path=/dashboard g2s_host_url=%s g2s_endpoint_path=%s egm_count=%d",
		*configPath,
		cfg.Database.Path,
		cfg.WebUI.BindAddress,
		cfg.G2S.HostURL,
		cfg.G2S.EndpointPath,
		len(cfg.EGMRoster),
	)
	log.Printf(
		"security mode tls_required=%t client_cert_required=%t web_login_required=%t admin_client_cert_required=%t simulated_trigger=%t",
		cfg.G2S.RequireTLS,
		cfg.G2S.RequireClientCert,
		cfg.WebUI.RequireLogin,
		cfg.WebUI.RequireClientCertForAdmin,
		*simulateTrigger,
	)

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
	log.Printf("certificate inventory %s", summarizeCertificateInventory(certInventory))

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
	mux.HandleFunc("/api/status", statusHandler(eng, auditStore, cfg, runtimeInfo{
		ConfigPath:       *configPath,
		StartedAt:        startedAt,
		SimulatedTrigger: *simulateTrigger,
	}))
	mux.HandleFunc("/readyz", readinessHandler(eng, auditStore, cfg, runtimeInfo{
		ConfigPath:       *configPath,
		StartedAt:        startedAt,
		SimulatedTrigger: *simulateTrigger,
	}))
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
	if cfg.G2S.RequireTLS {
		tlsConfig, err := tlsConfigFromConfig(cfg)
		if err != nil {
			log.Fatalf("configure TLS: %v", err)
		}
		server.TLSConfig = tlsConfig
	}

	go func() {
		var err error
		if cfg.G2S.RequireTLS {
			log.Printf("service ready protocol=https bind_address=%s health=/healthz ready=/readyz status=/api/status dashboard=/dashboard", cfg.WebUI.BindAddress)
			err = server.ListenAndServeTLS(cfg.Crypto.WebServerCertPath, cfg.Crypto.WebServerKeyPath)
		} else {
			log.Printf("service ready protocol=http bind_address=%s health=/healthz ready=/readyz status=/api/status dashboard=/dashboard", cfg.WebUI.BindAddress)
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
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

func tlsConfigFromConfig(cfg config.Config) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if !cfg.G2S.RequireClientCert {
		return tlsConfig, nil
	}

	raw, err := os.ReadFile(cfg.Crypto.G2SCAPath)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(raw) {
		return nil, fmt.Errorf("no CA certificate found in %s", cfg.Crypto.G2SCAPath)
	}
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	tlsConfig.ClientCAs = pool
	return tlsConfig, nil
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintln(w, "ok")
}

type runtimeInfo struct {
	ConfigPath       string
	StartedAt        time.Time
	SimulatedTrigger bool
}

type applianceStatus struct {
	engine.Snapshot
	Runtime   runtimeStatus   `json:"runtime"`
	Readiness readinessStatus `json:"readiness"`
}

type readinessResponse struct {
	Overall string   `json:"overall"`
	Issues  []string `json:"issues"`
}

type runtimeStatus struct {
	StartedAt               time.Time `json:"started_at"`
	UptimeSeconds           int64     `json:"uptime_seconds"`
	ConfigPath              string    `json:"config_path"`
	DatabasePath            string    `json:"database_path"`
	BindAddress             string    `json:"bind_address"`
	DashboardPath           string    `json:"dashboard_path"`
	HealthPath              string    `json:"health_path"`
	G2SEndpointPath         string    `json:"g2s_endpoint_path"`
	G2SHostURL              string    `json:"g2s_host_url"`
	TLSRequired             bool      `json:"tls_required"`
	ClientCertRequired      bool      `json:"client_cert_required"`
	WebLoginRequired        bool      `json:"web_login_required"`
	AdminClientCertRequired bool      `json:"admin_client_cert_required"`
	InputMode               string    `json:"input_mode"`
	SimulatedTrigger        bool      `json:"simulated_trigger"`
}

type readinessStatus struct {
	Overall            string         `json:"overall"`
	Issues             []string       `json:"issues"`
	Warnings           []string       `json:"warnings"`
	EGMCount           int            `json:"egm_count"`
	CertificateSummary map[string]int `json:"certificate_summary"`
}

func statusHandler(eng *engine.Engine, store *store.SQLiteStore, cfg config.Config, runtime runtimeInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		status, err := computeApplianceStatus(r.Context(), eng, store, cfg, runtime)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(status); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func readinessHandler(eng *engine.Engine, store *store.SQLiteStore, cfg config.Config, runtime runtimeInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		status, err := computeApplianceStatus(r.Context(), eng, store, cfg, runtime)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(readinessResponse{
				Overall: "DEGRADED",
				Issues:  []string{"readiness status unavailable"},
			})
			return
		}
		issues := append([]string{}, status.Readiness.Issues...)
		if status.Readiness.Overall != "READY" && status.Readiness.Overall != "READY_LAB" {
			if len(issues) == 0 {
				issues = append(issues, "readiness state is "+status.Readiness.Overall)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(readinessResponse{
				Overall: status.Readiness.Overall,
				Issues:  issues,
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(readinessResponse{
			Overall: status.Readiness.Overall,
			Issues:  issues,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func computeApplianceStatus(ctx context.Context, eng *engine.Engine, store *store.SQLiteStore, cfg config.Config, runtime runtimeInfo) (applianceStatus, error) {
	snapshot := eng.Snapshot()
	certificates, err := store.ListCertificateInventory(ctx)
	if err != nil {
		return applianceStatus{}, err
	}
	return applianceStatus{
		Snapshot:  snapshot,
		Runtime:   buildRuntimeStatus(cfg, runtime),
		Readiness: buildReadinessStatus(snapshot, cfg, certificates),
	}, nil
}

func buildRuntimeStatus(cfg config.Config, runtime runtimeInfo) runtimeStatus {
	return runtimeStatus{
		StartedAt:               runtime.StartedAt,
		UptimeSeconds:           int64(time.Since(runtime.StartedAt).Seconds()),
		ConfigPath:              runtime.ConfigPath,
		DatabasePath:            cfg.Database.Path,
		BindAddress:             cfg.WebUI.BindAddress,
		DashboardPath:           "/dashboard",
		HealthPath:              "/healthz",
		G2SEndpointPath:         cfg.G2S.EndpointPath,
		G2SHostURL:              cfg.G2S.HostURL,
		TLSRequired:             cfg.G2S.RequireTLS,
		ClientCertRequired:      cfg.G2S.RequireClientCert,
		WebLoginRequired:        cfg.WebUI.RequireLogin,
		AdminClientCertRequired: cfg.WebUI.RequireClientCertForAdmin,
		InputMode:               "SIMULATED_SOFTWARE_ONLY",
		SimulatedTrigger:        runtime.SimulatedTrigger,
	}
}

func buildReadinessStatus(snapshot engine.Snapshot, cfg config.Config, certificates []model.CertificateInventory) readinessStatus {
	status := readinessStatus{
		Overall:            "READY",
		Issues:             []string{},
		Warnings:           []string{},
		EGMCount:           len(snapshot.EGMs),
		CertificateSummary: certificateSummary(certificates),
	}
	if snapshot.State == model.StateDegraded || snapshot.AuditError != "" {
		status.Overall = "DEGRADED"
		if snapshot.AuditError != "" {
			status.Issues = append(status.Issues, snapshot.AuditError)
		}
	}
	if len(snapshot.EGMs) == 0 {
		status.Overall = "DEGRADED"
		status.Issues = append(status.Issues, "no EGMs configured")
	}
	for _, certificate := range certificates {
		if certificateBlocksRuntime(cfg, certificate) {
			status.Overall = "DEGRADED"
			status.Issues = append(status.Issues, certificate.Role+" certificate is "+certificateStatusKey(certificate.Status))
		}
	}
	if !cfg.G2S.RequireTLS {
		if status.Overall == "READY" {
			status.Overall = "READY_LAB"
		}
		status.Warnings = append(status.Warnings, "G2S TLS is disabled for local lab mode")
	}
	if !cfg.G2S.RequireClientCert {
		status.Warnings = append(status.Warnings, "G2S client certificate enforcement is disabled")
	}
	if !cfg.WebUI.RequireLogin {
		status.Warnings = append(status.Warnings, "web UI login is disabled")
	}
	return status
}

func certificateBlocksRuntime(cfg config.Config, certificate model.CertificateInventory) bool {
	key := certificateStatusKey(certificate.Status)
	if key == "OK" || key == "EXPIRING_SOON" || key == "NOT_CONFIGURED" {
		return false
	}
	switch certificate.Role {
	case "web_server_cert":
		return cfg.G2S.RequireTLS
	case "g2s_ca_cert", "g2s_client_cert":
		return cfg.G2S.RequireClientCert
	default:
		return false
	}
}

func certificateSummary(certificates []model.CertificateInventory) map[string]int {
	summary := map[string]int{}
	for _, certificate := range certificates {
		summary[certificateStatusKey(certificate.Status)]++
	}
	return summary
}

func certificateStatusKey(status string) string {
	for i, r := range status {
		if r == ':' {
			return status[:i]
		}
	}
	if status == "" {
		return "UNKNOWN"
	}
	return status
}

func summarizeCertificateInventory(certificates []model.CertificateInventory) string {
	summary := certificateSummary(certificates)
	return fmt.Sprintf("ok=%d expiring_soon=%d missing=%d not_configured=%d invalid=%d", summary["OK"], summary["EXPIRING_SOON"], summary["MISSING"], summary["NOT_CONFIGURED"], summary["INVALID"])
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
