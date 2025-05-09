package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/globals"

	// Import common packages
	apierrors "github.com/narender/common/apierrors"
	apiresponses "github.com/narender/common/apiresponses"
)

// RecoverMiddleware handles panics gracefully
func RecoverMiddleware() fiber.Handler {
	logger := globals.Logger()

	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("panic: %v", r)
				}

				stack := string(debug.Stack())

				logger.ErrorContext(c.UserContext(), "CRITICAL: Unhandled panic recovered",
					slog.String("error", err.Error()),
					slog.String("stack", stack),
					slog.String("path", c.Path()),
					slog.String("method", c.Method()),
				)

				appErr := apierrors.NewApplicationError(
					apierrors.ErrCodeSystemPanic,
					"A critical system error occurred. Our team has been notified.",
					err)

				// Handle through the normal error handler
				_ = ErrorHandler()(c, appErr)
			}
		}()
		return c.Next()
	}
}

// ErrorHandler creates a Fiber error handler middleware.
func ErrorHandler() fiber.ErrorHandler {
	logger := globals.Logger()

	return func(c *fiber.Ctx, err error) error {
		var appErr *apierrors.AppError
		var statusCode int = http.StatusInternalServerError
		var errCode string = apierrors.ErrCodeUnknown
		var message string = "An unexpected error occurred. Please try again later."

		if errors.As(err, &appErr) {
			// Handle our custom AppError
			errCode = appErr.Code
			message = appErr.Message

			// Map AppError Code to HTTP Status Code based on category and code
			if appErr.Category == apierrors.CategoryBusiness {
				switch appErr.Code {
				case apierrors.ErrCodeProductNotFound:
					statusCode = http.StatusNotFound
				case apierrors.ErrCodeInsufficientStock,
					apierrors.ErrCodeInvalidProductData,
					apierrors.ErrCodeOrderLimitExceeded,
					apierrors.ErrCodePriceMismatch:
					statusCode = http.StatusBadRequest
				default:
					statusCode = http.StatusBadRequest
				}
			} else {
				// Application category
				switch appErr.Code {
				case apierrors.ErrCodeDatabaseAccess,
					apierrors.ErrCodeInternalProcessing,
					apierrors.ErrCodeSystemPanic:
					statusCode = http.StatusInternalServerError
				case apierrors.ErrCodeServiceUnavailable,
					apierrors.ErrCodeNetworkError:
					statusCode = http.StatusServiceUnavailable
				case apierrors.ErrCodeRequestValidation,
					apierrors.ErrCodeMalformedData:
					statusCode = http.StatusBadRequest
				case apierrors.ErrCodeResourceConstraint:
					statusCode = http.StatusTooManyRequests
				case apierrors.ErrCodeRequestTimeout:
					statusCode = http.StatusRequestTimeout
				default:
					statusCode = http.StatusInternalServerError
				}
			}

			// Log with appropriate level based on category and status code
			if appErr.Category == apierrors.CategoryBusiness && statusCode < 500 {
				logger.WarnContext(c.UserContext(), "Business rule violation",
					slog.String("error_code", appErr.Code),
					slog.String("message", appErr.Message),
					slog.String("path", c.Path()),
				)
			} else {
				logger.ErrorContext(c.UserContext(), "Error occurred",
					slog.String("error_code", appErr.Code),
					slog.String("category", string(appErr.Category)),
					slog.String("message", appErr.Message),
					slog.Any("cause", appErr.Unwrap()),
					slog.String("path", c.Path()),
				)
			}
		} else {
			// Handle unexpected errors with better classification
			var netErr net.Error
			var jsonErr *json.SyntaxError

			switch {
			case errors.As(err, &netErr):
				errCode = apierrors.ErrCodeNetworkError
				statusCode = http.StatusServiceUnavailable
				message = "Network connectivity issue occurred"

			case errors.As(err, &jsonErr):
				errCode = apierrors.ErrCodeMalformedData
				statusCode = http.StatusBadRequest
				message = "Invalid data format in request"

			case errors.Is(err, context.DeadlineExceeded):
				errCode = apierrors.ErrCodeRequestTimeout
				statusCode = http.StatusRequestTimeout
				message = "Request processing timed out"

			case errors.Is(err, context.Canceled):
				errCode = apierrors.ErrCodeRequestTimeout
				statusCode = http.StatusRequestTimeout
				message = "Request was canceled"

			default:
				errCode = apierrors.ErrCodeUnknown
				statusCode = http.StatusInternalServerError
				message = "An unexpected error occurred"
			}

			logger.ErrorContext(c.UserContext(), "Unhandled error",
				slog.String("error_type", fmt.Sprintf("%T", err)),
				slog.String("error", err.Error()),
				slog.String("error_code", errCode),
				slog.String("path", c.Path()),
			)
		}

		// Send standardized JSON error response
		c.Status(statusCode)
		return c.JSON(apiresponses.ErrorResponse{
			Status: "error",
			Error: apiresponses.ErrorDetail{
				Code:      errCode,
				Message:   message,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			},
		})
	}
}
