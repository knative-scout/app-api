package handlers

import (
	"net/http"
)

// HealthHandler is used to detemrine if the server is running
type HealthHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.RespondJSON(w, http.StatusOK, map[string]bool{
		"ok": true,
	})
}
