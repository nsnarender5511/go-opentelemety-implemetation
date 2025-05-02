package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type ProductHandler struct {
	service ProductService
}

func NewProductHandler(service ProductService) *ProductHandler {
	handler := &ProductHandler{
		service: service,
	}
	return handler
}


type ErrorResponse struct {
	Message string `json:"message"`
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	products, err := h.service.GetAll()
	if err != nil {
		log.Printf("ERROR: Handler failed to get all products: %v", err)
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{Message: "Failed to retrieve products"})
	}
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	productID := c.Params("productId")
	product, err := h.service.GetByID(productID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Printf("WARN: Handler product not found for ID %s", productID)
			return c.Status(http.StatusNotFound).JSON(ErrorResponse{Message: "Product not found"})
		}
		log.Printf("ERROR: Handler failed to get product by ID %s: %v", productID, err)
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{Message: "Failed to retrieve product"})
	}
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	log.Println("INFO: Health check requested")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}
