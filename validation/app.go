package validation

import (
	"strings"
	
	"github.com/kscout/serverless-registry-api/models"
	
	"gopkg.in/go-playground/validator.v9"
)

// validCategories is a map set of valid category values
var validCategories map[string]bool = map[string]bool{
	"analytics": true,
	"automation": true,
	"entertainment": true,
	"hello world": true,
	"internet of things": true,
	"utilities": true,
	"virtual assistant": true,
	"other": true,
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

// validateCategories ensures that all items are a valid category value
func validateCategories(fl validator.FieldLevel) bool {
	iface := fl.Field().Interface()
	array, ok := iface.([]string)
	if !ok {
		return false
	}

	for _, value := range array {
		if _, ok := validCategories[value]; !ok {
			return false
		}
	}

	return true
}

// ValidateApp ensures that an App's data meets all constraints
func ValidateApp(app models.App) error {
	validate := validator.New()
	validate.RegisterValidation("lowercase", validateLowercase)
	validate.RegisterValidation("categories", validateCategories)

	return validate.Struct(app)
}
