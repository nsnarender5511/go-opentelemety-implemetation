package config

import (
	"fmt"
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

// RequireInRange validates that a numeric value is within a range
func (v *Validator) RequireInRange(field string, value, min, max interface{}) {
	switch value := value.(type) {
	case int:
		minVal, minOk := min.(int)
		maxVal, maxOk := max.(int)
		if minOk && maxOk && (value < minVal || value > maxVal) {
			v.AddError(field, fmt.Sprintf("must be between %d and %d", minVal, maxVal))
		}
	case float64:
		minVal, minOk := min.(float64)
		maxVal, maxOk := max.(float64)
		if minOk && maxOk && (value < minVal || value > maxVal) {
			v.AddError(field, fmt.Sprintf("must be between %f and %f", minVal, maxVal))
		}
	default:
		v.AddError(field, "invalid type for range check")
	}
}

// Errors returns all validation errors
func (v *Validator) Errors() []error {
	return v.errors
}
