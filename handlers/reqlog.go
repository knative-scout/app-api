package handlers

import (
	"net/http"
)

// ReqLoggerHandler logs every request. Additionally it records certain metrics about each request.
type ReqLoggerHandler struct {
	BaseHandler

	// Handler to actually handle requests
	Handler http.Handler
}

// ServeHTTP implements http.Handler
func (h ReqLoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Logger.Debugf("%s %s", r.Method, r.URL.String())
	h.Handler.ServeHTTP(w, r)
}
