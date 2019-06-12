package validation

import (
	"fmt"
	"context"
	"github.com/google/go-github/github"

	"github.com/kscout/serverless-registry-api/parsing"
)

// FormatValidator validates apps in the PR are in the correct format
type FormatValidator struct {
	// Ctx is the server context
	Ctx context.Context

	// Gh is a GitHub API client
	GH *github.Client

	// RepoOwner is the owner of the PR's GitHub repository
	RepoOwner string

	// RepoName is the name of the PR's GitHub repository
	RepoName string

	// RepoRef is the Git refernece at which to parse data
	RepoRef string

	// PRNumber is the pull request's unique user facing number
	PRNumber int
}

// Name implements Validator.Name() 
func (v FormatValidator) Name() string {
	return "Format validator"
}

// Summary implements Validator.Summary()
func (v FormatValidator) Summary() string {
	return "Ensures applications in PR are correctly formatted"
}

// Validate format is valid. Will return an array of parsing.ParseErrors if any
// applications have incorrect formatting. Will return an error if the validation
// process fails to complete successfully
func (v FormatValidator) Validate() ([]*parsing.ParseError, error) {
	// {{{1 Get applications which were modified in PR
	prParser := parsing.PRParser{
		Ctx: v.Ctx,
		GH: v.GH,
		RepoOwner: v.RepoOwner,
		RepoName: v.RepoName,
		RepoRef: v.RepoRef,
		PRNumber: v.PRNumber,
	}
	appIDs, err := prParser.GetModifiedAppIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to get IDs of app modified in PR: %s",
			err.Error())
	}

	// {{{1 Load each application
	repoParser := parsing.RepoParser{
		Ctx: v.Ctx,
		GH: v.GH,
		RepoOwner: v.RepoOwner,
		RepoName: v.RepoName,
		RepoRef: v.RepoRef,
	}

        parseErrs := []*parsing.ParseError{}

	for _, appID := range appIDs {
		_, err := repoParser.GetApp(appID)
		if parseErr, ok := err.(parsing.ParseError); ok {
			parseErrs = append(parseErrs, parseErr)
		} else {
			return nil, fmt.Errorf("failed to parse app with id %s: %s",
				appID, err.Error())
		}
	}

	return parseErrs, nil
}
