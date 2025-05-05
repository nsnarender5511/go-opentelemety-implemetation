package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	// Import the common errors package
	apierrors "github.com/narender/common/apierrors"
)

// Singleton validator instance
var validate = validator.New()

// ValidateRequest performs validation on the struct payload.
// Returns nil on success, or AppError with ErrCodeValidation on failure.
// Change function name to be exported and update return type
func ValidateRequest(payload interface{}) *apierrors.AppError {
	err := validate.Struct(payload)
	if err != nil {
		// Handle validation errors
		var validationErrors []string
		// Use type assertion to access validator specific error details
		if vErrs, ok := err.(validator.ValidationErrors); ok {
			for _, vErr := range vErrs {
				// Customize error messages based on tag/field if needed
				// Example: Provide more user-friendly messages based on vErr.Tag()
				validationErrors = append(validationErrors, fmt.Sprintf("Field '%s' failed validation on '%s' tag", vErr.Field(), vErr.Tag()))
			}
		} else {
			// Handle non-validator errors if necessary, though validate.Struct usually returns ValidationErrors
			validationErrors = append(validationErrors, err.Error())
		}

		errMsg := "Validation failed: " + strings.Join(validationErrors, "; ")
		// Use imported package's constants and constructor
		return apierrors.NewAppError(apierrors.ErrCodeValidation, errMsg, err) // Pass original validator error as cause
	}
	return nil // Validation passed 🎉
}
