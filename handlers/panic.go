package handlers

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/kscout/serverless-registry-api/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

// PanicHandler runs another http.Handler and recovers from any panics which occur
// Prevents server from crashing and recovers from panic.
// Also prints stack trace for panic.

type PanicHandler struct {
	BaseHandler

	// Handler to run
	Handler http.Handler
}

// ServeHTTP implements http.Handler
func (h PanicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Setup metrics.MetricsResponseWriter
	promLabels := prometheus.Labels{"path": r.URL.Path, "method": r.Method}

	w = metrics.MetricsResponseWriter{
		ResponseWriter: w,
		WriteHeaderCallback: func(code int) {
			thisPromLabels := promLabels
			thisPromLabels["status_code"] = fmt.Sprintf("%d", code)
			h.Metrics.APIErrorResponsesTotal.With(thisPromLabels).Inc()
		},
	}

	defer func() {
		if recovery := recover(); recovery != nil {
			// Metrics
			h.Metrics.APIHandlersPanicsTotal.With(promLabels).Inc()

			// Handle panic
			h.Logger.Error(string(debug.Stack()))
			h.Logger.Errorf("panicked while handling request: %#v", recovery)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, err := fmt.Fprintln(w, "{\"error\": \"internal server error\"}")
			if err != nil {
				h.Logger.Fatalf("failed to send panic response: %s", err.Error())
			}
		}
	}()

	h.Handler.ServeHTTP(w, r)
}
