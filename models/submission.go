package models

// Submission holds information about a serverless application which is
// being submitted to the registry repository.
// Currently applications are submitted via pull requests.
type Submission struct {
	// PRNumber is the user facing ID number of the pull request
	PRNumber int `bson:"pr_number"`

	// Apps are the applications which are currently in the pull request.
	// Keys are app IDs.
	// A value can be nil if an internal system error occurred while parsing /
	// loading the app.
	Apps map[string]*SubmissionApp `bson:"apps"`
}

// SubmissionApp associates an app in a submission with a verification status entry
type SubmissionApp struct {
	// App is an app which is present in the submission, nil if
	// the VerificationStatus.FormatCorrect field is false.
	App *App `bson:"app"`

	// VerificationStatus of the App
	VerificationStatus AppVerificationStatus `bson:"verification_status"`
}

// AppVerificationStatus holds the verification status of an app
type AppVerificationStatus struct {
	// FormatCorrect indicates if the app submission files are formatted correctly
	FormatCorrect bool `bson:"format_correct"`
}
