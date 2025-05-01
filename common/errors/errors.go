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
