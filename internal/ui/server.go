package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type Server struct {
	tmpl             *template.Template
	assetVersion     string
	dashboardJSETag  string
	dashboardCSSETag string
}

func NewServer() (*Server, error) {
	tmpl, err := template.New("dashboard").Parse(dashboardHTML)
	if err != nil {
		return nil, err
	}
	jsDigest := sha256.Sum256([]byte(dashboardJS))
	cssDigest := sha256.Sum256([]byte(dashboardCSS))
	versionDigest := sha256.Sum256([]byte(dashboardCSS + "\x00" + dashboardJS))
	return &Server{
		tmpl:             tmpl,
		assetVersion:     hex.EncodeToString(versionDigest[:8]),
		dashboardJSETag:  fmt.Sprintf("\"%s\"", hex.EncodeToString(jsDigest[:])),
		dashboardCSSETag: fmt.Sprintf("\"%s\"", hex.EncodeToString(cssDigest[:])),
	}, nil
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
	setDashboardNoCacheHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.tmpl.Execute(w, map[string]string{
		"Title":           "G2S Muting Controller",
		"DashboardCSSURL": "/static/dashboard.css?v=" + s.assetVersion,
		"DashboardJSURL":  "/static/dashboard.js?v=" + s.assetVersion,
	})
}

func (s *Server) styles(w http.ResponseWriter, r *http.Request) {
	if writeNotModifiedIfETagMatches(w, r, s.dashboardCSSETag) {
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	setStaticAssetRevalidateHeaders(w, s.dashboardCSSETag)
	_, _ = w.Write([]byte(dashboardCSS))
}

func (s *Server) script(w http.ResponseWriter, r *http.Request) {
	if writeNotModifiedIfETagMatches(w, r, s.dashboardJSETag) {
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	setStaticAssetRevalidateHeaders(w, s.dashboardJSETag)
	_, _ = w.Write([]byte(dashboardJS))
}

func setDashboardNoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func setStaticAssetRevalidateHeaders(w http.ResponseWriter, etag string) {
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	w.Header().Set("ETag", etag)
}

func writeNotModifiedIfETagMatches(w http.ResponseWriter, r *http.Request, etag string) bool {
	if etag == "" {
		return false
	}
	ifNoneMatch := strings.TrimSpace(r.Header.Get("If-None-Match"))
	if ifNoneMatch == "" {
		return false
	}
	if ifNoneMatch == "*" {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	for _, item := range strings.Split(ifNoneMatch, ",") {
		if strings.TrimSpace(item) == etag {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}
	return false
}
