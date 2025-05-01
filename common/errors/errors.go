package errors

import (
	stdErrors "errors" // Alias standard errors package
	"fmt"
)

// Standard application errors
var (
	// Define common error types using stdErrors.New
	ErrNotFound          = stdErrors.New("resource not found")
	ErrProductNotFound   = stdErrors.New("product not found")
	ErrDatabaseOperation = stdErrors.New("database operation failed")
	ErrBadRequest        = stdErrors.New("bad request")           // Adding a basic bad request error
	ErrInternalServer    = stdErrors.New("internal server error") // Adding a basic internal error

	// Unused errors from plan have been removed:
	// ErrUserNotFound, ErrCartNotFound, ErrOrderNotFound, ErrServiceCallFailed
)

// --- Typed Errors for more context ---

// ValidationError indicates an issue with input data.
type ValidationError struct {
	Field   string // Optional field name
	Message string // Specific validation message
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed: %s", e.Message)
}

// DatabaseError indicates an issue during a database operation.
type DatabaseError struct {
	Operation string // Description of the operation (e.g., "read", "unmarshal")
	Err       error  // Underlying driver/database/file error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database error during %s: %v", e.Operation, e.Err)
}

// Unwrap allows retrieving the underlying error.
func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// --- End Typed Errors ---

// Is wraps the standard library errors.Is function.
// This allows packages using common/errors to check error types
// without needing to directly import the standard "errors" package.
func Is(err, target error) bool {
	return stdErrors.Is(err, target) // Call standard errors.Is using the alias
}

// AppError represents a custom error type for the application.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	// Original error if available
	Err error `json:"-"`
	// Uncomment and use if you need HTTP status codes associated with errors
	// HTTPStatusCode int    `json:"-"`
}

// Error returns the string representation of the AppError.
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
