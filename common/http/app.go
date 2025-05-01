package http

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/http/middleware"
	"github.com/sirupsen/logrus"
)

// AppConfig holds configuration for the Fiber app
type AppConfig struct {
	Name              string
	Logger            *logrus.Logger
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	BodyLimit         int
	EnablePrefork     bool
	DisableStartupLog bool
	MiddlewareConfig  MiddlewareConfig
}

// DefaultAppConfig returns default app configuration
func DefaultAppConfig() AppConfig {
	return AppConfig{
		Name:              "service",
		Logger:            logrus.StandardLogger(),
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		BodyLimit:         4 * 1024 * 1024, // 4MB
		EnablePrefork:     false,
		DisableStartupLog: false,
		MiddlewareConfig:  DefaultMiddlewareConfig(),
	}
}

// NewApp creates a new Fiber app with common configuration
func NewApp(config ...AppConfig) *fiber.App {
	// Use default config if none provided
	cfg := DefaultAppConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Initialize logger if not provided
	if cfg.Logger == nil {
		cfg.Logger = logrus.StandardLogger()
	}

	// Create Fiber app with configuration
	app := fiber.New(fiber.Config{
		AppName:               cfg.Name,
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		IdleTimeout:           cfg.IdleTimeout,
		BodyLimit:             cfg.BodyLimit,
		Prefork:               cfg.EnablePrefork,
		DisableStartupMessage: cfg.DisableStartupLog,
		ErrorHandler:          middleware.ErrorHandler(cfg.Logger),
	})

	// Register middleware
	cfg.MiddlewareConfig.Logger = cfg.Logger
	RegisterMiddleware(app, cfg.MiddlewareConfig)

	return app
}
