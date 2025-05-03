package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/narender/common/config"
	"github.com/narender/common/middleware"
	"github.com/narender/common/telemetry"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logger, err := telemetry.InitTelemetry(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize telemetry and logging: %v", err)
	}
	log.Println("Telemetry and logging initialized successfully.")

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

	if err := app.Listen(addr); err != nil {
		logger.Error("Server listener failed", slog.Any("error", err))
		os.Exit(1)
	}
}
