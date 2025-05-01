package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Updated common imports to use new module path
	config "github.com/narender/common-module/config"
	commonErrors "github.com/narender/common-module/errors"
	"github.com/narender/common-module/telemetry"

	// Removed handler import as handler.go is now package main
	// "product-service/src/handler"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sirupsen/logrus"
	codes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Custom validation error sentinel
// var ErrValidation = fmt.Errorf("input validation failed")

func main() {
	// --- Initialize OpenTelemetry ---
	otelShutdown, err := telemetry.InitTelemetry()
	if err != nil {
		// Use standard log before Logrus is configured if OTel fails early
		logrus.WithError(err).Fatal("Failed to initialize OpenTelemetry")
	}
	// Defer the shutdown function to be called when main exits
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Allow 10 seconds for graceful shutdown
		defer cancel()
		if err := otelShutdown(ctx); err != nil {
			logrus.WithError(err).Error("Failed to shutdown OpenTelemetry gracefully")
		} else {
			logrus.Info("OpenTelemetry shutdown complete")
		}
	}()

	// --- Configure Logrus AFTER OTel Init ---
	// Set log level from config
	if level, err := logrus.ParseLevel(config.LOG_LEVEL); err == nil {
		logrus.SetLevel(level)
	} else {
		logrus.Warnf("Invalid log level '%s', using default 'info'", config.LOG_LEVEL)
		logrus.SetLevel(logrus.InfoLevel)
	}
	// Set formatter (e.g., JSON or Text)
	if config.LOG_FORMAT == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, TimestampFormat: time.RFC3339Nano})
	}
	logrus.SetOutput(os.Stdout) // Ensure logs go to stdout
	// Optional: logrus.SetReportCaller(true) // Adds overhead

	logrus.Info("Starting Product Service", "port", config.PRODUCT_SERVICE_PORT) // Use Logrus

	// --- Initialize Tracer and Meter --- Removed explicit Get global instances
	// tracer := otel.Tracer("product-service/main") // No longer needed here
	// meter := otel.Meter("product-service/main")   // No longer needed here

	// --- Inject Dependencies --- Removed passing tracer/meter
	productRepo, err := NewProductRepository() // Removed tracer
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize ProductRepository")
	}
	productService := NewProductService(productRepo)    // Removed tracer
	productHandler := NewProductHandler(productService) // Removed tracer and meter

	app := fiber.New(fiber.Config{
		AppName: fmt.Sprintf("%s v%s", config.SERVICE_NAME, config.SERVICE_VERSION),
		// --- Add Fiber Error Handler for OTel Integration ---
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			ctx := c.UserContext()
			span := trace.SpanFromContext(ctx)
			logEntry := logrus.WithContext(ctx).WithError(err)

			code := http.StatusInternalServerError                           // Default
			httpErrMessage := "An unexpected internal server error occurred" // Default

			var validationErr *commonErrors.ValidationError
			var dbErr *commonErrors.DatabaseError

			if errors.As(err, &validationErr) {
				code = http.StatusBadRequest
				httpErrMessage = validationErr.Error()
				span.SetStatus(codes.Error, httpErrMessage)
				logEntry.Warnf("Validation error: %v", err)
			} else if errors.Is(err, commonErrors.ErrProductNotFound) {
				code = http.StatusNotFound
				httpErrMessage = commonErrors.ErrProductNotFound.Error()
				span.SetStatus(codes.Error, httpErrMessage)
				logEntry.Warnf("Resource not found: %v", err)
			} else if errors.As(err, &dbErr) {
				code = http.StatusInternalServerError
				httpErrMessage = "An internal database error occurred" // Generic user message
				// Log the specific internal error details
				logEntry.Errorf("Database error during operation %s: %+v", dbErr.Operation, dbErr.Err)
				span.SetStatus(codes.Error, httpErrMessage)
			} else {
				// Default case for unhandled errors
				code = http.StatusInternalServerError
				httpErrMessage = "An unexpected internal server error occurred"
				logEntry.Errorf("Unhandled internal server error: %+v", err) // Log full error with stack trace if possible
				span.SetStatus(codes.Error, httpErrMessage)
			}

			// Record the error on the span (already done in handlers for specific cases, but good fallback)
			// Ensure RecordError is called appropriately within handlers or here if needed.
			// span.RecordError(err, trace.WithStackTrace(true)) // Might be redundant if handlers already do it

			// REMOVED: Generic span status set - span.SetStatus(codes.Error, httpErrMessage)

			// Return the error response to the client
			return c.Status(code).JSON(fiber.Map{
				"error": httpErrMessage,
			})
		},
	})

	// --- Base Middleware ---
	app.Use(recover.New())
	app.Use(logger.New())

	// --- Health Check Route (BEFORE OTel Middleware) ---
	app.Get("/healthz", productHandler.HealthCheck)

	// --- OTel Middleware (applied AFTER health check) ---
	// Using default otelfiber middleware configuration
	app.Use(otelfiber.Middleware())

	// --- Main API Routes ---
	api := app.Group("/products")
	api.Get("/", productHandler.GetAllProducts)
	api.Get("/:productId", productHandler.GetProductByID)
	api.Get("/:productId/stock", productHandler.GetProductStock)

	// --- Start Server in Goroutine ---
	go func() {
		logrus.Info("Server starting to listen...")
		if err := app.Listen(":" + config.PRODUCT_SERVICE_PORT); err != nil {
			// Check if the error is due to server closing, which is expected during shutdown
			if err != http.ErrServerClosed {
				logrus.WithError(err).Fatal("Server failed to start")
			}
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a signal is received

	logrus.Info("Received shutdown signal, initiating graceful shutdown...")

	// Allow timeout for Fiber shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // Slightly longer than OTel shutdown
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logrus.WithError(err).Error("Error during Fiber server shutdown")
	} else {
		logrus.Info("Fiber server shutdown complete")
	}

	// OTel shutdown is handled by the deferred function call
	logrus.Info("Product service shut down complete.")
}
