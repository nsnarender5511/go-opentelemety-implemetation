package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// LoggingConfig holds configuration options for the logging middleware
type LoggingConfig struct {
	Logger           *logrus.Logger
	SkipPaths        []string
	LogRequestBody   bool
	LogResponseBody  bool
	LogRequestHeader bool
}

// DefaultLoggingConfig returns the default configuration
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Logger:           logrus.StandardLogger(),
		SkipPaths:        []string{"/health", "/metrics", "/ready"},
		LogRequestBody:   false,
		LogResponseBody:  false,
		LogRequestHeader: false,
	}
}

// Logger returns a middleware that logs incoming HTTP requests
func Logger(config ...LoggingConfig) fiber.Handler {
	// Use default config if none provided
	cfg := DefaultLoggingConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Initialize logger if not provided
	if cfg.Logger == nil {
		cfg.Logger = logrus.StandardLogger()
	}

	// Create a map for faster lookups of paths to skip
	skipPaths := make(map[string]bool)
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *fiber.Ctx) error {
		// Skip logging for certain paths
		if skipPaths[c.Path()] {
			return c.Next()
		}

		// Record start time
		start := time.Now()

		// Create fields for the log entry
		fields := logrus.Fields{
			"method":     c.Method(),
			"path":       c.Path(),
			"ip":         c.IP(),
			"user_agent": c.Get("User-Agent"),
		}

		// Log request headers if enabled
		if cfg.LogRequestHeader {
			headers := make(map[string]string)
			c.Request().Header.VisitAll(func(key, value []byte) {
				headers[string(key)] = string(value)
			})
			fields["headers"] = headers
		}

		// Log request body if enabled and available
		if cfg.LogRequestBody && c.Request().Body() != nil {
			fields["request_body"] = string(c.Request().Body())
		}

		// Process request
		err := c.Next()

		// Record duration
		duration := time.Since(start)
		fields["duration_ms"] = duration.Milliseconds()
		fields["status"] = c.Response().StatusCode()
		fields["bytes_sent"] = len(c.Response().Body())

		// Log response body if enabled
		if cfg.LogResponseBody {
			fields["response_body"] = string(c.Response().Body())
		}

		// Get log level based on status code
		var logEntry *logrus.Entry
		logEntry = cfg.Logger.WithFields(fields)

		// Log based on status code and errors
		statusCode := c.Response().StatusCode()
		switch {
		case statusCode >= 500:
			logEntry.WithError(err).Error("Server error")
		case statusCode >= 400:
			logEntry.Warn("Client error")
		case statusCode >= 300:
			logEntry.Info("Redirection")
		default:
			logEntry.Info("Success")
		}

		return err
	}
}
