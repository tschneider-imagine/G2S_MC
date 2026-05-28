package g2s

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/inbound"
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
	req.RemoteAddr = "192.168.55.10:9443"
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "commsOnLineAck") {
		t.Fatalf("expected commsOnLineAck, got %s", body)
	}
	if !strings.Contains(body, `xmlns:g2s="http://www.gamingstandards.com/g2s/schemas/v1.0.3"`) {
		t.Fatalf("expected g2s namespace, got %s", body)
	}
	if !strings.Contains(body, `syncTimer="30000"`) {
		t.Fatalf("expected syncTimer=30000, got %s", body)
	}
	if strings.Contains(body, "g2sResponse") {
		t.Fatalf("unexpected g2sResponse wrapper in %s", body)
	}
	if strings.Contains(body, "commsOnLineResponse") {
		t.Fatalf("unexpected commsOnLineResponse in %s", body)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := eng.Snapshot()
		if len(snapshot.EGMs) == 1 &&
			snapshot.EGMs[0].Status == model.EGMGreen &&
			!snapshot.EGMs[0].LastSeen.IsZero() &&
			snapshot.EGMs[0].LastEndpointIP == "192.168.55.10" &&
			snapshot.EGMs[0].LastEndpointPort == 9443 &&
			!snapshot.EGMs[0].LastEndpointSeenAt.IsZero() &&
			len(snapshot.EGMs[0].RecentEndpoints) == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected EGM last seen and endpoint metadata to update")
}

func TestCommsOnlineDiscoversUnknownEGM(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	NewServer("HOST-1", eng).RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(`<g2sBody egmId="EGM-9"><commsOnLine/></g2sBody>`))
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
		for _, egm := range snapshot.EGMs {
			if egm.ID == "EGM-9" {
				if egm.Source != model.EGMSourceDiscovered {
					t.Fatalf("expected discovered source, got %q", egm.Source)
				}
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected discovered EGM to appear in snapshot")
}

func TestKeepAliveDiscoversUnknownEGM(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	NewServer("HOST-1", eng).RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(`<g2sBody egmId="EGM-8"><keepAlive/></g2sBody>`))
	req.RemoteAddr = "10.11.12.13:9555"
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "keepAliveAck") {
		t.Fatalf("expected keepAliveAck, got %s", body)
	}
	if strings.Contains(body, "g2sResponse") {
		t.Fatalf("unexpected g2sResponse wrapper in %s", body)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := eng.Snapshot()
		for _, egm := range snapshot.EGMs {
			if egm.ID == "EGM-8" {
				if egm.Source != model.EGMSourceDiscovered {
					t.Fatalf("expected discovered source, got %q", egm.Source)
				}
				if egm.Status != model.EGMGreen {
					t.Fatalf("expected GREEN status, got %s", egm.Status)
				}
				if egm.LastEndpointIP != "10.11.12.13" {
					t.Fatalf("last_endpoint_ip = %q, want 10.11.12.13", egm.LastEndpointIP)
				}
				if egm.LastEndpointPort != 9555 {
					t.Fatalf("last_endpoint_port = %d, want 9555", egm.LastEndpointPort)
				}
				if len(egm.RecentEndpoints) != 1 {
					t.Fatalf("recent_endpoints len = %d, want 1", len(egm.RecentEndpoints))
				}
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected discovered EGM to appear in snapshot")
}

func TestKeepAliveWithoutEGMIDFallsBackToSourceIPDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	NewServer("HOST-1", eng).RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(`<g2sBody><keepAlive/></g2sBody>`))
	req.RemoteAddr = "198.51.100.50:9443"
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "keepAliveAck") {
		t.Fatalf("expected keepAliveAck, got %s", rr.Body.String())
	}

	expectedID := "DISCOVERED-IP-198-51-100-50"
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := eng.Snapshot()
		for _, egm := range snapshot.EGMs {
			if egm.ID != expectedID {
				continue
			}
			if egm.Source != model.EGMSourceDiscovered {
				t.Fatalf("expected discovered source, got %q", egm.Source)
			}
			if egm.LastEndpointIP != "198.51.100.50" {
				t.Fatalf("last_endpoint_ip = %q, want 198.51.100.50", egm.LastEndpointIP)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected source-IP discovered EGM to appear in snapshot")
}

func TestInboundContactWithoutEGMIDStillAppearsInSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	NewServer("HOST-1", eng).RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(`<g2sBody><status/></g2sBody>`))
	req.RemoteAddr = "203.0.113.55:9444"
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "g2sAck") {
		t.Fatalf("expected g2sAck response, got %s", body)
	}
	if !strings.Contains(body, `egmId="DISCOVERED-IP-203-0-113-55"`) {
		t.Fatalf("expected fallback egmId in response, got %s", body)
	}

	expectedID := "DISCOVERED-IP-203-0-113-55"
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot := eng.Snapshot()
		for _, egm := range snapshot.EGMs {
			if egm.ID != expectedID {
				continue
			}
			if egm.Source != model.EGMSourceDiscovered {
				t.Fatalf("expected discovered source, got %q", egm.Source)
			}
			if egm.Status != model.EGMGreen {
				t.Fatalf("expected GREEN status, got %s", egm.Status)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected inbound contact to create discovered EGM entry")
}

func TestParseRemoteEndpoint(t *testing.T) {
	ip, port := parseRemoteEndpoint("203.0.113.40:9000")
	if ip != "203.0.113.40" || port != 9000 {
		t.Fatalf("got %q:%d", ip, port)
	}
	ip, port = parseRemoteEndpoint("[2001:db8::1]:9443")
	if ip != "2001:db8::1" || port != 9443 {
		t.Fatalf("got %q:%d", ip, port)
	}
	ip, port = parseRemoteEndpoint("not-a-socket")
	if ip != "not-a-socket" || port != 0 {
		t.Fatalf("got %q:%d", ip, port)
	}
}

func TestFallbackEGMIDFromSourceIP(t *testing.T) {
	if got := fallbackEGMIDFromSourceIP("192.168.10.10"); got != "DISCOVERED-IP-192-168-10-10" {
		t.Fatalf("fallback id = %q", got)
	}
	if got := fallbackEGMIDFromSourceIP("2001:db8::2"); got != "DISCOVERED-IP-2001-db8-2" {
		t.Fatalf("fallback id = %q", got)
	}
	if got := fallbackEGMIDFromSourceIP("   "); got != "" {
		t.Fatalf("fallback id = %q", got)
	}
}

type fakeInboundProcessor struct {
	calls          []inbound.InboundMessage
	outboundCalls  []inbound.OutboundResponse
	err            error
	processResult  inbound.ProcessResult
	recordOutbound error
}

func (f *fakeInboundProcessor) Process(_ context.Context, message inbound.InboundMessage) (inbound.ProcessResult, error) {
	f.calls = append(f.calls, message)
	if f.err != nil {
		return inbound.ProcessResult{}, f.err
	}
	if f.processResult.MessageID != 0 || strings.TrimSpace(f.processResult.EGMID) != "" || strings.TrimSpace(f.processResult.ActionRunID) != "" || f.processResult.OfferedMessage != nil {
		result := f.processResult
		if strings.TrimSpace(result.EGMID) == "" {
			result.EGMID = message.EGMID
		}
		return result, nil
	}
	return inbound.ProcessResult{MessageID: 1, EGMID: message.EGMID}, nil
}

func (f *fakeInboundProcessor) RecordOutboundResponse(_ context.Context, response inbound.OutboundResponse) error {
	f.outboundCalls = append(f.outboundCalls, response)
	return f.recordOutbound
}

func TestInboundProcessorReceivesMessageMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	server := NewServer("HOST-1", eng)
	processor := &fakeInboundProcessor{}
	server.SetInboundProcessor(processor)
	server.RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s?action_run_id=run-1", strings.NewReader(`<g2sBody egmId="EGM-1"><keepAlive/></g2sBody>`))
	req.Header.Set("X-Message-Type", "keepAlive")
	req.RemoteAddr = "192.0.2.10:9555"
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if len(processor.calls) != 1 {
		t.Fatalf("processor calls=%d want 1", len(processor.calls))
	}
	call := processor.calls[0]
	if call.EGMID != "EGM-1" {
		t.Fatalf("egm id=%q", call.EGMID)
	}
	if call.QueryParams["action_run_id"] != "run-1" {
		t.Fatalf("query action_run_id=%q", call.QueryParams["action_run_id"])
	}
	if call.Headers["X-Message-Type"] != "keepAlive" {
		t.Fatalf("x-message-type header=%q", call.Headers["X-Message-Type"])
	}
	if !strings.Contains(call.RawPayload, "<keepAlive/>") {
		t.Fatalf("unexpected payload: %s", call.RawPayload)
	}
}

func TestCommsAckUsesIncomingEGMIDAndIncludesRequiredEnvelopeAttrs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	server := NewServer("1", eng)
	processor := &fakeInboundProcessor{
		processResult: inbound.ProcessResult{
			MessageID:   12,
			EGMID:       "STORED-EGM-999",
			ActionRunID: "run-1",
		},
	}
	server.SetInboundProcessor(processor)
	server.RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(`<g2sBody egmId="WIRE-EGM-123"><commsOnLine/></g2sBody>`))
	req.RemoteAddr = "192.0.2.44:9443"
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	for _, expected := range []string{
		`hostId="1"`,
		`egmId="WIRE-EGM-123"`,
		`dateTimeSent="`,
		"commsOnLineAck",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %q: %s", expected, body)
		}
	}
	if strings.Contains(body, `egmId="STORED-EGM-999"`) {
		t.Fatalf("response used stored egm id instead of incoming payload: %s", body)
	}

	if len(processor.outboundCalls) != 1 {
		t.Fatalf("outbound calls=%d want 1", len(processor.outboundCalls))
	}
	if processor.outboundCalls[0].EGMID != "WIRE-EGM-123" {
		t.Fatalf("recorded outbound egm id=%q want WIRE-EGM-123", processor.outboundCalls[0].EGMID)
	}
}

func TestInboundProcessorErrorDoesNotBreakG2SResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	server := NewServer("HOST-1", eng)
	server.SetInboundProcessor(&fakeInboundProcessor{err: fmt.Errorf("store unavailable")})
	server.RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(`<g2sBody egmId="EGM-1"><commsOnLine/></g2sBody>`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "commsOnLineAck") {
		t.Fatalf("expected commsOnLineAck, got %s", body)
	}
	if strings.Contains(body, "g2sResponse") {
		t.Fatalf("unexpected g2sResponse wrapper in %s", body)
	}
	if strings.Contains(body, "commsOnLineResponse") {
		t.Fatalf("unexpected commsOnLineResponse in %s", body)
	}
}

func TestG2SResponseRecordedAsOutbound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	server := NewServer("HOST-1", eng)
	processor := &fakeInboundProcessor{
		processResult: inbound.ProcessResult{
			MessageID:   9001,
			EGMID:       "EGM-1",
			ActionRunID: "run-1",
		},
	}
	server.SetInboundProcessor(processor)
	server.RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(`<g2sBody egmId="EGM-1"><keepAlive/></g2sBody>`))
	req.RemoteAddr = "192.168.10.10:9443"
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if len(processor.outboundCalls) != 1 {
		t.Fatalf("outbound calls=%d want 1", len(processor.outboundCalls))
	}
	captured := processor.outboundCalls[0]
	if captured.MessageType != "keepAliveAck" {
		t.Fatalf("message_type=%q want keepAliveAck", captured.MessageType)
	}
	if captured.RelatedMessageID != 9001 {
		t.Fatalf("related_message_id=%d want 9001", captured.RelatedMessageID)
	}
	if captured.FromEndpoint != "/g2s" || captured.ToEndpoint != "192.168.10.10:9443" {
		t.Fatalf("endpoints from=%q to=%q", captured.FromEndpoint, captured.ToEndpoint)
	}
	if captured.TransportMode != "HOST_LISTENER" {
		t.Fatalf("transport_mode=%q", captured.TransportMode)
	}
	if !strings.Contains(captured.RawPayload, "keepAliveAck") {
		t.Fatalf("payload missing keepAliveAck: %s", captured.RawPayload)
	}
}

func TestG2SOutboundRecordIncludesOfferedMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	server := NewServer("HOST-1", eng)
	processor := &fakeInboundProcessor{
		processResult: inbound.ProcessResult{
			MessageID:   1001,
			EGMID:       "EGM-1",
			ActionRunID: "run-1",
			OfferedMessage: &inbound.OfferedMessage{
				MessageID:       77,
				ActionRunID:     "run-1",
				ActionStepID:    "step-2",
				TemplateID:      "template-generic-g2s-action",
				TemplateVersion: "3",
				MessageType:     "NOTICE",
				RawPayload:      "<command>silence</command>",
				OfferedAt:       time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC),
				OfferCount:      2,
			},
		},
	}
	server.SetInboundProcessor(processor)
	server.RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(`<g2sBody egmId="EGM-1"><keepAlive/></g2sBody>`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if len(processor.outboundCalls) != 1 {
		t.Fatalf("outbound calls=%d want 1", len(processor.outboundCalls))
	}
	captured := processor.outboundCalls[0]
	if captured.OfferedMessageID != 77 {
		t.Fatalf("offered_message_id=%d want 77", captured.OfferedMessageID)
	}
	if captured.ActionStepID != "step-2" {
		t.Fatalf("action_step_id=%q want step-2", captured.ActionStepID)
	}
	if captured.TemplateID != "template-generic-g2s-action" || captured.TemplateVersion != "3" {
		t.Fatalf("template ref=%s@%s", captured.TemplateID, captured.TemplateVersion)
	}
}

func TestCommsOnlineSOAPInboundGetsSOAPResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	NewServer("HOST-1", eng).RegisterRoutes(mux, "/g2s")

	soapReq := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">` +
		`<soap:Body><g2sBody egmId="EGM-1"><commsOnLine/></g2sBody></soap:Body>` +
		`</soap:Envelope>`
	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(soapReq))
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<soap:Envelope") {
		t.Fatalf("expected SOAP envelope, got %s", body)
	}
	if !strings.Contains(body, "commsOnLineAck") {
		t.Fatalf("expected commsOnLineAck, got %s", body)
	}
	if strings.Contains(body, "g2sResponse") {
		t.Fatalf("unexpected g2sResponse wrapper in %s", body)
	}
	if strings.Contains(body, "commsOnLineResponse") {
		t.Fatalf("unexpected commsOnLineResponse in %s", body)
	}
}

func TestInboundOfferedMessageIncludedInResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng := engine.New("controller", []config.EGM{{EGMID: "EGM-1", IPAddress: "127.0.0.1", Port: 9443}})
	eng.Start(ctx)

	mux := http.NewServeMux()
	server := NewServer("HOST-1", eng)
	server.SetInboundProcessor(&fakeInboundProcessorWithOffer{})
	server.RegisterRoutes(mux, "/g2s")

	req := httptest.NewRequest(http.MethodPost, "/g2s", strings.NewReader(`<g2sBody egmId="EGM-1"><keepAlive/></g2sBody>`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	for _, expected := range []string{
		"pendingDelivery",
		`messageId="77"`,
		`actionRunId="run-1"`,
		`<payload>&lt;command&gt;silence&lt;/command&gt;</payload>`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %q: %s", expected, body)
		}
	}
}

type fakeInboundProcessorWithOffer struct{}

func (f *fakeInboundProcessorWithOffer) Process(_ context.Context, message inbound.InboundMessage) (inbound.ProcessResult, error) {
	return inbound.ProcessResult{
		MessageID: 1,
		EGMID:     message.EGMID,
		OfferedMessage: &inbound.OfferedMessage{
			MessageID:       77,
			ActionRunID:     "run-1",
			ActionStepID:    "step-1",
			TemplateID:      "template-generic-g2s-action",
			TemplateVersion: "1",
			MessageType:     "NOTICE",
			RawPayload:      "<command>silence</command>",
			OfferedAt:       time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC),
			OfferCount:      1,
		},
	}, nil
}
