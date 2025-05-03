package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	_ "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	commonErrors "github.com/narender/common/errors"
	"github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type ErrorResponse struct {
	StatusCode int                    `json:"statusCode"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

func ErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		statusCode := http.StatusInternalServerError
		userMessage := "An unexpected error occurred. Please try again later."
		internalMessage := err.Error()
		var details map[string]interface{}
		errorType := commonErrors.TypeUnknown
		logLevel := slog.LevelError
		var appErr *commonErrors.AppError
		var fiberErr *fiber.Error
		var validationErr *commonErrors.ValidationError
		var dbErr *commonErrors.DatabaseError
		if errors.As(err, &appErr) {

			if appErr.StatusCode != 0 {
				statusCode = appErr.StatusCode
			}
			if appErr.UserMessage != "" {
				userMessage = appErr.UserMessage
			}
			internalMessage = appErr.Error()
			details = appErr.Context
			errorType = appErr.Type
		} else if errors.As(err, &fiberErr) {

			statusCode = fiberErr.Code
			userMessage = fiberErr.Message
			internalMessage = fiberErr.Error()
		} else if errors.As(err, &validationErr) {

			statusCode = http.StatusBadRequest
			userMessage = validationErr.Error()
			internalMessage = validationErr.Error()
			errorType = commonErrors.TypeValidation
		} else if errors.As(err, &dbErr) {

			statusCode = http.StatusInternalServerError
			userMessage = "A database error occurred."
			internalMessage = dbErr.Error()
			errorType = commonErrors.TypeDatabase
		} else {

			if errors.Is(err, commonErrors.ErrNotFound) {
				statusCode = http.StatusNotFound
				userMessage = "Resource not found."
				errorType = commonErrors.TypeNotFound
			} else if errors.Is(err, commonErrors.ErrInvalidInput) || errors.Is(err, commonErrors.ErrBadRequest) {
				statusCode = http.StatusBadRequest
				userMessage = "Invalid request."
				errorType = commonErrors.TypeBadRequest
			}
		}

		if statusCode >= 400 && statusCode < 500 {
			logLevel = slog.LevelWarn
		} else if statusCode >= 500 {
			logLevel = slog.LevelError
		} else {
			logLevel = slog.LevelInfo
		}

		ctx := c.UserContext()
		span := oteltrace.SpanFromContext(ctx)
		if span != nil && span.IsRecording() {
			trace.RecordSpanError(span, err)
			span.SetAttributes(
				attribute.String("error.message", internalMessage),
				attribute.Int("error.type", int(errorType)),
			)
		}

		slogAttrs := []slog.Attr{
			slog.Any("original_error", err),
			slog.String("internal_message", internalMessage),
			slog.Int("error_type", int(errorType)),
			slog.Int("status_code", statusCode),
			slog.String("user_message", userMessage),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.String("ip", c.IP()),
			slog.String("route", c.Route().Path),
			slog.Any("request_details", details),
		}

		logMessage := fmt.Sprintf("HTTP Error: %s %s -> %d", c.Method(), c.Path(), statusCode)

		logger.LogAttrs(ctx, logLevel, logMessage, slogAttrs...)

		resp := ErrorResponse{
			StatusCode: statusCode,
			Message:    userMessage,
			Details:    details,
		}
		c.Status(statusCode)
		return c.JSON(resp)
	}
}
