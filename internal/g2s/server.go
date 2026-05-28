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

type OutboundResponseRecorder interface {
	RecordOutboundResponse(ctx context.Context, response inbound.OutboundResponse) error
}

func (s *Server) SetInboundProcessor(processor InboundProcessor) {
	s.inbounds = processor
}

func (s *Server) RegisterRoutes(mux *http.ServeMux, endpointPath string) {
	mux.HandleFunc(endpointPath, s.handleG2S)
	if strings.EqualFold(strings.TrimSpace(endpointPath), "/g2s") {
		if endpointPath != "/g2s" {
			mux.HandleFunc("/g2s", s.handleG2S)
		}
		if endpointPath != "/G2s" {
			mux.HandleFunc("/G2s", s.handleG2S)
		}
	}
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
	incomingEGMID := findAttribute(message, "egmId")
	if incomingEGMID == "" {
		incomingEGMID = findElement(message, "egmId")
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
			EGMID:        incomingEGMID,
			RawPayload:   message,
			Headers:      headers,
			QueryParams:  query,
		})
	}
	egmID := strings.TrimSpace(incomingEGMID)
	if strings.TrimSpace(egmID) == "" {
		egmID = fallbackEGMIDFromSourceIP(sourceIP)
	}

	responseName := "g2sAck"
	responsePayload := ""
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
		responseName = "commsOnLineAck"
		responsePayload = buildSOAPCommsAck(s.hostID, egmID, "commsOnLineAck", "30000", inboundResult.OfferedMessage)
	case strings.Contains(message, "keepAlive"):
		s.engine.Submit(engine.Event{
			Type:       engine.EventKeepAlive,
			EGMID:      egmID,
			At:         time.Now(),
			Detail:     "keepalive",
			SourceIP:   sourceIP,
			SourcePort: sourcePort,
		})
		responseName = "keepAliveAck"
		responsePayload = buildSOAPCommsAck(s.hostID, egmID, "keepAliveAck", "", inboundResult.OfferedMessage)
	default:
		if strings.TrimSpace(egmID) != "" {
			s.engine.Submit(engine.Event{
				Type:       engine.EventKeepAlive,
				EGMID:      egmID,
				At:         time.Now(),
				Detail:     "inbound contact",
				SourceIP:   sourceIP,
				SourcePort: sourcePort,
			})
		}
		responseName = "g2sAck"
		responsePayload = buildSOAPG2SAck(s.hostID, egmID)
	}
	writeSOAPResponse(w, responsePayload)
	s.recordOutboundResponse(r.Context(), inboundResult, responseName, egmID, r.URL.Path, r.RemoteAddr, responsePayload)
}

func fallbackEGMIDFromSourceIP(sourceIP string) string {
	value := strings.TrimSpace(sourceIP)
	if value == "" {
		return ""
	}
	var normalized strings.Builder
	normalized.Grow(len(value))
	lastDash := false
	for _, ch := range value {
		switch {
		case ch >= 'a' && ch <= 'z',
			ch >= 'A' && ch <= 'Z',
			ch >= '0' && ch <= '9':
			normalized.WriteRune(ch)
			lastDash = false
		default:
			if !lastDash {
				normalized.WriteRune('-')
				lastDash = true
			}
		}
	}
	segment := strings.Trim(normalized.String(), "-")
	if segment == "" {
		return ""
	}
	return "DISCOVERED-IP-" + segment
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

func writeSOAPResponse(w http.ResponseWriter, payload string) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, _ = w.Write([]byte(payload))
}

func buildSOAPCommsAck(hostID string, egmID string, ackName string, syncTimer string, offered *inbound.OfferedMessage) string {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	builder.WriteString(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:g2s="` + g2sSchemaNamespace + `">`)
	builder.WriteString(`<soap:Body>`)
	builder.WriteString(`<g2s:g2sBody hostId="` + xmlEscape(hostID) + `" egmId="` + xmlEscape(egmID) + `" dateTimeSent="` + xmlEscape(timestamp) + `">`)
	builder.WriteString(`<g2s:communications>`)
	builder.WriteString(`<g2s:` + ackName)
	if strings.TrimSpace(syncTimer) != "" {
		builder.WriteString(` syncTimer="` + xmlEscape(syncTimer) + `"`)
	}
	builder.WriteString(`/>`)
	if offered != nil {
		builder.WriteString(`<pendingDelivery messageId="` + strconv.FormatInt(offered.MessageID, 10) + `" actionRunId="` + xmlEscape(offered.ActionRunID) + `" actionStepId="` + xmlEscape(offered.ActionStepID) + `" templateId="` + xmlEscape(offered.TemplateID) + `" templateVersion="` + xmlEscape(offered.TemplateVersion) + `" offerCount="` + strconv.Itoa(offered.OfferCount) + `" offeredAt="` + xmlEscape(offered.OfferedAt.UTC().Format(time.RFC3339)) + `">`)
		builder.WriteString(`<messageType>` + xmlEscape(offered.MessageType) + `</messageType>`)
		builder.WriteString(`<payload>` + xmlEscape(offered.RawPayload) + `</payload>`)
		builder.WriteString(`</pendingDelivery>`)
	}
	builder.WriteString(`</g2s:communications>`)
	builder.WriteString(`</g2s:g2sBody>`)
	builder.WriteString(`</soap:Body></soap:Envelope>`)
	return builder.String()
}

func buildSOAPG2SAck(hostID string, egmID string) string {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	builder.WriteString(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:g2s="` + g2sSchemaNamespace + `">`)
	builder.WriteString(`<soap:Body>`)
	builder.WriteString(`<g2s:g2sAck hostId="` + xmlEscape(hostID) + `" egmId="` + xmlEscape(egmID) + `" dateTimeSent="` + xmlEscape(timestamp) + `"/>`)
	builder.WriteString(`</soap:Body></soap:Envelope>`)
	return builder.String()
}

func (s *Server) recordOutboundResponse(ctx context.Context, inboundResult inbound.ProcessResult, messageType string, egmID string, fromEndpoint string, toEndpoint string, payload string) {
	recorder, ok := s.inbounds.(OutboundResponseRecorder)
	if !ok {
		return
	}
	record := inbound.OutboundResponse{
		SentAt:           time.Now().UTC(),
		FromEndpoint:     strings.TrimSpace(fromEndpoint),
		ToEndpoint:       strings.TrimSpace(toEndpoint),
		EGMID:            strings.TrimSpace(egmID),
		ActionRunID:      strings.TrimSpace(inboundResult.ActionRunID),
		MessageType:      strings.TrimSpace(messageType),
		RawPayload:       payload,
		RelatedMessageID: inboundResult.MessageID,
		TransportMode:    "HOST_LISTENER",
	}
	if inboundResult.OfferedMessage != nil {
		record.ActionRunID = firstNonEmpty(strings.TrimSpace(record.ActionRunID), strings.TrimSpace(inboundResult.OfferedMessage.ActionRunID))
		record.ActionStepID = strings.TrimSpace(inboundResult.OfferedMessage.ActionStepID)
		record.TemplateID = strings.TrimSpace(inboundResult.OfferedMessage.TemplateID)
		record.TemplateVersion = strings.TrimSpace(inboundResult.OfferedMessage.TemplateVersion)
		record.OfferedMessageID = inboundResult.OfferedMessage.MessageID
	}
	_ = recorder.RecordOutboundResponse(ctx, record)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
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
