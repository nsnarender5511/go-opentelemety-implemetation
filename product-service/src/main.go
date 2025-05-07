// Entry point for product service
package main

import (
	"fmt"
	"log/slog"
	"os"

	// "github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/narender/common/globals"
	// Import new common packages
	commonMiddleware "github.com/narender/common/middleware"

	// Import new structured packages
	"github.com/narender/product-service/src/handlers"
	"github.com/narender/product-service/src/repositories"
	"github.com/narender/product-service/src/services"
)

func main() {

	// --- Initialize Globals (Config & Logger/Telemetry) ---
	if err := globals.Init(); err != nil {
		fmt.Printf("Failed to initialize application globals: %v\n", err)
		panic(err)
	}
	logger := globals.Logger()
	logger.Debug("data file located at ", slog.String("path", globals.Cfg().PRODUCT_DATA_FILE_PATH))

	// --- Service and Handler Initialization with new packages ---
	repo := repositories.NewProductRepository()
	service := services.NewProductService(repo)
	handler := handlers.NewProductHandler(service)

	// --- Service Information Logging ---
	logger.Info("Starting product-service")

	// --- Fiber App Initialization with Error Handler ---
	app := fiber.New(fiber.Config{
		ErrorHandler: commonMiddleware.ErrorHandler(),
	})

	// --- Middleware Configuration ---
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Use(recover.New())          // Recover from panics
	// app.Use(otelfiber.Middleware()) // otelfiber instrumentation

	// --- Route Definitions ---
	setupRoutes(app, handler)
	logger.Info("Routes registered")

	// --- Server Startup ---
	addr := fmt.Sprintf(":%s", globals.Cfg().PRODUCT_SERVICE_PORT)
	logger.Info("Server starting to listen", slog.String("address", addr))

	if err := app.Listen(addr); err != nil {
		logger.Error("Server listener failed", slog.Any("error", err))
		os.Exit(1)
	}
}

// setupRoutes function to keep main clean
func setupRoutes(app *fiber.App, handler *handlers.ProductHandler) {
	app.Get("/health", handler.HealthCheck)
	app.Get("/products", handler.GetAllProducts)
	app.Get("/products/category", handler.GetProductsByCategory)
	app.Post("/products/details", handler.GetProductByName)
	app.Patch("/products/stock", handler.UpdateProductStock)
	app.Post("/products/buy", handler.BuyProduct)
}
