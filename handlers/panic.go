package handlers

import (
	"fmt"
	"net/http"
	"runtime/debug"
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
	defer func() {
		if r := recover(); r != nil {
			h.Logger.Error(string(debug.Stack()))
			h.Logger.Errorf("panicked while handling request: %#v", r)

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
	
