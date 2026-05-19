package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func requireMutationAuth(w http.ResponseWriter, r *http.Request, expectedToken string) bool {
	token := strings.TrimSpace(expectedToken)
	if token == "" {
		return true
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		writeUnauthorizedBearer(w, "bearer token required")
		return false
	}

	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		writeUnauthorizedBearer(w, "invalid bearer token")
		return false
	}
	providedToken := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
	if providedToken == "" || subtle.ConstantTimeCompare([]byte(providedToken), []byte(token)) != 1 {
		writeUnauthorizedBearer(w, "invalid bearer token")
		return false
	}

	return true
}

func writeUnauthorizedBearer(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="g2s-mute"`)
	http.Error(w, "unauthorized: "+message, http.StatusUnauthorized)
}
