package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	_ "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	commonErrors "github.com/narender/common/errors"
	commonTrace "github.com/narender/common/telemetry/trace"
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
		if err == nil {
			return nil
		}

		statusCode := http.StatusInternalServerError
		userMessage := "An unexpected error occurred. Please try again later."
		internalMessage := err.Error()
		details := make(map[string]interface{})
		errorType := commonErrors.TypeUnknown
		logLevel := slog.LevelError

		var appErr *commonErrors.AppError
		var fiberErr *fiber.Error
		var validationErr *commonErrors.ValidationError
		var dbErr *commonErrors.DatabaseError

		if errors.As(err, &validationErr) {
			statusCode = validationErr.StatusCode
			userMessage = validationErr.UserMessage
			errorType = validationErr.Type
			internalMessage = validationErr.Error()
			details = validationErr.Context
		} else if errors.As(err, &dbErr) {
			statusCode = dbErr.StatusCode
			userMessage = dbErr.UserMessage
			errorType = dbErr.Type
			internalMessage = dbErr.Error()
			details = dbErr.Context
		} else if errors.As(err, &appErr) {
			if appErr.StatusCode != 0 {
				statusCode = appErr.StatusCode
			}
			if appErr.UserMessage != "" {
				userMessage = appErr.UserMessage
			}
			errorType = appErr.Type
			internalMessage = appErr.Error()
			details = appErr.Context
		} else if errors.As(err, &fiberErr) {
			statusCode = fiberErr.Code
			userMessage = fiberErr.Message
			internalMessage = fiberErr.Error()
			if statusCode == http.StatusNotFound {
				errorType = commonErrors.TypeNotFound
			} else if statusCode >= 400 && statusCode < 500 {
				errorType = commonErrors.TypeBadRequest
			}
		} else {
			if errors.Is(err, commonErrors.ErrNotFound) {
				statusCode = http.StatusNotFound
				userMessage = "Resource not found."
				errorType = commonErrors.TypeNotFound
			} else if errors.Is(err, commonErrors.ErrValidation) || errors.Is(err, commonErrors.ErrInvalidInput) || errors.Is(err, commonErrors.ErrBadRequest) {
				statusCode = http.StatusBadRequest
				userMessage = "Invalid request details provided."
				errorType = commonErrors.TypeBadRequest
			} else if errors.Is(err, commonErrors.ErrUnauthorized) {
				statusCode = http.StatusUnauthorized
				userMessage = "Authentication required."
				errorType = commonErrors.TypeUnauthorized
			} else if errors.Is(err, commonErrors.ErrForbidden) {
				statusCode = http.StatusForbidden
				userMessage = "You do not have permission to access this resource."
				errorType = commonErrors.TypeForbidden
			} else if errors.Is(err, commonErrors.ErrInternal) || errors.Is(err, commonErrors.ErrDatabase) {
				errorType = commonErrors.TypeInternal
			}
		}

		if statusCode >= 500 {
			logLevel = slog.LevelError
		} else if statusCode >= 400 {
			logLevel = slog.LevelWarn
		} else {
			logLevel = slog.LevelInfo
		}

		ctx := c.UserContext()
		span := oteltrace.SpanFromContext(ctx)
		traceID := ""
		spanID := ""
		if span != nil && span.IsRecording() {
			commonTrace.RecordSpanError(span, err)
			span.SetAttributes(
				attribute.Int("http.status_code", statusCode),
				attribute.String("error.type", errorType.String()),
				attribute.String("error.message", internalMessage),
			)
			// Extract IDs if span is valid
			if span.SpanContext().IsValid() {
				traceID = span.SpanContext().TraceID().String()
				spanID = span.SpanContext().SpanID().String()
			}
		}

		slogAttrs := []slog.Attr{
			slog.Group("error",
				slog.String("internal_message", internalMessage),
				slog.String("type", errorType.String()),
				slog.Int("status_code", statusCode),
				slog.String("user_message", userMessage),
				slog.Any("original", err),
				slog.Any("details", details),
			),
			slog.Group("request",
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.String("ip", c.IP()),
				slog.String("route", c.Route().Path),
			),
		}

		// Add trace and span IDs if available
		if traceID != "" {
			slogAttrs = append(slogAttrs, slog.String("trace_id", traceID))
		}
		if spanID != "" {
			slogAttrs = append(slogAttrs, slog.String("span_id", spanID))
		}

		logMessage := fmt.Sprintf("HTTP Error Handled: %s %s -> %d", c.Method(), c.Path(), statusCode)
		logger.LogAttrs(ctx, logLevel, logMessage, slogAttrs...)

		resp := ErrorResponse{
			StatusCode: statusCode,
			Message:    userMessage,
			Details:    details,
		}

		c.Status(statusCode)
		if jsonErr := c.JSON(resp); jsonErr != nil {
			logger.ErrorContext(ctx, "Failed to marshal error response JSON", slog.Any("json_error", jsonErr), slog.Any("original_error", err))
			return c.SendString(userMessage)
		}
		return nil
	}
}
