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
	"github.com/narender/common/otel"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	// --- Initialize Telemetry using refactored otel builder ---
	// Pass context and shutdown manager to NewSetup (remove cfg argument)
	otelSetup, err := otel.NewSetup(ctx, nil, otel.WithLogger(logrus.StandardLogger()))
	if err != nil {
		logrus.WithError(err).Fatal("Failed during initial OTel setup")
	}

	// Initialize components (Resource is handled internally by NewSetup now)
	otelSetup = otelSetup.WithPropagator()
	otelSetup, err = otelSetup.WithTracing(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize telemetry tracing")
	}
	otelSetup, err = otelSetup.WithMetrics(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize telemetry metrics")
	}
	_, err = otelSetup.WithLogging(ctx)
	if err != nil {
		// Log provider setup already logs a warning if it fails, so just log info here.
		logrus.Info("Proceeding without OpenTelemetry Logging fully configured")
	}

	logrus.Info("OpenTelemetry initialization sequence completed.")

	// --- Dependencies ---
	// Use global config var for DataFilePath
	repo, err := NewProductRepository(config.DATA_FILE_PATH)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize repository")
	}
	productService := NewProductService(repo)
	productHandler := NewProductHandler(productService)

	// --- Fiber App Setup ---
	app := fiber.New(fiber.Config{
		ErrorHandler: productHandler.MapErrorToResponse,
	})
	app.Use(cors.New())
	app.Use(recover.New())
	app.Use(otelfiber.Middleware())

	// --- Routes ---
	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)
	v1.Get("/products/:productId", productHandler.GetProductByID)
	v1.Get("/products/:productId/stock", productHandler.GetProductStock)
	v1.Get("/healthz", productHandler.HealthCheck)

	// --- Start Server Goroutine ---
	go func() {
		// Use global config var for port
		addr := ":" + config.PRODUCT_SERVICE_PORT
		logrus.WithField("address", addr).Info("Server starting to listen...")
		logrus.WithFields(logrus.Fields{
			// Use global config vars
			"otel_endpoint": config.OTEL_EXPORTER_OTLP_ENDPOINT,
			"otel_insecure": config.OTEL_EXPORTER_INSECURE,
		}).Info("Configured OTLP Exporter Endpoint")
		if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithError(err).Fatal("Server failed to start listening")
		}
	}()

	// Keep the application running indefinitely
	select {}
}
