package ui

import (
	"html/template"
	"net/http"
)

type Server struct {
	tmpl *template.Template
}

func NewServer() (*Server, error) {
	tmpl, err := template.New("dashboard").Parse(dashboardHTML)
	if err != nil {
		return nil, err
	}
	return &Server{tmpl: tmpl}, nil
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.dashboard)
	mux.HandleFunc("/dashboard", s.dashboard)
	mux.HandleFunc("/static/dashboard.css", s.styles)
	mux.HandleFunc("/static/dashboard.js", s.script)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/dashboard" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.tmpl.Execute(w, map[string]string{
		"Title": "G2S Muting Controller",
	})
}

func (s *Server) styles(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write([]byte(dashboardCSS))
}

func (s *Server) script(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write([]byte(dashboardJS))
}
