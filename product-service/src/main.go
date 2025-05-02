package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/narender/common/config"
	_ "github.com/narender/common/errors"
	"github.com/narender/common/middleware"
	otel "github.com/narender/common/otel"
	"github.com/sirupsen/logrus"

	// Add imports needed for metric callback
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	ServiceName = "product-service"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfgProvider := config.NewEnvironmentProvider()
	cfg, err := config.LoadConfig(".env", cfgProvider)
	if err != nil {

		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	otelShutdown, err := otel.InitTelemetry(ctx, cfg)
	if err != nil {

		logrus.WithError(err).Fatal("Failed to initialize OpenTelemetry SDK")
	}

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		if err := otelShutdown(shutdownCtx); err != nil {

			otel.GetLogger().WithError(err).Error("Error shutting down OpenTelemetry SDK")
		}
	}()
	otel.GetLogger().Info("Logger and OpenTelemetry initialized globally.")

	// Initialize helper for recording specific HTTP metrics (e.g., in error handler)
	// Note: otelfiber middleware uses the global meter provider configured by InitTelemetry
	// for its automatic instrumentation.
	// Call renamed function NewHTTPMetrics
	httpMetricsHelper, err := otel.NewHTTPMetrics()
	if err != nil {
		otel.GetLogger().WithError(err).Fatal("Failed to create common HTTP metrics helper")
	}

	repo, err := NewProductRepository(cfg.DataFilePath)
	if err != nil {
		otel.GetLogger().WithError(err).Fatal("Failed to initialize repository")
	}

	// Define the gauge instrument using the function from common/otel
	stockGauge, err := otel.DefineProductStockGauge()
	if err != nil {
		otel.GetLogger().WithError(err).Fatal("Failed to define stock gauge")
	}

	// Register the callback here using the application's repo instance
	if stockGauge != nil { // Only register if gauge definition succeeded
		meter := otel.GetMeter(otel.ProductInstrumentationName) // Get meter using constant from otel pkg
		_, err = meter.RegisterCallback(
			func(ctx context.Context, o otelmetric.Observer) error {
				levels, err := repo.GetCurrentStockLevels(ctx) // Call repo directly
				if err != nil {
					otel.GetLogger().WithError(err).Error("Failed to get stock levels for metric callback")
					return nil // Don't propagate error to SDK
				}
				for id, stock := range levels {
					o.ObserveInt64(stockGauge, int64(stock), otelmetric.WithAttributes(
						otel.AttrAppProductIDKey.String(id), // Use attribute from otel pkg
					))
				}
				return nil
			},
			stockGauge,
		)
		if err != nil {
			otel.GetLogger().WithError(err).Fatal("Failed to register stock metrics callback")
		}
		otel.GetLogger().Info("Stock metrics callback registered.")
	}

	productService := NewProductService(repo)
	productHandler := NewProductHandler(productService)

	// Pass the helper for use cases like error handler metrics
	errorHandler := middleware.NewErrorHandler(otel.GetLogger(), httpMetricsHelper)
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})
	app.Use(recover.New())
	app.Use(cors.New())

	app.Use(otelfiber.Middleware(otelfiber.WithServerName(cfg.ServiceName)))

	app.Use(middleware.RequestLoggerMiddleware(otel.GetLogger()))

	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)
	v1.Get("/products/:productId", productHandler.GetProductByID)
	v1.Get("/products/:productId/stock", productHandler.GetProductStock)
	v1.Get("/healthz", productHandler.HealthCheck)

	go func() {
		addr := ":" + cfg.ProductServicePort
		otel.GetLogger().WithField("address", addr).Info("Server starting to listen...")
		if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			otel.GetLogger().WithError(err).Fatal("Server failed to start listening")
		}
	}()

	<-ctx.Done()
	otel.GetLogger().Info("Shutdown signal received, starting graceful shutdown...")

	shutdownTimeoutCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ServerShutdownTimeout)
	defer cancelShutdown()
	if err := app.ShutdownWithContext(shutdownTimeoutCtx); err != nil {
		otel.GetLogger().WithError(err).Errorf("Error during server graceful shutdown (timeout %s)", cfg.ServerShutdownTimeout)
	} else {
		otel.GetLogger().Info("Server gracefully shut down.")
	}

	otel.GetLogger().Info("Application exiting.")
}
