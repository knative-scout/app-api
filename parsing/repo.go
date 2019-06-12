package parsing

import (
	"strings"
	"fmt"
	"context"
	"encoding/json"
	"crypto/sha256"

	"github.com/kscout/serverless-registry-api/models"
	
	"github.com/google/go-github/v26/github"
	"gopkg.in/yaml.v2"
	"gopkg.in/go-playground/validator.v9"
)

// RepoParser reads GitHub repositories for serverless application information
type RepoParser struct {
	// Ctx is the server's context
	Ctx context.Context

	// GH is a GitHub API client
	GH *github.Client

	// GHDevTeamName is the name of the GitHub team to ping when an internal error occurs
	GHDevTeamName string

	// RepoOwner is the owner of the repository
	RepoOwner string

	// RepoName is the name of the repository
	RepoName string

	// RepoRef is the Git reference to parse data at
	RepoRef string
}

// GetAppIDs returns the IDs of all the serverless applications in a repository
func (p RepoParser) GetAppIDs() ([]string, error) {
	_, contents, _, err := p.GH.Repositories.GetContents(p.Ctx, p.RepoOwner, p.RepoName,
		"", &github.RepositoryContentGetOptions{
			Ref: p.RepoRef,
		})
	if err != nil {
		return nil, fmt.Errorf("error listing top level repository contents via "+
			"GitHub API: %s", err.Error())
	}

	ids := []string{}
	
	for _, content := range contents {
		if *content.Type == "file" {
			continue
		}

		ids = append(ids, *content.Name)
	}

	return ids, nil
}

// GetDownloadURLs returns the download URLs for files in a directory
func (p RepoParser) GetDownloadURLs(path string) ([]string, error) {
	// {{{1 Make API call
	_, contents, _, err := p.GH.Repositories.GetContents(p.Ctx, p.RepoOwner, p.RepoName,
		path, &github.RepositoryContentGetOptions{
			Ref: p.RepoRef,
		})
	if err != nil {
		return nil, fmt.Errorf("error listing directory contents with GitHub API: %s",
			err.Error())
	}

	// {{{1 Accumulate list of files
	urls := []string{}
	
	for _, content := range contents {
		if *content.Type == "dir" {
			continue
		}

		urls = append(urls, *content.HTMLURL)
	}

	return urls, nil
}

// GetFileContent retrieves the contents of a file
func (p RepoParser) GetFileContent(f string) (string, error) {
	content, _, _, err := p.GH.Repositories.GetContents(p.Ctx, p.RepoOwner,
		p.RepoName, f, &github.RepositoryContentGetOptions{
			Ref: p.RepoRef,
		})
	if err != nil {
		return "", fmt.Errorf("failed to get content from GitHub API: %s", err.Error())
	}

	txt, err := content.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode content: %s", err.Error())
	}

	return txt, nil
}

// GetApp marshals an app from the repository
func (p RepoParser) GetApp(id string) (*models.App, *ParseError) {
	// {{{1 Get contents of app directory
	_, dirContents, _, err := p.GH.Repositories.GetContents(p.Ctx, p.RepoOwner,
		p.RepoName, id, &github.RepositoryContentGetOptions{
			Ref: p.RepoRef,
		})
	if err != nil {
		return nil, &ParseError{
			What: fmt.Sprintf("app in %s directory", id),
			Why: "the GitHub API returned an error response",
			FixInstructions: fmt.Sprintf("%s will triage this issue", p.GHDevTeamName),
			InternalError: err,
		}
	}

	if len(dirContents) == 0 {
		return nil, &ParseError{
			What: fmt.Sprintf("app in %s directory", id),
			Why: "no content found",
			FixInstructions: "add required files",
			InternalError: nil,
		}
	}

	// {{{1 Parse contents into App
	app := models.App{}

	app.AppID = id
	app.VerificationStatus = models.VerificationStatusPending
	app.GitHubURL = fmt.Sprintf("https://github.com/%s/%s/tree/%s/%s",
		p.RepoOwner, p.RepoName, p.RepoRef, id)

	// found tracks if a file / directory has been found in the registry
	found := map[string]bool{
		"manifest.yaml": false,
		"README.md": false,
		"logo.png": false,
		"deployment": false,
	}
	
	for _, content := range dirContents {
		// {{{2 Check if file / directory is supposed to be there
		if _, ok := found[*content.Name]; !ok {
			return nil, &ParseError{
				What: fmt.Sprintf("%s %s for app in %s directory",
					*content.Name, *content.Type, id),
				Why: "should not exist",
				FixInstructions: fmt.Sprintf("delete this %s", *content.Type),
				InternalError: nil,
			}
		}

		found[*content.Name] = true

		// {{{2 Parse file / directory for app info
		switch *content.Type {
		case "file":
			switch *content.Name {
			case "manifest.yaml":
				// {{{2 Get manifest.yaml file content
				txt, err := p.GetFileContent(fmt.Sprintf("%s/%s", id, *content.Name))
				if err != nil {
					return nil, &ParseError{
						What: fmt.Sprintf("%s file for app in %s directory", *content.Name, id),
						Why: "failed to get contents from GitHub API, error response returned",
						FixInstructions: fmt.Sprintf("%s will triage this issue", p.GHDevTeamName),
						InternalError: err,
					}
				}
				
				// {{{2 Parse as YAML
				var manifest models.AppManifestFile
				err = yaml.UnmarshalStrict([]byte(txt), &manifest)
				if err != nil {
					return nil, &ParseError{
						What: fmt.Sprintf("%s file for app in %s directory", *content.Name, id),
						Why: fmt.Sprintf("failed to parse as YAML: %s", err.Error()),
						FixInstructions: "fix YAML syntax",
						InternalError: nil,
					}
				}

				// {{{2 Using custom validation for author and maintainer fields
				if !models.ContactStrExp.Match([]byte(manifest.Author)) {
					return nil, &ParseError{
						What: fmt.Sprintf("%s file for app in %s directory", *content.Name, id),
						Why: "author field not in format \"NAME <EMAIL>\"",
						FixInstructions: "make field conform to specified format",
						InternalError: nil,
					}
				}

				if !models.ContactStrExp.Match([]byte(manifest.Maintainer)) {
					return nil, &ParseError{
						What: fmt.Sprintf("%s file for app in %s directory", *content.Name, id),
						Why: "maintainer field not in format \"NAME <EMAIL>\"",
						FixInstructions: "make field conform to specified format",
						InternalError: nil,
					}
				}

				// {{{2 Assign values from manifestFile to app
				// {{{3 Downcase tags and categories
				for _, tag := range manifest.Tags {
					app.Tags = append(app.Tags, strings.ToLower(tag))
				}

				for _, category := range manifest.Categories {
					app.Categories = append(app.Categories, strings.ToLower(category))
				}
				
				// {{{3 Set rest normally
				app.Name = manifest.Name
				app.Tagline = manifest.Tagline
				app.Author = manifest.Author
				app.Maintainer = manifest.Maintainer
				
			case "README.md":
				// {{{2 Get content
				txt, err := p.GetFileContent(fmt.Sprintf("%s/%s", id, *content.Name))
				if err != nil {
					return nil, &ParseError{
						What: fmt.Sprintf("%s file for app in %s directory", *content.Name, id),
						Why: "failed to get contents of file from the GitHub API, an error response was returned",
						FixInstructions: fmt.Sprintf("%s will triage this issue", p.GHDevTeamName),
						InternalError: err,
					}
				}

				if len(txt) == 0 {
					return nil, &ParseError{
						What: fmt.Sprintf("%s file for app in %s directory", *content.Name, id),
						Why: "file is empty",
						FixInstructions: "add content to file",
						InternalError: nil,
					}
				}

				app.Description = txt
			case "logo.png":
				app.LogoURL = *content.DownloadURL
			}
		case "dir":
			switch *content.Name {
			case "screenshots":
				// {{{2 Get files in screenshots directory
				urls, err := p.GetDownloadURLs(fmt.Sprintf("%s/screenshots", id))
				if err != nil {
					return nil, &ParseError{
						What: fmt.Sprintf("%s directory for app in %s directory", *content.Name, id),
						Why: "failed to get files in directory from GitHub API, an error response was returned",
						FixInstructions: fmt.Sprintf("%s will triage this issue", p.GHDevTeamName),
						InternalError: err,
					}
				}
				
				app.ScreenshotURLs = urls
				
			case "deployment":
				// {{{2 Get files in deployment directory
				urls, err := p.GetDownloadURLs(fmt.Sprintf("%s/deployment", id))
				if err != nil {
					return nil, &ParseError{
						What: fmt.Sprintf("%s directory for app in %s directory", *content.Name, id),
						Why: "failed to get files in directory from GitHub API, an error response was returned",
						FixInstructions: fmt.Sprintf("%s will triage this issue", p.GHDevTeamName),
						InternalError: err,
					}
				}
				
				app.DeploymentFileURLs = urls
			}
		}
	}

	// {{{1 Create version hash of app
	asJSON, err := json.Marshal(app)
	if err != nil {
		return nil, &ParseError{
			What: fmt.Sprintf("app in %s directory", id),
			Why: "error when creating app version hash",
			FixInstructions: fmt.Sprintf("%s will triage this issue", p.GHDevTeamName),
			InternalError: err,
		}
	}

	app.Version = fmt.Sprintf("%x", sha256.Sum256([]byte(asJSON)))

	// {{{1 Validate app
	validate := validator.New()
	err = validate.Struct(app)
	if err != nil {
		return nil, &ParseError{
			What: fmt.Sprintf("app in %s directory", id),
			Why: fmt.Sprintf("the information gathered about the app is not valid: %s", err.Error()),
			FixInstructions: "fix validation error",
			InternalError: nil,
		}
	}

	return &app, nil
}
