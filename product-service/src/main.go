package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/narender/common/config"
	"github.com/narender/common/middleware"
	"github.com/narender/common/telemetry"
)

func main() {
	// --- Configuration Loading ---
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// --- Initialization (Telemetry & Logging) ---
	logger, err := telemetry.InitTelemetry(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize telemetry and logging: %v", err)
	}
	log.Println("Telemetry and logging initialized successfully.")

	// --- Service and Handler Initialization ---
	// Calculate path relative to the executable's directory or use an absolute path/config
	// Assuming main is run from project root for this relative path:
	productDataPath := filepath.Join("product-service", "data.json")
	// If running from product-service/src: productDataPath := "../data.json"
	logger.Info("Attempting to load product data", slog.String("path", productDataPath))

	service, err := NewJsonProductService(productDataPath, logger)
	if err != nil {
		logger.Error("Failed to initialize ProductService", slog.Any("error", err))
		os.Exit(1)
	}
	handler := NewProductHandler(service)

	// --- Service Information Logging ---
	logger.Info("Starting service",
		slog.String("service.name", cfg.ServiceName),
		slog.String("service.version", cfg.ServiceVersion),
		slog.String("environment", cfg.Environment),
	)

	// --- Fiber App Setup ---
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(logger),
	})

	// --- Middleware Configuration ---
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Use(recover.New())
	app.Use(otelfiber.Middleware())
	// app.Use(middleware.RequestLogger(logger))

	// --- Route Definitions ---
	app.Get("/health", func(c *fiber.Ctx) error {
		logger.Debug("Minimal health check endpoint hit")
		return c.Status(http.StatusOK).JSON(fiber.Map{"status": "ok (minimal)"})
	})

	// Add routes from handler
	app.Get("/products", handler.GetAllProducts)
	app.Get("/products/:productId", handler.GetProductByID)
	app.Get("/status", handler.HealthCheck) // Handler's detailed health check

	// --- Server Startup ---
	addr := fmt.Sprintf(":%s", cfg.ProductServicePort)
	logger.Info("Server starting to listen", slog.String("address", addr))

	if err := app.Listen(addr); err != nil {
		logger.Error("Server listener failed", slog.Any("error", err))
		os.Exit(1)
	}
}
