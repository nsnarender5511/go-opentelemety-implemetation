package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "signoz-common/config"
	"signoz-common/telemetry"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/sirupsen/logrus"
)

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

	productRepo := NewProductRepository()
	productService := NewProductService(productRepo)
	productHandler := NewProductHandler(productService)

	app := fiber.New(fiber.Config{
		// Consider adding ErrorHandler for better OTel error reporting
		// ErrorHandler: func(c *fiber.Ctx, err error) error { ... }
	})

	// --- Add OTel Middleware FIRST ---
	app.Use(otelfiber.Middleware())

	// Optional: Keep Fiber's built-in logger or rely solely on OTel/Logrus logs
	app.Use(fiberlogger.New())

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
