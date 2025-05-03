package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
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
)

var appConfig *config.Config

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	appConfig = cfg

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 15*time.Second)
	otelShutdown, err := telemetry.InitTelemetry(startupCtx, cfg)
	cancelStartup()
	if err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}
	log.Println("OpenTelemetry initialized successfully.")

	if err := commonlog.Init(cfg); err != nil {
		log.Fatalf("Failed to initialize application logger: %v", err)
	}
	defer commonlog.Cleanup()

	appLogger := commonlog.L

	appLogger.Info("Starting service",
		slog.String("service.name", cfg.ServiceName),
		slog.String("service.version", cfg.ServiceVersion),
		slog.String("environment", cfg.Environment),
	)

	repo, err := NewProductRepository(cfg.DataFilePath)
	if err != nil {
		appLogger.Error("Failed to initialize product repository", slog.Any("error", err))
		os.Exit(1)
	}
	productService := NewProductService(repo)
	productHandler := NewProductHandler(productService)

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(appLogger),
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	app.Use(recover.New())
	app.Use(otelfiber.Middleware())
	app.Use(middleware.RequestLogger(appLogger))

	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)
	v1.Get("/products/:productId", productHandler.GetProductByID)
	v1.Get("/healthz", productHandler.HealthCheck)

	port := cfg.ProductServicePort
	addr := ":" + port
	go func() {
		appLogger.Info("Server starting to listen", slog.String("address", addr))
		if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("Server listener failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutdown signal received, initiating graceful server shutdown...")
	serverShutdownCtx, serverCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer serverCancel()

	if err := app.ShutdownWithContext(serverShutdownCtx); err != nil {
		appLogger.Error("Fiber server graceful shutdown failed", slog.Any("error", err))
	} else {
		appLogger.Info("Fiber server shutdown complete.")
	}

	shutdownCtx, otelCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer otelCancel()

	if err := otelShutdown(shutdownCtx); err != nil {
		appLogger.Error("Error during OpenTelemetry shutdown", slog.Any("error", err))
	} else {
		appLogger.Info("OpenTelemetry shutdown complete.")
	}

	appLogger.Info("Application exiting.")
}
