package main

import (
	"net/http"
	"strings"
)

func updateActorNameFromRequest(r *http.Request) string {
	updatedBy := strings.TrimSpace(r.Header.Get("X-Operator"))
	if updatedBy == "" {
		updatedBy = "lab-api"
	}
	return updatedBy
}
