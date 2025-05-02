package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
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
	ErrDBConnection        = errors.New("database connection error")
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
	ErrDBConnection:        http.StatusServiceUnavailable,
	// Note: ErrInternalServer is handled by the default case below
}

type AppError struct {
	Err         error                  `json:"-"`
	StatusCode  int                    `json:"statusCode"`
	Message     string                 `json:"message"`
	UserMessage string                 `json:"userMessage,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.UserMessage != "" {
		return e.UserMessage
	}
	return "An unexpected error occurred"
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

func Wrap(err error, statusCode int, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Err:        err,
		StatusCode: statusCode,
		Message:    message,
	}
}

func Wrapf(err error, statusCode int, format string, args ...interface{}) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Err:        err,
		StatusCode: statusCode,
		Message:    fmt.Sprintf(format, args...),
	}
}

func (e *AppError) WithUserMessage(userMessage string) *AppError {
	e.UserMessage = userMessage
	return e
}

func NotFound(message string) *AppError {
	return &AppError{
		Err:         ErrNotFound,
		StatusCode:  http.StatusNotFound,
		Message:     message,
		UserMessage: message,
	}
}

func BadRequest(message string) *AppError {
	return &AppError{
		Err:         ErrBadRequest,
		StatusCode:  http.StatusBadRequest,
		Message:     message,
		UserMessage: message,
	}
}

func InternalServer(err error) *AppError {
	return &AppError{
		Err:         err,
		StatusCode:  http.StatusInternalServerError,
		Message:     fmt.Sprintf("Internal server error: %v", err),
		UserMessage: "An internal server error occurred. Please try again later.",
	}
}

func ToStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// --- Check for fiber.Error first (specifically NotFound) ---
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		if fiberErr.Code == http.StatusNotFound {
			return http.StatusNotFound
		}
		// Potentially handle other fiber errors here if needed
		// return fiberErr.Code // Or map specific fiber codes
	}
	// --- End fiber error check ---

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}

	for stdErr, statusCode := range standardErrorToStatusCode {
		if errors.Is(err, stdErr) {
			return statusCode
		}
	}

	// Default to 500 for unhandled/unmapped errors
	return http.StatusInternalServerError
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed: %s", e.Message)
}

type DatabaseError struct {
	Operation string
	Err       error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database error during %s operation: %v", e.Operation, e.Err)
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}
