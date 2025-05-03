package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RequestLogger accepts an slog.Logger and returns a Fiber handler
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

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

		if err != nil {
			attrs = append(attrs, slog.Any("error", err))
		}

		msg := "Request completed"
		if statusCode >= 500 {
			logger.LogAttrs(c.UserContext(), slog.LevelError, msg, attrs...)
		} else if statusCode >= 400 {
			logger.LogAttrs(c.UserContext(), slog.LevelWarn, msg, attrs...)
		} else {
			logger.LogAttrs(c.UserContext(), slog.LevelInfo, msg, attrs...)
		}

		return err
	}
}
