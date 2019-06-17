package parsing

import (
	"fmt"
)

// ParseError provides details about a failure to parse an object. ParseErrors are meant to
// be presented to users.
// All string fields will be interpreted with Markdown formatting.
type ParseError struct {
	// What indicates the object that failed to be parsed.
	// This field does not have to provide context about what is being parsed. Just
	// what part of the parsing process failed.
	What string

	// Why indicates why the object failed to be parsed
	Why string

	// FixInstructions for the user to remedy this error
	// Leave this field blank if there is nothing the user can do to fix the issue,
	// ex., internal server error
	FixInstructions string

	// InternalError is a non user presentable error which will be logged for
	// debug purposes. Can be nil if error is entirely caused by user's input.
	// If not nil will be treated as if the server messed up in some way and
	// the dev team will be notified.
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
