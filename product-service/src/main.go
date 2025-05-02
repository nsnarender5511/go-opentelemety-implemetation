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
	ServiceName = "product-service"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.LoadConfig(".env")
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

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

	meterProvider := otel.GetMeterProvider()
	tracerProvider := otel.GetTracerProvider()
	tracer := tracerProvider.Tracer(ServiceName)

	commonMetrics, err := otel.NewMetrics(meterProvider)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create common metrics helper")
	}

	repo, err := NewProductRepository(cfg.DataFilePath, logger, tracer)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize repository")
	}

	meter := meterProvider.Meter(otel.InstrumentationName)
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

	errorHandler := middleware.NewErrorHandler(logger, commonMetrics)

	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})

	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(middleware.OtelMiddleware())
	app.Use(middleware.RequestLoggerMiddleware(logger))

	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)
	v1.Get("/products/:productId", productHandler.GetProductByID)
	v1.Get("/products/:productId/stock", productHandler.GetProductStock)
	v1.Get("/healthz", productHandler.HealthCheck)

	go func() {
		addr := ":" + cfg.ProductServicePort
		logger.WithField("address", addr).Info("Server starting to listen...")
		if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithError(err).Fatal("Server failed to start listening")
		}
	}()

	<-ctx.Done()

	logger.Info("Shutdown signal received, starting graceful shutdown...")

	shutdownTimeoutCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ServerShutdownTimeout)
	defer cancelShutdown()

	if err := app.ShutdownWithContext(shutdownTimeoutCtx); err != nil {
		logger.WithError(err).Errorf("Error during server graceful shutdown (timeout %s)", cfg.ServerShutdownTimeout)
	} else {
		logger.Info("Server gracefully shut down.")
	}

	logger.Info("Application exiting.")
}
