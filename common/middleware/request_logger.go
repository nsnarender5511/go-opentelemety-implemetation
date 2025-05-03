package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
)

func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

		// Get trace context before calling Next()
		ctx := c.UserContext()
		span := trace.SpanFromContext(ctx)
		traceID := ""
		spanID := ""
		if span != nil && span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
			spanID = span.SpanContext().SpanID().String()
		}

		err := c.Next()

		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		attrs := []slog.Attr{
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status_code", statusCode),
			slog.Duration("duration", duration),
			slog.String("ip", c.IP()),
			slog.String("user_agent", string(c.Request().Header.UserAgent())),
		}

		// Add trace and span IDs if available
		if traceID != "" {
			attrs = append(attrs, slog.String("trace_id", traceID))
		}
		if spanID != "" {
			attrs = append(attrs, slog.String("span_id", spanID))
		}

		if err != nil {
			// Ensure error attribute is added *before* logging level decision
			attrs = append(attrs, slog.Any("error", err))
		}

		msg := "Request completed"
		if statusCode >= 500 {
			logger.LogAttrs(ctx, slog.LevelError, msg, attrs...) // Use ctx from UserContext
		} else if statusCode >= 400 {
			logger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...) // Use ctx from UserContext
		} else {
			logger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...) // Use ctx from UserContext
		}

		return err // Return the original error
	}
}
