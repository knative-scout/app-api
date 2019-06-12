package validation


// ValidatorDescriber provides basic information about a validator
type ValidatorDescriber interface {
	// Name returns the name of the validator
	Name() string

	// Summary returns a short description of what the check does
	Summary() string
}
