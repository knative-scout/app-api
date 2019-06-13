package handlers

import (
	"fmt"
	"net/http"
	"encoding/json"
	"crypto/hmac"
	"crypto/sha1"
	"io/ioutil"
	"encoding/hex"

	"github.com/kscout/serverless-registry-api/jobs"
	"github.com/google/go-github/v26/github"
)

// WebhookHandler handles GitHub App pull requests
type WebhookHandler struct {
	BaseHandler

	// JobRunner is used to run jobs
	JobRunner *jobs.JobRunner
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

	// {{{1 Spawn action depending on event type
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
		// {{{2 Parse so we can tell if the event is for a PR just getting merged
		var event github.PullRequestEvent

		if err := json.Unmarshal(bodyBytes, &event); err != nil {
			panic(fmt.Errorf("failed to parse pull request event body as JSON: %s",
				err.Error()))
		}

		h.Logger.Debugf("received pull request event: %#v", event)

		// {{{2 Start job if PR was just opened or just merged
		if *event.Action == "opened" {
			h.Logger.Debugf("started validate job for PR #%d",
				*event.PullRequest.Number)
			
			// {{{3 Marshal PR back to bytes
			prBytes, err := json.Marshal(*event.PullRequest)
			if err != nil {
				panic(fmt.Errorf("failed to marshal PR into JSON: %s",
					err.Error()))
			}
			h.JobRunner.Submit(jobs.JobTypeValidate, prBytes)
		} else if *event.Action == "closed" && *event.PullRequest.Merged { 
			h.Logger.Debugf("PR #%d was merged, submited update apps job",
				*event.PullRequest.Number)
			
			h.JobRunner.Submit(jobs.JobTypeUpdateApps, nil)
		}
	case "check_suite":
		// {{{2 Parse as CheckSuiteEvent so we can extract pull requests
		var event github.CheckSuiteEvent

		if err := json.Unmarshal(bodyBytes, &event); err != nil {
			panic(fmt.Errorf("failed to parse check run event body as JSON: %s",
				err.Error()))
		}

		h.Logger.Debugf("received check suite event: %#v", event)

		// {{{2 Start job for each pull request
		checkSuite := *event.CheckSuite
		prs := []*github.PullRequest{}

		for _, pr := range checkSuite.PullRequests {
			prs = append(prs, pr)
		}

		// {{{3 If no PRs in webhook then use head SHA
		if len(prs) == 0 {
			searchedPRs, _, err := h.Gh.PullRequests.ListPullRequestsWithCommit(h.Ctx,
				h.Cfg.GhRegistryRepoOwner, h.Cfg.GhRegistryRepoName,
				*checkSuite.HeadSHA, &github.PullRequestListOptions{
					State: "open",
				})
			if err != nil {
				panic(fmt.Errorf("failed to get PRs for check suite head SHA: %s",
					err.Error()))
			}

			for _, pr := range searchedPRs {
				prs = append(prs, pr)
			}
			
		}

		// {{{3 Submit validate job for each PR
		for _, pr := range prs {
			h.Logger.Debugf("submited validate job for PR #%d", *pr.Number)
			
			// {{{3 Marshal PR back to bytes
			prBytes, err := json.Marshal(*pr)
			if err != nil {
				panic(fmt.Errorf("failed to marshal PR into JSON: %s",
					err.Error()))
			}
			h.JobRunner.Submit(jobs.JobTypeValidate, prBytes)
		}
	default:
		h.RespondJSON(w, http.StatusNotAcceptable, map[string]string{
			"error": fmt.Sprintf("cannot handle event type: %s", eventType),
		})
		return
	}

	h.RespondJSON(w, http.StatusOK, map[string]bool{
		"ok": true,
	})
}
