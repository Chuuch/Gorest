package validation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Struct(value any) error {
	return validate.Struct(value)
}

func Errors(err error) map[string]string {
	var validationErrors validator.ValidationErrors

	if !errors.As(err, &validationErrors) {
		return nil
	}

	result := make(map[string]string, len(validationErrors))

	for _, fieldError := range validationErrors {
		result[fieldError.Field()] = message(fieldError)
	}

	return result
}

func message(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"

	case "email":
		return "must be a valid email address"

	case "min":
		return fmt.Sprintf("must be at least %s", err.Param())

	case "max":
		return fmt.Sprintf("must be at most %s characters", err.Param())

	default:
		return strings.ToLower(err.Tag())
	}
}
