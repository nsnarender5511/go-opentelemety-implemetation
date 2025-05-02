package middleware

import (
	"errors"
	"fmt"
	"time"

	_ "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	commonErrors "github.com/narender/common/errors"
	"github.com/narender/common/otel"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ErrorResponse defines the standard JSON error response body.
type ErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	// Optionally add RequestID, ErrorCode, etc.
	// RequestID string `json:"requestId,omitempty"`
}

// NewErrorHandler creates a Fiber error handler that logs the error,
// records metrics/traces, and returns a standardized JSON response.
func NewErrorHandler(logger *logrus.Logger, metrics *otel.Metrics) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Get status code from the error
		statusCode := commonErrors.ToStatusCode(err)

		// Prepare standard response
		resp := ErrorResponse{
			StatusCode: statusCode,
			Message:    "An unexpected error occurred. Please try again later.", // Default message
		}

		// Extract user-friendly message if available from AppError
		var appErr *commonErrors.AppError
		if errors.As(err, &appErr) {
			if appErr.UserMessage != "" {
				resp.Message = appErr.UserMessage
			} else if appErr.Message != "" {
				// Fallback to internal message if user message is empty but internal isn't
				// Potentially sanitize this before sending to client if sensitive
				resp.Message = appErr.Message
			}
		} else {
			// For non-AppError types, use the status text or a generic message
			// Keep the generic message for non-AppErrors for simplicity
			// statusText := fiber.ErrInternalServerError.Message // Removed unused variable
		}

		// --- Telemetry ---
		// 1. Record error on the span
		span := oteltrace.SpanFromContext(c.UserContext())
		if span != nil && span.IsRecording() {
			otel.RecordSpanError(span, err, otel.AttrHTTPResponseStatusCodeKey.Int(statusCode))
		}

		// 2. Record metrics (adjust attributes as needed)
		if metrics != nil {
			// Extract attributes from context set by otelfiber middleware
			attrs := []attribute.KeyValue{
				otel.AttrHTTPRequestMethod.String(c.Method()),
				otel.AttrHTTPRouteKey.String(c.Route().Path), // Use Route().Path for the template
				otel.AttrHTTPResponseStatus.Int(statusCode),
				// Potentially add otel.AttrNetHostName, otel.AttrNetHostPort if relevant
			}

			// Estimate duration if possible, otherwise record count with error status
			// otelfiber might store start time in UserCtx? Check its implementation.
			// Assuming we can't easily get duration here, just update counter
			metrics.RecordHTTPRequestDuration(c.UserContext(), 0*time.Second /* duration unknown here */, attrs...)
			// Alternatively, have separate counter for errors? Or rely on status code attribute.
		}

		// --- Logging ---
		entry := logger.WithFields(logrus.Fields{
			"error":       err.Error(), // Log the actual error message
			"status_code": statusCode,
			"method":      c.Method(),
			"path":        c.Path(),
			"ip":          c.IP(),
			"route":       c.Route().Path,
			// Add TraceID/SpanID if possible
		})
		if span != nil && span.SpanContext().IsValid() {
			entry = entry.WithFields(logrus.Fields{
				"trace_id": span.SpanContext().TraceID().String(),
				"span_id":  span.SpanContext().SpanID().String(),
			})
		}

		if statusCode >= 500 {
			// Log the actual error message for server errors
			entry.Error(fmt.Sprintf("Server error occurred: %s", err.Error()))
		} else {
			// Log the actual error message for client errors
			entry.Warn(fmt.Sprintf("Client error occurred: %s", err.Error()))
		}

		// Set header and send response
		c.Status(statusCode)
		return c.JSON(resp)
	}
}
