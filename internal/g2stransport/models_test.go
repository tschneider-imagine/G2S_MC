package g2stransport

import "testing"

func TestCaptureEndpointAllowed(t *testing.T) {
	cases := []struct {
		endpoint string
		allowed  bool
	}{
		{endpoint: "http://127.0.0.1:18080/capture", allowed: true},
		{endpoint: "http://localhost:18080/capture", allowed: true},
		{endpoint: "http://[::1]:18080/capture", allowed: true},
		{endpoint: "http://10.10.10.10:18080/capture", allowed: false},
	}
	for _, tc := range cases {
		got, _ := CaptureEndpointAllowed(tc.endpoint)
		if got != tc.allowed {
			t.Fatalf("CaptureEndpointAllowed(%q)=%v want %v", tc.endpoint, got, tc.allowed)
		}
	}
}
