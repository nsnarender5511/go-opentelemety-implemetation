package main

import (
	"context"
	"net/http"

	// Use correct module path for common telemetry
	commonErrors "github.com/narender/common-module/errors"
	"github.com/narender/common-module/telemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// Constants for telemetry and JSON field names
const (
	LogFieldProductID = "product_id"
	LogFieldStock     = "stock"
	// JSONField constants are already declared elsewhere, removing duplicates
)

// Attribute keys for telemetry
var (
	// Using string constants instead of attribute.Key
	AppProductIDKey     = "app.product.id"
	AppProductStockKey  = "app.product.stock"
	AppLookupSuccessKey = "app.lookup.success"
	AppStockCheckKey    = "app.stock.check.success"
)

// Package-level variables for instruments are managed via wrappers now

// ProductHandler handles HTTP requests for products
type ProductHandler struct {
	service ProductService
}

// NewProductHandler creates a new product handler
func NewProductHandler(service ProductService) *ProductHandler {
	return &ProductHandler{
		service: service,
	}
}

// GetAllProducts handles GET /products
func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	ctx := c.UserContext()
	log := logrus.WithContext(ctx)
	log.Info("Handler: Received request to get all products")

	// Start the span using wrapper
	ctx, span := telemetry.StartSpan(ctx, "product-service", "handler.GetAllProducts")
	defer span.End()

	// Call the service method with the span context
	products, err := h.service.GetAll(ctx)

	if err != nil {
		log.WithError(err).Error("Handler: Error calling service.GetAll")
		// Record error using wrapper
		telemetry.RecordError(span, err, "failed to get all products")
		return err
	}

	// Record metric for successful response
	telemetry.IncrementCounter(ctx, "product-service", "app.product.list.success", 1)
	telemetry.AddAttribute(span, "product.count", len(products))

	log.Infof("Handler: Responding with %d products", len(products))
	return c.Status(http.StatusOK).JSON(products)
}

// GetProductByID handles GET /products/:productId
func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	ctx := c.UserContext()
	log := logrus.WithContext(ctx)

	productID, validationErr := h.validatePathParam(ctx, c, JSONFieldProductID)
	if validationErr != nil {
		// No span started yet, just return the validation error
		return validationErr
	}
	log = log.WithField(LogFieldProductID, productID)
	log.Infof("Handler: Received request to get product by ID %s", productID)

	// Start span with wrapper
	ctx, span := telemetry.StartSpan(ctx, "product-service", "handler.GetProductByID")
	defer span.End()

	// Set attributes with wrapper
	telemetry.AddAttribute(span, "product.id", productID)

	// Call the service method with the span context
	product, err := h.service.GetByID(ctx, productID)

	// --- Metric Recording using wrapper ---
	success := err == nil
	telemetry.IncrementCounter(ctx, "product-service", "app.product.lookups", 1)
	telemetry.AddAttribute(span, AppProductIDKey, productID)
	telemetry.AddAttribute(span, AppLookupSuccessKey, success)

	// Handle error, log, and return if necessary
	if err != nil {
		log.WithError(err).Errorf("Handler: Error calling service.GetByID for ID %s", productID)
		telemetry.RecordError(span, err, "failed to get product by ID")
		return err
	}

	// If successful, log and return OK response
	log.Infof("Handler: Responding with product data for ID %s", productID)
	return c.Status(http.StatusOK).JSON(product)
}

// GetProductStock handles GET /products/:productId/stock
func (h *ProductHandler) GetProductStock(c *fiber.Ctx) error {
	ctx := c.UserContext()
	log := logrus.WithContext(ctx)

	productID, validationErr := h.validatePathParam(ctx, c, JSONFieldProductID)
	if validationErr != nil {
		// No span started yet, just return the validation error
		return validationErr
	}
	log = log.WithField(LogFieldProductID, productID)
	log.Infof("Handler: Received request to get product stock for ID %s", productID)

	// Start span with wrapper
	ctx, span := telemetry.StartSpan(ctx, "product-service", "handler.GetProductStock")
	defer span.End()

	// Set attributes with wrapper
	telemetry.AddAttribute(span, "product.id", productID)

	// Call the service method with the span context
	stock, err := h.service.GetStock(ctx, productID)

	// --- Metric Recording with wrapper ---
	success := err == nil
	telemetry.IncrementCounter(ctx, "product-service", "app.product.stock_checks", 1)
	telemetry.AddAttribute(span, AppProductIDKey, productID)
	telemetry.AddAttribute(span, AppStockCheckKey, success)

	if success {
		// Add stock attribute only on success
		telemetry.AddAttribute(span, AppProductStockKey, stock)
	}

	// Handle error, log, and return if necessary
	if err != nil {
		log.WithError(err).Errorf("Handler: Error calling service.GetStock for ID %s", productID)
		telemetry.RecordError(span, err, "failed to get product stock")
		return err
	}

	// If successful, log and return OK response
	response := fiber.Map{
		JSONFieldProductID: productID,
		JSONFieldStock:     stock,
	}
	log.Infof("Handler: Responding with product stock %d for ID %s", stock, productID)
	return c.Status(http.StatusOK).JSON(response)
}

// HealthCheck handles GET /healthz
// It provides a minimal liveness check.
func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	// Simply return 200 OK and a status message
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

// Helper to validate path parameters
func (h *ProductHandler) validatePathParam(ctx context.Context, c *fiber.Ctx, paramName string) (string, error) {
	id := c.Params(paramName)
	if id == "" {
		return "", &commonErrors.ValidationError{
			Field:   paramName,
			Message: "must not be empty",
		}
	}
	return id, nil
}
