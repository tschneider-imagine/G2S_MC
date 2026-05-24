package api

import (
	"encoding/json"
	"net/http"
)

func RegisterPhase1ARoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v2/inputs", phase1ANotImplemented)
	mux.HandleFunc("/api/v2/actions", phase1ANotImplemented)
	mux.HandleFunc("/api/v2/templates", phase1ANotImplemented)
	mux.HandleFunc("/api/v2/egms", phase1ANotImplemented)
	mux.HandleFunc("/api/v2/comms/messages", phase1ANotImplemented)
	mux.HandleFunc("/api/v2/audit/timeline", phase1ANotImplemented)
}

func phase1ANotImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "phase 1A route stub not wired",
	})
}
