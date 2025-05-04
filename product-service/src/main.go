package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/narender/common/globals"
)

func main() {

	// --- Initialize Globals (Config & Logger/Telemetry) ---
	if err := globals.Init(); err != nil {
		fmt.Printf("Failed to initialize application globals: %v\n", err)
		panic(err)
	}
	logger := globals.Logger()

	// --- Service and Handler Initialization ---
	productDataPath := filepath.Join("product-service", "data.json")
	repo := NewProductRepository(productDataPath)
	service := NewProductService(repo)
	handler := NewProductHandler(service)

	// --- Service Information Logging ---
	logger.InfoContext(context.Background(), "Starting product-service")
	app := fiber.New(fiber.Config{})

	// --- Middleware Configuration ---
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Use(recover.New())
	app.Use(otelfiber.Middleware()) // otelfiber instrumentation

	// --- Route Definitions ---

	app.Get("/health", handler.HealthCheck)
	app.Get("/products", handler.GetAllProducts)
	app.Get("/products/:productId", handler.GetProductByID)
	logger.InfoContext(context.Background(), "Routes registered")

	// --- Server Startup ---
	addr := fmt.Sprintf(":%s", globals.Cfg().PRODUCT_SERVICE_PORT)
	logger.InfoContext(context.Background(), "Server starting to listen", slog.String("address", addr))

	if err := app.Listen(addr); err != nil {
		logger.Error("Server listener failed", slog.Any("error", err))
		os.Exit(1)
	}
}
