package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/errors"
	"github.com/sirupsen/logrus"
)

// ErrorResponse is the standard error response structure
type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// ErrorHandler returns a middleware that handles errors
func ErrorHandler(logger *logrus.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// Get the status code
		statusCode := errors.ToStatusCode(err)

		// Create response
		response := ErrorResponse{
			Status:  statusCode,
			Message: err.Error(),
		}

		// Add additional details for app errors
		var appErr *errors.AppError
		if errors.As(err, &appErr) {
			// Include context in the log but not in the response
			logFields := logrus.Fields{
				"status_code": statusCode,
				"path":        c.Path(),
				"method":      c.Method(),
			}

			// Add context to log fields
			if appErr.Context != nil {
				for k, v := range appErr.Context {
					logFields[k] = v
				}
			}

			// Log error with context
			logger.WithFields(logFields).WithError(appErr.Err).Error(appErr.Message)

			// Set error name if available but not in production
			if appErr.Err != nil && c.App().Config().AppName != "production" {
				response.Error = appErr.Err.Error()
			}
		} else {
			// Log standard error
			logger.WithFields(logrus.Fields{
				"status_code": statusCode,
				"path":        c.Path(),
				"method":      c.Method(),
			}).WithError(err).Error("Request error")
		}

		// Return JSON response
		return c.Status(statusCode).JSON(response)
	}
}

// DefaultErrorMapper maps errors to HTTP responses for Fiber's ErrorHandler
func DefaultErrorMapper(err error, c *fiber.Ctx) error {
	// Create response
	errorHandler := ErrorHandler(logrus.StandardLogger())
	return errorHandler(c, err)
}
