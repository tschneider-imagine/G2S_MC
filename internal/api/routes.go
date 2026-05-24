package api

import "net/http"

func RegisterRoutes(mux *http.ServeMux, server *Server) {
	if server == nil {
		return
	}
	server.RegisterRoutes(mux)
}
