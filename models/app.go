package models

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

// AppDeployParameter holds information about a parameter in an app's deployment resources
type AppDeployParameter struct {
	// Substitution is the value which should be substitured for the parameter's value
	Substitution string `json:"substitution" bson:"substitution" validate:"required"`

	// DisplayName is a user friendly name to describe the parameter
	DisplayName string `json:"display_name" bson:"display_name" validate:"required"`

	// DefaultValue of parameter
	DefaultValue string `json:"default_value" bson:"default_value" validate:"required"`

	// RequiresBase64 indicates if the parameter should be encoded in base64 before being
	// placed in the template
	RequiresBase64 bool `json:"requires_base64" bson:"requires_base64" validate:"required"`
}

// AppDeployment holds deployment information about an app
type AppDeployment struct {
	// Resources is the raw JSON for each deployment resource
	Resources []string `json:"resources" bson:"resources" validate:"required"`

	// ParameterizedResources is the JSON for each deployment resource, except values in
	// ConfigMap and Secret resources are replaced with their
	// AppDeployParameter.SubstituionVariable value
	ParameterizedResources []string `json:"parameterized_resources" bson:"parameterized_resources" validate:"required"`

	// Parameters holds metadata about the parameters in PrameterizedYAML
	Parameters []AppDeployParameter `json:"parameters" bson:"parameters" validate:"required"`

	// DeployScript is a custom deployment script for the app
	DeployScript string `json:"deploy_script" bson:"deploy_script" validate:"required"`
}

// App is a serverless application from the repository
// Stores the json file format of the response
type App struct {
	// Name to display to users
	Name string `json:"name" bson:"name" validate:"required"`
	
	// Tagline is a short description of the app
	Tagline string `json:"tagline" bson:"tagline" validate:"required"`

	// Tags is a lists of tags
	Tags []string `json:"tags" bson:"tags" validate:"required,lowercase"`

	// Categories is a list of categories
	Categories []string `json:"categories" bson:"categories" validate:"required,lowercase"`

	// Author is the person who created the app
	Author string `yaml:"author" json:"author" bson:"author" validate:"required,contact_info"`

	// Maintainer is the person who will support the app
	Maintainer string `yaml:"maintainer" json:"maintainer" bson:"maintainer" validate:"required,contact_info"`
	
	// AppID is a human and computer readable identifier for the application
	AppID string `json:"app_id" bson:"app_id" validate:"required"`

	// Description is more detailed markdown formatted information about the app
	Description string `json:"description" bson:"description" validate:"required"`

	// ScreenshotURLs are links to app screenshots
	ScreenshotURLs []string `json:"screenshot_urls" bson:"screenshot_urls"`

	// LogoURL is a link to the app logo
	LogoURL string `json:"logo_url" bson:"logo_url" validate:"required"`

	// VerificationStatus indicates the stage of the verification process the app
	// is currently in. Can be one of: "pending", "verifying", "good", "bad"
	VerificationStatus VerStatusT `json:"verification_status" bson:"verification_status" validate:"required,verification_status_t"`

	// GitHubURL is a link to the GitHub files for the app
	GitHubURL string `json:"github_url" bson:"github_url" validate:"required"`

	// Deployment datan
	Deployment AppDeployment `json:"deployment" bson:"deployment" validate:"required"`

	// Version is the semantic version of the app
	Version string `json:"version" bson:"version" validate:"required"`
}

