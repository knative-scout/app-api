package jobs

import (
	"fmt"
	"context"
	"path/filepath"
	"strings"

	"github.com/kscout/serverless-registry-api/config"
	"github.com/kscout/serverless-registry-api/models"

	"github.com/Noah-Huppert/golog"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/google/go-github/v25/github"
)

// PREvalSubmission holds the information required by the PullRequestEvaluator to evaluate a
// GitHub pull request
type PREvalSubmission struct {
	// PR is the GitHub pull request to evaluate
	PR github.PullRequest

	// OnlyUpdateDB indicates if the changes made to the serverless applications in the PR
	// should be saved in the database.
	// This field will be set to true by submitters when a PR is merged into master.
	// If true the PullRequestEvaluator will not make a comment or set a status.
	OnlyUpdateDB bool
}

// PullRequestEvaluator evaluates a GitHub pull request in the registry repository for correctness
// For each pull request:
//
//    - The format of files in the PR are checked
//    - A comment is made by Scout Bot informing the user of their PR's status
//    - Set a status on the PR ref based on the results of the earlier checks
//
// Additionally if the PREvalSubmission.UpdateDB field is true the serverless applications
// modified in the PR will be saved in the database.
type PullRequestEvaluator struct {
	// submitChan is a channel to which new PREvalSubmissions will be pushed
	submitChan chan PREvalSubmission
	
	// Ctx is the application context
	Ctx context.Context

	// Logger logs information
	Logger golog.Logger

	// Cfg is the application configuration
	Cfg *config.Config

	// MDbApps is the MongoDB apps collection instance
	MDbApps *mongo.Collection

	// MDbSubmissions is the MongoDB submissions collection instance
	MDbSubmissions *mongo.Collection

	// Gh is the GitHub API client
	Gh *github.Client
}

// Init initializes a PullRequestEvaluator, no methods can be called before this
func (e *PullRequestEvaluator) Init() {
	e.submitChan = make(chan PREvalSubmission)
}

// Submit a PREvalSubmission to be evaluated
func (e PullRequestEvaluator) Submit(evalSubmission PREvalSubmission) {
	e.submitChan <- evalSubmission
}

// Run a goroutine which will call evaluate for each PREvalSubmission it receives on submitChan
// Cancel Ctx to stop this method. Doing so will make the method finish evaluating any pull
// requests which it is currently evaluating and exit.
func (e PullRequestEvaluator) Run() {
	e.Logger.Info("running")
	for {
		select {
		case <-e.Ctx.Done():
			return

		case evalSubmission := <-e.submitChan:
			e.Logger.Debugf("received evaluation request: %#v", evalSubmission)
			
			if err := e.evaluate(evalSubmission); err != nil {
				e.Logger.Errorf("failed to evaluate %#v: %s", evalSubmission,
					err.Error())
			}
		}
	}
}

func (e PullRequestEvaluator) evaluate(evalSubmission PREvalSubmission) error {
	// {{{1 Get apps edited in PR
	// {{{2 Get files in PR
	prFiles, _, err := e.Gh.PullRequests.ListFiles(e.Ctx, e.Cfg.GhRegistryRepoOwner,
		e.Cfg.GhRegistryRepoName, *evalSubmission.PR.Number, &github.ListOptions{
			Page: 1,
			PerPage: 300, // 300 is the max number ever returned by this endpoint
		})
	if err != nil {
		return fmt.Errorf("failed to list files in PR: %s", err.Error())
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
		Ctx: e.Ctx,
		Gh: e.Gh,
		Cfg: e.Cfg,
	}
	
	presentAppIDs, err := appLoader.GetAppIDsFromRegistry(*evalSubmission.PR.Head.Ref)
	if err != nil {
		return fmt.Errorf("failed to get IDs of all apps present in PR head: %s",
			err.Error())
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
		return nil
	}

	// {{{1 Parse / load modified apps
	apps := map[string]models.App{}
	
	appLoadInternalErrs := map[string]error{}
	appLoadFormatErrs := map[string]models.AppSrcFormatError{}
	
	for appID, _ := range modifiedApps {
		app, err := appLoader.LoadAppFromRegistry(*evalSubmission.PR.Head.Ref, appID)
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
		PRNumber: *evalSubmission.PR.Number,
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
		e.Logger.Errorf("internal error occurred when parsing / loading app with "+
			"ID \"%s\": %s", appID, err.Error())
	}

	// {{{2 Save in DB
	trueValue := true
	_, err = e.MDbSubmissions.UpdateOne(e.Ctx, bson.D{{"pr_number", submission.PRNumber}},
		bson.D{{"$set", submission}},
		&options.UpdateOptions{
			Upsert: &trueValue,
		})
	if err != nil {
		return fmt.Errorf("failed to save submission in db: %s", err.Error())
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
			comment = fmt.Sprintf("%s please triage", e.Cfg.GhDevTeamName)
		} else if subApp.VerificationStatus.FormatCorrect {
			status = "Good"
			comment = ":+1:"
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

	_, _, err = e.Gh.Issues.CreateComment(e.Ctx, e.Cfg.GhRegistryRepoOwner,
		e.Cfg.GhRegistryRepoName, *evalSubmission.PR.Number, &github.IssueComment{
			Body: &commentBody,
		})

	if err != nil {
		return fmt.Errorf("failed to create comment on PR: %s", err.Error())
	}

	return nil
}
