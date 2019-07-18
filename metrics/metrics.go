package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all the available internal metrics
//
// Fields which are prefixed with API will have the following common labels:
//
// * path - HTTP request path
//
// * method - HTTP request method
//
// Fields which are prefixed with Jobs will have the following common labels:
//
// * job_type - The .Type field of the associated job request
type Metrics struct {
	// APIRequestsTotal is the number of HTTP request made to the API
	APIRequestsTotal *prometheus.CounterVec

	// APIRequestsDurationMilliseconds is the number of milliseconds it takes to
	// complete API requests
	APIRequestsDurationMilliseconds *prometheus.HistogramVec

	// APIHandlersPanicsTotal is the number of times HTTP request handlers have paniced
	APIHandlersPanicsTotal *prometheus.CounterVec

	// APIErrorResponsesTotal is the number of HTTP responses which do not have a 2xx status code
	//
	// Has the additional label: status_code. This label records the status code of error responses.
	APIErrorResponsesTotal *prometheus.CounterVec

	// JobsSubmittedTotal is the number of jobs which are submitted
	JobsSubmittedTotal *prometheus.CounterVec

	// JobsRunDurationMilliseconds is the number of milliseconds jobs run for
	JobsRunDurationMilliseconds *prometheus.HistogramVec

	// JobsFailuresTotal is the number of jobs which fail
	//
	// Has the additional label: failure_type. This label can have the following values:
	//
	// * internal - Error occurred within job
	//
	// * invalid_type - Job was submitted with an incorrect type
	JobsFailuresTotal *prometheus.CounterVec
}

// NewMetrics creates a Metrics struct with all the Prometheus metrics recorders initialized
func NewMetrics() Metrics {
	metrics := Metrics{
		APIRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "api",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests made to the API",
		}, []string{"path", "method"}),
		APIRequestsDurationMilliseconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "api",
			Name:      "requests_duration_milliseconds",
			Help:      "Duration, in milliseconds, of API requests",
		}, []string{"path", "method"}),
		APIHandlersPanicsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "api",
			Name:      "handlers_panics_total",
			Help:      "Total number of HTTP handlers which have panicked while processing a request",
		}, []string{"path", "method"}),
		APIErrorResponsesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "api",
			Name:      "error_responses_total",
			Help:      "Total number of HTTP responses with a non 2xx status code",
		}, []string{"path", "method", "status_code"}),
		JobsSubmittedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "jobs",
			Name:      "submitted_total",
			Help:      "Total number of jobs submitted",
		}, []string{"job_type"}),
		JobsRunDurationMilliseconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "jobs",
			Name:      "run_duration_milliseconds",
			Help:      "Duration, in milliseconds, of jobs",
		}, []string{"job_type"}),
		JobsFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "serverless_registry_api",
			Subsystem: "jobs",
			Name:      "failures_total",
			Help:      "Total number of jobs which fail",
		}, []string{"job_type", "failure_type"}),
	}

	prometheus.MustRegister(metrics.APIRequestsTotal)
	prometheus.MustRegister(metrics.APIRequestsDurationMilliseconds)
	prometheus.MustRegister(metrics.APIHandlersPanicsTotal)
	prometheus.MustRegister(metrics.APIErrorResponsesTotal)
	prometheus.MustRegister(metrics.JobsSubmittedTotal)
	prometheus.MustRegister(metrics.JobsRunDurationMilliseconds)
	prometheus.MustRegister(metrics.JobsFailuresTotal)

	return metrics
}

// StartTimer starts a Timer for the provided Prometheus observer. Calling .Finish()
// on the returned timer will records the time elapsed in milliseconds.
func (m Metrics) StartTimer(observer prometheus.Observer) Timer {
	return Timer{
		startTime: time.Now(),
		observer:  observer,
	}
}
