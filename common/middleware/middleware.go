package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/globals"

	// Import common packages
	apierrors "github.com/narender/common/apierrors"
	apiresponses "github.com/narender/common/apiresponses"
)

// ErrorHandler creates a Fiber error handler middleware.
func ErrorHandler() fiber.ErrorHandler {
	logger := globals.Logger()

	return func(c *fiber.Ctx, err error) error {
		var appErr *apierrors.AppError                                               // Use imported type
		var statusCode int = http.StatusInternalServerError                          // Default to 500
		var errCode string = apierrors.ErrCodeUnknown                                // Default code
		var message string = "An unexpected error occurred. Please try again later." // Generic default message

		if errors.As(err, &appErr) {
			// Handle our custom AppError
			errCode = appErr.Code
			message = appErr.Message // Use the specific message from AppError

			// Map AppError Code to HTTP Status Code
			switch appErr.Code {
			case apierrors.ErrCodeNotFound:
				statusCode = http.StatusNotFound // 404
			case apierrors.ErrCodeValidation, apierrors.ErrCodeInsufficientStock:
				statusCode = http.StatusBadRequest // 400
			case apierrors.ErrCodeDatabase: // Will become INVENTORY_ACCESS_ERROR
				statusCode = http.StatusInternalServerError // 500
			// Add mappings for other codes if needed
			default:
				statusCode = http.StatusInternalServerError
			}
			// Use the passed-in logger - Refined log message
			logger.ErrorContext(c.UserContext(), "API Error Handled",
				slog.String("msg", message),
				slog.Any("cause", appErr.Unwrap()),
			)
		} else {
			// Use the passed-in logger - Refined log message
			logger.ErrorContext(c.UserContext(), "API Unhandled Error",
				slog.String("type", fmt.Sprintf("%T", err)),
				slog.String("error", err.Error()),

			)
			message = "An internal server error occurred."
		}

		// Send standardized JSON error response using the structure from responses.go
		c.Status(statusCode)
		return c.JSON(apiresponses.ErrorResponse{
			Status: "error",
			Error: apiresponses.ErrorDetail{
				Code:    errCode,
				Message: message,
			},
		})
	}
}
