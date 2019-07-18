package handlers

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

// ReqLoggerHandler logs every request. Additionally it records certain metrics about each request.
type ReqLoggerHandler struct {
	BaseHandler

	// Handler to actually handle requests
	Handler http.Handler
}

// ServeHTTP implements http.Handler
func (h ReqLoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Pre-request metrics
	promLabels := prometheus.Labels{"path": r.URL.Path, "method": r.Method}
	h.Metrics.APIRequestsTotal.With(promLabels).Inc()
	durationTimer := h.Metrics.StartTimer(h.Metrics.APIRequestsDurationMilliseconds.With(promLabels))

	// Log
	h.Logger.Debugf("%s %s", r.Method, r.URL.String())

	// Handle
	h.Handler.ServeHTTP(w, r)

	// Post-request metrics
	durationTimer.Finish()
}
