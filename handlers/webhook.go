package handlers

import (
	"fmt"
	"net/http"
	"crypto/hmac"
	"crypto/sha1"
	"io/ioutil"
	"encoding/hex"
	"encoding/json"

	"github.com/knative-scout/app-api/models"

	"go.mongodb.org/mongo-driver/bson"
)

// WebhookHandler handles registry repository pull request webhook requests
type WebhookHandler struct {
	BaseHandler
}

// webhookRequest is a request made by GitHub to the webhook endpoint
type webhookRequest struct {
	// Action is the pull request action which the request describes
	Action string `json:"action"`

	// Number is the pull request number
	Number int `json:"number"`

	// PullRequest itself
	PullRequest pullRequest `json:"pull_request"`
}

// pullRequest holds the relevant fields of a GitHub API pull request object
type pullRequest struct {
	// Merged indicates if the pull request has been merged yet
	Merged bool `json:"merged"`
}

// ServeHTTP implements net.Handler
func (h WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// {{{1 Verify request came from GitHub
	var bodyBytes []byte = []byte{}
	
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
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(fmt.Errorf("failed to ready body bytes when verifying request: %s",
				err.Error()))
		}
		bodyBytes = b
		
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

	// {{{1 Check if PR is merged as a result
	var req webhookRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		panic(fmt.Errorf("failed to parse request body as JSON: %s", err.Error()))
	}
	
	h.Logger.Debugf("webhook request: %#v", req)

	if !req.PullRequest.Merged {
		h.Logger.Debug("not merged yet")

		h.RespondJSON(w, http.StatusOK, map[string]bool{
			"ok": true,
		})
		return
	}

	// {{{1 Rebuild database
	// {{{2 Load all apps
	appLoader := models.AppLoader{
		Ctx: h.Ctx,
		Gh: h.Gh,
		Cfg: h.Cfg,
	}
	
	apps, err := appLoader.LoadAllAppsFromRegistry(*req.PullRequest.Head.Ref)
	if err != nil {
		panic(fmt.Errorf("failed to load apps: %s", err.Error()))
	}

	// {{{2 Delete old apps
	_, err = h.MDbApps.DeleteMany(h.Ctx, bson.D{}, nil)
	if err != nil {
		panic(fmt.Errorf("failed to delete old apps from db: %s", err.Error()))
	}

	// {{{2 Insert
	insertDocs := []interface{}{}

	for _, app := range apps {
		insertDocs = append(insertDocs, *app)
	}
	
	_, err = h.MDbApps.InsertMany(h.Ctx, insertDocs, nil)
	if err != nil {
		panic(fmt.Errorf("failed to insert apps into db: %s", err.Error()))
	}

	h.RespondJSON(w, http.StatusOK, map[string]bool{
		"ok": true,
	})
}
