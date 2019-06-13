package validation

import (
	"regexp"
	"strings"
	
	"github.com/kscout/serverless-registry-api/models"
	
	"gopkg.in/go-playground/validator.v9"
)

// contactInfoExp matches a string which holds contact information in the format:
//
//    NAME <EMAIL>
//
// Groups:
//    1. NAME
//    2. EMAIL
var contactInfoExp *regexp.Regexp = regexp.MustCompile("(.+) (<.+@.+>)")

// validateContactInfo is a custom validation which ensures a string matches the contactInfoExp
// Only works with fields which are strings.
func validateContactInfo(fl validator.FieldLevel) bool {
	return contactInfoExp.MatchString(fl.Field().String())
}

// validateLowercase ensures that all items in field are lowercase.
// Only works on fields which are string arrays.
func validateLowercase(fl validator.FieldLevel) bool {
	i := fl.Field().Interface()
	a, ok := i.([]string)
	if !ok {
		return false
	}

	for _, v := range a {
		if v != strings.ToLower(v) {
			return false
		}
	}

	return true
}

// validateVerificationStatusT ensures that a field matches one of the values of models.VerStatusT
// Only works on fields which are strings.
func validateVerificationStatusT(fl validator.FieldLevel) bool {
	s := models.VerStatusT(fl.Field().String())
	return s == models.VerificationStatusPending ||
		s == models.VerificationStatusVerifying ||
		s == models.VerificationStatusGood ||
		s == models.VerificationStatusBad
}

// ValidateApp ensures that an App's data meets all constraints
func ValidateApp(app models.App) error {
	validate := validator.New()
	validate.RegisterValidation("contact_info", validateContactInfo)
	validate.RegisterValidation("lowercase", validateLowercase)
	validate.RegisterValidation("verification_status_t", validateVerificationStatusT)

	return validate.Struct(app)
}
