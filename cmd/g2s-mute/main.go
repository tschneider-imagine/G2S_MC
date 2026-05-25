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
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	rebuildapi "github.com/tschneider-imagine/G2S_MC/internal/api"
	"github.com/tschneider-imagine/G2S_MC/internal/certs"
	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/g2s"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/operatorui"
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

	profileOnStartup, err := resolveCabinetProfile(ctx, auditStore, cfg.CabinetProfile)
	if err != nil {
		log.Fatalf("load cabinet profile override: %v", err)
	}
	log.Printf(
		"cabinet profile source=%s wire_host_url=%s host_id=%s first_test_egm_ids=%d differs_from_file=%t",
		profileOnStartup.ProfileSource,
		profileOnStartup.Effective.WireHostURL,
		profileOnStartup.Effective.HostID,
		len(profileOnStartup.Effective.FirstTestEGMIDs),
		profileOnStartup.ProfileDiffersFromFile,
	)
	if profileOnStartup.Warning != "" {
		log.Printf("cabinet profile warning: %s", profileOnStartup.Warning)
	}

	eng := engine.NewWithAuditSink(cfg.ControllerID, cfg.EGMRoster, auditStore)
	eng.Start(ctx)
	eng.Submit(engine.Event{Type: engine.EventBootComplete, At: time.Now(), Detail: "startup complete"})
	drillManager := newOperatorDrillManager(eng, cfg.EGMRoster, cfg.Timeouts.EGMHeartbeatIntervalMS)

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
	rebuildV2API := &rebuildapi.Server{
		Store: auditStore,
		AuthorizeMutation: func(w http.ResponseWriter, r *http.Request) bool {
			return requireMutationAuth(w, r, cfg)
		},
	}
	rebuildV2API.RegisterRoutes(mux)
	operatorServer := operatorui.NewServer(
		auditStore,
		operatorui.Options{
			AppVersion:              "operator-console",
			DatabasePath:            cfg.Database.Path,
			BindAddress:             cfg.WebUI.BindAddress,
			RealSendDefaultDisabled: true,
		},
		func(w http.ResponseWriter, r *http.Request) bool {
			return requireMutationAuth(w, r, cfg)
		},
	)
	operatorServer.RegisterRoutes(mux)
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
	mux.HandleFunc("/api/egm-registry", egmRegistryHandler(eng, auditStore, cfg))
	mux.HandleFunc(
		"/api/egm-registry/promote",
		requireMutationAuthForMethods(
			egmRegistryPromoteHandler(eng, auditStore, cfg),
			cfg,
			http.MethodPost,
		),
	)
	mux.HandleFunc(
		"/api/egm-registry/promote-bulk",
		requireMutationAuthForMethods(
			egmRegistryPromoteBulkHandler(eng, auditStore, cfg),
			cfg,
			http.MethodPost,
		),
	)
	mux.HandleFunc(
		"/api/egm-registry/apply-to-cabinet-profile",
		requireMutationAuthForMethods(
			egmRegistryApplyToCabinetProfileHandler(eng, auditStore, cfg),
			cfg,
			http.MethodPost,
		),
	)
	mux.HandleFunc(
		"/api/egm-registry/",
		requireMutationAuthForMethods(
			egmRegistryByIDHandler(eng, auditStore, cfg),
			cfg,
			http.MethodPut,
			http.MethodDelete,
		),
	)
	mux.HandleFunc("/api/endpoint-integrity/alerts", endpointIntegrityAlertsHandler(eng, auditStore, cfg))
	mux.HandleFunc(
		"/api/endpoint-integrity/alerts/",
		requireMutationAuthForMethods(
			endpointIntegrityAlertActionHandler(eng, auditStore, cfg),
			cfg,
			http.MethodPost,
		),
	)
	mux.HandleFunc("/api/compliance", complianceHandler(auditStore))
	mux.HandleFunc("/api/state-history", stateHistoryHandler(auditStore))
	mux.HandleFunc("/api/certificates", certificatesHandler(auditStore))
	mux.HandleFunc("/api/operator-audit", operatorAuditHandler(auditStore))
	mux.HandleFunc("/api/session-evidence", sessionEvidenceHandler(auditStore, cfg))
	mux.HandleFunc("/api/session-evidence/export-all", sessionEvidenceExportAllHandler(auditStore, cfg))
	mux.HandleFunc("/api/session-package/export", sessionPackageExportHandler(eng, auditStore, cfg, runtimeInfo{
		ConfigPath:       *configPath,
		StartedAt:        startedAt,
		SimulatedTrigger: *simulateTrigger,
	}))
	mux.HandleFunc("/api/runtime-overrides/snapshot", runtimeOverridesSnapshotHandler(auditStore, cfg))
	mux.HandleFunc(
		"/api/runtime-overrides/restore",
		requireMutationAuthForMethods(
			runtimeOverridesRestoreHandler(eng, auditStore, cfg),
			cfg,
			http.MethodPost,
		),
	)
	mux.HandleFunc("/api/runtime-overrides/presets", runtimeOverridesPresetsHandler(auditStore))
	mux.HandleFunc(
		"/api/runtime-overrides/presets/save",
		requireMutationAuthForMethods(
			runtimeOverridesPresetsSaveHandler(auditStore, cfg),
			cfg,
			http.MethodPost,
		),
	)
	mux.HandleFunc(
		"/api/runtime-overrides/presets/load",
		requireMutationAuthForMethods(
			runtimeOverridesPresetsLoadHandler(eng, auditStore, cfg),
			cfg,
			http.MethodPost,
		),
	)
	mux.HandleFunc(
		"/api/runtime-overrides/presets/",
		requireMutationAuthForMethods(
			runtimeOverridesPresetByNameHandler(auditStore, cfg),
			cfg,
			http.MethodDelete,
		),
	)
	mux.HandleFunc(
		"/api/session-evidence/",
		requireMutationAuthForMethods(
			sessionEvidenceByIDHandler(auditStore, cfg),
			cfg,
			http.MethodDelete,
		),
	)
	mux.HandleFunc(
		"/api/session-workflow",
		requireMutationAuthForMethods(
			sessionWorkflowHandler(auditStore, cfg),
			cfg,
			http.MethodPut,
			http.MethodDelete,
		),
	)
	mux.HandleFunc("/api/run-markers", runMarkersHandler(auditStore, cfg))
	mux.HandleFunc("/api/operator-drill", operatorDrillHandler(drillManager, cfg))
	mux.HandleFunc(
		"/api/heartbeat-policy",
		requireMutationAuthForMethods(
			heartbeatPolicyHandler(auditStore, cfg),
			cfg,
			http.MethodPut,
			http.MethodDelete,
		),
	)
	mux.HandleFunc("/api/certificates/backups", certificateBackupsHandler(cfg))
	mux.HandleFunc("/api/certificates/restore", certificateRestoreHandler(auditStore, cfg))
	mux.HandleFunc("/api/certificates/preview", certificatePreviewHandler(auditStore, cfg))
	mux.HandleFunc("/api/certificates/import", certificateImportHandler(auditStore, cfg))
	mux.HandleFunc("/api/certificates/export", certificateExportHandler(cfg))
	mux.HandleFunc(
		"/api/cabinet-profile",
		requireMutationAuthForMethods(
			cabinetProfileHandler(auditStore, cfg),
			cfg,
			http.MethodPut,
			http.MethodDelete,
		),
	)
	mux.HandleFunc("/api/cabinet-profile/suggestions", cabinetProfileSuggestionsHandler(eng, auditStore, cfg))
	mux.HandleFunc("/api/cabinet-preflight", cabinetPreflightHandler(eng, auditStore, cfg, runtimeInfo{
		ConfigPath:       *configPath,
		StartedAt:        startedAt,
		SimulatedTrigger: *simulateTrigger,
	}))

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
	drillManager.shutdown(shutdownCtx)
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
	Runtime                      runtimeStatus         `json:"runtime"`
	Readiness                    readinessStatus       `json:"readiness"`
	CabinetProfile               config.CabinetProfile `json:"cabinet_profile"`
	HeartbeatPolicy              heartbeatPolicy       `json:"heartbeat_policy"`
	HeartbeatPolicySource        string                `json:"heartbeat_policy_source"`
	HeartbeatPolicyLastUpdatedAt *time.Time            `json:"heartbeat_policy_last_updated_at,omitempty"`
	ProfileSource                string                `json:"profile_source"`
	ProfileLastUpdatedAt         *time.Time            `json:"profile_last_updated_at,omitempty"`
	ProfileDiffersFromFile       bool                  `json:"profile_differs_from_file"`
}

type readinessResponse struct {
	Overall string   `json:"overall"`
	Issues  []string `json:"issues"`
}

type runtimeStatus struct {
	StartedAt                   time.Time `json:"started_at"`
	UptimeSeconds               int64     `json:"uptime_seconds"`
	ConfigPath                  string    `json:"config_path"`
	DatabasePath                string    `json:"database_path"`
	BindAddress                 string    `json:"bind_address"`
	DashboardPath               string    `json:"dashboard_path"`
	HealthPath                  string    `json:"health_path"`
	G2SEndpointPath             string    `json:"g2s_endpoint_path"`
	G2SHostURL                  string    `json:"g2s_host_url"`
	TLSRequired                 bool      `json:"tls_required"`
	ClientCertRequired          bool      `json:"client_cert_required"`
	WebLoginRequired            bool      `json:"web_login_required"`
	AdminClientCertRequired     bool      `json:"admin_client_cert_required"`
	APIMutationAuthRequired     bool      `json:"api_mutation_auth_required"`
	TrustedMutationBypassActive bool      `json:"trusted_mutation_bypass_active"`
	AllowPrivateKeyExport       bool      `json:"allow_private_key_export"`
	EGMHeartbeatIntervalMS      int       `json:"egm_heartbeat_interval_ms"`
	InputMode                   string    `json:"input_mode"`
	SimulatedTrigger            bool      `json:"simulated_trigger"`
}

type readinessStatus struct {
	Overall            string         `json:"overall"`
	Issues             []string       `json:"issues"`
	Warnings           []string       `json:"warnings"`
	EGMCount           int            `json:"egm_count"`
	CertificateSummary map[string]int `json:"certificate_summary"`
}

type resolvedCabinetProfile struct {
	Effective              config.CabinetProfile
	File                   config.CabinetProfile
	Override               *store.CabinetProfileOverride
	ProfileSource          string
	ProfileLastUpdatedAt   *time.Time
	ProfileDiffersFromFile bool
	Warning                string
}

type heartbeatPolicy struct {
	IntervalMS         int `json:"interval_ms"`
	WarningAfterMissed int `json:"warning_after_missed"`
	BlockAfterMissed   int `json:"block_after_missed"`
}

type resolvedHeartbeatPolicy struct {
	Effective           heartbeatPolicy
	File                heartbeatPolicy
	Override            *store.HeartbeatPolicyOverride
	PolicySource        string
	PolicyLastUpdatedAt *time.Time
}

type heartbeatPolicyResponse struct {
	Effective           heartbeatPolicy              `json:"effective"`
	PolicySource        string                       `json:"policy_source"`
	PolicyLastUpdatedAt *time.Time                   `json:"policy_last_updated_at,omitempty"`
	OverridePresent     bool                         `json:"override_present"`
	Override            *heartbeatPolicyOverrideView `json:"override,omitempty"`
}

type heartbeatPolicyOverrideView struct {
	IntervalMS         int       `json:"interval_ms"`
	WarningAfterMissed int       `json:"warning_after_missed"`
	BlockAfterMissed   int       `json:"block_after_missed"`
	UpdatedAt          time.Time `json:"updated_at"`
	UpdatedBy          string    `json:"updated_by,omitempty"`
}

type cabinetProfileResponse struct {
	Effective              config.CabinetProfile       `json:"effective"`
	ProfileSource          string                      `json:"profile_source"`
	ProfileLastUpdatedAt   *time.Time                  `json:"profile_last_updated_at,omitempty"`
	ProfileDiffersFromFile bool                        `json:"profile_differs_from_file"`
	OverridePresent        bool                        `json:"override_present"`
	Override               *cabinetProfileOverrideView `json:"override,omitempty"`
	Warning                string                      `json:"warning,omitempty"`
}

type cabinetProfileOverrideView struct {
	Profile   config.CabinetProfile `json:"profile"`
	UpdatedAt time.Time             `json:"updated_at"`
	UpdatedBy string                `json:"updated_by,omitempty"`
}

type sessionWorkflowResponse struct {
	CurrentPhase   string     `json:"current_phase"`
	CompletedSteps []string   `json:"completed_steps"`
	OperatorNotes  string     `json:"operator_notes"`
	LastUpdatedAt  *time.Time `json:"last_updated_at,omitempty"`
	Persisted      bool       `json:"persisted"`
}

type sessionWorkflowRequest struct {
	CurrentPhase   string   `json:"current_phase"`
	CompletedSteps []string `json:"completed_steps"`
	OperatorNotes  string   `json:"operator_notes"`
}

type cabinetProfileSuggestionsResponse struct {
	ObservedEGMIDs             []string `json:"observed_egm_ids"`
	RecommendedFirstTestEGMIDs []string `json:"recommended_first_test_egm_ids"`
	PlaceholderDetected        bool     `json:"placeholder_detected"`
	Reason                     string   `json:"reason"`
	Messages                   []string `json:"messages"`
}

func statusHandler(eng *engine.Engine, store *store.SQLiteStore, cfg config.Config, runtime runtimeInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		status, err := computeApplianceStatus(r.Context(), eng, store, cfg, runtime, r)
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
		status, err := computeApplianceStatus(r.Context(), eng, store, cfg, runtime, r)
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

func computeApplianceStatus(ctx context.Context, eng *engine.Engine, store *store.SQLiteStore, cfg config.Config, runtime runtimeInfo, request *http.Request) (applianceStatus, error) {
	snapshot := eng.Snapshot()
	egmRegistryOverrides, err := store.ListEGMRegistryOverrides(ctx)
	if err != nil {
		return applianceStatus{}, err
	}
	snapshot = applyEGMRegistryOverrides(snapshot, egmRegistryOverrides)
	certificates, err := store.ListCertificateInventory(ctx)
	if err != nil {
		return applianceStatus{}, err
	}
	profile, err := resolveCabinetProfile(ctx, store, cfg.CabinetProfile)
	if err != nil {
		return applianceStatus{}, err
	}
	heartbeat, err := resolveHeartbeatPolicy(ctx, store, cfg.Timeouts)
	if err != nil {
		return applianceStatus{}, err
	}
	return applianceStatus{
		Snapshot:                     snapshot,
		Runtime:                      buildRuntimeStatus(cfg, runtime, request),
		Readiness:                    buildReadinessStatus(snapshot, cfg, certificates, profile.Warning),
		CabinetProfile:               profile.Effective,
		HeartbeatPolicy:              heartbeat.Effective,
		HeartbeatPolicySource:        heartbeat.PolicySource,
		HeartbeatPolicyLastUpdatedAt: heartbeat.PolicyLastUpdatedAt,
		ProfileSource:                profile.ProfileSource,
		ProfileLastUpdatedAt:         profile.ProfileLastUpdatedAt,
		ProfileDiffersFromFile:       profile.ProfileDiffersFromFile,
	}, nil
}

func buildRuntimeStatus(cfg config.Config, runtime runtimeInfo, request *http.Request) runtimeStatus {
	authRequired := requestRequiresMutationAuth(request, cfg)
	trustedBypass := trustedPrivateNetworkMutationBypassAllowed(request, cfg)
	return runtimeStatus{
		StartedAt:                   runtime.StartedAt,
		UptimeSeconds:               int64(time.Since(runtime.StartedAt).Seconds()),
		ConfigPath:                  runtime.ConfigPath,
		DatabasePath:                cfg.Database.Path,
		BindAddress:                 cfg.WebUI.BindAddress,
		DashboardPath:               "/dashboard",
		HealthPath:                  "/healthz",
		G2SEndpointPath:             cfg.G2S.EndpointPath,
		G2SHostURL:                  cfg.G2S.HostURL,
		TLSRequired:                 cfg.G2S.RequireTLS,
		ClientCertRequired:          cfg.G2S.RequireClientCert,
		WebLoginRequired:            cfg.WebUI.RequireLogin,
		AdminClientCertRequired:     cfg.WebUI.RequireClientCertForAdmin,
		APIMutationAuthRequired:     authRequired,
		TrustedMutationBypassActive: trustedBypass,
		AllowPrivateKeyExport:       cfg.WebUI.AllowPrivateKeyExport,
		EGMHeartbeatIntervalMS:      cfg.Timeouts.EGMHeartbeatIntervalMS,
		InputMode:                   "SIMULATED_SOFTWARE_ONLY",
		SimulatedTrigger:            runtime.SimulatedTrigger,
	}
}

func buildReadinessStatus(snapshot engine.Snapshot, cfg config.Config, certificates []model.CertificateInventory, profileWarning string) readinessStatus {
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
		status.Warnings = append(status.Warnings, "No EGM traffic has been observed yet")
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
	if cfg.WebUI.AllowTrustedPrivateNetworkMutations {
		status.Warnings = append(status.Warnings, "trusted private network mutation bypass is enabled for lab mode")
	}
	if profileWarning != "" {
		status.Warnings = append(status.Warnings, profileWarning)
	}
	return status
}

func certificateBlocksRuntime(cfg config.Config, certificate model.CertificateInventory) bool {
	key := certificateStatusKey(certificate.Status)
	if key == "OK" || key == "VALID" || key == "EXPIRING_SOON" || key == "NOT_CONFIGURED" {
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
		if err == nil && queryBool(r, "rollup_heartbeat", false) {
			history = rollupHeartbeatHistory(history)
		}
		writeJSON(w, history, err)
	}
}

func rollupHeartbeatHistory(history []model.EGMStatusSnapshot) []model.EGMStatusSnapshot {
	if len(history) == 0 {
		return history
	}
	rolled := make([]model.EGMStatusSnapshot, 0, len(history))
	for i := 0; i < len(history); {
		current := history[i]
		if !isKeepAliveHistoryEvent(current) {
			rolled = append(rolled, current)
			i++
			continue
		}
		egmID := current.EGMID
		newest := current.CreatedAt
		oldest := current.CreatedAt
		count := 0
		for i < len(history) {
			row := history[i]
			if !isKeepAliveHistoryEvent(row) || row.EGMID != egmID {
				break
			}
			count++
			if row.CreatedAt.After(newest) {
				newest = row.CreatedAt
			}
			if row.CreatedAt.Before(oldest) {
				oldest = row.CreatedAt
			}
			i++
		}
		bucket := current
		bucket.CreatedAt = newest
		bucket.HeartbeatRollup = true
		bucket.HeartbeatRollupCount = count
		bucket.HeartbeatRollupFirstSeenAt = &oldest
		bucket.HeartbeatRollupLastSeenAt = &newest
		bucket.Detail = heartbeatRollupSummary(count, oldest, newest)
		rolled = append(rolled, bucket)
	}
	return rolled
}

func isKeepAliveHistoryEvent(snapshot model.EGMStatusSnapshot) bool {
	return strings.EqualFold(strings.TrimSpace(snapshot.EventType), string(engine.EventKeepAlive))
}

func heartbeatRollupSummary(count int, firstSeen, lastSeen time.Time) string {
	if count <= 0 {
		count = 1
	}
	span := lastSeen.Sub(firstSeen)
	if span < 0 {
		span = -span
	}
	return fmt.Sprintf("keepAlive x%d over %s", count, formatRollupDuration(span))
}

func formatRollupDuration(duration time.Duration) string {
	if duration < 0 {
		duration = -duration
	}
	rounded := duration.Round(time.Second)
	seconds := int64(rounded / time.Second)
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	remSeconds := seconds % 60
	if minutes < 60 {
		if remSeconds == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm%ds", minutes, remSeconds)
	}
	hours := minutes / 60
	remMinutes := minutes % 60
	if remMinutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, remMinutes)
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

type sessionEvidencePayload struct {
	CapturedAt    time.Time `json:"captured_at"`
	OperatorNotes string    `json:"operator_notes"`
	Session       struct {
		OverallState   string `json:"overall_state"`
		ReadyzState    string `json:"readyz_state"`
		PreflightState string `json:"preflight_state"`
	} `json:"session"`
	CabinetProfile struct {
		HostID      string `json:"host_id"`
		WireHostURL string `json:"wire_host_url"`
	} `json:"cabinet_profile"`
}

type runMarkerPayload struct {
	CreatedAt   time.Time `json:"created_at"`
	MarkerType  string    `json:"marker_type"`
	Title       string    `json:"title"`
	Notes       string    `json:"notes"`
	HostID      string    `json:"host_id"`
	WireHostURL string    `json:"wire_host_url"`
	Operator    string    `json:"operator"`
}

type sessionEvidenceArchive struct {
	GeneratedAt  time.Time                    `json:"generated_at"`
	SummaryIndex sessionEvidenceArchiveIndex  `json:"summary_index"`
	CaptureFiles []sessionEvidenceArchiveFile `json:"capture_files"`
}

type sessionEvidenceArchiveIndex struct {
	CaptureCount int                          `json:"capture_count"`
	Captures     []sessionEvidenceArchiveItem `json:"captures"`
}

type sessionEvidenceArchiveItem struct {
	ID               int64     `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	OverallState     string    `json:"overall_state"`
	ReadyzState      string    `json:"readyz_state"`
	PreflightState   string    `json:"preflight_state"`
	HostID           string    `json:"host_id"`
	WireHostURL      string    `json:"wire_host_url"`
	OperatorNotes    string    `json:"operator_notes,omitempty"`
	JSONFileName     string    `json:"json_file_name"`
	MarkdownFileName string    `json:"markdown_file_name"`
}

type sessionEvidenceArchiveFile struct {
	ID               int64  `json:"id"`
	JSONFileName     string `json:"json_file_name"`
	MarkdownFileName string `json:"markdown_file_name"`
	JSONCapture      any    `json:"json_capture"`
	MarkdownReport   string `json:"markdown_report"`
}

func sessionEvidenceHandler(store *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			records, err := store.ListSessionEvidence(r.Context(), queryLimit(r, 20))
			writeJSON(w, records, err)
		case http.MethodDelete:
			if !requireMutationAuth(w, r, cfg) {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.delete", "fail", "Session evidence delete rejected", "mutation authorization failed")
				return
			}
			id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
			if err != nil || id <= 0 {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.delete", "fail", "Session evidence delete rejected", "valid id query parameter is required")
				http.Error(w, "valid id query parameter is required", http.StatusBadRequest)
				return
			}
			deleted, err := store.DeleteSessionEvidenceByID(r.Context(), id)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.delete", "fail", "Session evidence delete failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			if !deleted {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.delete", "fail", "Session evidence delete failed", "session evidence record not found")
				http.Error(w, "session evidence record not found", http.StatusNotFound)
				return
			}
			recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.delete", "success", "Session evidence deleted", fmt.Sprintf("id=%d", id))
			writeJSON(w, map[string]any{"deleted_id": id}, nil)
		case http.MethodPost:
			if !requireMutationAuth(w, r, cfg) {
				return
			}
			var raw json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			var payload sessionEvidencePayload
			if err := json.Unmarshal(raw, &payload); err != nil {
				http.Error(w, "invalid session evidence payload", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(payload.Session.OverallState) == "" {
				http.Error(w, "session.overall_state is required", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(payload.CabinetProfile.HostID) == "" {
				http.Error(w, "cabinet_profile.host_id is required", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(payload.CabinetProfile.WireHostURL) == "" {
				http.Error(w, "cabinet_profile.wire_host_url is required", http.StatusBadRequest)
				return
			}
			capturedAt := payload.CapturedAt
			if capturedAt.IsZero() {
				capturedAt = time.Now().UTC()
			}
			record := model.SessionEvidenceRecord{
				CreatedAt:      capturedAt,
				OverallState:   payload.Session.OverallState,
				ReadyzState:    payload.Session.ReadyzState,
				PreflightState: payload.Session.PreflightState,
				HostID:         payload.CabinetProfile.HostID,
				WireHostURL:    payload.CabinetProfile.WireHostURL,
				OperatorNotes:  payload.OperatorNotes,
				PayloadJSON:    string(raw),
			}
			id, err := store.RecordSessionEvidence(r.Context(), record)
			if err != nil {
				writeJSON(w, nil, err)
				return
			}
			record.ID = id
			writeJSON(w, record, nil)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func sessionEvidenceByIDHandler(store *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id, err := sessionEvidenceIDFromPath(r.URL.Path)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.delete", "fail", "Session evidence delete rejected", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		deleted, err := store.DeleteSessionEvidenceByID(r.Context(), id)
		if err != nil {
			recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.delete", "fail", "Session evidence delete failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		if !deleted {
			recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.delete", "fail", "Session evidence delete failed", "session evidence record not found")
			http.Error(w, "session evidence record not found", http.StatusNotFound)
			return
		}
		recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.delete", "success", "Session evidence deleted", fmt.Sprintf("id=%d", id))
		writeJSON(w, map[string]any{"deleted_id": id}, nil)
	}
}

func sessionEvidenceExportAllHandler(store *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		records, err := store.ListAllSessionEvidence(r.Context())
		if err != nil {
			recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.export_all", "fail", "Session evidence export-all failed", err.Error())
			writeJSON(w, nil, err)
			return
		}
		archive := buildSessionEvidenceArchive(records)
		filename := "session-evidence-archive-" + archive.GeneratedAt.Format("20060102T150405Z") + ".json"
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		if err := json.NewEncoder(w).Encode(archive); err != nil {
			recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.export_all", "fail", "Session evidence export-all failed", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_evidence.export_all", "success", "Session evidence export-all completed", fmt.Sprintf("captures=%d", len(records)))
	}
}

func runMarkersHandler(store *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			records, err := store.ListRunMarkers(r.Context(), queryLimit(r, 20))
			writeJSON(w, records, err)
		case http.MethodPost:
			if !requireMutationAuth(w, r, cfg) {
				return
			}
			var payload runMarkerPayload
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&payload); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			markerType := strings.ToLower(strings.TrimSpace(payload.MarkerType))
			switch markerType {
			case "start", "note", "end":
			default:
				http.Error(w, "marker_type must be start, note, or end", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(payload.Title) == "" {
				http.Error(w, "title is required", http.StatusBadRequest)
				return
			}
			createdAt := payload.CreatedAt
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			record := model.RunMarker{
				CreatedAt:   createdAt,
				MarkerType:  markerType,
				Title:       strings.TrimSpace(payload.Title),
				Notes:       strings.TrimSpace(payload.Notes),
				HostID:      strings.TrimSpace(payload.HostID),
				WireHostURL: strings.TrimSpace(payload.WireHostURL),
				Operator:    strings.TrimSpace(payload.Operator),
			}
			id, err := store.RecordRunMarker(r.Context(), record)
			if err != nil {
				writeJSON(w, nil, err)
				return
			}
			record.ID = id
			writeJSON(w, record, nil)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func cabinetProfileHandler(store *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			profile, err := resolveCabinetProfile(r.Context(), store, cfg.CabinetProfile)
			if err != nil {
				writeJSON(w, nil, err)
				return
			}
			writeJSON(w, buildCabinetProfileResponse(profile), nil)
		case http.MethodPut:
			var profile config.CabinetProfile
			if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "cabinet_profile.save", "fail", "Cabinet profile save rejected", "invalid JSON body")
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			if err := config.ValidateCabinetProfile(profile); err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "cabinet_profile.save", "fail", "Cabinet profile save rejected", err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			updatedBy := strings.TrimSpace(r.Header.Get("X-Operator"))
			if updatedBy == "" {
				updatedBy = "lab-api"
			}
			if err := store.UpsertCabinetProfileOverride(r.Context(), profile, updatedBy); err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "cabinet_profile.save", "fail", "Cabinet profile save failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			resolved, err := resolveCabinetProfile(r.Context(), store, cfg.CabinetProfile)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "cabinet_profile.save", "fail", "Cabinet profile save failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(
				r.Context(),
				store,
				r,
				cfg,
				"cabinet_profile.save",
				"success",
				"Cabinet profile override saved",
				fmt.Sprintf("host_id=%s wire_host_url=%s first_test_egm_ids=%d", resolved.Effective.HostID, resolved.Effective.WireHostURL, len(resolved.Effective.FirstTestEGMIDs)),
			)
			writeJSON(w, buildCabinetProfileResponse(resolved), nil)
		case http.MethodDelete:
			if err := store.ClearCabinetProfileOverride(r.Context()); err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "cabinet_profile.clear", "fail", "Cabinet profile override clear failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			resolved, err := resolveCabinetProfile(r.Context(), store, cfg.CabinetProfile)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "cabinet_profile.clear", "fail", "Cabinet profile override clear failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(r.Context(), store, r, cfg, "cabinet_profile.clear", "success", "Cabinet profile override cleared", "profile_source="+resolved.ProfileSource)
			writeJSON(w, buildCabinetProfileResponse(resolved), nil)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func cabinetProfileSuggestionsHandler(eng *engine.Engine, store *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		profile, err := resolveCabinetProfile(r.Context(), store, cfg.CabinetProfile)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}

		suggestions := buildCabinetProfileSuggestions(eng.Snapshot(), profile.Effective.FirstTestEGMIDs)
		writeJSON(w, suggestions, nil)
	}
}

func sessionWorkflowHandler(store *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			progress, err := store.GetSessionWorkflowProgress(r.Context())
			if err != nil {
				writeJSON(w, nil, err)
				return
			}
			writeJSON(w, buildSessionWorkflowResponse(progress), nil)
		case http.MethodPut:
			var payload sessionWorkflowRequest
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&payload); err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_workflow.save", "fail", "Session workflow save rejected", "invalid JSON body")
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			normalized, err := validateSessionWorkflowRequest(payload)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_workflow.save", "fail", "Session workflow save rejected", err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := store.UpsertSessionWorkflowProgress(r.Context(), normalized.CurrentPhase, normalized.CompletedSteps, normalized.OperatorNotes); err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_workflow.save", "fail", "Session workflow save failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			progress, err := store.GetSessionWorkflowProgress(r.Context())
			if err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_workflow.save", "fail", "Session workflow save failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(
				r.Context(),
				store,
				r,
				cfg,
				"session_workflow.save",
				"success",
				"Session workflow progress saved",
				fmt.Sprintf("current_phase=%s completed_steps=%d", normalized.CurrentPhase, len(normalized.CompletedSteps)),
			)
			writeJSON(w, buildSessionWorkflowResponse(progress), nil)
		case http.MethodDelete:
			if err := store.ClearSessionWorkflowProgress(r.Context()); err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_workflow.clear", "fail", "Session workflow progress clear failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(r.Context(), store, r, cfg, "session_workflow.clear", "success", "Session workflow progress cleared", "")
			writeJSON(w, buildSessionWorkflowResponse(nil), nil)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func heartbeatPolicyHandler(store *store.SQLiteStore, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			policy, err := resolveHeartbeatPolicy(r.Context(), store, cfg.Timeouts)
			if err != nil {
				writeJSON(w, nil, err)
				return
			}
			writeJSON(w, buildHeartbeatPolicyResponse(policy), nil)
		case http.MethodPut:
			var payload struct {
				IntervalMS         int `json:"interval_ms"`
				WarningAfterMissed int `json:"warning_after_missed"`
				BlockAfterMissed   int `json:"block_after_missed"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "heartbeat_policy.save", "fail", "Heartbeat policy save rejected", "invalid JSON body")
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			if payload.IntervalMS <= 0 {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "heartbeat_policy.save", "fail", "Heartbeat policy save rejected", "interval_ms must be greater than zero")
				http.Error(w, "interval_ms must be greater than zero", http.StatusBadRequest)
				return
			}
			if payload.WarningAfterMissed <= 0 {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "heartbeat_policy.save", "fail", "Heartbeat policy save rejected", "warning_after_missed must be greater than zero")
				http.Error(w, "warning_after_missed must be greater than zero", http.StatusBadRequest)
				return
			}
			if payload.BlockAfterMissed < payload.WarningAfterMissed {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "heartbeat_policy.save", "fail", "Heartbeat policy save rejected", "block_after_missed must be greater than or equal to warning_after_missed")
				http.Error(w, "block_after_missed must be greater than or equal to warning_after_missed", http.StatusBadRequest)
				return
			}
			updatedBy := strings.TrimSpace(r.Header.Get("X-Operator"))
			if updatedBy == "" {
				updatedBy = "lab-api"
			}
			if err := store.UpsertHeartbeatPolicyOverride(r.Context(), payload.IntervalMS, payload.WarningAfterMissed, payload.BlockAfterMissed, updatedBy); err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "heartbeat_policy.save", "fail", "Heartbeat policy save failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			resolved, err := resolveHeartbeatPolicy(r.Context(), store, cfg.Timeouts)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "heartbeat_policy.save", "fail", "Heartbeat policy save failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(
				r.Context(),
				store,
				r,
				cfg,
				"heartbeat_policy.save",
				"success",
				"Heartbeat policy override saved",
				fmt.Sprintf("interval_ms=%d warning_after_missed=%d block_after_missed=%d", resolved.Effective.IntervalMS, resolved.Effective.WarningAfterMissed, resolved.Effective.BlockAfterMissed),
			)
			writeJSON(w, buildHeartbeatPolicyResponse(resolved), nil)
		case http.MethodDelete:
			if err := store.ClearHeartbeatPolicyOverride(r.Context()); err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "heartbeat_policy.clear", "fail", "Heartbeat policy override clear failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			resolved, err := resolveHeartbeatPolicy(r.Context(), store, cfg.Timeouts)
			if err != nil {
				recordOperatorAuditEvent(r.Context(), store, r, cfg, "heartbeat_policy.clear", "fail", "Heartbeat policy override clear failed", err.Error())
				writeJSON(w, nil, err)
				return
			}
			recordOperatorAuditEvent(r.Context(), store, r, cfg, "heartbeat_policy.clear", "success", "Heartbeat policy override cleared", "policy_source="+resolved.PolicySource)
			writeJSON(w, buildHeartbeatPolicyResponse(resolved), nil)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func buildCabinetProfileResponse(profile resolvedCabinetProfile) cabinetProfileResponse {
	response := cabinetProfileResponse{
		Effective:              profile.Effective,
		ProfileSource:          profile.ProfileSource,
		ProfileLastUpdatedAt:   profile.ProfileLastUpdatedAt,
		ProfileDiffersFromFile: profile.ProfileDiffersFromFile,
		OverridePresent:        profile.Override != nil,
		Warning:                profile.Warning,
	}
	if profile.Override != nil {
		response.Override = &cabinetProfileOverrideView{
			Profile:   profile.Override.Profile,
			UpdatedAt: profile.Override.UpdatedAt,
			UpdatedBy: profile.Override.UpdatedBy,
		}
	}
	return response
}

var sessionWorkflowPhases = map[string]struct{}{
	"pre_check":        {},
	"connect_observe":  {},
	"run_active":       {},
	"capture_evidence": {},
	"session_complete": {},
}

func buildSessionWorkflowResponse(progress *model.SessionWorkflowProgress) sessionWorkflowResponse {
	if progress == nil {
		return sessionWorkflowResponse{
			CurrentPhase:   "pre_check",
			CompletedSteps: []string{},
			OperatorNotes:  "",
			Persisted:      false,
		}
	}
	updatedAt := progress.LastUpdatedAt
	return sessionWorkflowResponse{
		CurrentPhase:   progress.CurrentPhase,
		CompletedSteps: append([]string{}, progress.CompletedSteps...),
		OperatorNotes:  progress.OperatorNotes,
		LastUpdatedAt:  &updatedAt,
		Persisted:      true,
	}
}

func validateSessionWorkflowRequest(payload sessionWorkflowRequest) (sessionWorkflowRequest, error) {
	normalized := sessionWorkflowRequest{
		CurrentPhase:   strings.TrimSpace(payload.CurrentPhase),
		CompletedSteps: []string{},
		OperatorNotes:  strings.TrimSpace(payload.OperatorNotes),
	}
	if normalized.CurrentPhase == "" {
		return sessionWorkflowRequest{}, fmt.Errorf("current_phase is required")
	}
	if _, ok := sessionWorkflowPhases[normalized.CurrentPhase]; !ok {
		return sessionWorkflowRequest{}, fmt.Errorf("current_phase must be one of pre_check, connect_observe, run_active, capture_evidence, session_complete")
	}
	if len(normalized.OperatorNotes) > 4000 {
		return sessionWorkflowRequest{}, fmt.Errorf("operator_notes must be 4000 characters or fewer")
	}

	seen := map[string]struct{}{}
	for _, raw := range payload.CompletedSteps {
		stepID := strings.TrimSpace(raw)
		if stepID == "" {
			return sessionWorkflowRequest{}, fmt.Errorf("completed_steps entries must be non-empty")
		}
		if _, ok := sessionWorkflowPhases[stepID]; !ok {
			return sessionWorkflowRequest{}, fmt.Errorf("completed_steps contains invalid step %q", stepID)
		}
		if _, duplicated := seen[stepID]; duplicated {
			return sessionWorkflowRequest{}, fmt.Errorf("completed_steps contains duplicate step %q", stepID)
		}
		seen[stepID] = struct{}{}
		normalized.CompletedSteps = append(normalized.CompletedSteps, stepID)
	}

	return normalized, nil
}

func buildHeartbeatPolicyResponse(policy resolvedHeartbeatPolicy) heartbeatPolicyResponse {
	response := heartbeatPolicyResponse{
		Effective:           policy.Effective,
		PolicySource:        policy.PolicySource,
		PolicyLastUpdatedAt: policy.PolicyLastUpdatedAt,
		OverridePresent:     policy.Override != nil,
	}
	if policy.Override != nil {
		response.Override = &heartbeatPolicyOverrideView{
			IntervalMS:         policy.Override.IntervalMS,
			WarningAfterMissed: policy.Override.WarningAfterMissed,
			BlockAfterMissed:   policy.Override.BlockAfterMissed,
			UpdatedAt:          policy.Override.UpdatedAt,
			UpdatedBy:          policy.Override.UpdatedBy,
		}
	}
	return response
}

func resolveCabinetProfile(ctx context.Context, store *store.SQLiteStore, fileProfile config.CabinetProfile) (resolvedCabinetProfile, error) {
	resolved := resolvedCabinetProfile{
		Effective:              fileProfile,
		File:                   fileProfile,
		ProfileSource:          "file",
		ProfileDiffersFromFile: false,
	}

	override, err := store.GetCabinetProfileOverride(ctx)
	if err != nil {
		return resolved, err
	}
	if override == nil {
		return resolved, nil
	}

	merged, usage := mergeCabinetProfile(fileProfile, override.Profile)
	resolved.Override = override
	resolved.Effective = merged
	resolved.ProfileLastUpdatedAt = &override.UpdatedAt
	resolved.ProfileDiffersFromFile = !cabinetProfilesEqual(merged, fileProfile)
	if usage.allFieldsSet {
		resolved.ProfileSource = "override"
	} else {
		resolved.ProfileSource = "mixed"
	}

	if err := config.ValidateCabinetProfile(merged); err != nil {
		resolved.Warning = "cabinet profile override invalid: " + err.Error() + "; using file defaults"
		resolved.Effective = fileProfile
		resolved.ProfileDiffersFromFile = false
		resolved.ProfileSource = "mixed"
	}

	return resolved, nil
}

func resolveHeartbeatPolicy(ctx context.Context, store *store.SQLiteStore, fileTimeouts config.Timeouts) (resolvedHeartbeatPolicy, error) {
	filePolicy := heartbeatPolicy{
		IntervalMS:         fileTimeouts.EGMHeartbeatIntervalMS,
		WarningAfterMissed: config.EffectiveHeartbeatWarningAfterMissed(fileTimeouts),
		BlockAfterMissed:   config.EffectiveHeartbeatBlockAfterMissed(fileTimeouts),
	}
	resolved := resolvedHeartbeatPolicy{
		Effective:    filePolicy,
		File:         filePolicy,
		PolicySource: "file",
	}

	override, err := store.GetHeartbeatPolicyOverride(ctx)
	if err != nil {
		return resolved, err
	}
	if override == nil {
		return resolved, nil
	}

	resolved.Override = override
	if override.IntervalMS > 0 {
		resolved.Effective.IntervalMS = override.IntervalMS
	}
	resolved.Effective.WarningAfterMissed = override.WarningAfterMissed
	resolved.Effective.BlockAfterMissed = override.BlockAfterMissed
	resolved.PolicySource = "override"
	resolved.PolicyLastUpdatedAt = &override.UpdatedAt
	return resolved, nil
}

func buildCabinetProfileSuggestions(snapshot engine.Snapshot, currentFirstTestEGMIDs []string) cabinetProfileSuggestionsResponse {
	observed := observedEGMIDsNewestFirst(snapshot)
	recommended := recommendedFirstTestEGMIDs(observed)
	placeholderDetected := cabinetProfilePlaceholderDetected(currentFirstTestEGMIDs)
	response := cabinetProfileSuggestionsResponse{
		ObservedEGMIDs:             observed,
		RecommendedFirstTestEGMIDs: recommended,
		PlaceholderDetected:        placeholderDetected,
		Messages:                   []string{},
	}

	if len(observed) == 0 {
		response.Reason = "No observed EGM traffic is available yet."
		response.Messages = append(response.Messages, "Start cabinet session traffic (commsOnLine/keepAlive) to generate observed EGM IDs.")
	} else {
		response.Reason = fmt.Sprintf("Recommended first-test EGM IDs use the newest observed EGMs (%d of %d).", len(recommended), len(observed))
		response.Messages = append(response.Messages, "Use Observed EGMs fills the form only; save to persist cabinet profile changes.")
	}

	if placeholderDetected {
		response.Messages = append(response.Messages, "Current first-test EGM IDs include placeholder-style values.")
	}
	if len(recommended) > 0 {
		response.Messages = append(response.Messages, "Recommended first-test EGM IDs: "+strings.Join(recommended, ", "))
	}

	return response
}

func observedEGMIDsNewestFirst(snapshot engine.Snapshot) []string {
	type observedEGM struct {
		id       string
		lastSeen time.Time
	}

	lastSeenByID := map[string]time.Time{}
	for _, egm := range snapshot.EGMs {
		id := strings.TrimSpace(egm.ID)
		if id == "" || egm.LastSeen.IsZero() {
			continue
		}
		if previous, ok := lastSeenByID[id]; !ok || egm.LastSeen.After(previous) {
			lastSeenByID[id] = egm.LastSeen
		}
	}

	rows := make([]observedEGM, 0, len(lastSeenByID))
	for id, lastSeen := range lastSeenByID {
		rows = append(rows, observedEGM{id: id, lastSeen: lastSeen})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].lastSeen.Equal(rows[j].lastSeen) {
			return rows[i].id < rows[j].id
		}
		return rows[i].lastSeen.After(rows[j].lastSeen)
	})

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.id)
	}
	return ids
}

func recommendedFirstTestEGMIDs(observed []string) []string {
	if len(observed) == 0 {
		return []string{}
	}
	limit := len(observed)
	if limit > 3 {
		limit = 3
	}
	return append([]string{}, observed[:limit]...)
}

func cabinetProfilePlaceholderDetected(ids []string) bool {
	for _, id := range ids {
		if looksPlaceholderText(id) {
			return true
		}
	}
	return false
}

type cabinetProfileOverrideUsage struct {
	allFieldsSet bool
}

func mergeCabinetProfile(fileProfile config.CabinetProfile, overrideProfile config.CabinetProfile) (config.CabinetProfile, cabinetProfileOverrideUsage) {
	merged := fileProfile
	fieldsSet := 0
	totalFields := 7

	if value := strings.TrimSpace(overrideProfile.WireHostURL); value != "" {
		merged.WireHostURL = value
		fieldsSet++
	}
	if value := strings.TrimSpace(overrideProfile.ListenerDNSName); value != "" {
		merged.ListenerDNSName = value
		fieldsSet++
	}
	if value := strings.TrimSpace(overrideProfile.ListenerIP); value != "" {
		merged.ListenerIP = value
		fieldsSet++
	}
	if len(overrideProfile.RequiredSANDNS) > 0 {
		merged.RequiredSANDNS = append([]string{}, overrideProfile.RequiredSANDNS...)
		fieldsSet++
	}
	if len(overrideProfile.RequiredSANIPs) > 0 {
		merged.RequiredSANIPs = append([]string{}, overrideProfile.RequiredSANIPs...)
		fieldsSet++
	}
	if value := strings.TrimSpace(overrideProfile.HostID); value != "" {
		merged.HostID = value
		fieldsSet++
	}
	if len(overrideProfile.FirstTestEGMIDs) > 0 {
		merged.FirstTestEGMIDs = append([]string{}, overrideProfile.FirstTestEGMIDs...)
		fieldsSet++
	}

	return merged, cabinetProfileOverrideUsage{
		allFieldsSet: fieldsSet == totalFields,
	}
}

func cabinetProfilesEqual(a config.CabinetProfile, b config.CabinetProfile) bool {
	if a.WireHostURL != b.WireHostURL ||
		a.ListenerDNSName != b.ListenerDNSName ||
		a.ListenerIP != b.ListenerIP ||
		a.HostID != b.HostID {
		return false
	}
	if len(a.RequiredSANDNS) != len(b.RequiredSANDNS) ||
		len(a.RequiredSANIPs) != len(b.RequiredSANIPs) ||
		len(a.FirstTestEGMIDs) != len(b.FirstTestEGMIDs) {
		return false
	}
	for i := range a.RequiredSANDNS {
		if a.RequiredSANDNS[i] != b.RequiredSANDNS[i] {
			return false
		}
	}
	for i := range a.RequiredSANIPs {
		if a.RequiredSANIPs[i] != b.RequiredSANIPs[i] {
			return false
		}
	}
	for i := range a.FirstTestEGMIDs {
		if a.FirstTestEGMIDs[i] != b.FirstTestEGMIDs[i] {
			return false
		}
	}
	return true
}

func sessionEvidenceIDFromPath(path string) (int64, error) {
	const prefix = "/api/session-evidence/"
	if !strings.HasPrefix(path, prefix) {
		return 0, fmt.Errorf("session evidence id path is invalid")
	}
	trimmed := strings.TrimPrefix(path, prefix)
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return 0, fmt.Errorf("session evidence id path is invalid")
	}
	id, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("session evidence id must be a positive integer")
	}
	return id, nil
}

func buildSessionEvidenceArchive(records []model.SessionEvidenceRecord) sessionEvidenceArchive {
	archive := sessionEvidenceArchive{
		GeneratedAt:  time.Now().UTC(),
		SummaryIndex: buildSessionEvidenceArchiveIndex(records),
		CaptureFiles: []sessionEvidenceArchiveFile{},
	}

	for _, record := range records {
		base := sessionEvidenceArchiveBaseName(record)
		jsonName := base + ".json"
		markdownName := base + ".md"
		payload := parseSessionEvidencePayload(record.PayloadJSON)
		archive.CaptureFiles = append(archive.CaptureFiles, sessionEvidenceArchiveFile{
			ID:               record.ID,
			JSONFileName:     jsonName,
			MarkdownFileName: markdownName,
			JSONCapture:      payload,
			MarkdownReport:   buildSessionEvidenceArchiveMarkdown(record, payload),
		})
	}

	return archive
}

func buildSessionEvidenceArchiveIndex(records []model.SessionEvidenceRecord) sessionEvidenceArchiveIndex {
	index := sessionEvidenceArchiveIndex{
		CaptureCount: len(records),
		Captures:     []sessionEvidenceArchiveItem{},
	}
	for _, record := range records {
		base := sessionEvidenceArchiveBaseName(record)
		jsonName := base + ".json"
		markdownName := base + ".md"
		index.Captures = append(index.Captures, sessionEvidenceArchiveItem{
			ID:               record.ID,
			CreatedAt:        record.CreatedAt,
			OverallState:     record.OverallState,
			ReadyzState:      record.ReadyzState,
			PreflightState:   record.PreflightState,
			HostID:           record.HostID,
			WireHostURL:      record.WireHostURL,
			OperatorNotes:    record.OperatorNotes,
			JSONFileName:     jsonName,
			MarkdownFileName: markdownName,
		})
	}
	return index
}

func parseSessionEvidencePayload(raw string) any {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return map[string]any{
			"payload_parse_error": err.Error(),
			"payload_raw":         raw,
		}
	}
	return payload
}

func sessionEvidenceArchiveBaseName(record model.SessionEvidenceRecord) string {
	host := strings.TrimSpace(record.HostID)
	if host == "" {
		host = "cabinet"
	}
	host = sanitizeArchiveName(host)
	stamp := record.CreatedAt.UTC().Format("20060102T150405Z")
	return fmt.Sprintf("%s-session-evidence-%d-%s", host, record.ID, stamp)
}

func sanitizeArchiveName(raw string) string {
	builder := strings.Builder{}
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteRune('-')
	}
	value := strings.Trim(builder.String(), "-")
	if value == "" {
		return "cabinet"
	}
	return value
}

func buildSessionEvidenceArchiveMarkdown(record model.SessionEvidenceRecord, payload any) string {
	lines := []string{
		"# Session Evidence Capture",
		"",
		"- Record ID: " + strconv.FormatInt(record.ID, 10),
		"- Captured at: " + record.CreatedAt.UTC().Format(time.RFC3339),
		"- Session state: " + record.OverallState,
		"- Readyz state: " + record.ReadyzState,
		"- Preflight state: " + record.PreflightState,
		"- Host ID: " + record.HostID,
		"- Wire host URL: " + record.WireHostURL,
		"- Operator notes: " + strings.TrimSpace(record.OperatorNotes),
		"",
		"## Payload",
		"",
		"```json",
	}
	pretty, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		lines = append(lines, "{}")
	} else {
		lines = append(lines, string(pretty))
	}
	lines = append(lines, "```")
	return strings.Join(lines, "\n")
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

func queryBool(r *http.Request, key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	case "":
		return fallback
	default:
		return fallback
	}
}
