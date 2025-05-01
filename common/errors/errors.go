package errors

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrInvalidInput        = errors.New("invalid input")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrInternalServer      = errors.New("internal server error")
	ErrBadRequest          = errors.New("bad request")
	ErrConflict            = errors.New("conflict")
	ErrUnavailable         = errors.New("service unavailable")
	ErrTimeout             = errors.New("timeout")
	ErrTooManyRequests     = errors.New("too many requests")
	ErrUnprocessableEntity = errors.New("unprocessable entity")
)

var standardErrorToStatusCode = map[error]int{
	ErrNotFound:            http.StatusNotFound,
	ErrInvalidInput:        http.StatusBadRequest,
	ErrBadRequest:          http.StatusBadRequest,
	ErrUnauthorized:        http.StatusUnauthorized,
	ErrForbidden:           http.StatusForbidden,
	ErrConflict:            http.StatusConflict,
	ErrUnavailable:         http.StatusServiceUnavailable,
	ErrTimeout:             http.StatusGatewayTimeout,
	ErrTooManyRequests:     http.StatusTooManyRequests,
	ErrUnprocessableEntity: http.StatusUnprocessableEntity,
	// Note: ErrInternalServer is handled by the default case below
}

type AppError struct {
	Err        error
	StatusCode int
	Message    string
	Context    map[string]interface{}
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown error"
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

func New(err error, statusCode int, message string) *AppError {
	return &AppError{
		Err:        err,
		StatusCode: statusCode,
		Message:    message,
	}
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

func NotFound(message string) *AppError {
	return &AppError{
		Err:        ErrNotFound,
		StatusCode: http.StatusNotFound,
		Message:    message,
	}
}

func BadRequest(message string) *AppError {
	return &AppError{
		Err:        ErrBadRequest,
		StatusCode: http.StatusBadRequest,
		Message:    message,
	}
}

func InternalServer(err error) *AppError {
	return &AppError{
		Err:        err,
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal server error",
	}
}

func ToStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}

	// Check against the standard error map
	for stdErr, statusCode := range standardErrorToStatusCode {
		if errors.Is(err, stdErr) {
			return statusCode
		}
	}

	// Default to internal server error for unmapped errors
	return http.StatusInternalServerError
}


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

type DatabaseError struct {
	Operation string // Description of the operation (e.g., "read", "unmarshal")
	Err       error  // Underlying driver/database/file error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database error during %s: %v", e.Operation, e.Err)
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}

