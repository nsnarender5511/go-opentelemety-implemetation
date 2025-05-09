package apierrors

import (
	"fmt"
	"time"
)

// AppError defines a standard application error.
type AppError struct {
	Code        string                 // Application-specific error code
	Message     string                 // User-friendly error message
	Err         error                  // Original underlying error (optional)
	Timestamp   time.Time              // When error occurred
	ContextData map[string]interface{} // Additional context
	Category    ErrorCategory          // Business or Application
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		// Include cause for better internal logging
		return fmt.Sprintf("AppError(Code=%s, Category=%s, Message=%s, Cause=%v)",
			e.Code, e.Category, e.Message, e.Err)
	}
	return fmt.Sprintf("AppError(Code=%s, Category=%s, Message=%s)",
		e.Code, e.Category, e.Message)
}

// Unwrap provides compatibility for errors.Is and errors.As.
// This allows checking the underlying error cause.
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithCategory sets the error category
func (e *AppError) WithCategory(category ErrorCategory) *AppError {
	e.Category = category
	return e
}

// WithContext adds context data to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.ContextData == nil {
		e.ContextData = make(map[string]interface{})
	}
	e.ContextData[key] = value
	return e
}

// NewAppError creates a new AppError with defaults
func NewAppError(code, message string, cause error) *AppError {
	// Determine category based on code prefix
	category := CategoryApplication
	for _, prefix := range []string{
		ErrCodeProductNotFound,
		ErrCodeInsufficientStock,
		ErrCodeInvalidProductData,
		ErrCodeOrderLimitExceeded,
		ErrCodePriceMismatch,
	} {
		if code == prefix {
			category = CategoryBusiness
			break
		}
	}

	return &AppError{
		Code:      code,
		Message:   message,
		Err:       cause,
		Timestamp: time.Now(),
		Category:  category,
	}
}

// NewBusinessError creates a business domain error
func NewBusinessError(code, message string, cause error) *AppError {
	return NewAppError(code, message, cause).WithCategory(CategoryBusiness)
}

// NewApplicationError creates a technical/infrastructure error
func NewApplicationError(code, message string, cause error) *AppError {
	return NewAppError(code, message, cause).WithCategory(CategoryApplication)
}
