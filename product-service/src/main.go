package main

import (
	"fmt"
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
		fmt.Println("Failed to load configuration: %v", err)
		panic(err)
	}

	// --- Initialization (Telemetry & Logging) ---
	logger, err := telemetry.InitTelemetry(cfg)
	if err != nil {
		fmt.Println("Failed to initialize telemetry and logging: %v", err)
		panic(err)
	}

	// --- Service and Handler Initialization ---
	logger.Debug("Initializing Product Repository and Service")
	productDataPath := filepath.Join("product-service", "data.json")
	logger.Info("Attempting to load product data for repository", slog.String("path", productDataPath))

	repo, err := NewProductRepository(productDataPath)
	if err != nil {
		logger.Error("Failed to initialize ProductRepository", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Debug("Product Repository initialized successfully")

	service := NewProductService(repo)
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
	logger.Debug("Middleware configured")

	// --- Route Definitions ---
	logger.Debug("Registering routes")
	app.Get("/health", func(c *fiber.Ctx) error {
		logger.DebugContext(c.UserContext(), "Minimal health check endpoint hit")
		return c.Status(http.StatusOK).JSON(fiber.Map{"status": "ok (minimal)"})
	})

	app.Get("/products", handler.GetAllProducts)
	app.Get("/products/:productId", handler.GetProductByID)
	app.Put("/products/:productId/stock", handler.UpdateStock)
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
