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
			h.Logger.Error("panicked while handling request:", r)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "{\"error\": \"internal server error\"}")
		}
	}()

	h.Handler.ServeHTTP(w, r)
}
	
