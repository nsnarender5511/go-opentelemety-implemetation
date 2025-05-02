package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/narender/common/config"
	_ "github.com/narender/common/errors"
	"github.com/narender/common/middleware"
	"github.com/narender/common/otel"
	"github.com/sirupsen/logrus"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	ServiceName = "product-service" // Or derive from config
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// --- Load Configuration ---
	cfg, err := config.LoadConfig(".env") // Load from .env file relative to executable
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// --- Initialize Logger & Telemetry ---
	logger, otelShutdown, err := otel.SetupOTelSDK(ctx, cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize OpenTelemetry SDK")
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		if err := otelShutdown(shutdownCtx); err != nil {
			logger.WithError(err).Error("Error shutting down OpenTelemetry SDK")
		}
	}()

	logger.Info("Logger and OpenTelemetry initialized.")

	// --- Dependencies ---
	// Get Meter & Tracer
	meterProvider := otel.GetMeterProvider()
	tracerProvider := otel.GetTracerProvider()
	tracer := tracerProvider.Tracer(ServiceName) // Use service name for the main tracer

	// Common Metrics Helper
	commonMetrics, err := otel.NewMetrics(meterProvider)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create common metrics helper")
	}

	// Repository, Service, Handler initialization (Requires updated constructors)
	repo, err := NewProductRepository(cfg.DataFilePath, logger, tracer)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize repository")
	}

	// --- Register Observable Metrics (e.g., Stock Levels) ---
	// Ensure repo implements a method suitable for callback, e.g., ObserveStockLevels
	// Assume repo has: ObserveStockLevels(ctx context.Context, obs metric.Observer, gauge metric.Int64ObservableGauge) error
	meter := meterProvider.Meter(otel.InstrumentationName) // Use common instrumentation name
	productStockGauge, err := meter.Int64ObservableGauge(
		"product.stock.level",
		otelmetric.WithDescription("Current stock level for each product"),
		otelmetric.WithUnit("{items}"),
	)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create product stock gauge")
	}

	_, err = meter.RegisterCallback(
		func(ctx context.Context, obs otelmetric.Observer) error {
			// Call the method directly on the interface
			return repo.ObserveStockLevels(ctx, obs, productStockGauge)
		},
		productStockGauge,
	)
	if err != nil {
		logger.WithError(err).Fatal("Failed to register product stock callback")
	}
	logger.Info("Registered product stock observable gauge callback.")

	productService := NewProductService(repo, logger, tracer)
	productHandler := NewProductHandler(productService, logger, tracer, commonMetrics)

	// --- Fiber App Setup ---
	// Common Error Handler
	errorHandler := middleware.NewErrorHandler(logger, commonMetrics)

	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler, // Use the common error handler
		// Increase read timeout for server robustness
		// ReadTimeout: 5 * time.Second,
	})

	// Middleware
	app.Use(recover.New()) // Recover should be early
	app.Use(cors.New())    // Configure CORS as needed
	// Use the common Otel middleware wrapper
	// TODO: Fix otelfiber import/usage issue
	// app.Use(middleware.OtelMiddleware(otelfiber.WithServerName(cfg.ServiceName)))
	app.Use(middleware.OtelMiddleware()) // Use without options for now
	// Use the common Request Logger middleware
	app.Use(middleware.RequestLoggerMiddleware(logger))

	// --- Routes ---
	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)
	v1.Get("/products/:productId", productHandler.GetProductByID)
	v1.Get("/products/:productId/stock", productHandler.GetProductStock)
	v1.Get("/healthz", productHandler.HealthCheck)

	// --- Start Server ---
	go func() {
		addr := ":" + cfg.ProductServicePort
		logger.WithField("address", addr).Info("Server starting to listen...")
		if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithError(err).Fatal("Server failed to start listening")
		}
	}()

	// --- Wait for shutdown signal ---
	<-ctx.Done()

	logger.Info("Shutdown signal received, starting graceful shutdown...")

	// --- Graceful Shutdown ---
	shutdownTimeoutCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ServerShutdownTimeout)
	defer cancelShutdown()

	if err := app.ShutdownWithContext(shutdownTimeoutCtx); err != nil {
		logger.WithError(err).Errorf("Error during server graceful shutdown (timeout %s)", cfg.ServerShutdownTimeout)
	} else {
		logger.Info("Server gracefully shut down.")
	}

	logger.Info("Application exiting.")
}
