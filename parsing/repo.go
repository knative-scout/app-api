package parsing

import (
	"strings"
	"fmt"
	"context"
	"encoding/json"
	"crypto/sha256"

	"github.com/kscout/serverless-registry-api/models"
	"github.com/kscout/serverless-registry-api/validation"
	
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
func (p RepoParser) GetApp(id string) (*models.App, []ParseError) {
	// {{{1 Get contents of app directory
	_, dirContents, _, err := p.GH.Repositories.GetContents(p.Ctx, p.RepoOwner,
		p.RepoName, id, &github.RepositoryContentGetOptions{
			Ref: p.RepoRef,
		})
	if err != nil {
		return nil, []ParseError{ParseError{
			What: "all files in the app directory",
			Why: "the GitHub API returned an error response",
			InternalError: err,
		}}
	}

	if len(dirContents) == 0 {
		return nil, []ParseError{ParseError{
			What: "all files in the app directory",
			Why: "no files were found",
			FixInstructions: "add required files",
		}}
	}

	// {{{1 Parse contents into App
	errs := []ParseError{}
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
		// whatFile will be used as the ParseError.What field value if necessary
		whatFile := fmt.Sprintf("`%s` file", *content.Name)
		
		// {{{2 Check if file / directory is supposed to be there
		if _, ok := found[*content.Name]; !ok {
			errs = append(errs, ParseError{
				What: whatFile,
				Why: fmt.Sprintf("not allowed in an app directory",
					*content.Name),
				FixInstructions: "delete this file",
			})
			continue
		}

		found[*content.Name] = true

		// {{{2 Parse file / directory for app info
		switch *content.Type {
		case "file":
			switch *content.Name {
			case "manifest.yaml":
				// {{{2 Get manifest.yaml file content
				txt, err := p.GetFileContent(fmt.Sprintf("%s/%s", id,
					*content.Name))
				if err != nil {
					errs = append(errs, ParseError{
						What: whatFile,
						Why: "failed to get contents from the GitHub API",
						InternalError: err,
					})
					continue
				}
				
				// {{{2 Parse as YAML
				var manifest models.AppManifestFile
				err = yaml.UnmarshalStrict([]byte(txt), &manifest)
				if err != nil {
					errs = append(errs, ParseError{
						What: whatFile,
						Why: fmt.Sprintf("failed to parse file as "+
							"YAML: %s", err.Error()),
						FixInstructions: "fix any YAML syntax errors",
					})
					continue
				}

				// {{{2 Downcase tags and categories
				for _, tag := range manifest.Tags {
					app.Tags = append(app.Tags, strings.ToLower(tag))
				}

				for _, category := range manifest.Categories {
					app.Categories = append(app.Categories, strings.ToLower(category))
				}
				
				// {{{3 Set App fields from manifest values
				app.Name = manifest.Name
				app.Tagline = manifest.Tagline
				app.Author = manifest.Author
				app.Maintainer = manifest.Maintainer
				
			case "README.md":
				// {{{2 Get content
				txt, err := p.GetFileContent(fmt.Sprintf("%s/%s", id,
					*content.Name))
				if err != nil {
					errs = append(errs, ParseError{
						What: whatFile,
						Why: "failed to get file content, the GitHub "+
							"API returned any error response",
						InternalError: err,
					})
					continue
				}

				app.Description = txt
			case "logo.png":
				app.LogoURL = *content.DownloadURL
			}
		case "dir":
			// whatDir is used as the ParseError.What field value is necessary
			whatDir := fmt.Sprintf("`%s` directory", *content.Name)
			
			switch *content.Name {
			case "screenshots":
				// {{{2 Get files in screenshots directory
				urls, err := p.GetDownloadURLs(fmt.Sprintf("%s/screenshots", id))
				if err != nil {
					errs = append(errs, ParseError{
						What: whatDir,
						Why: "failed to list files in the directory "+
							"using the GitHub API, an error "+
							"response was returned",
						InternalError: err,
					})
					continue
				}
				
				app.ScreenshotURLs = urls
				
			case "deployment":
				// {{{2 Get files in deployment directory
				urls, err := p.GetDownloadURLs(fmt.Sprintf("%s/deployment", id))
				if err != nil {
					errs = append(errs, ParseError{
						What: whatDir,
						Why: "failed to list files in the directory "+
							"using the GitHub API, an error "+
							"response was returned",
						InternalError: err,
					})
					continue
				}
				
				app.DeploymentFileURLs = urls
			}
		}
	}

	// {{{1 Create version hash of app
	asJSON, err := json.Marshal(app)
	if err != nil {
		errs = append(errs, ParseError{
			What: "the process which computes the app's `version` field",
			Why: "interal server error",
			InternalError: err,
		})
		return nil, errs
	}

	app.Version = fmt.Sprintf("%x", sha256.Sum256([]byte(asJSON)))

	// {{{1 Validate app
	// Don't validate if there were errors parsing the content
	if len(errs) > 0 {
		return nil, errs
	}
	
	err = validation.ValidateApp(app)

	// {{{2 Convert validation errors to ParseErrors
	if err != nil {
		// If field validation errors (Most times)
		if fieldErrs, ok := err.(validator.ValidationErrors); ok {
			// whatMap maps models.App field names to names of files and fields
			// which user submited. Keys are App field names. Values
			// are strings describing the field in a context the user understands.
			// If a field is not in this map it means the field is a value computed
			// by this RepoParser.GetApp method, not provided by the user.
			whatMap := map[string]string{
				"Name": "`name` field in the `manifest.yaml` file",
				"Tagline": "`tagline` field in the `manifest.yaml` file",
				"Tags": "`tags` array in the `manifest.yaml` file",
				"Categories": "`categories` array in the `manifest.yaml` file",
				"Author": "`author` field in the `manifest.yaml` file",
				"Maintainer": "`maintainer` field in the `manifest.yaml` file",
				"Description": "`README.md` file",
				"ScreenshotURLs": "`screenshots` directory",
				"LogoURL": "`logo.png` file",
				"DeploymentFileURLs": "`deployment` directory",
			}

			// whyMap maps validation tags to user readable reasons for the validation
			// failing. Keys are tag names, values are arrays which always have 2
			// items. The first item will be the reason why, the second item will
			// be the fix instructions.
			// If a tag isn't in the map it means the validation should never fail
			// in this method. It failing means an internal error occured, unrelated
			// to the user's input.
			whyMap := map[string][]string{
				"required": []string{
					"a value must be provided",
					"set a value",
				},
				"contact_info": []string{
					"must be in format: `NAME <EMAIL>`",
					"ensure value matches specified format",
				},
			}
			
			for _, fieldErr := range fieldErrs {
				// If a field the user provides a value for
				if what, ok := whatMap[fieldErr.Field()]; ok {
					// If validation error is caused by user's input
					if why, ok := whyMap[fieldErr.Tag()]; ok {
						errs = append(errs, ParseError{
							What: what,
							Why: why[0],
							FixInstructions: why[1],
						})
					} else { // error caused by this method, not user input
						errs = append(errs, ParseError{
							What: what,
							Why: "internal server error occurred",
							InternalError: fmt.Errorf("the \"%s\" "+
								"validation tag failed",
								fieldErr.Tag()),
						})
					}
				} else { // If a field computed by this method, not user provided
					errs = append(errs, ParseError{
						What: fmt.Sprintf("the `%s` internal "+
							"meta field", fieldErr.Field()),
						Why: "internal server error occurred",
						InternalError: fmt.Errorf("the \"%s\" field "+
							"failed the \"%s\" validation tag",
							fieldErr.Field(), fieldErr.Tag()),
					})
				}
			}
		} else { // Rarely, an internal error will occur when validating
			errs = append(errs, ParseError{
				What: "the app validation process failed",
				Why: "internal server error occurred",
				InternalError: err,
			})
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return &app, nil
}
