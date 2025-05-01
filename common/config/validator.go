package config

import (
	"fmt"

	"golang.org/x/exp/constraints" // Added for generic constraints
)

// Validator provides helper methods for configuration validation
type Validator struct {
	errors []error
}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	return &Validator{
		errors: []error{},
	}
}

// AddError adds an error for a field
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, fmt.Errorf("%s: %s", field, message))
}

// RequireNonEmpty validates that a string field is not empty
func (v *Validator) RequireNonEmpty(field, value string) {
	if value == "" {
		v.AddError(field, "cannot be empty")
	}
}

// RequireOneOf validates that a string value is one of the allowed values
func (v *Validator) RequireOneOf(field, value string, allowed []string) {
	for _, a := range allowed {
		if value == a {
			return
		}
	}
	v.AddError(field, fmt.Sprintf("must be one of: %v", allowed))
}

// RequireInRange validates that a numeric value is within a specified inclusive range.
// It uses generics (Go 1.18+) for type safety.
func RequireInRange[T constraints.Ordered](v *Validator, field string, value, min, max T) {
	if value < min || value > max {
		// Use %v for generic representation, which works for most numeric types.
		v.AddError(field, fmt.Sprintf("must be between %v and %v", min, max))
	}
}

// Errors returns all validation errors
func (v *Validator) Errors() []error {
	return v.errors
}
