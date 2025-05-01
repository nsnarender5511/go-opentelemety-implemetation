package http

import (
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/narender/common/http/middleware"
	"github.com/sirupsen/logrus"
)

// MiddlewareConfig holds configuration for middleware
type MiddlewareConfig struct {
	Logger         *logrus.Logger
	EnableOTel     bool
	EnableLogger   bool
	EnableCORS     bool
	EnableRecovery bool
	LoggingConfig  middleware.LoggingConfig
	CORSConfig     cors.Config
}

// DefaultMiddlewareConfig returns default middleware configuration
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		Logger:         logrus.StandardLogger(),
		EnableOTel:     true,
		EnableLogger:   true,
		EnableCORS:     true,
		EnableRecovery: true,
		LoggingConfig:  middleware.DefaultLoggingConfig(),
		CORSConfig:     cors.Config{},
	}
}

// RegisterMiddleware registers common middleware with a Fiber app
func RegisterMiddleware(app *fiber.App, config ...MiddlewareConfig) {
	// Use default config if none provided
	cfg := DefaultMiddlewareConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Initialize logger if not provided
	if cfg.Logger == nil {
		cfg.Logger = logrus.StandardLogger()
	}

	// Let the user know about the error handler requirement
	cfg.Logger.Warn("Remember to set the error handler during app creation: fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler()})")

	// Register OpenTelemetry middleware
	if cfg.EnableOTel {
		app.Use(otelfiber.Middleware())
	}

	// Register recovery middleware
	if cfg.EnableRecovery {
		app.Use(middleware.Recovery(cfg.Logger))
	}

	// Register CORS middleware
	if cfg.EnableCORS {
		app.Use(cors.New(cfg.CORSConfig))
	}

	// Register logger middleware
	if cfg.EnableLogger {
		// Update logger in config
		cfg.LoggingConfig.Logger = cfg.Logger
		app.Use(middleware.Logger(cfg.LoggingConfig))
	}
}
