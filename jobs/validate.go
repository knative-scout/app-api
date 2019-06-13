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
	checkRun, _, err := j.GH.Checks.CreateCheckRun(j.Ctx, j.Cfg.GhRegistryRepoOwner,
		j.Cfg.GhRegistryRepoName, github.CreateCheckRunOptions{
			Name: "KScout Validation",
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
		RepoOwner: j.Cfg.GhRegistryRepoOwner,
		RepoName: j.Cfg.GhRegistryRepoName,
		RepoRef: *pr.Head.Ref,
	}

        parseErrs := map[string]*parsing.ParseError{}

	for _, appID := range appIDs {
		_, err := repoParser.GetApp(appID)
		if err != nil {
			parseErrs[appID] = err
		}
	}

	// {{{1 Comment with validation result
	// {{{2 Build app status overview table
	commentBody := "I've taken a look at your pull request, here is the "+
		"current status of the applications you modified:  \n"+
		"\n  "

	statusTable := ""+
		"| App ID | Status | Comment |  \n"+
		"| ------ | ------ | ------- |  \n"

	for _, appID := range appIDs {
		status := ""
		comment := ""

		if err, ok := parseErrs[appID]; ok {
			if err.InternalError != nil {
				status = "Internal error"
				comment = fmt.Sprintf("%s please triage", j.Cfg.GhDevTeamName)
			} else {
				status = "Formatting error"
				comment = "See error bellow"
			}
		} else {
			status = "Good"
			comment = ":+1:"
		}

		statusTable += fmt.Sprintf("| %s | %s | %s |", appID, status, comment)
	}

	commentBody += statusTable

	// {{{2 Place any detailed error messages
	if len(parseErrs) > 0 {
		commentBody += "  \n"+
			"# Errors  \n"+
			"I found some errors with the changes made in this pull request:  \n"
	}

	errsDetails := ""
	for appID, err := range parseErrs {
		errsDetails += fmt.Sprintf("## App ID %s\n", appID)

		if err.InternalError != nil {
			errsDetails += "> Sometime went wrong on our servers when parsing this "+
				"serverless application. The development team has been notified "+
				"and will triage this issue as soon as they can.  \n"
		}

		errsDetails += fmt.Sprintf("### What failed?\n%s  \n", err.What)
		errsDetails += fmt.Sprintf("### Why did it fail?\n```\n%s\n```  \n", err.Why)
		errsDetails += fmt.Sprintf("### How to fix it\n%s  \n", err.FixInstructions)
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

	for _, err := range parseErrs {
		if err.InternalError != nil {
			title = "Internal Error"
			conclusion = "cancelled"
			break
		}
	}

	checkRunStatus = "completed"
	
	_, _, err = j.GH.Checks.UpdateCheckRun(j.Ctx, j.Cfg.GhRegistryRepoOwner,
		j.Cfg.GhRegistryRepoName, *checkRun.ID, github.UpdateCheckRunOptions{
			Name: "validate",
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
