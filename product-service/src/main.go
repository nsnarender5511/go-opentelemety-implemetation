package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/narender/common/config"
	commonlog "github.com/narender/common/log"
	"github.com/narender/common/middleware"
	"github.com/narender/common/telemetry"
)

var appConfig *config.Config

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	appConfig = cfg

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 15*time.Second)
	otelShutdown, err := telemetry.InitTelemetry(startupCtx, cfg)
	cancelStartup()
	if err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}
	log.Println("OpenTelemetry initialized successfully.")

	if err := commonlog.Init(cfg); err != nil {
		log.Fatalf("Failed to initialize application logger: %v", err)
	}
	defer commonlog.Cleanup()

	logger := commonlog.L

	logger.Info("Starting service",
		slog.String("service.name", cfg.ServiceName),
		slog.String("service.version", cfg.ServiceVersion),
		slog.String("environment", cfg.Environment),
	)

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(logger),
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Use(recover.New())
	app.Use(otelfiber.Middleware())
	app.Use(middleware.RequestLogger(logger))

	app.Get("/healthz", func(c *fiber.Ctx) error {
		logger.Debug("Minimal health check endpoint hit")
		return c.Status(http.StatusOK).JSON(fiber.Map{"status": "ok (minimal)"})
	})

	addr := fmt.Sprintf(":%s", cfg.ProductServicePort)
	logger.Info("Server starting to listen", slog.String("address", addr))

	go func() {
		if err := app.Listen(addr); err != nil && err != http.ErrServerClosed {
			logger.Error("Fiber listener failed", slog.Any("error", err))
			os.Exit(1) // Exit if the listener fails
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutdown signal received, initiating graceful server shutdown...")
	serverShutdownCtx, serverCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer serverCancel()

	if err := app.ShutdownWithContext(serverShutdownCtx); err != nil {
		logger.Error("Fiber server graceful shutdown failed", slog.Any("error", err))
	} else {
		logger.Info("Fiber server shutdown complete.")
	}

	// Shutdown OpenTelemetry
	shutdownCtx, otelCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer otelCancel()

	if err := otelShutdown(shutdownCtx); err != nil {
		logger.Error("Error during OpenTelemetry shutdown", slog.Any("error", err))
	} else {
		logger.Info("OpenTelemetry shutdown complete.")
	}

	logger.Info("Application exiting.")
}
