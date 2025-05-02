package main

import (
	"context"
	"log"
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
	"github.com/narender/common/logging"
	"github.com/narender/common/middleware"
	"github.com/narender/common/telemetry"
)

const (
	ServiceName = "product-service"
)

func main() {
	config, err := config.LoadConfig(".env", config.NewEnvironmentProvider())
	if err != nil {
		log.Fatalf("Initial config load failed: %v", err)
	}
	config.ServiceName = ServiceName

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Telemetry Setup --- Must happen before logger setup for the hook
	shutdownTelemetry, err := telemetry.InitTelemetry(ctx, config)
	if err != nil {
		// Use standard log here as our logger isn't initialized yet
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			// Use standard log or try GetLogger if it might be initialized by now
			if logger := logging.GetLogger(); logger != nil {
				logger.Errorf("Error shutting down telemetry: %v", err)
			} else {
				log.Printf("Error shutting down telemetry: %v", err) // Fallback
			}
		}
	}()

	// --- Logging Setup ---
	logger := logging.SetupLogrus(config)
	if logger == nil {
		log.Fatalf("Failed to initialize logger.") // Handle nil logger case
	}
	logger.Info("Initializing application...")

	// --- Dependency Injection ---
	repo := NewProductRepository()
	productService := NewProductService(repo)
	productHandler := NewProductHandler(productService)

	// --- Fiber App Setup ---
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(),
	})

	app.Use(recover.New())
	app.Use(cors.New())

	app.Use(otelfiber.Middleware(
		otelfiber.WithTracerProvider(telemetry.GetTracerProvider()),
		otelfiber.WithMeterProvider(telemetry.GetMeterProvider()),
		otelfiber.WithPropagators(telemetry.GetTextMapPropagator()),
	))

	app.Use(middleware.RequestLoggerMiddleware())

	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)
	v1.Get("/products/:productId", productHandler.GetProductByID)
	v1.Get("/healthz", productHandler.HealthCheck)

	// --- Server Start & Shutdown ---
	port := config.ProductServicePort
	addr := ":" + port

	go func() {
		logging.GetLogger().Infof("Server starting on %s", addr)
		if err := app.Listen(addr); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start listening: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		logger.Errorf("Error shutting down Fiber server: %v", err)
	}

	logger.Info("Application exiting gracefully.")
}
