package models

// AppManifestFile is the format of an app's manifest file in the registry repository.
// This structure holds metadata about an application. This data will be merged into the
// App model.
type AppManifestFile struct {
	// Name to display to users
	Name string `yaml:"name"`

	// HomepageURL is a link to the website for this app.
	HomepageURL string `yaml:"homepage_url"`

	// Tagline is a short description of the app
	Tagline string `yaml:"tagline"`

	// Tags is a lists of tags
	Tags []string `yaml:"tags"`

	// Categories is a list of categories
	Categories []string `yaml:"categories"`

	// Author is the person who created the app
	Author ContactInfo `yaml:"author"`
}
