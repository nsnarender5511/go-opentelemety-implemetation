// Entry point for master store
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/narender/common/globals"
	// Import new common packages
	commonMiddleware "github.com/narender/common/middleware"

	// Import new structured packages
	"github.com/narender/master-store/src/handlers"
	"github.com/narender/master-store/src/repositories"
	"github.com/narender/master-store/src/services"
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
	service := services.NewMasterStoreService(repo)
	handler := handlers.NewMasterStoreHandler(service)

	// --- Service Information Logging ---
	logger.Info("Starting master-store")

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
	app.Use(otelfiber.Middleware()) // otelfiber instrumentation

	// --- Route Definitions ---
	setupRoutes(app, handler)
	logger.Info("Routes registered")

	// --- Server Startup ---
	addr := fmt.Sprintf(":%s", globals.Cfg().MASTER_STORE_SERVICE_PORT)
	logger.Info("Server starting to listen", slog.String("address", addr))

	if err := app.Listen(addr); err != nil {
		logger.Error("Server listener failed", slog.Any("error", err))
		os.Exit(1)
	}
}

// setupRoutes function to keep main clean
func setupRoutes(app *fiber.App, handler *handlers.MasterStoreHandler) {
	app.Get("/health", handler.HealthCheck)
	app.Get("/products", handler.GetAllProducts)
	app.Post("/products/update-stock", handler.UpdateStoreStock)
}
