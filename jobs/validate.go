package jobs

import (
	"fmt"
	"time"
	"context"
	"encoding/json"

	"github.com/kscout/serverless-registry-api/parsing"
	"github.com/kscout/serverless-registry-api/config"
	
	"github.com/google/go-github/v26/github"
	"github.com/Noah-Huppert/golog"
)

// ValidateJob updates the apps collection based on the current master branch state
// Expects the data passed to Do() to be a github.PullRequest in JSON form. This
// pull request will be validated.
type ValidateJob struct {
	// Ctx
	Ctx context.Context

	// Logger
	Logger golog.Logger

	// Cfg is the server configuration
	Cfg *config.Config
	
	// GH is a GitHub API client
	GH *github.Client
}

// Do implments Job
func (j ValidateJob) Do(data []byte) error {
	// {{{1 Parse PullRequestEvent
	var pr github.PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return fmt.Errorf("failed to unmarshal data as github.PullRequest: %s",
			err.Error())
	}

	// {{{1 Create check run
	checkRunStatus :="in_progress"
	checkRunName := "KScout Format Validation"
	checkRun, _, err := j.GH.Checks.CreateCheckRun(j.Ctx, j.Cfg.GhRegistryRepoOwner,
		j.Cfg.GhRegistryRepoName, github.CreateCheckRunOptions{
			Name: checkRunName,
			HeadBranch: *pr.Head.Ref,
			HeadSHA: *pr.Head.SHA,
			StartedAt: &github.Timestamp{ time.Now() },
			Status: &checkRunStatus,
		})
	if err != nil {
		return fmt.Errorf("failed to create initial check run status: %s", err.Error())
	}
	
	// {{{1 Get applications which were modified in PR
	prParser := parsing.PRParser{
		Ctx: j.Ctx,
		GH: j.GH,
		RepoOwner: j.Cfg.GhRegistryRepoOwner,
		RepoName: j.Cfg.GhRegistryRepoName,
		RepoRef: *pr.Head.Ref,
		PRNumber: *pr.Number,
	}
	appIDs, err := prParser.GetModifiedAppIDs()
	if err != nil {
		return fmt.Errorf("failed to get IDs of app modified in PR: %s",
			err.Error())
	}

	if len(appIDs) == 0 {
		return nil
	}

	// {{{1 Load each application
	repoParser := parsing.RepoParser{
		Ctx: j.Ctx,
		GH: j.GH,
		GHDevTeamName: j.Cfg.GhDevTeamName,
		SiteURL: j.Cfg.SiteURL,
		RepoOwner: j.Cfg.GhRegistryRepoOwner,
		RepoName: j.Cfg.GhRegistryRepoName,
		RepoRef: *pr.Head.Ref,
	}

        parseErrs := map[string][]parsing.ParseError{}

	for _, appID := range appIDs {
		_, errs := repoParser.GetApp(appID)
		if len(errs) > 0 {
			parseErrs[appID] = errs
		}
	}

	// {{{1 Comment with validation result
	// {{{2 Build app status overview table
	commentBody := "I've taken a look at your pull request, here is the "+
		"current status of the applications you modified:  \n"+
		"\n  "

	statusTable := ""+
		"| App ID | Status | Comment |\n"+
		"| ------ | ------ | ------- |\n"

	internalErr := false
	
	for _, appID := range appIDs {
		status := ""
		comment := ""

		if errs, ok := parseErrs[appID]; ok {
			for _, err := range errs {
				if err.InternalError != nil {
					internalErr = true
					break
				}
			}
			
			if internalErr {
				status = "Internal error"
				comment = fmt.Sprintf("%s please triage", j.Cfg.GhDevTeamName)
			} else {
				status = "Formatting error"
				comment = "See details bellow"
			}
		} else {
			status = "Good"
			comment = ":+1:"
		}

		statusTable += fmt.Sprintf("| %s | %s | %s |\n", appID, status, comment)
	}

	commentBody += statusTable

	// {{{2 Place any detailed error messages
	if len(parseErrs) > 0 {
		commentBody += "  \n"+
			"# Errors  \n"+
			"I found some errors with the changes made in this pull request:  \n"
	}

	errsDetails := ""
	for appID, errs := range parseErrs {
		errsDetails += fmt.Sprintf("## App ID %s\n", appID)

		if internalErr {
			errsDetails += "> Sometime went wrong on our servers when "+
				"parsing this serverless application. The development "+
				"team has been notified and will triage this issue "+
				"as soon as they can.  \n"
		}

		for i, err := range errs {
			errsDetails += fmt.Sprintf("- Error %d\n", i+1)

			errsDetails += fmt.Sprintf("  - **What failed?**: %s\n", err.What)
			errsDetails += fmt.Sprintf("  - **Why did it fail?**: %s\n", err.Why)
			errsDetails += "  - **How to fix**: "
			
			if err.InternalError != nil {
				errsDetails += "This issue was caused by an error with the "+
					"KScout servers. The development team will fix this "+
					"error for you.\n"
				j.Logger.Errorf("internal error when validating app \"%s\": %s",
					appID, err.InternalError.Error())
			} else {
				errsDetails += fmt.Sprintf("%s\n", err.FixInstructions)
			}
		}
	}

	commentBody += errsDetails

	commentBody += "  \n---  \n"+
		"*I am a bot*"

	// {{{2 Make comment
	_, _, err = j.GH.Issues.CreateComment(j.Ctx, j.Cfg.GhRegistryRepoOwner,
		j.Cfg.GhRegistryRepoName, *pr.Number, &github.IssueComment{
			Body: &commentBody,
		})

	if err != nil {
		return fmt.Errorf("failed to create comment on PR: %s", err.Error())
	}

	// {{{1 Update check run status
	title := "Passed"
	conclusion := "success"
	
	if len(parseErrs) > 0 {
		title = "Failed"
		conclusion = "failure"
	}

	if internalErr {
		title = "Internal Error"
		conclusion = "cancelled"
	}

	checkRunStatus = "completed"
	
	_, _, err = j.GH.Checks.UpdateCheckRun(j.Ctx, j.Cfg.GhRegistryRepoOwner,
		j.Cfg.GhRegistryRepoName, *checkRun.ID, github.UpdateCheckRunOptions{
			Name: checkRunName,
			CompletedAt: &github.Timestamp{ time.Now() },
			Status: &checkRunStatus,
			Conclusion: &conclusion,
			Output: &github.CheckRunOutput{
				Title: &title,
				Summary: &statusTable,
				Text: &errsDetails,
			},
		})
	if err != nil {
		return fmt.Errorf("failed to update check run: %s", err.Error())
	}

	return nil
}
