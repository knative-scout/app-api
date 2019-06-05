package handlers

import (
	"net/http"
)

// PreFlightOptionsHandler responds to OPTIONS requests with headers which set headers
// required to allow CORS
type PreFlightOptionsHandler struct {
	BaseHandler
}

// ServeHTTP implements http.Handler
func (h PreFlightOptionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}
