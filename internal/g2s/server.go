package g2s

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tschneider-imagine/G2S_MC/internal/engine"
)

type Server struct {
	hostID string
	engine *engine.Engine
}

func NewServer(hostID string, engine *engine.Engine) *Server {
	return &Server{hostID: hostID, engine: engine}
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

	switch {
	case strings.Contains(message, "commsOnLine") || strings.Contains(message, "commsOnline"):
		s.engine.Submit(engine.Event{Type: engine.EventG2SSessionOnline, EGMID: egmID, At: time.Now(), Detail: "comms online"})
		writeSOAP(w, "commsOnLineAck", s.hostID, egmID)
	case strings.Contains(message, "keepAlive"):
		s.engine.Submit(engine.Event{Type: engine.EventKeepAlive, EGMID: egmID, At: time.Now(), Detail: "keepalive"})
		writeSOAP(w, "keepAliveAck", s.hostID, egmID)
	default:
		writeSOAP(w, "g2sAck", s.hostID, egmID)
	}
}

func writeSOAP(w http.ResponseWriter, name string, hostID string, egmID string) {
	w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>`))
	_, _ = w.Write([]byte(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">`))
	_, _ = w.Write([]byte(`<soap:Body>`))
	_, _ = w.Write([]byte(`<g2sResponse hostId="` + xmlEscape(hostID) + `" egmId="` + xmlEscape(egmID) + `">`))
	_, _ = w.Write([]byte(`<` + name + `/>`))
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
