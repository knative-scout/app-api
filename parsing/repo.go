package parsing

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"fmt"
	"context"
	"encoding/json"
	"crypto/sha256"

	"github.com/kscout/serverless-registry-api/models"
	"github.com/kscout/serverless-registry-api/validation"
	
	"github.com/google/go-github/v26/github"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"gopkg.in/go-playground/validator.v9"
	v1Meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1Core "k8s.io/api/core/v1"
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
				err = yaml.Unmarshal([]byte(txt), &manifest)
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
				// {{{2 Get YAML for each resource
				// {{{3 Get files in directory
				_, dirContents, _, err := p.GH.Repositories.GetContents(p.Ctx, p.RepoOwner,
					p.RepoName, fmt.Sprintf("%s/deployment", id),
					&github.RepositoryContentGetOptions{
						Ref: p.RepoRef,
					})
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

				// {{{3 Get content of each file
				filesTxt := []string{}
				for _, deployContent := range dirContents {
					if *deployContent.Type == "dir" {
						continue
					}

					txt, err := p.GetFileContent(
						fmt.Sprintf("%s/deployment/%s",
							id, *deployContent.Name))
					if err != nil {
						errs = append(errs, ParseError{
							What: fmt.Sprintf("`%s/deployment/%s`"+
								"file", id, *deployContent.Name),
							Why: "failed to get content of file "+
								"using the GitHub API, an error "+
								"response was returned",
							InternalError: err,
						})
						continue
					}
					
					filesTxt = append(filesTxt, txt)
				}

				// {{{3 Split content up by resource
				resourcesYAML := [][]byte{}

				for _, fileTxt := range filesTxt {
					lines := []string{}
					for _, line := range strings.Split(fileTxt, "\n") {
						if strings.ReplaceAll(line, " ", "") == "---" {
							if len(lines) > 0 {
								resourcesYAML = append(resourcesYAML,
									[]byte(strings.Join(lines, "\n")))
								lines = []string{}
							}
						} else {
							lines = append(lines, line)
						}
					}
					
					if len(lines) > 0 {
						resourcesYAML = append(resourcesYAML,
							[]byte(strings.Join(lines, "\n")))
					}
				}

				// {{{2 Parse resources
				resourcesJSON := [][]byte{}

				params := []models.AppDeployParameter{}				
				paramdResourcesJSON := [][]byte{}

				for _, resourceYAML := range resourcesYAML {
					// {{{3 Convert to JSON
					resourceJSON, err := yaml.YAMLToJSON(resourceYAML)
					if err != nil {
						errs = append(errs, ParseError{
							What: whatDir,
							Why: "failed to convert YAML to JSON",
							InternalError: err,
						})
						continue
					}

					// {{{3 Parse type data
					var resourceType v1Meta.TypeMeta
					
					err = json.Unmarshal(resourceJSON, &resourceType)
					if err != nil {
						errs = append(errs, ParseError{
							What: whatDir,
							Why: "failed to parse resource type information",
							InternalError: err,
						})
						continue
					}

					// {{{3 Do not allow namespace resources in the deployment
					if resourceType.Kind == "Namespace" {
						errs = append(errs, ParseError{
							What: whatDir,
							Why: "resources of type Namespace are not allowed",
							FixInstructions: "remove all Namespace resources",
						})
						continue
					}

					// {{{3 Parse metadata
					var resourceMeta v1Meta.ObjectMeta

					err = json.Unmarshal(resourceJSON, &resourceMeta)
					if err != nil {
						errs = append(errs, ParseError{
							What: whatDir,
							Why: "failed to parse resource metadata information",
							InternalError: err,
						})
						continue
					}

					// {{{3 Do not allow resources with a namespace field
					if len(resourceMeta.Namespace) > 0 {
						errs = append(errs, ParseError{
							What: whatDir,
							Why: "resources may not have a metadata.namespace field",
							FixInstructions: "ensure resources do not have a metadata.namespace field",
						})
						continue
					}

					// {{{3 Parameterize
					// {{{4 Save un-parameterized resource
					resourcesJSON = append(resourcesJSON, resourceJSON)

					// Only parameterize v1 API resources
					if resourceType.APIVersion != "v1" {
						paramdResourcesJSON = append(paramdResourcesJSON, resourceJSON)
						continue
					}
					
					switch resourceType.Kind {
					case "Namespace":
						// Do not include Namespaces in deployment resources
						continue
					case "Secret":
						var secret v1Core.Secret
						
						err := json.Unmarshal(resourceJSON, &secret)
						if err != nil {
							errs = append(errs, ParseError{
								What: whatDir,
								Why: "failed to parse resource as v1.Secret",
								InternalError: err,
							})
							continue
						}

						newData := map[string][]byte{}
						
						for key, data := range secret.Data {
							param := models.AppDeployParameter{
								Substitution: uuid.New().String(),
								DisplayName: fmt.Sprintf("\"%s\" key in \"%s\" Secret",
									key, secret.Name),
								DefaultValue: string(data),
								RequiresBase64: true,
							}
							params = append(params, param)
							
							newData[key] = []byte(fmt.Sprintf("%s", param.Substitution))
						}

						secret.Data = newData

						resourceJSON, err = json.Marshal(secret)
						if err != nil {
							errs = append(errs, ParseError{
								What: whatDir,
								Why: "failed to save resource as JSON",
								InternalError: err,
							})
							continue
						}

						paramdResourcesJSON = append(paramdResourcesJSON, resourceJSON)
					case "ConfigMap":
						var configMap v1Core.ConfigMap
						
						err := json.Unmarshal(resourceJSON, &configMap)
						if err != nil {
							errs = append(errs, ParseError{
								What: whatDir,
								Why: "failed to parse resource as v1.ConfigMap",
								InternalError: err,
							})
							continue
						}

						newData := map[string]string{}
						
						for key, data := range configMap.Data {
							param := models.AppDeployParameter{
								Substitution: uuid.New().String(),
								DisplayName: fmt.Sprintf("\"%s\" key in \"%s\" ConfigMap",
									key, configMap.Name),
								DefaultValue: data,
								RequiresBase64: false,
							}
							params = append(params, param)
							
							newData[key] = fmt.Sprintf("%s", param.Substitution)
						}

						configMap.Data = newData

						resourceJSON, err = json.Marshal(configMap)
						if err != nil {
							errs = append(errs, ParseError{
								What: whatDir,
								Why: "failed to save resource as JSON",
								InternalError: err,
							})
							continue
						}

						paramdResourcesJSON = append(paramdResourcesJSON, resourceJSON)
					}
				}

				resourcesStr := []string{}
				for _, resource := range resourcesJSON {
					resourcesStr = append(resourcesStr, string(resource))
				}

				paramdResourcesStr := []string{}
				for _, resource := range paramdResourcesJSON {
					paramdResourcesStr = append(paramdResourcesStr, string(resource))
				}

				deploymentScript := CreateDeploymentScript(id, params, strings.Join(paramdResourcesStr, "\n"))

				app.Deployment = models.AppDeployment{
					Resources: resourcesStr,
					ParameterizedResources: paramdResourcesStr,
					Parameters: params,
					DeployScript: deploymentScript,
				}
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
				"Deployment": "`deployment` directory",
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


func CreateDeploymentScript(id string, params []models.AppDeployParameter, ymlfile string) string {

	//opening deploy.sh file
	file, err := os.Open("parsing/deploy.sh")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			panic(fmt.Errorf("unable to close file : %s",err))
		}
	}()

	script, err := ioutil.ReadAll(file)
	bashrc := string(script)

	secrets := "\n"+
	"ID=\"{{param.id}}\"\n"+
	"KEY=\"{{param.key}}\"\n"+
	"DFLT=\"{{param.dflt}}\"\n"+
	"BASE64=\"{{param.base64}}\"\n"+

	"echo  #new line\n"+
	"echo \"Default Value for $KEY is '$DFLT'\"\n"+
	"read -p \"Do you want to change it ? (y/n): \" choice\n"+

	"case \"$choice\" in\n"+
	"y|Y|yes|YES|Yes )\n"+
	"read -p \"Enter new value for $KEY : \" value\n"+
	"if [[ \"$BASE64\" == \"Y\" ]]\n"+
	"then\n"+
	"value=$(echo \"${value}\" | base64)\n"+
	"else\n"+
	"value=\"${value}\"\n"+
	"fi\n"+
	"SED_DATA=\"$SED_DATA ; s/$ID/$value/\" ;;\n"+
	"n|N|no|NO|No )\n"+
	"if [[ \"$BASE64\" == \"Y\" ]]\n"+
	"then\n"+
	"DFLT=\"${value}\"\n"+
	"else\n"+
	"DFLT=$(echo \"${DFLT}\" | base64 -d)\n"+
	"fi\n"+
	"SED_DATA=\"$SED_DATA ; s/$ID/$DFLT/\";;\n"+
	"* ) echo \"invalid input, Please run the script again\";;\n"+
	"esac\n"+
	"\n"

	secretsRes:= ""

	for _,parameter := range params{
		ID := parameter.Substitution
		KEY := parameter.DisplayName
		DFLT := parameter.DefaultValue
		BASE64 := "N"
		if parameter.RequiresBase64{
			BASE64 = "Y"
		}



		tempSec := secrets
		tempSec = strings.ReplaceAll(tempSec, "{{param.id}}", ID)
		tempSec = strings.ReplaceAll(tempSec, "{{param.key}}", KEY)
		tempSec = strings.ReplaceAll(tempSec, "{{param.dflt}}", DFLT)
		tempSec = strings.ReplaceAll(tempSec, "{{param.base64}}", BASE64)

		secretsRes = secretsRes + tempSec

	}

	bashrc = strings.ReplaceAll(bashrc, "{{{replacement.script}}}", secretsRes)
	bashrc = strings.ReplaceAll(bashrc, "{{{yaml.file}}}", ymlfile)


	return bashrc
}
