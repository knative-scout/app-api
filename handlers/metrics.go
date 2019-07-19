package handlers

import (
	"fmt"
	"net/http"

	"github.com/kscout/serverless-registry-api/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricsHandler exports custom API metrics.
//
// The PanicHandler is the only other middleware handler that exports metrics.
// This is neccessary due to the nature of the metrics it collections.
type MetricsHandler struct {
	BaseHandler

	// Handler will actually handle requests
	Handler http.Handler
}

// ServeHTTP will observe custom metrics and let the .Handler handle the request
func (h MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Pre-request metrics
	respCode := http.StatusOK

	// metricsW will be used as a ResponseWriter by all child handlers. We will
	// capture response HTTP codes using OnWriteHeader.
	metricsW := metrics.MetricsResponseWriter{
		ResponseWriter: w,
		OnWriteHeader: func(code int) {
			respCode = code
		},
	}

	durationTimer := h.Metrics.StartTimer()

	// Handle
	h.Handler.ServeHTTP(metricsW, r)

	// Post-request metrics
	durationTimer.Finish(h.Metrics.APIResponseDurationsMilliseconds.With(prometheus.Labels{
		"path":        r.URL.Path,
		"method":      r.Method,
		"status_code": fmt.Sprintf("%d", respCode),
	}))
}
