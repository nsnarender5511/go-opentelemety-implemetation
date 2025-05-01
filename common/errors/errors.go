package errors

import (
	stdErrors "errors" // Alias standard errors package
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// Standard application errors
var (
	ErrNotFound          = stdErrors.New("resource not found")                    // Use aliased import
	ErrProductNotFound   = stdErrors.New("product not found")                     // Use aliased import
	ErrUserNotFound      = stdErrors.New("user not found")                        // Example
	ErrCartNotFound      = stdErrors.New("cart not found")                        // Example
	ErrOrderNotFound     = stdErrors.New("order not found")                       // Example
	ErrDatabaseOperation = stdErrors.New("database operation failed")             // Use aliased import
	ErrServiceCallFailed = stdErrors.New("internal service communication failed") // Example
	// Add other common errors here
)

// Is wraps the standard library errors.Is function.
// This allows packages using common/errors to check error types
// without needing to directly import the standard "errors" package.
func Is(err, target error) bool {
	return stdErrors.Is(err, target) // Call standard errors.Is using the alias
}

// HandleServiceError logs and maps service layer errors to appropriate HTTP responses for Fiber.
// It centralizes error handling logic and uses logrus for logging.
func HandleServiceError(c *fiber.Ctx, err error, action string) error {
	// Log the error with context
	logrus.WithContext(c.UserContext()).WithError(err).Errorf("Failed to %s", action)

	var statusCode int
	var response fiber.Map

	// Use the local Is wrapper function (which calls stdErrors.Is)
	switch {
	case Is(err, ErrProductNotFound):
		statusCode = http.StatusNotFound
		response = fiber.Map{"error": ErrProductNotFound.Error()}
	case Is(err, ErrUserNotFound): // Example mapping
		statusCode = http.StatusNotFound
		response = fiber.Map{"error": ErrUserNotFound.Error()}
	case Is(err, ErrCartNotFound): // Example mapping
		statusCode = http.StatusNotFound
		response = fiber.Map{"error": ErrCartNotFound.Error()}
	case Is(err, ErrOrderNotFound): // Example mapping
		statusCode = http.StatusNotFound
		response = fiber.Map{"error": ErrOrderNotFound.Error()}
	case Is(err, ErrDatabaseOperation):
		statusCode = http.StatusInternalServerError
		response = fiber.Map{"error": fmt.Sprintf("Failed to %s due to internal database error", action)}
	case Is(err, ErrServiceCallFailed):
		statusCode = http.StatusInternalServerError
		response = fiber.Map{"error": fmt.Sprintf("Failed to %s due to an internal service communication error", action)}
	default:
		// Check if the error wraps the generic ErrNotFound as a fallback
		if Is(err, ErrNotFound) {
			statusCode = http.StatusNotFound
			response = fiber.Map{"error": ErrNotFound.Error()} // Use the generic message
		} else {
			// Default internal server error for unmapped errors
			statusCode = http.StatusInternalServerError
			response = fiber.Map{"error": fmt.Sprintf("Failed to %s due to an unexpected internal error", action)}
		}
	}

	return c.Status(statusCode).JSON(response)
}
