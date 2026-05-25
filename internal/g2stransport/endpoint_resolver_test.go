package g2stransport

import (
	"strings"
	"testing"

	"github.com/tschneider-imagine/G2S_MC/internal/egms"
	"github.com/tschneider-imagine/G2S_MC/internal/templates"
)

func TestResolveDeliveryTargetUsesFullEndpointURL(t *testing.T) {
	target, err := ResolveDeliveryTarget(DeliveryTargetResolveRequest{
		EGMRecord: &egms.EGMRecord{
			EGMID:        "EGM-001",
			EndpointPath: "https://egm.local:9443/g2s",
		},
		FallbackMethod:      "POST",
		FallbackContentType: "application/xml",
		FallbackTimeoutMS:   4000,
	})
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	if target.EndpointURL != "https://egm.local:9443/g2s" {
		t.Fatalf("endpoint=%q", target.EndpointURL)
	}
}

func TestResolveDeliveryTargetRequiresDefaultsForIPAndPath(t *testing.T) {
	record := &egms.EGMRecord{
		EGMID:        "EGM-001",
		IPAddress:    "10.1.2.3",
		EndpointPath: "/g2s",
	}
	_, err := ResolveDeliveryTarget(DeliveryTargetResolveRequest{
		EGMRecord:         record,
		FallbackMethod:    "POST",
		FallbackTimeoutMS: 3000,
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "missing configured delivery scheme") {
		t.Fatalf("expected missing scheme error, got %v", err)
	}

	target, err := ResolveDeliveryTarget(DeliveryTargetResolveRequest{
		EGMRecord:         record,
		FallbackMethod:    "POST",
		FallbackTimeoutMS: 3000,
		Defaults: EndpointDefaults{
			Scheme: "https",
			Port:   8443,
		},
	})
	if err != nil {
		t.Fatalf("resolve with defaults: %v", err)
	}
	if target.EndpointURL != "https://10.1.2.3:8443/g2s" {
		t.Fatalf("endpoint=%q", target.EndpointURL)
	}
}

func TestResolveDeliveryTargetAppliesTemplateEndpointQuirks(t *testing.T) {
	target, err := ResolveDeliveryTarget(DeliveryTargetResolveRequest{
		EGMRecord: &egms.EGMRecord{
			EGMID:        "EGM-001",
			EndpointPath: "https://egm.local/g2s",
		},
		TemplateVersion: &templates.G2STemplateVersion{
			EndpointQuirksJSON: `{"method":"PUT","content_type":"application/soap+xml","headers":{"SOAPAction":"urn:g2s","X-Test":"1"},"timeout_ms":2500}`,
		},
		FallbackMethod:      "POST",
		FallbackContentType: "application/xml",
		FallbackTimeoutMS:   4000,
	})
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	if target.Method != "PUT" {
		t.Fatalf("method=%q", target.Method)
	}
	if target.ContentType != "application/soap+xml" {
		t.Fatalf("content-type=%q", target.ContentType)
	}
	if target.TimeoutMS != 2500 {
		t.Fatalf("timeout=%d", target.TimeoutMS)
	}
	if target.Headers["SOAPAction"] != "urn:g2s" {
		t.Fatalf("headers=%v", target.Headers)
	}
}

func TestResolveDeliveryTargetInvalidEndpointQuirksFailsClearly(t *testing.T) {
	_, err := ResolveDeliveryTarget(DeliveryTargetResolveRequest{
		EGMRecord: &egms.EGMRecord{
			EGMID:        "EGM-001",
			EndpointPath: "https://egm.local/g2s",
		},
		TemplateVersion: &templates.G2STemplateVersion{
			EndpointQuirksJSON: `{"method":"POST",`,
		},
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "endpoint quirks") {
		t.Fatalf("expected endpoint quirks error, got %v", err)
	}
}

func TestEndpointDefaultsFromHostURL(t *testing.T) {
	defaults := EndpointDefaultsFromHostURL("https://controller.local:9443/g2s")
	if defaults.Scheme != "https" || defaults.Port != 9443 {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
}
