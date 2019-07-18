package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all the available internal metrics
type Metrics struct {
	// APIRequestsTotal is the number of HTTP request made to the API
	APIRequestsTotal prometheus.Counter

	// APIRequestsDurationMilliseconds is the number of milliseconds it takes to
	// complete API requests
	APIRequestsDurationMilliseconds prometheus.Histogram

	// APIHandlersPanicsTotal is the number of times HTTP request handlers have paniced
	APIHandlersPanicsTotal prometheus.Counter

	// APIErrorResponsesTotal is the number of HTTP responses which do not have a 2xx status code
	APIErrorResponsesTotal prometheus.Counter

	// JobsSubmittedTotal is the number of jobs which are submitted
	JobsSubmittedTotal prometheus.Counter

	// JobsRunDurationMilliseconds is the number of milliseconds jobs run for
	JobsRunDurationMilliseconds prometheus.Histogram

	// JobsFailuresTotal is the number of jobs which fail
	JobsFailuresTotal prometheus.Counter
}

// NewMetrics creates a Metrics struct with all the Prometheus metrics recorders initialized
func NewMetrics() Metrics {
	return Metrics{
		APIRequestsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "api",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests made to the API",
		}),
		APIRequestsDurationMilliseconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "api",
			Name:      "requests_duration_milliseconds",
			Help:      "Duration, in milliseconds, of API requests",
		}),
		APIHandlersPanicsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "api",
			Name:      "handlers_panics_total",
			Help:      "Total number of HTTP handlers which have panicked while processing a request",
		}),
		APIErrorResponsesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "api",
			Name:      "error_responses_total",
			Help:      "Total number of HTTP responses with a non 2xx status code",
		}),
		JobsSubmittedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "jobs",
			Name:      "submitted_total",
			Help:      "Total number of jobs submitted",
		}),
		JobsRunDurationMilliseconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "jobs",
			Name:      "run_duration_milliseconds",
			Help:      "Duration, in milliseconds, of jobs",
		}),
		JobsFailuresTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "jobs",
			Name:      "failures_total",
			Help:      "Total number of jobs which fail",
		}),
	}
}
