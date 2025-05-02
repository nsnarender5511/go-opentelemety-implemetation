package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// RequestLoggerMiddleware logs the start and end of each HTTP request.
func RequestLoggerMiddleware(logger *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

		// Log start (optional, can be noisy)
		// logger.WithFields(logrus.Fields{
		// 	"method": method,
		// 	"path":   path,
		// 	"ip":     c.IP(),
		// }).Info("Request started")

		// Process request
		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		// Extract TraceID/SpanID from context if available
		span := oteltrace.SpanFromContext(c.UserContext())
		traceID := ""
		spanID := ""
		if span != nil && span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
			spanID = span.SpanContext().SpanID().String()
		}

		entry := logger.WithFields(logrus.Fields{
			"method":      method,
			"path":        path,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
			"ip":          c.IP(),
			"trace_id":    traceID,
			"span_id":     spanID,
		})

		if err != nil {
			// Error was handled by the ErrorHandler, but we log it here too
			// The ErrorHandler should have already logged the full error details
			entry = entry.WithError(err) // Add error context if available
			// Note: Status code might already be set by ErrorHandler
			// We log based on the final status code set on the response
		}

		// Log based on status code
		if statusCode >= 500 {
			entry.Error("Request completed with server error")
		} else if statusCode >= 400 {
			entry.Warn("Request completed with client error")
		} else {
			entry.Info("Request completed successfully")
		}

		return err // Return the error for Fiber to handle (it might already be handled)
	}
}
