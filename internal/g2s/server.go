package g2s

import (
	"context"
	"encoding/xml"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/engine"
	"github.com/tschneider-imagine/G2S_MC/internal/inbound"
)

const g2sSchemaNamespace = "http://www.gamingstandards.com/g2s/schemas/v1.0.3"

type Server struct {
	hostID   string
	engine   *engine.Engine
	inbounds InboundProcessor
}

func NewServer(hostID string, engine *engine.Engine) *Server {
	return &Server{hostID: hostID, engine: engine}
}

type InboundProcessor interface {
	Process(ctx context.Context, message inbound.InboundMessage) (inbound.ProcessResult, error)
}

func (s *Server) SetInboundProcessor(processor InboundProcessor) {
	s.inbounds = processor
}

func (s *Server) RegisterRoutes(mux *http.ServeMux, endpointPath string) {
	mux.HandleFunc(endpointPath, s.handleG2S)
}

func (s *Server) handleG2S(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	message := string(body)
	egmID := findAttribute(message, "egmId")
	if egmID == "" {
		egmID = findElement(message, "egmId")
	}
	sourceIP, sourcePort := parseRemoteEndpoint(r.RemoteAddr)

	var inboundResult inbound.ProcessResult
	if s.inbounds != nil {
		headers := map[string]string{}
		for key, values := range r.Header {
			if len(values) == 0 {
				continue
			}
			headers[key] = strings.Join(values, ", ")
		}
		query := map[string]string{}
		for key, values := range r.URL.Query() {
			if len(values) == 0 {
				continue
			}
			query[key] = values[0]
		}
		inboundResult, _ = s.inbounds.Process(r.Context(), inbound.InboundMessage{
			ReceivedAt:   time.Now().UTC(),
			FromEndpoint: strings.TrimSpace(r.RemoteAddr),
			ToEndpoint:   strings.TrimSpace(r.URL.Path),
			RemoteAddr:   strings.TrimSpace(r.RemoteAddr),
			EGMID:        egmID,
			RawPayload:   message,
			Headers:      headers,
			QueryParams:  query,
		})
	}

	switch {
	case strings.Contains(message, "commsOnLine") || strings.Contains(message, "commsOnline"):
		s.engine.Submit(engine.Event{
			Type:       engine.EventG2SSessionOnline,
			EGMID:      egmID,
			At:         time.Now(),
			Detail:     "comms online",
			SourceIP:   sourceIP,
			SourcePort: sourcePort,
		})
		writeSOAP(w, `g2s:commsSyncAck xmlns:g2s="`+g2sSchemaNamespace+`" syncTimer="30000"`, s.hostID, egmID, inboundResult.OfferedMessage)
	case strings.Contains(message, "keepAlive"):
		s.engine.Submit(engine.Event{
			Type:       engine.EventKeepAlive,
			EGMID:      egmID,
			At:         time.Now(),
			Detail:     "keepalive",
			SourceIP:   sourceIP,
			SourcePort: sourcePort,
		})
		writeSOAP(w, "keepAliveAck", s.hostID, egmID, inboundResult.OfferedMessage)
	default:
		writeSOAP(w, "g2sAck", s.hostID, egmID, inboundResult.OfferedMessage)
	}
}

func parseRemoteEndpoint(remoteAddr string) (string, int) {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if remoteAddr == "" {
		return "", 0
	}
	host, portText, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr, 0
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 {
		port = 0
	}
	if parsed := net.ParseIP(host); parsed != nil {
		host = parsed.String()
	}
	return strings.TrimSpace(host), port
}

func writeSOAP(w http.ResponseWriter, name string, hostID string, egmID string, offered *inbound.OfferedMessage) {
	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>`))
	_, _ = w.Write([]byte(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">`))
	_, _ = w.Write([]byte(`<soap:Body>`))
	_, _ = w.Write([]byte(`<g2sResponse hostId="` + xmlEscape(hostID) + `" egmId="` + xmlEscape(egmID) + `">`))
	_, _ = w.Write([]byte(`<` + name + `/>`))
	if offered != nil {
		_, _ = w.Write([]byte(`<pendingDelivery messageId="` + strconv.FormatInt(offered.MessageID, 10) + `" actionRunId="` + xmlEscape(offered.ActionRunID) + `" actionStepId="` + xmlEscape(offered.ActionStepID) + `" templateId="` + xmlEscape(offered.TemplateID) + `" templateVersion="` + xmlEscape(offered.TemplateVersion) + `" offerCount="` + strconv.Itoa(offered.OfferCount) + `" offeredAt="` + xmlEscape(offered.OfferedAt.UTC().Format(time.RFC3339)) + `">`))
		_, _ = w.Write([]byte(`<messageType>` + xmlEscape(offered.MessageType) + `</messageType>`))
		_, _ = w.Write([]byte(`<payload>` + xmlEscape(offered.RawPayload) + `</payload>`))
		_, _ = w.Write([]byte(`</pendingDelivery>`))
	}
	_, _ = w.Write([]byte(`</g2sResponse>`))
	_, _ = w.Write([]byte(`</soap:Body></soap:Envelope>`))
}

func findAttribute(body string, name string) string {
	needle := name + `="`
	start := strings.Index(body, needle)
	if start < 0 {
		return ""
	}
	start += len(needle)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		return ""
	}
	return body[start : start+end]
}

func findElement(body string, name string) string {
	open := "<" + name + ">"
	close := "</" + name + ">"
	start := strings.Index(body, open)
	if start < 0 {
		return ""
	}
	start += len(open)
	end := strings.Index(body[start:], close)
	if end < 0 {
		return ""
	}
	return body[start : start+end]
}

func xmlEscape(value string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}
