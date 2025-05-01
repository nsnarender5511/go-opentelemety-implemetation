package main

import (
	"context"
	"errors"
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
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/codes"
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
		// --- Add Fiber Error Handler for OTel Integration ---
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			ctx := c.UserContext()
			// Get the current span from the context
			span := trace.SpanFromContext(ctx)

			// Determine the correct HTTP status code based on error type
			code := http.StatusInternalServerError  // Default
			var httpErrMessage string = err.Error() // Default message

			if errors.Is(err, commonErrors.ErrProductNotFound) {
				code = http.StatusNotFound
				httpErrMessage = commonErrors.ErrProductNotFound.Error()
			} else if errors.Is(err, commonErrors.ErrDatabaseOperation) {
				// Keep 500 but maybe provide a less specific message to client
				httpErrMessage = "An internal database error occurred"
			} else if errors.Is(err, ErrValidation) { // Check against ErrValidation (local package)
				code = http.StatusBadRequest
				// Extract the original validation message if wrapped
				unwrappedErr := errors.Unwrap(err)
				if unwrappedErr != nil {
					httpErrMessage = unwrappedErr.Error()
				}
			}
			// Add checks for other specific commonErrors if needed

			// Log the original error with context (includes trace/span IDs via hook)
			logEntry := logrus.WithContext(ctx).WithError(err)
			if code >= 500 {
				logEntry.Errorf("Unhandled error in handler: %v", err)
			} else {
				// Log client errors (4xx) at Warn level
				logEntry.Warnf("Client error in handler: %v", err)
			}

			// Record the error on the span
			span.RecordError(err)
			// Set span status to Error (standard practice for any handler error)
			span.SetStatus(codes.Error, err.Error())

			// Return the error response to the client using the determined code and message
			return c.Status(code).JSON(fiber.Map{
				// Use the JSONFieldError constant (defined in handler.go, now part of package main)
				JSONFieldError: httpErrMessage,
			})
		},
	})

	// --- Add OTel Middleware FIRST ---
	app.Use(otelfiber.Middleware())

	// Optional: Keep Fiber's built-in logger or rely solely on OTel/Logrus logs
	// app.Use(fiberlogger.New()) // REMOVED

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
