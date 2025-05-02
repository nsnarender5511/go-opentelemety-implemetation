package main

import (
	"context"
	"log" // Use standard log for critical bootstrap errors
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	// Assuming these common packages exist and are correct
	"github.com/narender/common/config"
	"github.com/narender/common/logging"
	"github.com/narender/common/middleware"
	"github.com/narender/common/telemetry"

	"go.opentelemetry.io/otel"
	// Needed for GetTextMapPropagator
	// Needed for logger type in DI
	// TODO: Add import for metric.HTTPMetrics if needed for error handler
	// "github.com/narender/common/telemetry/metric"
)

const (
	ServiceName = "product-service" // Define service name here
)

func main() {
	// --- Hardcoded Configuration ---
	cfg := config.GetHardcodedConfig()
	cfg.ServiceName = ServiceName // Override service name if needed

	// --- Telemetry Setup ---
	setupCtx := context.Background()
	shutdownTelemetry, err := telemetry.InitTelemetry(setupCtx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			otelLogger := logging.GetLogger()
			if otelLogger != nil {
				otelLogger.Errorf("Error shutting down telemetry: %v", err)
			} else {
				log.Printf("Error shutting down telemetry: %v", err)
			}
		}
	}()

	// --- Logging Setup ---
	logger := logging.SetupLogrus(cfg)
	if logger == nil {
		log.Fatalf("Failed to initialize logger.")
	}
	logger.Info("Logger initialized.")

	// --- Application Context ---
	appCtx, cancelApp := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelApp()

	logger.Info("Initializing application dependencies...")
	// --- Dependency Injection ---
	// Repository only takes path and returns an error
	repo, err := NewProductRepository(cfg.DataFilePath)
	if err != nil {
		// Log fatal here as repo is critical
		logger.Fatalf("Failed to initialize product repository: %v", err)
	}
	// Service only takes repo
	productService := NewProductService(repo)
	// Handler only takes service
	productHandler := NewProductHandler(productService)

	logger.Info("Setting up Fiber application...")
	// --- Fiber App Setup ---
	// TODO: Create/pass a valid *metric.HTTPMetrics instance if required by error handler
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(logger, nil), // Passing nil for metrics for now
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	})

	// --- Fiber Middleware ---
	app.Use(recover.New())
	app.Use(cors.New())

	// OTel Middleware - uses global providers set by telemetry.InitTelemetry
	propagator := otel.GetTextMapPropagator() // Get global propagator

	app.Use(otelfiber.Middleware(
		// Default options use global providers for Tracer and Meter
		otelfiber.WithPropagators(propagator),
	))
	logger.Info("OpenTelemetry middleware configured for Fiber.")

	// Custom Request Logger Middleware
	// TODO: Ensure RequestLoggerMiddleware takes logger - If not, remove logger arg
	app.Use(middleware.RequestLoggerMiddleware(logger)) // Pass logger if needed

	// --- API Routes ---
	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)
	v1.Get("/products/:productId", productHandler.GetProductByID)
	v1.Get("/healthz", productHandler.HealthCheck)
	logger.Info("API routes configured.")

	// --- Server Start ---
	port := cfg.ProductServicePort
	addr := ":" + port
	go func() {
		logger.Infof("Server starting on %s", addr)
		if err := app.Listen(addr); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start listening: %v", err)
		}
	}()

	// --- Graceful Shutdown ---
	<-appCtx.Done()
	logger.Info("Shutdown signal received, initiating graceful shutdown...")

	shutdownTimeout := cfg.ServerShutdownTimeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	logger.Infof("Attempting to shut down Fiber server within %v...", shutdownTimeout)
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Errorf("Error during Fiber server shutdown: %v", err)
	} else {
		logger.Info("Fiber server shut down successfully.")
	}

	logger.Info("Application exiting gracefully.")
}

// Removed all placeholder declarations as they exist in other files
