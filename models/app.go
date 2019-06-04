package models

// App is a serverless application from the repository
// Stores the json file format of the response
// Fields in structures can have tags, which hold metadata about a field.
// The `json:"VALUE"` tag tells the JSON library to place the value of that field under the `VALUE` key in the json object.
type App struct {
	// ID is a human and computer readable identifier for the application
	ID string `json:"id"`

	// Name to display to users
	Name string `json:"name"`

	// Tagline is a short description of the app
	Tagline string `json:"tagline"`

	// Description is more detailed markdown formatted information about the app
	Description string `json:"description"`

	// ScreenshotURLs are links to app screenshots
	ScreenshotURLs []string `json:"screenshot_urls"`

	// LogoURL is a link to the app logo
	LogoURL string `json:"logo_url"`

	// Tags are lists of tags
	Tags []string `json:"tags"`

	// VerificationStatus indicates the stage of the verification process the app
	// is currently in. Can be one of: "pending", "verifying", "good", "bad"
	VerificationStatus string `json:"verification_status"`

	// GitHubURL is a link to the GitHub files for the app
	GitHubURL string `json:"github_url"`

	// Version is the semantic version of the app
	Verison string `json:"version"`

	// Author is the person who created the app
	Author string `json:"author"`

	// Maintainer is the person who will support the app
	Maintainer string `json:"maintainer"`
}
