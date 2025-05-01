package main

import (
	"context"
	"fmt"
	"net/http"

	// Use correct module path for common telemetry
	commonErrors "github.com/narender/common-module/errors"
	"github.com/narender/common-module/telemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Constants for telemetry and JSON field names
const (
	LogFieldProductID = "product_id"
	LogFieldStock     = "stock"
	// JSONField constants are already declared elsewhere, removing duplicates
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

// OTel-Aligned hypothetical wrapper function
func (h *ProductHandler) handleRequest(c *fiber.Ctx, operationName string, attributes map[string]interface{}, serviceCall func(ctx context.Context) (interface{}, error)) error {
	ctx := c.UserContext() // Get context from Fiber
	log := logrus.WithContext(ctx).WithFields(logrus.Fields(attributes))
	log.Infof("Handler: Received request for %s", operationName)

	// OTel Standard: Get the span created by otelfiber from context
	span := trace.SpanFromContext(ctx)

	// OTel Standard: Add attributes to the *existing* span
	// Convert interface{} attributes safely
	otelAttributes := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		// Reverted: Handle only string keys from the map
		switch val := v.(type) {
		case string:
			otelAttributes = append(otelAttributes, attribute.String(k, val))
		case int:
			otelAttributes = append(otelAttributes, attribute.Int(k, val))
		case bool:
			otelAttributes = append(otelAttributes, attribute.Bool(k, val))
			// Add more types as needed
		}
	}
	span.SetAttributes(otelAttributes...)

	// Execute the core service logic
	result, err := serviceCall(ctx)

	// Generic metric recording (using existing wrapper - acceptable)
	success := err == nil
	metricBaseName := operationName // Or parse operationName
	telemetry.IncrementCounter(ctx, "product-service", fmt.Sprintf("app.%s.attempts", metricBaseName), 1)
	// Maybe add success as a span attribute too - USE CONSTANT
	// Determine success key based on operationName (example)
	var successKey attribute.Key
	if operationName == "handler.GetProductByID" {
		successKey = telemetry.AttrAppLookupSuccess
	} else if operationName == "handler.GetProductStock" {
		successKey = telemetry.AttrAppStockCheck
	} // Add more cases if needed or use a map
	if successKey.Defined() {
		span.SetAttributes(successKey.Bool(success))
	}

	if err != nil {
		log.WithError(err).Errorf("Handler: Error during %s", operationName)
		// OTel Standard: Record the error on the existing span
		span.RecordError(err)
		// Let the central error handler (and otelfiber) set final span status based on HTTP code
		return err // Propagate error
	}

	log.Infof("Handler: Responding successfully for %s", operationName)
	return c.Status(http.StatusOK).JSON(result)
}

// GetAllProducts handles GET /products
func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	attributes := map[string]interface{}{}
	return h.handleRequest(c, "handler.GetAllProducts", attributes, func(ctx context.Context) (interface{}, error) {
		products, err := h.service.GetAll(ctx)
		if err == nil {
			// Add count attribute only on success, retrieve span again
			span := trace.SpanFromContext(ctx)
			// USE CONSTANT
			span.SetAttributes(telemetry.AttrProductCount.Int(len(products)))
		}
		return products, err
	})
}

// GetProductByID handles GET /products/:productId
func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	productID, validationErr := h.validatePathParam(c.UserContext(), c, JSONFieldProductID)
	if validationErr != nil {
		return validationErr
	}

	// USE CONSTANT Key's string value for map key
	attributes := map[string]interface{}{
		string(telemetry.AttrAppProductID): productID,
		LogFieldProductID:                  productID, // Keep string key for logger
	}

	return h.handleRequest(c, "handler.GetProductByID", attributes, func(ctx context.Context) (interface{}, error) {
		return h.service.GetByID(ctx, productID)
	})
}

// GetProductStock handles GET /products/:productId/stock
func (h *ProductHandler) GetProductStock(c *fiber.Ctx) error {
	productID, validationErr := h.validatePathParam(c.UserContext(), c, JSONFieldProductID)
	if validationErr != nil {
		return validationErr
	}

	// USE CONSTANT Key's string value for map key
	attributes := map[string]interface{}{
		string(telemetry.AttrAppProductID): productID,
		LogFieldProductID:                  productID, // Keep string key for logger
	}

	// Call handleRequest, passing the GetStock service call
	return h.handleRequest(c, "handler.GetProductStock", attributes, func(ctx context.Context) (interface{}, error) {
		// The service call itself
		stock, err := h.service.GetStock(ctx, productID)
		if err != nil {
			return nil, err // Return nil result on error
		}
		// On success, return the result formatted as expected by the original handler
		return fiber.Map{
			JSONFieldProductID: productID,
			JSONFieldStock:     stock,
		}, nil
	})
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
