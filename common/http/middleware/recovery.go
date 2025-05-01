package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/errors"
	"github.com/sirupsen/logrus"
)

// Recovery returns a middleware that recovers from panics and returns error 500
func Recovery(logger *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic with stack trace
				stackTrace := string(debug.Stack())
				logger.WithFields(logrus.Fields{
					"panic_value": fmt.Sprintf("%v", r),
					"stack_trace": stackTrace,
					"path":        c.Path(),
					"method":      c.Method(),
					"ip":          c.IP(),
				}).Error("Recovered from panic")

				// Convert to error
				var err error
				switch x := r.(type) {
				case string:
					err = fmt.Errorf("server error: %s", x)
				case error:
					err = x
				default:
					err = fmt.Errorf("unknown panic: %v", r)
				}

				// Create app error
				appErr := errors.InternalServer(err)

				// Send error response
				errResp := ErrorResponse{
					Status:  appErr.StatusCode,
					Message: "Internal Server Error",
				}

				// Show detailed error in non-production environments
				if c.App().Config().AppName != "production" {
					errResp.Error = appErr.Error()
				}

				// Send error response and abort chain
				_ = c.Status(appErr.StatusCode).JSON(errResp)
				_ = c.App().Config().ErrorHandler(c, appErr)
			}
		}()

		return c.Next()
	}
}
