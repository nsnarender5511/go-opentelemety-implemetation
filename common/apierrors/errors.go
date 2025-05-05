package apierrors

import "fmt"

// Application-specific error codes
const (
	ErrCodeUnknown           = "UNKNOWN_ERROR"
	ErrCodeNotFound          = "RESOURCE_NOT_FOUND" // Will be PRODUCT_NOT_FOUND later
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeInsufficientStock = "INSUFFICIENT_STOCK"
	ErrCodeDatabase          = "DATABASE_ERROR" // Will be INVENTORY_ACCESS_ERROR later
	// Add more codes as needed ✍️
)

// AppError defines a standard application error.
type AppError struct {
	Code    string // Application-specific error code
	Message string // User-friendly error message
	Err     error  // Original underlying error (optional)
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		// Include cause for better internal logging
		return fmt.Sprintf("AppError(Code=%s, Message=%s, Cause=%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("AppError(Code=%s, Message=%s)", e.Code, e.Message)
}

// Unwrap provides compatibility for errors.Is and errors.As.
// This allows checking the underlying error cause.
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new AppError. Use this for generating errors.
func NewAppError(code, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     cause,
	}
}
