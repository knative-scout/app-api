package handlers

import (
	"net/http"
)

// CORSHandler enables cross origin resource sharing (CORS)
type CORSHandler struct {
	BaseHandler

	// Handler to enabled CORS for
	Handler http.Handler
}

// ServeHTTP runs CorsHandler.Handler with CORS enabled
func (h CORSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	h.Handler.ServeHTTP(w, r)
}
