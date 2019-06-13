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

		urls = append(urls, *content.DownloadURL)
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
			What: fmt.Sprintf("app in `%s` directory", id),
			Why: "the GitHub API returned an error response",
			FixInstructions: fmt.Sprintf("%s will triage this issue", p.GHDevTeamName),
			InternalError: err,
		}
	}

	if len(dirContents) == 0 {
		return nil, &ParseError{
			What: fmt.Sprintf("app in `%s` directory", id),
			Why: "no content found",
			FixInstructions: "add required files",
			InternalError: nil,
		}
	}

	// {{{1 Parse contents into App
	app := models.App{}

	app.AppID = id
	app.VerificationStatus = models.VerificationStatusPending

	ghURLRef := "master"
	if len(p.RepoRef) > 0 {
		ghURLRef = p.RepoRef
	}
	app.GitHubURL = fmt.Sprintf("https://github.com/%s/%s/tree/%s/%s",
		p.RepoOwner, p.RepoName, ghURLRef, id)

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
				What: fmt.Sprintf("`%s` %s for app in `%s` directory",
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
						What: fmt.Sprintf("`%s` file for app in `%s` directory", *content.Name, id),
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
						What: fmt.Sprintf("`%s` file for app in `%s` directory", *content.Name, id),
						Why: fmt.Sprintf("failed to parse as YAML: %s", err.Error()),
						FixInstructions: "fix YAML syntax",
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
						What: fmt.Sprintf("`%s` file for app in `%s` directory", *content.Name, id),
						Why: "failed to get contents of file from the GitHub API, an error response was returned",
						FixInstructions: fmt.Sprintf("%s will triage this issue", p.GHDevTeamName),
						InternalError: err,
					}
				}

				if len(txt) == 0 {
					return nil, &ParseError{
						What: fmt.Sprintf("`%s` file for app in `%s` directory", *content.Name, id),
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
						What: fmt.Sprintf("`%s` directory for app in `%s` directory", *content.Name, id),
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
						What: fmt.Sprintf("`%s` directory for app in `%s` directory", *content.Name, id),
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
			What: fmt.Sprintf("app in `%s` directory", id),
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
		// {{{2 If error is a validation error
		if vErr, ok := err.(validator.ValidationErrors); ok {
			// whatMap keys are models.App field names, values are the files
			// which these keys get their values in the github repository
			whatMap := map[string]string{
				"name": "`manifest.yaml` file",
				"tagline": "`manifest.yaml` file",
				"tags": "`manifest.yaml` file",
				"categories": "`manifest.yaml` file",
				"author": "`manifest.yaml` file",
				"maintainer": "`manifest.yaml` file",
				"description": "`README.md` file",
				"logo_url": "`logo.png` file",
				"screenshot_urls": "`screenshots` directory",
				"deployment_file_urls": "`deployment` directory",
			}

			// metaParseErrs will hold a ParseError for each field in
			// the models.App which has an error. If the ParseError.What
			// field is empty in any of the items this indicates it was an error
			// with the parsing process itself.
			metaParseErrs := []ParseError{}

			// {{{3 Build ParseError for each validation error
			for _, fieldErr := range vErr {
				parseErr := ParseError{}
				
				// {{{3 Determine which PR file is responsible for field
				// If field in whatMap then the validation error involves a user specified value
				if prFile, ok := whatMap[fieldErr.Field()]; ok {
					parseErr.What = prFile

					switch fieldErr.Tag() {
					case "required":
						parseErr.Why = "value is required"
						parseErr.FixInstructions = "add value"
					case "lowercase":
						parseErr.Why = "internal error, server failed to "+
							"pre-process value correctly"
						parseErr.FixInstructions = fmt.Sprintf("%s will triage this issue", p.GHDevTeamName)
						parseErr.InternalError = fmt.Errorf("server did not lowercase %s field values", fieldErr.Field())
					case "contact_info":
						parseErr.Why = fmt.Sprintf("%s field is not in format field not in format \"NAME <EMAIL>\"",
							fieldErr.Field())
						parseErr.FixInstructions = "make field conform to specified format"
					default:
						parseErr.Why = "unknown internal server error"
						parseErr.FixInstructions = fmt.Sprintf("%s will triage this issue", p.GHDevTeamName)
						parseErr.InternalError = fmt.Errorf("unexpected validation tag \"%s\" failed on field \"%s\"",
							fieldErr.Tag(), fieldErr.Field())
					}
				} else {
					parseErr.What = fmt.Sprintf("`%s` field in app data", fieldErr.Field())
					parseErr.Why = fmt.Sprintf("failed \"%s\" validation", fieldErr.Tag())
					parseErr.FixInstructions = fmt.Sprintf("%s will triage this issue", p.GHDevTeamName)
					parseErr.InternalError = fmt.Errorf("a computed field \"%s\" which the user does not enter information for "+
						"failed a validation \"%s\"", fieldErr.Field(), fieldErr.Tag())
				}

				metaParseErrs = append(metaParseErrs, parseErr)
			}

			// {{{3 Combine all field errors into one
			if len(metaParseErrs) == 1 {
				return nil, &ParseError{
					What: fmt.Sprintf("%s for app in the `%s` directory", metaParseErrs[0].What, id),
					Why: metaParseErrs[0].Why,
					FixInstructions: metaParseErrs[0].FixInstructions,
					InternalError: metaParseErrs[0].InternalError,
				}
			} else {
				parseErr := ParseError{}
				
				// {{{4 Combine all whats
				parseErr.What = fmt.Sprintf("multiple items related to an app in the `%s` directory  \n\n", id)
				parseErr.Why = "reason for each item listed below:  \n\n"
				parseErr.FixInstructions = "fix instructions for each item listed below:  \n\n"

				internalErrs := []string{}

				for _, metaParseErr := range metaParseErrs {
					parseErr.What += fmt.Sprintf("- %s\n", metaParseErr.What)
					parseErr.Why += fmt.Sprintf("- %s: %s\n", metaParseErr.What, metaParseErr.Why)
					parseErr.FixInstructions += fmt.Sprintf("- %s: %s\n", metaParseErr.What, metaParseErr.FixInstructions)

					if metaParseErr.InternalError != nil {
						internalErrs = append(internalErrs, metaParseErr.InternalError.Error())
					}
				}

				if len(internalErrs) > 0 {
					parseErr.InternalError = fmt.Errorf("multiple errors: %s", strings.Join(internalErrs, ", "))
				}

				return nil, &parseErr
			}
		} else {
			return nil, &ParseError{
				What: fmt.Sprintf("app in `%s` directory", id),
				Why: fmt.Sprintf("the app could not be validated: %s", err.Error()),
				FixInstructions: fmt.Sprintf("%s will triage this issue", p.GHDevTeamName),
				InternalError: err,
			}
		}
	}

	return &app, nil
}
