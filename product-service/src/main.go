package main

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/narender/common/config"
	"github.com/narender/common/lifecycle"
	"github.com/narender/common/otel"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	// --- Load Configuration using new pattern ---
	cfg := config.NewConfig().WithEnv()
	if validationErrs := cfg.Validate(); len(validationErrs) > 0 {
		errMsgs := make([]string, len(validationErrs))
		for i, err := range validationErrs {
			errMsgs[i] = err.Error()
		}
		logrus.Fatalf("Configuration validation failed: %s", strings.Join(errMsgs, "; "))
	}
	cfg.Log()

	// --- Initialize Telemetry using new otel builder ---
	otelSetup := otel.NewSetup(cfg)

	// Initialize components (Resource, Propagator, Tracing, Metrics, Logging)
	// The order might matter for dependencies (e.g., tracing needs resource)
	otelSetup, err := otelSetup.WithResource(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize telemetry resource")
	}
	otelSetup = otelSetup.WithPropagator()
	otelSetup, err = otelSetup.WithTracing(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize telemetry tracing")
	}
	otelSetup, err = otelSetup.WithMetrics(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize telemetry metrics")
	}
	otelSetup, err = otelSetup.WithLogging(ctx)
	if err != nil {
		logrus.WithError(err).Error("Proceeding without OpenTelemetry Logging configured")
	}

	logrus.Info("OpenTelemetry initialization sequence completed.")

	// --- Dependencies ---
	repo, err := NewProductRepository(cfg.DataFilePath)
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
		addr := ":" + cfg.ProductServicePort
		logrus.WithField("address", addr).Info("Server starting to listen...")
		logrus.WithFields(logrus.Fields{
			"otel_endpoint": cfg.OtelEndpoint,
			"otel_insecure": cfg.OtelInsecure,
		}).Info("Configured OTLP Exporter Endpoint")
		if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithError(err).Fatal("Server failed to start listening")
		}
	}()

	// --- Graceful Shutdown ---
	// Create shutdown manager
	shutdownManager := lifecycle.NewShutdownManager(logrus.StandardLogger()).WithTimeout(cfg.ShutdownTotalTimeout)

	// Register OTel shutdown using the new adapter constructor
	shutdownManager.Register("opentelemetry", lifecycle.NewFuncAdapter(otelSetup.Shutdown), cfg.ShutdownOtelMinTimeout)

	// Register Fiber app shutdown
	shutdownManager.Register("fiber-server", &lifecycle.FiberAdapter{App: app}, cfg.ShutdownServerTimeout)

	// Start listening for signals and wait for shutdown completion
	shutdownManager.Start(ctx)

	// The shutdown manager now handles blocking and exiting.
}
