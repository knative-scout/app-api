package handlers

import (
	"fmt"
	"net/http"
	"crypto/hmac"
	"crypto/sha1"
	"io/ioutil"
	"encoding/hex"
	"encoding/json"

	"github.com/kscout/serverless-registry-api/jobs"

	"github.com/google/go-github/v25/github"
)

// WebhookHandler handles registry repository pull request webhook requests
type WebhookHandler struct {
	BaseHandler

	// PullRequestEvaluator evaluates new pull requests and provides feedback to the user
	PullRequestEvaluator jobs.PullRequestEvaluator
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

	// {{{1 Determine if webhook should do anything
	// {{{2 Check if we can handle this type of event
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
	case "push":
		break
	default:
		h.RespondJSON(w, http.StatusNotAcceptable, map[string]string{
			"error": fmt.Sprintf("cannot handle event type: %s", eventType),
		})
	}

	// {{{2 Parse request body to determine what triggered event
	var prEvalSubmissions []jobs.PREvalSubmission
	var repoOwner string
	var repoName string
	var validEvent bool

	switch eventType {
	case "pull_request":
		// {{{3 Parse
		var req github.PullRequestEvent

		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			panic(fmt.Errorf("failed to parse pull request event body as JSON: %s",
				err.Error()))
		}

		repoOwner = *req.PullRequest.Base.Repo.Owner.Login
		repoName = *req.PullRequest.Base.Repo.Name

		// {{{3 Exit early if the PR event action does not indicate code changes
		// isMergeEvent indicates if the merge button was just clicked on a PR
		// button on the PR.
		isMergeEvent := *req.Action == "closed" && *req.PullRequest.Merged
		validEvent = *req.Action == "opened" || isMergeEvent

		// {{{3 Make PREvalSubmission
		prEvalSubmissions = append(prEvalSubmissions, jobs.PREvalSubmission{
			PR: *req.PullRequest,
			OnlyUpdateDB: isMergeEvent,
		})
	case "push":
		// {{{3 Parse
		var req github.PushEvent

		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			panic(fmt.Errorf("failed to parse push event body as JSON: %s",
				err.Error()))
		}

		repoOwner = *req.Repo.Owner.Login
		repoName = *req.Repo.Name

		// {{{3 Find PRs which include commits that were pushed
		// relevantPRs is a set which holds the pull requests which have code changes
		// keys are pull request numbers, values are pull requests
		relevantPRs := map[int]github.PullRequest{}

		for _, commit := range req.Commits {
			// {{{4 Get PRs to which commit belongs
			commitPRs, _, err := h.Gh.PullRequests.ListPullRequestsWithCommit(h.Ctx,
				h.Cfg.GhRegistryRepoOwner, h.Cfg.GhRegistryRepoName,
				*commit.ID, nil)
			if err != nil {
				panic(fmt.Errorf("failed to list PRs which include commit "+
					"with sha \"%s\": %s", *commit.ID, err.Error()))
			}

			// {{{4 Catalog relevant commits
			for _, pr := range commitPRs {
				// Ignore PRs where this commit is the merge commit
				if pr.MergeCommitSHA != nil &&
					*pr.MergeCommitSHA == *commit.ID {
					continue
				}
				
				relevantPRs[*pr.Number] = *pr
			}
		}

		// {{{3 Create PREvalSubmissions
		for _, pr := range relevantPRs {
			prEvalSubmissions = append(prEvalSubmissions, jobs.PREvalSubmission{
				PR: pr,
				OnlyUpdateDB: false,
			})
		}

		// {{{3 Exit early if none of the pushed commits reference a pull request
		validEvent = len(relevantPRs) > 0
	}

	// {{{2 Exit early if the event we received is not one we want to handle
	if !validEvent {
		h.RespondJSON(w, http.StatusOK, map[string]bool{
			"ok": true,
		})
		return
	}

	// {{{2 Check if we can handler events from this repository
	if repoOwner != h.Cfg.GhRegistryRepoOwner || repoName != h.Cfg.GhRegistryRepoName {
		h.RespondJSON(w, http.StatusNotAcceptable, map[string]string{
			"error": "endpoint does not handle requests from this repository",
		})
		return
	}

	// {{{1 Submit PR evaluation requests
	for _, evalSubmission := range prEvalSubmissions {
		h.PullRequestEvaluator.Submit(evalSubmission)
	}

	// {{{1 Return success response to GitHub
	h.RespondJSON(w, http.StatusOK, map[string]bool{
		"ok": true,
	})
}
