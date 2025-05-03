package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorType defines the category of an application error.
type ErrorType int

const (
	TypeUnknown ErrorType = iota // 0
	TypeValidation
	TypeDatabase
	TypeNotFound
	TypeBadRequest
	TypeInternal
	TypeForbidden
	TypeUnauthorized
	TypeConflict // Added for consistency
	// Add other types as needed
)

func (et ErrorType) String() string {
	switch et {
	case TypeValidation:
		return "Validation Error"
	case TypeDatabase:
		return "Database Error"
	case TypeNotFound:
		return "Not Found Error"
	case TypeBadRequest:
		return "Bad Request Error"
	case TypeInternal:
		return "Internal Server Error"
	case TypeForbidden:
		return "Forbidden Error"
	case TypeUnauthorized:
		return "Unauthorized Error"
	case TypeConflict:
		return "Conflict Error"
	default:
		return "Unknown Error"
	}
}

// Standard application sentinel errors
var (
	ErrNotFound     = errors.New("resource not found")
	ErrValidation   = errors.New("validation failed") // Basic sentinel
	ErrInternal     = errors.New("internal server error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input provided")     // Added
	ErrBadRequest   = errors.New("bad request")                // Added
	ErrDatabase     = errors.New("database operation failed")  // Basic sentinel
	ErrConflict     = errors.New("resource conflict occurred") // Added
	// Add other common errors as needed
)

// AppError represents a general application error with context.
type AppError struct {
	Type        ErrorType
	StatusCode  int
	UserMessage string
	OriginalErr error // The underlying error
	Context     map[string]interface{}
}

func (e *AppError) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("%s: %v", e.Type.String(), e.OriginalErr)
	}
	return e.Type.String()
}

// Unwrap allows AppError to be used with errors.Is and errors.As
func (e *AppError) Unwrap() error {
	return e.OriginalErr
}

// NewAppError creates a new AppError.
func NewAppError(errType ErrorType, statusCode int, userMessage string, originalErr error, context map[string]interface{}) *AppError {
	return &AppError{
		Type:        errType,
		StatusCode:  statusCode,
		UserMessage: userMessage,
		OriginalErr: originalErr,
		Context:     context,
	}
}

// --- Specific Error Types (embedding AppError or standalone) ---

// ValidationError represents an error during data validation.
type ValidationError struct {
	AppError                   // Embed AppError for common fields
	Fields   map[string]string // Map of field names to validation error messages
}

func (e *ValidationError) Error() string {
	var msgs []string
	for field, msg := range e.Fields {
		msgs = append(msgs, fmt.Sprintf("'%s': %s", field, msg))
	}
	baseMsg := "Validation failed"
	if len(msgs) > 0 {
		baseMsg += ": " + strings.Join(msgs, ", ")
	}
	if e.OriginalErr != nil {
		return fmt.Sprintf("%s: %v", baseMsg, e.OriginalErr)
	}
	return baseMsg
}

// NewValidationError creates a new ValidationError.
func NewValidationError(fields map[string]string, originalErr error) *ValidationError {
	return &ValidationError{
		AppError: AppError{
			Type:        TypeValidation,
			StatusCode:  400, // Typically Bad Request
			UserMessage: "Validation failed. Please check the provided data.",
			OriginalErr: originalErr,
			Context:     map[string]interface{}{"validation_fields": fields},
		},
		Fields: fields,
	}
}

// DatabaseError represents an error during a database operation.
type DatabaseError struct {
	AppError        // Embed AppError for common fields
	Query    string // Optional: The query that failed
}

func (e *DatabaseError) Error() string {
	baseMsg := "Database operation failed"
	if e.Query != "" {
		baseMsg += fmt.Sprintf(" (query: %s)", e.Query)
	}
	if e.OriginalErr != nil {
		return fmt.Sprintf("%s: %w", baseMsg, e.OriginalErr)
	}
	return baseMsg
}

// NewDatabaseError creates a new DatabaseError.
func NewDatabaseError(query string, originalErr error) *DatabaseError {
	// Wrap the original error with the basic sentinel if not already done
	wrappedErr := originalErr
	if !errors.Is(originalErr, ErrDatabase) {
		wrappedErr = fmt.Errorf("%w: %v", ErrDatabase, originalErr)
	}

	return &DatabaseError{
		AppError: AppError{
			Type:        TypeDatabase,
			StatusCode:  500, // Typically Internal Server Error
			UserMessage: "An internal error occurred while accessing data.",
			OriginalErr: wrappedErr,
			Context:     map[string]interface{}{"db_query": query},
		},
		Query: query,
	}
}
