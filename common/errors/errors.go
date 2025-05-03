package errors

import (
	"errors"
	"fmt"
	"strings"
)


type ErrorType int

const (
	TypeUnknown ErrorType = iota 
	TypeValidation
	TypeDatabase
	TypeNotFound
	TypeBadRequest
	TypeInternal
	TypeForbidden
	TypeUnauthorized
	TypeConflict 
	
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


var (
	ErrNotFound     = errors.New("resource not found")
	ErrValidation   = errors.New("validation failed") 
	ErrInternal     = errors.New("internal server error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input provided")     
	ErrBadRequest   = errors.New("bad request")                
	ErrDatabase     = errors.New("database operation failed")  
	ErrConflict     = errors.New("resource conflict occurred") 
	
)


type AppError struct {
	Type        ErrorType
	StatusCode  int
	UserMessage string
	OriginalErr error 
	Context     map[string]interface{}
}

func (e *AppError) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("%s: %v", e.Type.String(), e.OriginalErr)
	}
	return e.Type.String()
}


func (e *AppError) Unwrap() error {
	return e.OriginalErr
}


func NewAppError(errType ErrorType, statusCode int, userMessage string, originalErr error, context map[string]interface{}) *AppError {
	return &AppError{
		Type:        errType,
		StatusCode:  statusCode,
		UserMessage: userMessage,
		OriginalErr: originalErr,
		Context:     context,
	}
}




type ValidationError struct {
	AppError                   
	Fields   map[string]string 
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


func NewValidationError(fields map[string]string, originalErr error) *ValidationError {
	return &ValidationError{
		AppError: AppError{
			Type:        TypeValidation,
			StatusCode:  400, 
			UserMessage: "Validation failed. Please check the provided data.",
			OriginalErr: originalErr,
			Context:     map[string]interface{}{"validation_fields": fields},
		},
		Fields: fields,
	}
}


type DatabaseError struct {
	AppError        
	Query    string 
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


func NewDatabaseError(query string, originalErr error) *DatabaseError {
	
	wrappedErr := originalErr
	if !errors.Is(originalErr, ErrDatabase) {
		wrappedErr = fmt.Errorf("%w: %v", ErrDatabase, originalErr)
	}

	return &DatabaseError{
		AppError: AppError{
			Type:        TypeDatabase,
			StatusCode:  500, 
			UserMessage: "An internal error occurred while accessing data.",
			OriginalErr: wrappedErr,
			Context:     map[string]interface{}{"db_query": query},
		},
		Query: query,
	}
}
