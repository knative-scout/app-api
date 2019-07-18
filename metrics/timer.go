package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Timer measures the duration an operation and saves the result in a Prometheus observer.
// Time units are milliseconds.
type Timer struct {
	// startTime is the time the timer was started
	startTime time.Time

	// observer used to record duration in milliseconds
	observer prometheus.Observer
}

// Finish writes the duration between time.Now() and .startTime, in milliseconds, to
// the .observer.
func (t Timer) Finish() {
	duration := time.Since(t.startTime)
	ms := float64(duration / time.Millisecond)
	t.observer.Observe(ms)
}
