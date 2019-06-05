package handlers

import (
	"net/http"
)

// WebhookHandler handles registry repository pull request webhook requests
type WebhookHandler struct {
	BaseHandler
}

// ServeHTTP implements net.Handler
func (h WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	
}
