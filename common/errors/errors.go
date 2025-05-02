package errors

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"net/http"
)

type ErrorType int

const (
	TypeNotFound ErrorType = iota + 1
	TypeInvalidInput
	TypeUnauthorized
	TypeForbidden
	TypeInternalServer
	TypeBadRequest
	TypeConflict
	TypeUnavailable
	TypeTimeout
	TypeTooManyRequests
	TypeUnprocessableEntity
	TypeDBConnection
	TypeValidation
	TypeDatabase
	TypeUnknown
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
var errorTypeToStatusCode = map[ErrorType]int{
	TypeNotFound:            http.StatusNotFound,
	TypeInvalidInput:        http.StatusBadRequest,
	TypeBadRequest:          http.StatusBadRequest,
	TypeUnauthorized:        http.StatusUnauthorized,
	TypeForbidden:           http.StatusForbidden,
	TypeConflict:            http.StatusConflict,
	TypeUnavailable:         http.StatusServiceUnavailable,
	TypeTimeout:             http.StatusGatewayTimeout,
	TypeTooManyRequests:     http.StatusTooManyRequests,
	TypeUnprocessableEntity: http.StatusUnprocessableEntity,
	TypeDBConnection:        http.StatusServiceUnavailable,
	TypeValidation:          http.StatusBadRequest,
	TypeDatabase:            http.StatusInternalServerError,
	TypeInternalServer:      http.StatusInternalServerError,
}

type AppError struct {
	Err         error                  `json:"-"`
	Type        ErrorType              `json:"-"`
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
	return "An unexpected application error occurred"
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
func New(errType ErrorType, message string) *AppError {
	statusCode := errorTypeToStatusCode[errType]
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	return &AppError{
		Type:       errType,
		StatusCode: statusCode,
		Message:    message,
	}
}
func Wrap(err error, errType ErrorType, message string) *AppError {
	if err == nil {
		return nil
	}
	statusCode := errorTypeToStatusCode[errType]
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	return &AppError{
		Err:        err,
		Type:       errType,
		StatusCode: statusCode,
		Message:    message,
	}
}
func Wrapf(err error, errType ErrorType, format string, args ...interface{}) *AppError {
	if err == nil {
		return nil
	}
	statusCode := errorTypeToStatusCode[errType]
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	return &AppError{
		Err:        err,
		Type:       errType,
		StatusCode: statusCode,
		Message:    fmt.Sprintf(format, args...),
	}
}
func (e *AppError) WithUserMessage(userMessage string) *AppError {
	e.UserMessage = userMessage
	return e
}
func NotFound(message string) *AppError {
	return New(TypeNotFound, message).WithUserMessage(message)
}
func BadRequest(message string) *AppError {
	return New(TypeBadRequest, message).WithUserMessage(message)
}
func InternalServer(err error) *AppError {

	userMsg := "An internal server error occurred. Please try again later."
	return Wrap(err, TypeInternalServer, fmt.Sprintf("Internal server error: %v", err)).WithUserMessage(userMsg)
}
func ToStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		if appErr.StatusCode != 0 {
			return appErr.StatusCode
		}

		statusCode := errorTypeToStatusCode[appErr.Type]
		if statusCode != 0 {
			return statusCode
		}
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {

		return fiberErr.Code
	}

	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return errorTypeToStatusCode[TypeValidation]
	}
	var dbErr *DatabaseError
	if errors.As(err, &dbErr) {
		return errorTypeToStatusCode[TypeDatabase]
	}

	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrBadRequest) {
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}

type ValidationError struct {
	Field   string
	Message string
	Err     error
}

func (e *ValidationError) Error() string {
	base := "validation failed"
	if e.Field != "" {
		base += fmt.Sprintf(" for field '%s'", e.Field)
	}
	if e.Message != "" {
		base += ": " + e.Message
	}
	if e.Err != nil {
		base += fmt.Sprintf(" (caused by: %v)", e.Err)
	}
	return base
}
func (e *ValidationError) Unwrap() error {
	return e.Err
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
