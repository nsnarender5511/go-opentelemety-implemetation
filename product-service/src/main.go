package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

const (
	ServiceName         = "product-service"
	DefaultDataFilePath = "data.json" 
	DefaultServerPort   = "8080"      
)

func main() {
	fmt.Println("Initializing application...")

	
	dataPath := DefaultDataFilePath
	
	
	

	
	repo, err := NewProductRepository(dataPath)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err) 
	}

	
	productService := NewProductService(repo)
	productHandler := NewProductHandler(productService)

	

	
	app := fiber.New() 

	
	app.Use(recover.New())
	app.Use(cors.New())

	
	

	
	
	api := app.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/products", productHandler.GetAllProducts)            
	v1.Get("/products/:productId", productHandler.GetProductByID) 
	
	v1.Get("/healthz", productHandler.HealthCheck) 

	

	
	port := DefaultServerPort
	
	
	
	addr := ":" + port

	fmt.Printf("Server starting on %s\n", addr)

	
	if err := app.Listen(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start listening: %v", err) 
	}

	

	fmt.Println("Application exiting.")
}
