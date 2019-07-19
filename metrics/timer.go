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
}

// Finish writes the duration between time.Now() and .startTime, in milliseconds, to
// the observer.
func (t Timer) Finish(observer prometheus.Observer) {
	duration := time.Since(t.startTime)
	ms := float64(duration / time.Millisecond)
	observer.Observe(ms)
}
