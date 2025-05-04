package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/narender/common/globals"
	"github.com/narender/common/middleware"
)

func main() {

	// --- Initialize Globals (Config & Logger/Telemetry) ---
	if err := globals.Init(); err != nil {
		fmt.Printf("Failed to initialize application globals: %v\n", err)
		panic(err)
	}
	logger := globals.Logger()

	// --- Service and Handler Initialization ---
	repo := NewProductRepository()
	service := NewProductService(repo)
	handler := NewProductHandler(service)

	// --- Service Information Logging ---
	logger.Info("Starting product-service")
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(logger),
	})

	// --- Middleware Configuration ---
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Use(recover.New())
	app.Use(otelfiber.Middleware()) // otelfiber instrumentation

	// --- Route Definitions ---
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(fiber.Map{"status": "ok (minimal)"})
	})

	app.Get("/products", handler.GetAllProducts)
	app.Get("/products/:productId", handler.GetProductByID)
	app.Patch("/products/:productId/stock", handler.UpdateProductStock)
	app.Get("/status", handler.HealthCheck)
	logger.Info("Routes registered")

	// --- Server Startup ---
	addr := fmt.Sprintf(":%s", globals.Cfg().PRODUCT_SERVICE_PORT)
	logger.Info("Server starting to listen", slog.String("address", addr))

	if err := app.Listen(addr); err != nil {
		logger.Error("Server listener failed", slog.Any("error", err))
		os.Exit(1)
	}
}
