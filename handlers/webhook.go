package handlers

import (
	"fmt"
	"net/http"
	"crypto/hmac"
	"crypto/sha1"
	"hash"
	"io/ioutil"
)

// WebhookHandler handles registry repository pull request webhook requests
type WebhookHandler struct {
	BaseHandler
}

// ServeHTTP implements net.Handler
func (h WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// {{{1 Verify request came from GitHub
	if hashSig, ok := r.Header["X-Hub-Signature"]; !ok {
		h.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "X-Hub-Signature header not present",
		})
		return
	} else {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(fmt.Errorf("failed to ready body bytes when verifying request: %s",
				err.Error()))
		}
		
		bodyHMAC := hmac.New(sha1.New, []byte(h.cfg.GhWebhookSecret))
		bodyHMAC.Write(bodyBytes)

		matchHashSig := fmt.Sprintf("sha1=%s", bodyHMAC.Sum(nil))

		if !hmac.Equal([]byte(hashSig), []byte(matchHashSig)) {
			h.RespondJSON(w, http.StatusBadRequest, map[string]string{
				"error": "failed to verify request signature",
			})
			return
		}
	}
	
	// {{{1 Check if we can handle the event
	if eventType, ok := r.Header["X-GitHub-Event"]; !ok {
		h.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "X-GitHub-Event header not present",
		})
		return
	} else if eventType != "pull_request" {
		h.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "can only handle \"pull_request\" events",
		})
		return
	}

	h.RespondJSON(w, http.StatusOK, map[string]bool{
		"ok": true,
	})
}
