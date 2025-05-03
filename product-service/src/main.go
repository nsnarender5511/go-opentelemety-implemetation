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
	// Use a temporary logger only for config loading errors
	tempLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tempLogger.Debug("Starting main execution")

	// --- Configuration Loading ---
	tempLogger.Debug("Loading configuration")
	cfg, err := config.LoadConfig()
	if err != nil {
		// Use tempLogger here as the main logger isn't initialized yet
		tempLogger.Error("Failed to load configuration", slog.Any("error", err))
		log.Fatalf("Failed to load configuration: %v", err) // Keep Fatalf which exits
	}
	tempLogger.Debug("Configuration loaded successfully")

	// --- Initialization (Telemetry & Logging) ---
	tempLogger.Debug("Initializing telemetry and main logger")
	logger, err := telemetry.InitTelemetry(cfg)
	if err != nil {
		// We can use the partially initialized logger from telemetry if it returns one on error,
		// or fallback to tempLogger/log.Fatal if not.
		tempLogger.Error("Failed to initialize telemetry and logging", slog.Any("error", err))
		log.Fatalf("Failed to initialize telemetry and logging: %v", err)
	}
	logger.Info("Telemetry and logging initialized successfully.")
	logger.Debug("Telemetry initialization complete")

	// --- Service and Handler Initialization (Now using the main logger) ---
	logger.Debug("Initializing Product Service")
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
	logger.Debug("Product Service initialized successfully")
	logger.Debug("Initializing Product Handler")
	handler := NewProductHandler(service)
	logger.Debug("Product Handler initialized successfully")

	// --- Service Information Logging ---
	logger.Info("Starting service")

	// --- Fiber App Setup ---
	logger.Debug("Setting up Fiber app")
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(logger),
	})
	logger.Debug("Fiber app created")

	// --- Middleware Configuration ---
	logger.Debug("Configuring middleware")
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Use(recover.New())
	app.Use(otelfiber.Middleware())
	// app.Use(middleware.RequestLogger(logger))
	logger.Debug("Middleware configured")

	// --- Route Definitions ---
	logger.Debug("Registering routes")
	app.Get("/health", func(c *fiber.Ctx) error {
		logger.DebugContext(c.UserContext(), "Minimal health check endpoint hit")
		return c.Status(http.StatusOK).JSON(fiber.Map{"status": "ok (minimal)"})
	})

	// Add routes from handler
	app.Get("/products", handler.GetAllProducts)
	app.Get("/products/:productId", handler.GetProductByID)
	app.Get("/status", handler.HealthCheck) // Handler's detailed health check
	logger.Info("All routes registered successfully")

	// --- Server Startup ---
	addr := fmt.Sprintf(":%s", cfg.ProductServicePort)
	logger.Info("Server starting to listen", slog.String("address", addr))

	if err := app.Listen(addr); err != nil {
		logger.Error("Server listener failed", slog.Any("error", err))
		os.Exit(1)
	}
}
