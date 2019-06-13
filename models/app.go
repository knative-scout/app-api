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

	// DeploymentFileURLs are links to the Knative deployment resource files
	DeploymentFileURLs []string `json:"deployment_file_urls" bson:"deployment_file_urls" validate:"required"`

	// Version is the semantic version of the app
	Version string `json:"version" bson:"version" validate:"required"`
}

