package g2stransport

import "testing"

func TestNewSenderAlwaysReturnsHTTPSender(t *testing.T) {
	for _, mode := range []Mode{
		ModeHTTP,
		Mode("UNKNOWN"),
	} {
		sender := NewSender(mode)
		if _, ok := sender.(*HTTPSender); !ok {
			t.Fatalf("mode %q returned %T, want *HTTPSender", mode, sender)
		}
	}
}
