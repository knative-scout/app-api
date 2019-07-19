package metrics

import (
	"net/http"
)

// MetricsResponseWriter wraps an net/http.ResponseWriter and records metrics when certain interface methods are called
type MetricsResponseWriter struct {
	// ResponseWriter which will actually perform work
	ResponseWriter http.ResponseWriter

	// OnWriteHeader is a called any time ResponseWriter.WriteHeader is called
	OnWriteHeader OnWriteHeaderFunc
}

// OnWriteHeaderFunc is a function which will be called any time ResponseWriter.WriteHeader is called
type OnWriteHeaderFunc func(code int)

// Header calls ResponseWriter.Header
func (r MetricsResponseWriter) Header() http.Header {
	return r.ResponseWriter.Header()
}

// Write calls ResponseWriter.Write
func (r MetricsResponseWriter) Write(b []byte) (int, error) {
	return r.ResponseWriter.Write(b)
}

// WriteHeader increments WriteHeaderCounterVec if present and calls ResponseWriter.WriteHeader
func (r MetricsResponseWriter) WriteHeader(code int) {
	defer func() {
		r.OnWriteHeader(code)
	}()

	r.ResponseWriter.WriteHeader(code)
}
