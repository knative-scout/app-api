package validation

import (
	"strings"
	
	"github.com/kscout/serverless-registry-api/models"
	
	"gopkg.in/go-playground/validator.v9"
)

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

// ValidateApp ensures that an App's data meets all constraints
func ValidateApp(app models.App) error {
	validate := validator.New()
	validate.RegisterValidation("lowercase", validateLowercase)

	return validate.Struct(app)
}
