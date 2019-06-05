package handlers

import (
	"fmt"
	"net/http"
	"crypto/hmac"
	"crypto/sha1"
	"io/ioutil"
	"encoding/hex"

	"github.com/knative-scout/app-api/models"
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
	} else if len(hashSig) != 1 {
		h.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "X-Hub-Signature header must have 1 value",
		})
		return
	} else {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(fmt.Errorf("failed to ready body bytes when verifying request: %s",
				err.Error()))
		}
		
		bodyHMAC := hmac.New(sha1.New, []byte(h.Cfg.GhWebhookSecret))
		bodyHMAC.Write(bodyBytes)

		matchHashSig := fmt.Sprintf("sha1=%s", hex.EncodeToString(bodyHMAC.Sum(nil)))

		if !hmac.Equal([]byte(hashSig[0]), []byte(matchHashSig)) {
			h.RespondJSON(w, http.StatusBadRequest, map[string]string{
				"error": "failed to verify request signature",
			})
			return
		}
	}
	
	// {{{1 Check if we can handle the event
	if eventType, ok := r.Header["X-Github-Event"]; !ok {
		h.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "X-Github-Event header not present",
		})
		return
	} else if len(eventType) != 1 {
		h.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "X-Github-Event header must have 1 value",
		})
		return;
	} else if eventType[0] == "ping" {
		h.RespondJSON(w, http.StatusOK, map[string]bool{
			"ok": true,
		})
		return
	} else if eventType[0] != "pull_request" {
		h.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "can only handle \"pull_request\" events",
		})
		return
	}

	// {{{1 Rebuild database
	// {{{2 Load all apps
	apps, err := models.LoadAllAppsFromRegistry(h.Ctx, h.Gh, h.Cfg)
	if err != nil {
		panic(fmt.Errorf("failed to load apps: %s", err.Error()))
	}

	// {{{2 Upsert
	insertDocs := []interface{}{}

	for _, app := range apps {
		insertDocs = append(insertDocs, *app)
	}
	
	_, err = mDbApps.InsertMany(ctx, insertDocs, nil)
	if err != nil {
		loadLogger.Fatalf("failed to insert apps into db: %s", err.Error())
	}

	h.RespondJSON(w, http.StatusOK, map[string]bool{
		"ok": true,
	})
}
