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
	"github.com/google/go-github/github"
)

// WebhookHandler handles registry repository pull request webhook requests
type WebhookHandler struct {
	BaseHandler
}

// ServeHTTP implements net.Handler
func (h WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// {{{1 Verify request came from GitHub
	// {{{2 Get header value
	hubSigHeader, ok := r.Header["X-Hub-Signature"]
	if !ok || len(hubSigHeader) != 1 {
		h.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "X-Hub-Signature header must have a value",
		})
		return
	}

	expectedSig := hubSigHeader[0]

	// {{{2 Create HMAC of request
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(fmt.Errorf("failed to read request body: %s", err.Error()))
	}

	bodyHMAC := hmac.New(sha1.New, []byte(h.Cfg.GhWebhookSecret))
	bodyHMAC.Write(bodyBytes)

	actualSig := fmt.Sprintf("sha1=%s", hex.EncodeToString(bodyHMAC.Sum(nil)))

	// {{{2 Compare
	if !hmac.Equal([]byte(expectedSig), []byte(actualSig)) {
		h.RespondJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "could not verify request",
		})
		return
	}
		
	// {{{1 Check if we can handle this type of event
	eventTypeHeader, ok := r.Header["X-Github-Event"]
	if !ok || len(eventTypeHeader) != 1 {
		h.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "X-Github-Event header must have a value",
		})
		return
	}
	
	eventType := eventTypeHeader[0]

	switch eventType {
	case "ping":
		h.RespondJSON(w, http.StatusOK, map[string]bool{
			"pong": true,
		})
		return
	case "pull_request":
		break
	default:
		h.RespondJSON(w, http.StatusNotAcceptable, map[string]string{
			"error": fmt.Sprintf("cannot handle event type: %s", eventType),
		})
	}

	// {{{1 Parse body
	var req github.PullRequestEvent

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		panic(fmt.Errorf("failed to parse request body as JSON: %s", err.Error()))
	}

	// {{{1 Check if we can handler events from this repository
	if *req.Repo.Owner.Login != h.Cfg.GhRegistryRepoOwner ||
		*req.Repo.Name != h.Cfg.GhRegistryRepoName {
		h.RespondJSON(w, http.StatusNotAcceptable, map[string]string{
			"error": "endpoint does not handle requests from this repository",
		})
		return
	}

	// {{{1 Check if PR is merged as a result
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		panic(fmt.Errorf("failed to parse request body as JSON: %s", err.Error()))
	}
	
	h.Logger.Debugf("webhook request: %#v", req)

	if !*req.PullRequest.Merged {
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
