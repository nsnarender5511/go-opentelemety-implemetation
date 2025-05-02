package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func RequestLoggerMiddleware(logger *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()


		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

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
			entry = entry.WithError(err)
		}

		if statusCode >= 500 {
			entry.Error("Request completed with server error")
		} else if statusCode >= 400 {
			entry.Warn("Request completed with client error")
		} else {
			entry.Info("Request completed successfully")
		}

		return err
	}
}
