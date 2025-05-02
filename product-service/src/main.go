package main

import (
	"context"
	"log"       
	"math/rand" 
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	
	"github.com/narender/common/config"
	"github.com/narender/common/logging"
	"github.com/narender/common/middleware"
	"github.com/narender/common/telemetry"
	"github.com/narender/common/telemetry/manager"
	"github.com/narender/common/telemetry/metric"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap" 
	
	
	
)

const (
	ServiceName = "product-service" 
)


func simulateDelayIfEnabled() { 
	
	cfg := config.GetHardcodedConfig() 
	if !cfg.SimulateDelayEnabled || cfg.SimulateDelayMaxMs <= 0 || cfg.SimulateDelayMinMs < 0 || cfg.SimulateDelayMinMs > cfg.SimulateDelayMaxMs {
		return
	}
	minMs := cfg.SimulateDelayMinMs
	maxMs := cfg.SimulateDelayMaxMs
	
	delayMs := rand.Intn(maxMs-minMs+1) + minMs
	time.Sleep(time.Duration(delayMs) * time.Millisecond)
}

func main() {
	
	rand.Seed(time.Now().UnixNano())

	
	cfg := config.GetHardcodedConfig()
	cfg.ServiceName = ServiceName 

	
	baseLogger, err := logging.InitZapLogger(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	if baseLogger == nil {
		log.Fatalf("Zap logger initialization returned nil logger.")
	}
	
	defer func() {
		_ = baseLogger.Sync() 
	}()

	
	setupCtx := context.Background()
	shutdownTelemetry, err := telemetry.InitTelemetry(setupCtx, cfg)
	if err != nil {
		baseLogger.Fatal("Failed to initialize telemetry", zap.Error(err))
	}
	
	meter := manager.GetMeter(ServiceName) 
	if err := metric.InitializeCommonMetrics(meter); err != nil {
		
		baseLogger.Error("Failed to initialize common metrics", zap.Error(err))
	}
	
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			baseLogger.Error("Error shutting down telemetry", zap.Error(err))
		}
	}()

	baseLogger.Info("Zap Logger, Telemetry, and Common Metrics initialized.")

	
	appCtx, cancelApp := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancelApp()

	baseLogger.Info("Initializing application dependencies...")
	
	repo, err := NewProductRepository(cfg.DataFilePath) 
	if err != nil {
		baseLogger.Fatal("Failed to initialize product repository", zap.Error(err))
	}
	productService := NewProductService(repo) 
	productHandler := &ProductHandler{service: productService}

	baseLogger.Info("Setting up Fiber application...")
	
	
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(baseLogger, nil),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	})

	
	app.Use(recover.New())
	app.Use(cors.New())

	
	propagator := otel.GetTextMapPropagator()
	app.Use(otelfiber.Middleware(
		otelfiber.WithPropagators(propagator),
	))

	
	app.Use(middleware.ContextLoggerMiddleware(baseLogger))

	
	app.Use(middleware.RequestLoggerMiddleware()) 

	baseLogger.Info("Middleware configured.")

	
	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)
	v1.Get("/products/:productId", productHandler.GetProductByID)
	v1.Get("/healthz", productHandler.HealthCheck)
	baseLogger.Info("API routes configured.")

	
	port := cfg.ProductServicePort
	addr := ":" + port
	go func() {
		baseLogger.Info("Server starting", zap.String("address", addr))
		if err := app.Listen(addr); err != nil && err != http.ErrServerClosed {
			baseLogger.Fatal("Server failed to start listening", zap.Error(err))
		}
	}()

	
	<-appCtx.Done()
	baseLogger.Info("Shutdown signal received, initiating graceful shutdown...")

	shutdownTimeout := cfg.ServerShutdownTimeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	baseLogger.Info("Attempting to shut down Fiber server", zap.Duration("timeout", shutdownTimeout))
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		baseLogger.Error("Error during Fiber server shutdown", zap.Error(err))
	} else {
		baseLogger.Info("Fiber server shut down successfully.")
	}

	baseLogger.Info("Application exiting gracefully.")
}


