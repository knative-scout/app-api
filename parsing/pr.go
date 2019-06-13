package parsing

import (
	"fmt"
	"context"
	"path/filepath"
	"strings"
	
	"github.com/google/go-github/v26/github"
)

// PRParser parses a pull request
type PRParser struct {
	// Ctx is the server context
	Ctx context.Context

	// Gh is a GitHub API client
	GH *github.Client

	// RepoOwner is the owner of the PR's GitHub repository
	RepoOwner string

	// RepoName is the name of the PR's GitHub repository
	RepoName string

	// RepoRef is the Git reference to parse data at
	RepoRef string

	// PRNumber is the pull request's unique user facing number
	PRNumber int
}

// GetModifiedAppIDs returns the IDs of the serverless applications modified in a pull request.
func (p PRParser) GetModifiedAppIDs() ([]string, error) {
	// {{{1 Get files in PR
	prFiles, _, err := p.GH.PullRequests.ListFiles(p.Ctx, p.RepoOwner, p.RepoName, 
		p.PRNumber, &github.ListOptions{
			Page: 1,
			PerPage: 300, // 300 is the max number ever returned by this endpoint
		})
	if err != nil {
		return nil, fmt.Errorf("error listing PR files: %s", err.Error())
	}
	
	// {{{1 Parse file paths
	// modifiedApps is a map set which holds the names of modified apps as keys
	modifiedApps := map[string]interface{}{}
	
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

	// {{{1 Determine if any modified apps in PR are those apps being deleted
	// At this point modifiedApps would have a deleted app's ID in it b/c the
	// commit would show this deleted app's files as being modified.
	//
	// We can tell by listing the folders present in the PR's head, and if any folders
	// are not present in the PR's head but are in the modifiedApps set then these apps
	// were deleted.
	repoParser := RepoParser{
		Ctx: p.Ctx,
		GH: p.GH,
		RepoOwner: p.RepoOwner,
		RepoName: p.RepoName,
		RepoRef: p.RepoRef,
	}
	
	presentAppIDs, err := repoParser.GetAppIDs()
	if err != nil {
		return nil, fmt.Errorf("error getting IDs of apps in PR head: %s", err.Error())
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

	// {{{1 Make into array
	appIDs := []string{}

	for appID, _ := range modifiedApps {
		appIDs = append(appIDs, appID)
	}

	return appIDs, nil
}
