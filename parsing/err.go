package parsing

import (
	"fmt"
)

// ParseError provides details about a failure to parse an object. ParseErrors are meant to
// be presented to users.
type ParseError struct {
	// What indicates the object that failed to be parsed
	What string

	// Why indicates why the object failed to be parsed
	Why string

	// FixInstructions for the user to remedy this error
	FixInstructions string

	// InternalError is a non user presentable error which will be logged for
	// debug purposes. Can be nil if error is entirely user's fault.
	InternalError error
}

// Error returns an internal error string which should not be shown to the user
func (e ParseError) Error() string {
	if e.InternalError != nil {
		return fmt.Sprintf("%s (%s)", e.UserError(), e.InternalError.Error())
	} else {
		return e.UserError()
	}
}

// UserError returns an error string meant to be displayed to the user
func (e ParseError) UserError() string {
	return fmt.Sprintf("failed to parse %s: %s: %s",
		e.What, e.Why, e.FixInstructions)
}
