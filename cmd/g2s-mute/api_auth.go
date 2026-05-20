package main

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	"github.com/tschneider-imagine/G2S_MC/internal/config"
)

func requireMutationAuthForMethods(next http.HandlerFunc, cfg config.Config, methods ...string) http.HandlerFunc {
	methodSet := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		methodSet[method] = struct{}{}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if _, guarded := methodSet[r.Method]; guarded {
			if !requireMutationAuth(w, r, cfg) {
				return
			}
		}
		next(w, r)
	}
}

func requireMutationAuth(w http.ResponseWriter, r *http.Request, cfg config.Config) bool {
	if !requestRequiresMutationAuth(r, cfg) {
		return true
	}
	token := strings.TrimSpace(cfg.API.AuthToken)

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

func requestRequiresMutationAuth(r *http.Request, cfg config.Config) bool {
	token := strings.TrimSpace(cfg.API.AuthToken)
	if token == "" {
		return false
	}
	return !trustedPrivateNetworkMutationBypassAllowed(r, cfg)
}

func trustedPrivateNetworkMutationBypassAllowed(r *http.Request, cfg config.Config) bool {
	if cfg.WebUI.RequireLogin || !cfg.WebUI.AllowTrustedPrivateNetworkMutations {
		return false
	}
	return isTrustedPrivateNetworkRequest(r)
}

func certificateMaterialRequestAllowed(r *http.Request, cfg config.Config) bool {
	return isLoopbackRequest(r) || trustedPrivateNetworkMutationBypassAllowed(r, cfg)
}

func isTrustedPrivateNetworkRequest(r *http.Request) bool {
	ip := requestIP(r)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

func isLoopbackRequest(r *http.Request) bool {
	ip := requestIP(r)
	return ip != nil && ip.IsLoopback()
}

func requestIP(r *http.Request) net.IP {
	if r == nil {
		return nil
	}
	remote := strings.TrimSpace(r.RemoteAddr)
	if remote == "" {
		return nil
	}

	host := remote
	if parsedHost, _, err := net.SplitHostPort(remote); err == nil {
		host = parsedHost
	}
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	return net.ParseIP(host)
}

func writeUnauthorizedBearer(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="g2s-mute"`)
	http.Error(w, "unauthorized: "+message, http.StatusUnauthorized)
}
