package handlers

import (
	"fmt"
	"net/http"
	"crypto/hmac"
	"crypto/sha1"
	"io/ioutil"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/kscout/serverless-registry-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/google/go-github/v25/github"
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

	// {{{1 Get apps edited in PR
	// {{{2 Get files in PR
	prFiles, _, err := h.Gh.PullRequests.ListFiles(h.Ctx, h.Cfg.GhRegistryRepoOwner,
		h.Cfg.GhRegistryRepoName, *req.Number, &github.ListOptions{
			Page: 1,
			PerPage: 300, // 300 is the max number ever returned by this endpoint
		})
	if err != nil {
		panic(fmt.Errorf("failed to list files in PR: %s", err.Error()))
	}

	// {{{2 Parse file paths
	// modifiedApps is a map set which holds the names of modified apps as keys
	modifiedApps := map[string]bool{}
	
	for _, prFile := range prFiles {
		// {{{3 Get old and new filepath of commit file
		// This accounts for a file being moved from one app directory to another
		dirs := []string{}
		
		curDir, _ := filepath.Split(*prFile.Filename)
		dirs = append(dirs, curDir)

		if prFile.PreviousFilename != nil {
			oldDir, _ := filepath.Split(*prFile.PreviousFilename)
			dirs = append(dirs, oldDir)
		}

		// {{{3 Parse for app directories
		for _, dir := range dirs {
			// If file in base dir
			if len(dir) == 0 {
				continue
			}

			parts := strings.Split(dir, "/")
			
			modifiedApps[parts[0]] = true
		}
	}

	// {{{2 Determine if any modified apps in PR are those apps being deleted
	// At this point modifiedApps would have a deleted app's ID in it b/c the
	// commit would show this deleted app's files as being modified.
	//
	// We can tell by listing the folders present in the PR's head, and if any folders
	// are not present in the PR's head but are in the modifiedApps set then these apps
	// were deleted.
	appLoader := models.AppLoader{
		Ctx: h.Ctx,
		Gh: h.Gh,
		Cfg: h.Cfg,
	}
	
	presentAppIDs, err := appLoader.GetAppIDsFromRegistry(*req.PullRequest.Head.Ref)
	if err != nil {
		panic(fmt.Errorf("failed to get IDs of all apps present in PR head: %s",
			err.Error()))
	}

	presentAppIDsSet := map[string]bool{}
	for _, presentAppID := range presentAppIDs {
		presentAppIDsSet[presentAppID] = true
	}

	for modifiedAppID, _ := range modifiedApps {
		// If app in modifiedApps but not in PR head
		if _, ok := presentAppIDsSet[modifiedAppID]; !ok {
			delete(modifiedApps, modifiedAppID)
		}
	}

	// {{{1 End early if PR does not include any modified apps
	if len(modifiedApps) == 0 {
		h.RespondJSON(w, http.StatusOK, map[string]bool{
			"ok": true,
		})
		return
	}

	// {{{1 Parse / load modified apps
	apps := map[string]models.App{}
	
	appLoadInternalErrs := map[string]error{}
	appLoadFormatErrs := map[string]models.AppSrcFormatError{}
	
	for appID, _ := range modifiedApps {
		app, err := appLoader.LoadAppFromRegistry(*req.PullRequest.Head.Ref, appID)
		if err != nil {
			// Check if a formatting error or an internal error
			if fmtErr, ok := err.(models.AppSrcFormatError); ok {
				appLoadFormatErrs[appID] = fmtErr
			} else {
				appLoadInternalErrs[appID] = err
			}
		} else {
			apps[appID] = *app
		}
	}

	// {{{1 Save submission entry in db
	// {{{2 Build submission entry
	submission := models.Submission{
		PRNumber: *req.Number,
		Apps: map[string]*models.SubmissionApp{},
	}

	// {{{3 Add correctly formatted apps
	for appID, app := range apps {
		submission.Apps[appID] = &models.SubmissionApp{
			App: &app,
			VerificationStatus: models.AppVerificationStatus{
				FormatCorrect: true,
			},
		}
	}

	// {{{3 Add apps with format errors
	for appID, _ := range appLoadFormatErrs {
		submission.Apps[appID] = &models.SubmissionApp{
			App: nil,
			VerificationStatus: models.AppVerificationStatus{
				FormatCorrect: false,
			},
		}
	}

	// {{{3 Add apps with internal errors
	for appID, err := range appLoadInternalErrs {
		submission.Apps[appID] = nil
		h.Logger.Errorf("internal error occurred when parsing / loading app with "+
			"ID \"%s\": %s", appID, err.Error())
	}

	// {{{2 Save in DB
	trueValue := true
	_, err = h.MDbSubmissions.UpdateOne(h.Ctx, bson.D{{"pr_number", submission.PRNumber}},
		bson.D{{"$set", submission}},
		&options.UpdateOptions{
			Upsert: &trueValue,
		})
	if err != nil {
		panic(fmt.Errorf("failed to save submission in db: %s", err.Error()))
	}

	// {{{1 Comment on PR
	// {{{2 Build PR comment body
	// {{{3 Generate table with app statuses
	commentBody := "I've taken a look at your pull request, here is the "+
		"current status of the applications you modified:  \n"+
		"\n  "+
		"| App ID | Status | Comment |  \n"+
		"| ------ | ------ | ------- |  \n"

	for appID, subApp := range submission.Apps {
		status := ""
		comment := ""

		if subApp == nil {
			status = "Internal error"
			comment = fmt.Sprintf("%s please triage", h.Cfg.GhDevTeamName)
		} else if subApp.VerificationStatus.FormatCorrect {
			status = "Good"
		} else {
			status = "Formating Error"
			comment = "See error below"
		}

		commentBody += fmt.Sprintf("| %s | %s | %s |", appID, status, comment)
	}

	// {{{3 Place any detailed error messages
	if len(appLoadFormatErrs) > 0 || len(appLoadInternalErrs) > 0 {
		commentBody += "  \n"+
			"# Errors  \n"+
			"I found some errors with the changes made in this pull request:  \n"
	}

	for appID, err := range appLoadFormatErrs {
		commentBody += fmt.Sprintf("## App ID: %s  \n", appID)
		commentBody += "The application's formatting was incorrect:  \n"
		commentBody += fmt.Sprintf("```\n%s\n```\n", err.PublicError())
	}

	for appID, _ := range appLoadInternalErrs {
		commentBody += fmt.Sprintf("## App ID: %s  \n", appID)
		commentBody += "An internal error occurred on our servers when we were "+
			"processing this serverless application.  \n"+
			"  \n"+
			"Do not worry the team has been notified will fix the issue "+
			"as soon as possible"
	}

	commentBody += "  \n---  \n"+
		"*I am a bot*"

	_, _, err = h.Gh.Issues.CreateComment(h.Ctx, h.Cfg.GhRegistryRepoOwner,
		h.Cfg.GhRegistryRepoName, *req.Number, &github.IssueComment{
			Body: &commentBody,
		})

	if err != nil {
		panic(fmt.Errorf("failed to create comment on PR: %s", err.Error()))
	}
	
	// {{{1 Done
	h.RespondJSON(w, http.StatusOK, map[string]bool{
		"ok": true,
	})
}
