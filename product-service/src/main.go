package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
	"go.opentelemetry.io/otel"
)

// Custom validation error sentinel
// var ErrValidation = fmt.Errorf("input validation failed")

// logrusLoggerHook forwards logrus entries to our custom logger instance
type logrusLoggerHook struct {
	logger *logrus.Logger
}

// Fire implements logrus.Hook.Fire
func (h *logrusLoggerHook) Fire(entry *logrus.Entry) error {
	// Forward to our logger instance
	newEntry := h.logger.WithFields(entry.Data)
	newEntry.Time = entry.Time
	newEntry.Level = entry.Level
	newEntry.Message = entry.Message
	return nil
}

// Levels implements logrus.Hook.Levels
func (h *logrusLoggerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func main() {
	// --- Initialize Logrus FIRST ---
	// Set log level from config
	log := logrus.New()
	if level, err := logrus.ParseLevel(config.LOG_LEVEL); err == nil {
		log.SetLevel(level)
	} else {
		log.Warnf("Invalid log level '%s', using default 'info'", config.LOG_LEVEL)
		log.SetLevel(logrus.InfoLevel)
	}
	// Set formatter (e.g., JSON or Text)
	if config.LOG_FORMAT == "json" {
		log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	} else {
		log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, TimestampFormat: time.RFC3339Nano})
	}
	log.SetOutput(os.Stdout) // Ensure logs go to stdout

	// --- Configure Telemetry ---
	// Create telemetry configuration
	sampleRatio, _ := strconv.ParseFloat(config.OTEL_SAMPLE_RATIO, 64)
	insecure, _ := strconv.ParseBool(config.OTEL_EXPORTER_INSECURE)

	telemetryConfig := telemetry.TelemetryConfig{
		ServiceName:        config.SERVICE_NAME,
		Endpoint:           config.OTEL_EXPORTER_OTLP_ENDPOINT,
		Insecure:           insecure,
		SampleRatio:        sampleRatio,
		BatchTimeoutMS:     5000, // 5 seconds batch timeout
		MaxExportBatchSize: 512,  // Reasonable batch size
		Headers:            make(map[string]string),
		Logger:             log,
	}

	// Set global logger for telemetry package
	telemetry.SetLogger(log)

	// Initialize telemetry with context
	ctx := context.Background()
	otelShutdown, err := telemetry.InitTelemetry(ctx, telemetryConfig)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize OpenTelemetry")
	}

	// Use the application logger for all further logging
	logrus.SetOutput(io.Discard)                                    // Disable standard logrus output
	logrus.StandardLogger().AddHook(&logrusLoggerHook{logger: log}) // Forward to our logger

	log.Info("Starting Product Service", "port", config.PRODUCT_SERVICE_PORT)

	// --- Inject Dependencies ---
	productRepo, err := NewProductRepository()
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize ProductRepository")
	}
	productService := NewProductService(productRepo)
	productHandler := NewProductHandler(productService)

	app := fiber.New(fiber.Config{
		AppName: fmt.Sprintf("%s v%s", config.SERVICE_NAME, config.SERVICE_VERSION),
		// --- Add Fiber Error Handler for OTel Integration ---
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			ctx := c.UserContext()

			// Default status code
			code := http.StatusInternalServerError
			httpErrMessage := "An unexpected internal server error occurred"

			// Map error types to appropriate status codes
			var validationErr *commonErrors.ValidationError
			var dbErr *commonErrors.DatabaseError

			if errors.As(err, &validationErr) {
				code = http.StatusBadRequest
				httpErrMessage = validationErr.Error()
				log.WithContext(ctx).WithError(err).Warn("Validation error")
			} else if errors.Is(err, commonErrors.ErrProductNotFound) {
				code = http.StatusNotFound
				httpErrMessage = commonErrors.ErrProductNotFound.Error()
				log.WithContext(ctx).WithError(err).Warn("Resource not found")
			} else if errors.As(err, &dbErr) {
				code = http.StatusInternalServerError
				httpErrMessage = "An internal database error occurred" // Generic user message
				// Log the specific internal error details
				log.WithContext(ctx).WithFields(logrus.Fields{
					"operation": dbErr.Operation,
				}).WithError(dbErr.Err).Error("Database error")
			} else {
				// Default case for unhandled errors
				log.WithContext(ctx).WithError(err).Error("Unhandled internal server error")
			}

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
	// Using otelfiber middleware with default configuration
	app.Use(otelfiber.Middleware(
		otelfiber.WithPropagators(otel.GetTextMapPropagator()),
	))

	// --- Main API Routes ---
	api := app.Group("/products")
	api.Get("/", productHandler.GetAllProducts)
	api.Get("/:productId", productHandler.GetProductByID)
	api.Get("/:productId/stock", productHandler.GetProductStock)

	// --- Start Server in Goroutine ---
	go func() {
		log.Info("Server starting to listen...")
		if err := app.Listen(":" + config.PRODUCT_SERVICE_PORT); err != nil {
			// Check if the error is due to server closing, which is expected during shutdown
			if err != http.ErrServerClosed {
				log.WithError(err).Fatal("Server failed to start")
			}
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a signal is received

	log.Info("Received shutdown signal, initiating graceful shutdown...")

	// Create overall shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First shut down the HTTP server
	serverShutdownCtx, serverCancel := context.WithTimeout(shutdownCtx, 15*time.Second)
	defer serverCancel()

	if err := app.ShutdownWithContext(serverShutdownCtx); err != nil {
		log.WithError(err).Error("Error during Fiber server shutdown")
	} else {
		log.Info("Server shutdown complete")
	}

	// Then shut down telemetry with remaining time
	telemetryShutdownCtx, telemetryCancel := context.WithTimeout(
		shutdownCtx,
		getRemainingTime(shutdownCtx, 10*time.Second),
	)
	defer telemetryCancel()

	if err := otelShutdown(telemetryShutdownCtx); err != nil {
		log.WithError(err).Error("Error during OpenTelemetry shutdown")
	} else {
		log.Info("OpenTelemetry shutdown complete")
	}

	log.Info("Application shutdown complete")
}

// Helper to calculate remaining time with a minimum fallback
func getRemainingTime(ctx context.Context, minDuration time.Duration) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return minDuration
	}

	remaining := time.Until(deadline)
	if remaining < minDuration {
		return minDuration
	}
	return remaining
}
