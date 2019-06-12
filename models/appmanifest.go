package models

// AppManifestFile is the format of an app's manifest file in the registry repository.
// This structure holds metadata about an application. This data will be merged into the
// App model.
type AppManifestFile struct {
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
