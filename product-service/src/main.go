package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/narender/common/config"
	commonlog "github.com/narender/common/log"
	"github.com/narender/common/middleware"
	"github.com/narender/common/telemetry"

	"go.uber.org/zap"
)

var appConfig *config.Config

func main() {

	tempLogger := zap.NewExample()
	cfg, err := config.LoadConfig(tempLogger)
	if err != nil {
		tempLogger.Fatal("Failed to load configuration", zap.Error(err))
	}

	appConfig = cfg

	if err := commonlog.Init(cfg); err != nil {
		tempLogger.Fatal("Failed to initialize application logger", zap.Error(err))
	}
	defer commonlog.Cleanup()

	appLogger := commonlog.L

	appLogger.Info("Starting service",
		zap.String("service.name", cfg.ServiceName),
		zap.String("service.version", cfg.ServiceVersion),
		zap.String("environment", cfg.Environment),
	)

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 15*time.Second)
	otelShutdown, err := telemetry.InitTelemetry(startupCtx, cfg)
	cancelStartup()
	if err != nil {
		appLogger.Fatal("Failed to initialize OpenTelemetry", zap.Error(err))
	}
	appLogger.Info("OpenTelemetry initialized successfully.")

	defer func() {
		appLogger.Info("Shutting down OpenTelemetry...")
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		if err := otelShutdown(shutdownCtx); err != nil {
			appLogger.Error("Error during OpenTelemetry shutdown", zap.Error(err))
		} else {
			appLogger.Info("OpenTelemetry shutdown complete.")
		}
	}()

	appCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	appLogger.Info("Initializing application dependencies...")
	repo, err := NewProductRepository(cfg.DataFilePath)
	if err != nil {
		appLogger.Fatal("Failed to initialize product repository", zap.Error(err))
	}
	productService := NewProductService(repo)
	productHandler := NewProductHandler(productService)

	appLogger.Info("Setting up Fiber application...")
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(commonlog.L),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	})

	app.Use(recover.New())
	app.Use(cors.New())
	app.Use(otelfiber.Middleware())
	app.Use(middleware.RequestLoggerMiddleware())
	appLogger.Info("Fiber middleware configured.")

	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)
	v1.Get("/products/:productId", productHandler.GetProductByID)
	v1.Get("/healthz", productHandler.HealthCheck)
	appLogger.Info("API routes configured.")

	port := cfg.ProductServicePort
	addr := ":" + port
	go func() {
		appLogger.Info("Server starting to listen", zap.String("address", addr))
		if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Fatal("Server listener failed", zap.Error(err))
		}
	}()

	<-appCtx.Done()

	appLogger.Info("Shutdown signal received, initiating graceful server shutdown...")
	serverShutdownCtx, cancelServerShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelServerShutdown()

	if err := app.ShutdownWithContext(serverShutdownCtx); err != nil {
		appLogger.Error("Fiber server graceful shutdown failed", zap.Error(err))
	} else {
		appLogger.Info("Fiber server shutdown complete.")
	}

	appLogger.Info("Application exiting.")
}
