package deliverycheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/actions"
	"github.com/tschneider-imagine/G2S_MC/internal/certs"
	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/g2stransport"
	"github.com/tschneider-imagine/G2S_MC/internal/model"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

type fakeStore struct {
	egms      map[string]egms.EGMRecord
	actions   map[string]actions.ActionDefinition
	templates map[string]templates.G2STemplate
	versions  map[string]templates.G2STemplateVersion
	certs     []model.CertificateInventory
}

func (f *fakeStore) GetEGMRecord(_ context.Context, egmID string) (*egms.EGMRecord, error) {
	row, ok := f.egms[egmID]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) GetActionDefinition(_ context.Context, id string) (*actions.ActionDefinition, error) {
	row, ok := f.actions[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) GetG2STemplate(_ context.Context, id string) (*templates.G2STemplate, error) {
	row, ok := f.templates[id]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) GetActiveG2STemplateVersion(_ context.Context, templateID string) (*templates.G2STemplateVersion, error) {
	row, ok := f.versions[templateID]
	if !ok {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (f *fakeStore) ListCertificateInventory(_ context.Context) ([]model.CertificateInventory, error) {
	rows := make([]model.CertificateInventory, len(f.certs))
	copy(rows, f.certs)
	return rows, nil
}

func TestCheckMissingEGMReturnsError(t *testing.T) {
	service := Service{Store: seedStore(t), Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "UNKNOWN"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.OverallStatus != "ERROR" || len(result.Errors) == 0 {
		t.Fatalf("expected error result: %+v", result)
	}
}

func TestCheckHostListenerMissingEndpointReturnsWarn(t *testing.T) {
	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.EndpointPath = ""
	st.egms["EGM-001"] = row

	service := Service{Store: st, Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.OverallStatus != "WARN" {
		t.Fatalf("expected WARN result for host listener missing endpoint: %+v", result)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("did not expect errors for host listener missing endpoint: %+v", result)
	}
	if result.EndpointRequired {
		t.Fatalf("expected endpoint_required=false for host listener: %+v", result)
	}
	if !containsJoined(result.Warnings, "not required for host listener delivery") {
		t.Fatalf("expected host listener warning, got %+v", result)
	}
}

func TestCheckHostListenerMissingEndpointSkipsNetworkAndTLSChecks(t *testing.T) {
	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.EndpointPath = ""
	st.egms["EGM-001"] = row

	service := Service{Store: st, Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{
		EGMID:               "EGM-001",
		IncludeNetworkCheck: true,
		IncludeTLSCheck:     true,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.NetworkCheck.Status != "SKIPPED" || !containsJoined([]string{result.NetworkCheck.Detail}, "not required for host listener delivery") {
		t.Fatalf("expected network check skipped for host listener: %+v", result.NetworkCheck)
	}
	if result.TLSCheck.Status != "SKIPPED" || !containsJoined([]string{result.TLSCheck.Detail}, "not required for host listener delivery") {
		t.Fatalf("expected tls check skipped for host listener: %+v", result.TLSCheck)
	}
}

func TestCheckHostListenerIncludesListenerAndHostID(t *testing.T) {
	options := seedOptions()
	options.ListenerURL = "https://controller.local/g2s"
	options.HostID = "HOST-1"
	service := Service{Store: seedStore(t), Options: options}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.DeliveryTopology != string(g2stransport.DeliveryTopologyHostListener) {
		t.Fatalf("topology=%q", result.DeliveryTopology)
	}
	if result.ListenerURL != "https://controller.local/g2s" || result.HostID != "HOST-1" {
		t.Fatalf("expected listener/host in result, got %+v", result)
	}
}

func TestCheckFullEndpointResolves(t *testing.T) {
	service := Service{Store: seedStore(t), Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001", ActionID: "emergency-broadcast-trigger"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !result.EndpointConfigured || result.EndpointURL == "" {
		t.Fatalf("expected resolved endpoint: %+v", result)
	}
}

func TestCheckOutboundEndpointWithEndpointBehavesAsExpected(t *testing.T) {
	options := seedOptions()
	options.DeliveryTopology = string(g2stransport.DeliveryTopologyOutboundEndpoint)
	service := Service{Store: seedStore(t), Options: options}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.OverallStatus != "OK" || !result.EndpointRequired || !result.EndpointConfigured {
		t.Fatalf("expected outbound endpoint check to succeed: %+v", result)
	}
}

func TestCheckFullHTTPSEndpointResolves(t *testing.T) {
	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.EndpointPath = "https://egm.local:9443/g2s"
	st.egms["EGM-001"] = row

	service := Service{Store: st, Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.EndpointURL != "https://egm.local:9443/g2s" {
		t.Fatalf("endpoint=%q", result.EndpointURL)
	}
}

func TestCheckTemplateEndpointQuirksApplied(t *testing.T) {
	st := seedStore(t)
	version := st.versions["template-generic-g2s-action"]
	version.EndpointQuirksJSON = `{"method":"PUT","content_type":"application/soap+xml","headers":{"SOAPAction":"urn:test"},"timeout_ms":9000}`
	st.versions["template-generic-g2s-action"] = version

	service := Service{Store: st, Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.Method != "PUT" || result.ContentType != "application/soap+xml" || result.Headers["SOAPAction"] != "urn:test" {
		t.Fatalf("quirks not applied: %+v", result)
	}
}

func TestCheckTemplateEndpointPathQuirksApplied(t *testing.T) {
	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.IPAddress = "10.1.2.3"
	row.EndpointPath = "/g2s"
	st.egms["EGM-001"] = row

	version := st.versions["template-generic-g2s-action"]
	version.EndpointQuirksJSON = `{"endpoint_path":"/vendor-g2s","method":"POST","content_type":"application/xml"}`
	st.versions["template-generic-g2s-action"] = version

	service := Service{Store: st, Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.EndpointURL != "http://10.1.2.3:8444/vendor-g2s" {
		t.Fatalf("expected endpoint override, got %q", result.EndpointURL)
	}
}

func TestCheckOutboundInvalidEndpointURLReturnsError(t *testing.T) {
	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.EndpointPath = "http://example.com/%zz"
	st.egms["EGM-001"] = row

	options := seedOptions()
	options.DeliveryTopology = string(g2stransport.DeliveryTopologyOutboundEndpoint)
	service := Service{Store: st, Options: options}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.OverallStatus != "ERROR" || !containsJoined(result.Errors, "invalid endpoint url") {
		t.Fatalf("expected invalid endpoint error: %+v", result)
	}
}

func TestCheckHostListenerEndpointPathWithoutDefaultsReturnsWarn(t *testing.T) {
	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.IPAddress = "10.1.2.3"
	row.EndpointPath = "/g2s"
	st.egms["EGM-001"] = row

	service := Service{
		Store: st,
		Options: Options{
			EndpointDefaults: g2stransport.EndpointDefaults{},
			ClientConfig: g2stransport.HTTPClientConfig{
				DefaultTimeoutMS: 3000,
			},
			DeliveryMode:     "DISABLED",
			DefaultTimeoutMS: 3000,
		},
	}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.OverallStatus != "WARN" {
		t.Fatalf("expected host listener warning status: %+v", result)
	}
	if !containsJoined(result.Warnings, "host listener delivery") {
		t.Fatalf("expected host listener warning, got %+v", result)
	}
}

func TestCheckOutboundEndpointPathWithoutDefaultsReturnsError(t *testing.T) {
	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.IPAddress = "10.1.2.3"
	row.EndpointPath = "/g2s"
	st.egms["EGM-001"] = row

	service := Service{
		Store: st,
		Options: Options{
			DeliveryTopology: string(g2stransport.DeliveryTopologyOutboundEndpoint),
			EndpointDefaults: g2stransport.EndpointDefaults{},
			ClientConfig: g2stransport.HTTPClientConfig{
				DefaultTimeoutMS: 3000,
			},
			DeliveryMode:     "DISABLED",
			DefaultTimeoutMS: 3000,
		},
	}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.OverallStatus != "ERROR" || !containsJoined(result.Errors, "missing configured delivery scheme") {
		t.Fatalf("expected missing scheme error: %+v", result)
	}
}

func TestCheckHostListenerDoesNotUseFallbackEndpointWhenEndpointMissing(t *testing.T) {
	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.IPAddress = "10.1.2.3"
	row.EndpointPath = ""
	st.egms["EGM-001"] = row

	service := Service{
		Store: st,
		Options: Options{
			EndpointDefaults: g2stransport.EndpointDefaultsFromHostURL("https://controller.local:9443/g2s"),
			ClientConfig: g2stransport.HTTPClientConfig{
				DefaultTimeoutMS: 3000,
			},
			DeliveryMode:     "DISABLED",
			DefaultTimeoutMS: 3000,
		},
	}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.EndpointConfigured {
		t.Fatalf("expected no fallback endpoint in host listener mode: %+v", result)
	}
	if result.OverallStatus == "ERROR" || len(result.Errors) != 0 {
		t.Fatalf("expected no error for host listener missing endpoint: %+v", result)
	}
}

func TestCheckOutboundMissingEndpointReturnsError(t *testing.T) {
	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.EndpointPath = ""
	st.egms["EGM-001"] = row

	options := seedOptions()
	options.DeliveryTopology = string(g2stransport.DeliveryTopologyOutboundEndpoint)
	service := Service{Store: st, Options: options}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.OverallStatus != "ERROR" || !containsJoined(result.Errors, "missing endpoint") {
		t.Fatalf("expected missing endpoint error: %+v", result)
	}
}

func TestCheckCaptureEndpointMissingEndpointReturnsError(t *testing.T) {
	options := seedOptions()
	options.DeliveryTopology = string(g2stransport.DeliveryTopologyCaptureEndpoint)
	options.CaptureEndpoint = ""
	service := Service{Store: seedStore(t), Options: options}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.OverallStatus != "ERROR" || !containsJoined(result.Errors, "capture endpoint") {
		t.Fatalf("expected capture endpoint error: %+v", result)
	}
}

func TestCheckHostListenerInvalidTemplateQuirksJSONReturnsWarn(t *testing.T) {
	st := seedStore(t)
	version := st.versions["template-generic-g2s-action"]
	version.EndpointQuirksJSON = `{"method":"POST",`
	st.versions["template-generic-g2s-action"] = version

	service := Service{Store: st, Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.OverallStatus != "WARN" || !containsJoined(result.Warnings, "endpoint quirks") {
		t.Fatalf("expected quirks warning: %+v", result)
	}
}

func TestCheckHostListenerMissingTemplateActionKeyReturnsError(t *testing.T) {
	service := Service{Store: seedStore(t), Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{
		EGMID:             "EGM-001",
		TemplateID:        "template-generic-g2s-action",
		TemplateActionKey: "missing_action_key",
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.OverallStatus != "ERROR" || !containsJoined(result.Errors, "not found in template") {
		t.Fatalf("expected missing action key error: %+v", result)
	}
}

func TestCheckIncludesCertificateSummaryAndNoPrivateKeyLeak(t *testing.T) {
	service := Service{Store: seedStore(t), Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(result.CertificateSummary) == 0 {
		t.Fatal("expected certificate summary")
	}
	serialized := stringifyResult(result)
	if strings.Contains(serialized, "BEGIN PRIVATE KEY") {
		t.Fatalf("result leaked private key material: %s", serialized)
	}
}

func TestCheckSanitizesPrivateKeyMaterialFromCertificateErrors(t *testing.T) {
	st := seedStore(t)
	now := time.Now().UTC()
	st.certs = []model.CertificateInventory{
		{
			Role:          "g2s_client_key",
			Path:          "/certs/client.key",
			Status:        "INVALID",
			Error:         "failed parse: -----BEGIN PRIVATE KEY-----\nabc123\n-----END PRIVATE KEY-----",
			LastCheckedAt: now,
		},
	}

	service := Service{Store: st, Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	serialized := stringifyResult(result)
	if strings.Contains(serialized, "PRIVATE KEY") || strings.Contains(serialized, "abc123") {
		t.Fatalf("expected key material to be redacted: %s", serialized)
	}
}

func TestCheckNetworkCheckSkippedByDefault(t *testing.T) {
	service := Service{Store: seedStore(t), Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.NetworkCheck.Status != "SKIPPED" {
		t.Fatalf("expected network check skipped: %+v", result.NetworkCheck)
	}
}

func TestCheckNetworkCheckCanSucceed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.EndpointPath = server.URL
	st.egms["EGM-001"] = row

	service := Service{Store: st, Options: seedOptions()}
	result, err := service.Check(context.Background(), CheckRequest{
		EGMID:               "EGM-001",
		IncludeNetworkCheck: true,
		TimeoutMS:           2000,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.NetworkCheck.Status != "OK" {
		t.Fatalf("expected network check OK: %+v", result.NetworkCheck)
	}
}

func TestCheckTLSCheckCanSucceedWithConfiguredCA(t *testing.T) {
	paths := generateCerts(t)
	server := newTLSServer(t, paths)
	defer server.Close()

	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.EndpointPath = server.URL
	st.egms["EGM-001"] = row

	service := Service{
		Store: st,
		Options: Options{
			EndpointDefaults: g2stransport.EndpointDefaultsFromHostURL("https://localhost:443/g2s"),
			ClientConfig: g2stransport.HTTPClientConfig{
				TLSRequired:      true,
				RootCAPath:       paths.CACert,
				DefaultTimeoutMS: 2000,
			},
			DeliveryMode:     "HTTP",
			DefaultTimeoutMS: 2000,
		},
	}
	result, err := service.Check(context.Background(), CheckRequest{
		EGMID:           "EGM-001",
		IncludeTLSCheck: true,
		TimeoutMS:       2000,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.TLSCheck.Status != "OK" {
		t.Fatalf("expected tls check OK: %+v", result.TLSCheck)
	}
}

func TestCheckTLSCheckTrustFailure(t *testing.T) {
	paths := generateCerts(t)
	server := newTLSServer(t, paths)
	defer server.Close()

	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.EndpointPath = server.URL
	st.egms["EGM-001"] = row

	service := Service{
		Store: st,
		Options: Options{
			EndpointDefaults: g2stransport.EndpointDefaultsFromHostURL("https://localhost:443/g2s"),
			ClientConfig: g2stransport.HTTPClientConfig{
				TLSRequired:      true,
				RootCAPath:       "",
				DefaultTimeoutMS: 2000,
			},
			DeliveryMode:     "HTTP",
			DefaultTimeoutMS: 2000,
		},
	}
	result, err := service.Check(context.Background(), CheckRequest{
		EGMID:           "EGM-001",
		IncludeTLSCheck: true,
		TimeoutMS:       2000,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.TLSCheck.Status != "ERROR" {
		t.Fatalf("expected tls error: %+v", result.TLSCheck)
	}
	if strings.TrimSpace(result.TLSCheck.Detail) == "" {
		t.Fatalf("expected non-empty TLS error detail: %+v", result.TLSCheck)
	}
	if strings.Contains(strings.ToUpper(result.TLSCheck.Detail), "PRIVATE KEY") {
		t.Fatalf("tls error leaked key material: %+v", result.TLSCheck)
	}
}

func TestCheckTLSCheckSkippedForHTTPEndpoint(t *testing.T) {
	st := seedStore(t)
	row := st.egms["EGM-001"]
	row.EndpointPath = "http://127.0.0.1:18080/g2s"
	st.egms["EGM-001"] = row

	service := Service{
		Store: st,
		Options: Options{
			EndpointDefaults: seedOptions().EndpointDefaults,
			ClientConfig: g2stransport.HTTPClientConfig{
				TLSRequired:      true,
				DefaultTimeoutMS: 2000,
			},
			DeliveryMode:     "HTTP",
			DefaultTimeoutMS: 2000,
		},
	}
	result, err := service.Check(context.Background(), CheckRequest{
		EGMID:           "EGM-001",
		IncludeTLSCheck: true,
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.TLSCheck.Status != "SKIPPED" || !strings.Contains(strings.ToLower(result.TLSCheck.Detail), "non-https") {
		t.Fatalf("expected TLS check skipped, got %+v", result.TLSCheck)
	}
}

func TestCheckReadOnlyDoesNotRequireClientCertFiles(t *testing.T) {
	service := Service{
		Store: seedStore(t),
		Options: Options{
			EndpointDefaults: seedOptions().EndpointDefaults,
			ClientConfig: g2stransport.HTTPClientConfig{
				TLSRequired:      true,
				RootCAPath:       "/path/does/not/exist/ca.crt",
				ClientCertPath:   "/path/does/not/exist/client.crt",
				ClientKeyPath:    "/path/does/not/exist/client.key",
				DefaultTimeoutMS: 3000,
			},
			DeliveryMode:     "DISABLED",
			DefaultTimeoutMS: 3000,
		},
	}
	result, err := service.Check(context.Background(), CheckRequest{EGMID: "EGM-001"})
	if err != nil {
		t.Fatalf("read-only check should not require local cert file access: %v", err)
	}
	if result.OverallStatus == "" {
		t.Fatalf("expected populated result, got %+v", result)
	}
}

func seedStore(t *testing.T) *fakeStore {
	t.Helper()
	now := time.Now().UTC()
	return &fakeStore{
		egms: map[string]egms.EGMRecord{
			"EGM-001": {
				EGMID:            "EGM-001",
				DisplayName:      "Cabinet 001",
				IPAddress:        "127.0.0.1",
				EndpointPath:     "http://127.0.0.1:18080/g2s",
				TemplateID:       "template-generic-g2s-action",
				Enabled:          true,
				EmergencyEnabled: true,
			},
		},
		actions: map[string]actions.ActionDefinition{
			"emergency-broadcast-trigger": {
				ID:       "emergency-broadcast-trigger",
				Name:     "Emergency Broadcast Trigger",
				Severity: actions.SeverityEmergency,
				Steps: []actions.ActionStep{{
					ID:                "step-1",
					TemplateActionKey: "emergency_broadcast_silence",
				}},
			},
		},
		templates: map[string]templates.G2STemplate{
			"template-generic-g2s-action": {
				ID:     "template-generic-g2s-action",
				Name:   "Generic G2S Action Template",
				Vendor: "Generic",
				Status: templates.TemplateStatusActive,
			},
		},
		versions: map[string]templates.G2STemplateVersion{
			"template-generic-g2s-action": {
				ID:           "template-generic-g2s-action-v1",
				TemplateID:   "template-generic-g2s-action",
				VersionLabel: "1",
				ActionsJSON:  `{"actions":{"emergency_broadcast_silence":{"message_type":"NOTICE","content_type":"application/xml","payload_template":"<msg/>"}}}`,
			},
		},
		certs: []model.CertificateInventory{
			{Role: "g2s_ca_cert", Path: "/certs/ca.crt", Status: "VALID", LastCheckedAt: now},
			{Role: "g2s_client_cert", Path: "/certs/client.crt", Status: "VALID", LastCheckedAt: now},
			{Role: "g2s_client_key", Path: "/certs/client.key", Status: "VALID", LastCheckedAt: now},
			{Role: "web_server_cert", Path: "/certs/web.crt", Status: "VALID", LastCheckedAt: now},
		},
	}
}

func seedOptions() Options {
	return Options{
		EndpointDefaults: g2stransport.EndpointDefaultsFromHostURL("http://127.0.0.1:8444/g2s"),
		ClientConfig: g2stransport.HTTPClientConfig{
			TLSRequired:      false,
			DefaultTimeoutMS: 3000,
		},
		DeliveryMode:     "DISABLED",
		DefaultTimeoutMS: 3000,
	}
}

func containsJoined(rows []string, fragment string) bool {
	joined := strings.ToLower(strings.Join(rows, " "))
	return strings.Contains(joined, strings.ToLower(fragment))
}

func stringifyResult(result CheckResult) string {
	certDetails := make([]string, 0, len(result.CertificateSummary))
	for _, row := range result.CertificateSummary {
		certDetails = append(certDetails, row.Detail)
	}
	return result.EndpointURL + " " + strings.Join(result.Errors, " ") + " " + strings.Join(result.Warnings, " ") + " " + strings.Join(certDetails, " ")
}

func generateCerts(t *testing.T) certs.DevCertPaths {
	t.Helper()
	paths, err := certs.GenerateDevCerts(certs.DevCertOptions{
		OutputDir: t.TempDir(),
		HostDNS:   []string{"localhost"},
		Now:       time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("generate certs: %v", err)
	}
	return paths
}

func newTLSServer(t *testing.T, paths certs.DevCertPaths) *httptest.Server {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewUnstartedServer(handler)
	certPair, err := tls.LoadX509KeyPair(paths.HostCert, paths.HostKey)
	if err != nil {
		t.Fatalf("load host cert: %v", err)
	}
	server.TLS = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certPair},
	}
	caBytes, err := os.ReadFile(paths.CACert)
	if err != nil {
		t.Fatalf("read ca: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caBytes) {
		t.Fatal("append ca failed")
	}
	server.TLS.ClientCAs = pool
	server.StartTLS()
	return server
}
