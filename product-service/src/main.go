package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/narender/common/config"
	"github.com/narender/common/lifecycle"
	"github.com/narender/common/telemetry"
	"github.com/sirupsen/logrus"
)

func main() {
	// --- Load Configuration ---
	if err := config.LoadConfig(); err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// --- Initialize Telemetry ---
	telemetryConfig := telemetry.TelemetryConfig{
		ServiceName: config.ServiceName(),
		Endpoint:    config.OtelExporterEndpoint(),
		Insecure:    config.IsOtelExporterInsecure(),
		SampleRatio: config.OtelSampleRatio(),
		LogLevel:    config.LogLevel(),
	}
	otelShutdown, err := telemetry.InitTelemetry(context.Background(), telemetryConfig)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize telemetry")
	}
	logrus.Info("Telemetry initialization sequence completed.")

	// --- Dependencies ---
	repo, err := NewProductRepository()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize repository")
	}
	productService := NewProductService(repo)
	productHandler := NewProductHandler(productService)

	// --- Fiber App Setup ---
	app := fiber.New(fiber.Config{
		// Use custom error handler to map errors centrally
		ErrorHandler: productHandler.MapErrorToResponse,
	})
	app.Use(cors.New())
	app.Use(recover.New())
	// Use otelfiber middleware - Server name comes from the resource
	app.Use(otelfiber.Middleware())

	// --- Routes ---
	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)
	v1.Get("/products/:productId", productHandler.GetProductByID)
	// Add other routes if necessary
	v1.Get("/products/:productId/stock", productHandler.GetProductStock)
	v1.Get("/healthz", productHandler.HealthCheck) // Add health check route

	// --- Start Server Goroutine ---
	go func() {
		addr := ":" + config.ProductServicePort()
		logrus.WithField("address", addr).Info("Server starting to listen...")
		logrus.WithFields(logrus.Fields{
			"otel_endpoint": config.OtelExporterEndpoint(),
			"otel_insecure": config.IsOtelExporterInsecure(),
		}).Info("Configured OTLP Exporter Endpoint")
		if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithError(err).Fatal("Server failed to start listening")
		}
	}()

	// --- Graceful Shutdown (using common helper) ---
	lifecycle.WaitForGracefulShutdown(context.Background(), &lifecycle.FiberShutdownAdapter{App: app}, otelShutdown)

	// Code here will not be reached as WaitForGracefulShutdown blocks and exits.
}
