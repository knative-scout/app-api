package models

import (
	"context"
	"fmt"
	"regexp"
	"crypto/sha256"
	"encoding/json"

	"github.com/knative-scout/app-api/config"
	
	"github.com/google/go-github/v25/github"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

// contactStringExp matches a string which holds contact information in the format:
//
//    NAME <EMAIL>
//
// Groups:
//    1. NAME
//    2. EMAIL
var contactStringExp *regexp.Regexp = regexp.MustCompile("(.+) (<.+@.+>)")

// VerStatusT is a type which valid values for App.VerificationStatus are
// represented as
type VerStatusT string

// VerificationStatusPending indicates the app has not been verified yet, and is not
// currently being verified
const VerificationStatusPending VerStatusT = "pending"

// VerificationStatusVerifying indicates the app is currently being verified
const VerificationStatusVerifying VerStatusT = "verifying"

// VerificationStatusGood indicates the app is verified and safe
const VerificationStatusGood VerStatusT = "good"

// VerificationStatusBad indicates the app has been processed and is not safe
const VerificationStatusBad VerStatusT = "bad"

// App is a serverless application from the repository
// Stores the json file format of the response
type App struct {
	manifestFile
	
	// AppID is a human and computer readable identifier for the application
	AppID string `json:"id" bson:"app_id" validate:"required"`

	// Description is more detailed markdown formatted information about the app
	Description string `json:"description" bson:"description" validate:"required"`

	// ScreenshotURLs are links to app screenshots
	ScreenshotURLs []string `json:"screenshot_urls" bson:"screenshot_urls"`

	// LogoURL is a link to the app logo
	LogoURL string `json:"logo_url" bson:"logo_url" validate:"required"`

	// VerificationStatus indicates the stage of the verification process the app
	// is currently in. Can be one of: "pending", "verifying", "good", "bad"
	VerificationStatus VerStatusT `json:"verification_status" bson:"verification_status" validate:"required"`

	// GitHubURL is a link to the GitHub files for the app
	GitHubURL string `json:"github_url" bson:"github_url" validate:"required"`

	// DeploymentFileURLs are links to the Knative deployment resource files
	DeploymentFileURLs []string `json:"deployment_file_urls" bson:"deployment_file_urls" validate:"required"`

	// Version is the semantic version of the app
	Version string `json:"version" bson:"version" validate:"required"`
}

// manifestFile holds some of the metadata about a serverless application.
type manifestFile struct {
	// Name to display to users
	Name string `yaml:"name" json:"name" bson:"name" validate:"required"`
	
	// Tagline is a short description of the app
	Tagline string `yaml:"tagline" json:"tagline" bson:"tagline" validate:"required"`

	// Tags is a lists of tags
	Tags []string `yaml:"tags" json:"tags" bson:"tags" validate:"required"`

	// Categories is a list of categories
	Categories []string `yaml:"categories" json:"categories" bson:"categories" validate:"required"`

	// Author is the person who created the app
	Author string `yaml:"author" json:"author" bson:"author" validate:"required"`

	// Maintainer is the person who will support the app
	Maintainer string `yaml:"maintainer" json:"author" bson:"maintainer" validate:"required"`
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
	out := ""

	if len(e.name) > 0 {
		out += fmt.Sprintf("formatting error in %s: ", e.name)
	}

	out += e.public

	if e.error != nil {
		out += fmt.Sprintf(": %s", e.error.Error())
	}

	return out
}

// getGhURLsFromDir returns an array of GitHub HTML links to files in the specified directory
// in the app registry repository.
func getGhURLsFromDir(ctx context.Context, gh *github.Client, cfg *config.Config, prefix string) ([]string, error) {
	// {{{1 Make API call
	_, contents, _, err := gh.Repositories.GetContents(ctx, cfg.GhRegistryRepoOwner,
		cfg.GhRegistryRepoName, prefix, nil)
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

// getGhFileContent returns the string contents of a file in the registry repository on GitHub
func getGhFileContent(ctx context.Context, gh*github.Client, cfg *config.Config, path string) (string, error) {
	content, _, _, err := gh.Repositories.GetContents(ctx, cfg.GhRegistryRepoOwner,
		cfg.GhRegistryRepoName, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get content from GitHub API: %s", err.Error())
	}

	txt, err := content.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode content: %s", err.Error())
	}

	return txt, nil
}

// LoadAllAppsFromRegistry loads all serverless applications from the registry repository
func LoadAllAppsFromRegistry(ctx context.Context, gh *github.Client, cfg *config.Config) ([]*App, error) {
	// {{{1 Get names of all folders at the top level
	_, contents, _, err := gh.Repositories.GetContents(ctx, cfg.GhRegistryRepoOwner,
		cfg.GhRegistryRepoName, "/", nil)
	if err != nil {
		return nil, fmt.Errorf("error listing top levle repository contents via "+
			"GitHub API: %s", err.Error())
	}

	// {{{1 Loads each folder as an app
	apps := []*App{}
	
	for _, content := range contents {
		if *content.Type == "file" {
			continue
		}

		app, err := LoadAppFromRegistry(ctx, gh, cfg, *content.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to load \"%s\" app: %s",
				*content.Name, err.Error())
		}

		apps = append(apps, app)
	}

	return apps, nil
}

// LoadAppFromRegistry loads a single serverless application from the serverless application
// registry repository.
func LoadAppFromRegistry(ctx context.Context, gh *github.Client, cfg *config.Config, appName string) (*App, error) {
	// {{{1 Get contents of app directory
	_, dirContents, _, err := gh.Repositories.GetContents(ctx, cfg.GhRegistryRepoOwner,
		cfg.GhRegistryRepoName, appName, nil)
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
	app.GitHubURL = fmt.Sprintf("https://github.com/%s/%s/tree/master/%s",
		cfg.GhRegistryRepoOwner, cfg.GhRegistryRepoName, appName)

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
		if *content.Type == "file" {
			if *content.Name == "manifest.yaml" {
				// {{{2 Get manifest.yaml file content
				txt, err := getGhFileContent(ctx, gh, cfg,
					fmt.Sprintf("%s/manifest.yaml", appName))
				if err != nil {
					return nil, fmt.Errorf("failed to get content of "+
						"manifest.yaml file: %s", err.Error())
				}
				
				// {{{2 Parse as YAML
				err = yaml.UnmarshalStrict([]byte(txt), &app.manifestFile)
				if err != nil {
					return nil, AppSrcFormatError{
						*content.Name,
						fmt.Sprintf("failed to parse as YAML: %s", err.Error()),
						nil,
					}
				}

				// {{{2 Using custom validation for author and maintainer fields
				if !contactStringExp.Match([]byte(app.manifestFile.Author)) {
					return nil, AppSrcFormatError{
						*content.Name,
						"author field must be in format \"NAME <EMAIL>\"",
						nil,
					}
				}

				if !contactStringExp.Match([]byte(app.manifestFile.Maintainer)) {
					return nil, AppSrcFormatError{
						*content.Name,
						"maintainer field must be in format \"NAME <EMAIL>\"",
						nil,
					}
				}
			} else if *content.Name == "README.md" {
				// {{{2 Get content
				txt, err := getGhFileContent(ctx, gh, cfg,
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
			} else if *content.Name == "logo.png" {
				app.LogoURL = *content.HTMLURL
			}
		} else { // dir
			if *content.Name == "screenshots" {
				// {{{2 Get files in screenshots directory
				urls, err := getGhURLsFromDir(ctx, gh, cfg,
					fmt.Sprintf("%s/screenshots", appName))
				if err != nil {
					return nil, fmt.Errorf("failed to get files in "+
						"screenshots directory: %s", err.Error())
				}
				app.ScreenshotURLs = urls
			} else if *content.Name == "deployment" {
				// {{{2 Get files in deployment directory
				urls, err := getGhURLsFromDir(ctx, gh, cfg,
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
