package models

import (
	"context"
	"fmt"
	"crypto/sha256"
	"encoding/json"

	"github.com/kscout/serverless-registry-api/config"
	
	"github.com/google/go-github/v25/github"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

// AppLoader loads applications from the registry repository
type AppLoader struct {
	// Ctx is the application context
	Ctx context.Context

	// Gh is a GitHub API client
	Gh *github.Client

	// Cfg is application configuration
	Cfg *config.Config
}

// manifestFile holds some of the metadata about a serverless application.
type manifestFile struct {
	// Name to display to users
	Name string `yaml:"name"`
	
	// Tagline is a short description of the app
	Tagline string `yaml:"tagline"`

	// Tags is a lists of tags
	Tags []string `yaml:"tags"`

	// Categories is a list of categories
	Categories []string `yaml:"categories"`

	// Author is the person who created the app
	Author string `yaml:"author"`

	// Maintainer is the person who will support the app
	Maintainer string `yaml:"maintainer"`
}

// AppSrcFormatError indicates the source files in the registry repository for an application are misformatted
// These errors should be presentable to the user
type AppSrcFormatError struct {
	// name of file or directory which error relates to, can be empty if error refers to the general app
	name string
	
	// public is a user facing error description
	public string

	// error holds the technical non-user facing error details, if nil public field will be displayed
	error error
}

// Error implements the Error interface
func (e AppSrcFormatError) Error() string {
	out := e.PublicError()

	if e.error != nil {
		out += fmt.Sprintf(": %s", e.error.Error())
	}

	return out
}

// PublicError returns a public user facing error
func (e AppSrcFormatError) PublicError() string {
	out := ""

	if len(e.name) > 0 {
		out += fmt.Sprintf("formatting error in %s: ", e.name)
	}

	out += e.public

	return out
}

// getGhURLsFromDir returns an array of GitHub HTML links to files in the specified directory
// in the app registry repository.
// The ref argument can be used to load GH URLs from commits other than HEAD. Pass an empty
// string to read from HEAD.
func (l AppLoader) getGhURLsFromDir(ref string, prefix string) ([]string, error) {
	// {{{1 Make API call
	_, contents, _, err := l.Gh.Repositories.GetContents(l.Ctx, l.Cfg.GhRegistryRepoOwner,
		l.Cfg.GhRegistryRepoName, prefix, &github.RepositoryContentGetOptions{
			Ref: ref,
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

// getGhFileContent returns the string contents of a file in the registry repository on GitHub.
// The ref argument can be used to load content from commits other than HEAD. Pass an empty
// string to read from HEAD.
func (l AppLoader) getGhFileContent(ref string, path string) (string, error) {
	content, _, _, err := l.Gh.Repositories.GetContents(l.Ctx, l.Cfg.GhRegistryRepoOwner,
		l.Cfg.GhRegistryRepoName, path, &github.RepositoryContentGetOptions{
			Ref: ref,
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

// GetAppIDsFromRegistry gets a list of app IDs from the registry repository.
// The ref argument can be used to load the app ID list from commits other than HEAD. Pass an
// empty string to read from HEAD.
func (l AppLoader) GetAppIDsFromRegistry(ref string) ([]string, error) {
	_, contents, _, err := l.Gh.Repositories.GetContents(l.Ctx, l.Cfg.GhRegistryRepoOwner,
		l.Cfg.GhRegistryRepoName, "", &github.RepositoryContentGetOptions{
			Ref: ref,
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

// LoadAllAppsFromRegistry loads all serverless applications from the registry repository.
// The ref argument can be used to load applications from commits other than HEAD. Pass an empty
// string to read from HEAD.
func (l AppLoader) LoadAllAppsFromRegistry(ref string) ([]*App, error) {
	// {{{1 Get names of all apps in repository
	appIDs, err := l.GetAppIDsFromRegistry(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get IDs of all apps in repository: %s",
			err.Error())
	}

	// {{{1 Loads each folder as an app
	apps := []*App{}
	
	for _, appID := range appIDs {
		app, err := l.LoadAppFromRegistry(ref, appID)
		if err != nil {
			return nil, fmt.Errorf("failed to load \"%s\" app: %s",
				appID, err.Error())
		}

		apps = append(apps, app)
	}

	return apps, nil
}

// LoadAppFromRegistry loads a single serverless application from the serverless application
// registry repository.
// The ref argument can be used to load an application from commits other than HEAD. Pass an
// empty string to read from HEAD.
func (l AppLoader) LoadAppFromRegistry(ref string, appName string) (*App, error) {
	// {{{1 Get contents of app directory
	_, dirContents, _, err := l.Gh.Repositories.GetContents(l.Ctx, l.Cfg.GhRegistryRepoOwner,
		l.Cfg.GhRegistryRepoName, appName, &github.RepositoryContentGetOptions{
			Ref: ref,
		})
	if err != nil {
		return nil, fmt.Errorf("error listing app directory contents: %s", err.Error())
	}

	if len(dirContents) == 0 {
		return nil, AppSrcFormatError{"", "app directory is empty", nil}
	}

	// {{{1 Parse contents into App
	app := App{}

	app.AppID = appName
	app.VerificationStatus = VerificationStatusPending
	app.GitHubURL = fmt.Sprintf("https://github.com/%s/%s/tree/%s/%s",
		l.Cfg.GhRegistryRepoOwner, l.Cfg.GhRegistryRepoName, ref, appName)

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
			return nil, AppSrcFormatError{
				*content.Name,
				fmt.Sprintf("%s not allowed to exist", content.Type),
				nil,
			}
		}

		found[*content.Name] = true

		// {{{2 Parse file / directory for app info
		switch *content.Type {
		case "file":
			switch *content.Name {
			case "manifest.yaml":
				// {{{2 Get manifest.yaml file content
				txt, err := l.getGhFileContent(ref,
					fmt.Sprintf("%s/manifest.yaml", appName))
				if err != nil {
					return nil, fmt.Errorf("failed to get content of "+
						"manifest.yaml file: %s", err.Error())
				}
				
				// {{{2 Parse as YAML
				var manifest manifestFile
				err = yaml.UnmarshalStrict([]byte(txt), &manifest)
				if err != nil {
					return nil, AppSrcFormatError{
						*content.Name,
						fmt.Sprintf("failed to parse as YAML: %s",
							err.Error()),
						nil,
					}
				}

				// {{{2 Using custom validation for author and maintainer fields
				if !contactStringExp.Match([]byte(manifest.Author)) {
					return nil, AppSrcFormatError{
						*content.Name,
						"author field must be in format \"NAME <EMAIL>\"",
						nil,
					}
				}

				if !contactStringExp.Match([]byte(manifest.Maintainer)) {
					return nil, AppSrcFormatError{
						*content.Name,
						"maintainer field must be in format \"NAME <EMAIL>\"",
						nil,
					}
				}

				// {{{2 Assign values from manifestFile to app
				app.Name = manifest.Name
				app.Tagline = manifest.Tagline
				app.Tags = manifest.Tags
				app.Categories = manifest.Categories
				app.Author = manifest.Author
				app.Maintainer = manifest.Maintainer
				
			case "README.md":
				// {{{2 Get content
				txt, err := l.getGhFileContent(ref,
					fmt.Sprintf("%s/README.md", appName))
				if err != nil {
					return nil, fmt.Errorf("failed to get content of "+
						"README.md file: %s", err.Error())
				}

				if len(txt) == 0 {
					return nil, AppSrcFormatError{
						*content.Name,
						"file cannot be empty",
						nil,
					}
				}

				app.Description = txt
			case "logo.png":
				app.LogoURL = *content.HTMLURL
			}
		case "dir":
			switch *content.Name {
			case "screenshots":
				// {{{2 Get files in screenshots directory
				urls, err := l.getGhURLsFromDir(ref,
					fmt.Sprintf("%s/screenshots", appName))
				if err != nil {
					return nil, fmt.Errorf("failed to get files in "+
						"screenshots directory: %s", err.Error())
				}
				app.ScreenshotURLs = urls
				
			case "deployment":
				// {{{2 Get files in deployment directory
				urls, err := l.getGhURLsFromDir(ref,
					fmt.Sprintf("%s/deployment", appName))
				if err != nil {
					return nil, fmt.Errorf("failed to get files in "+
						"deployment directory: %s", err.Error())
				}
				
				app.DeploymentFileURLs = urls
			}
		}
	}

	// {{{1 Create version hash of app
	asJSON, err := json.Marshal(app)
	if err != nil {
		return nil, fmt.Errorf("error marshalling app into JSON so it could be "+
			"hashed: %s", err.Error())
	}

	app.Version = fmt.Sprintf("%x", sha256.Sum256([]byte(asJSON)))

	// {{{1 Validate app
	validate := validator.New()
	err = validate.Struct(app)
	if err != nil {
		return nil, AppSrcFormatError{
			"",
			fmt.Sprintf("a piece of data in your app is invalid: %s", err.Error()),
			nil,
		}
	}

	return &app, nil
}
