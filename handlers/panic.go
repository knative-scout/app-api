package handlers

import (
	"net/http"
	"runtime/debug"
)

// PanicHandler runs another http.Handler and recovers from any panics which occur
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
		}
	}()

	h.Handler.ServeHTTP(w, r)
}
	
