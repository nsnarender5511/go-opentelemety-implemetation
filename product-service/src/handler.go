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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

// Package-level variables for instruments
var (
	productLookupsCounter    metric.Int64Counter
	productStockCheckCounter metric.Int64Counter
)

// Initialize instruments using global Meter provider
func init() {
	meter := otel.Meter("product-service/handler") // Use correct instrumentation scope name
	var err error
	productLookupsCounter, err = meter.Int64Counter(
		"app.product.lookups", // Metric name
		metric.WithDescription("Counts product lookup attempts by ID"),
		metric.WithUnit("{lookup}"),
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create productLookupsCounter")
	}

	productStockCheckCounter, err = meter.Int64Counter(
		"app.product.stock_checks", // Metric name
		metric.WithDescription("Counts product stock check attempts by ID"),
		metric.WithUnit("{check}"),
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create productStockCheckCounter")
	}
}

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

	var products []Product
	var err error

	tracer := otel.Tracer("product-service/handler") // Get tracer directly from global provider
	// Start the span manually
	ctx, span := tracer.Start(ctx, "handler.GetAllProducts")
	defer span.End() // Ensure the span is ended

	// Call the service method with the span context
	products, err = h.service.GetAll(ctx)

	if err != nil {
		log.WithError(err).Error("Handler: Error calling service.GetAll")
		// Manually record error and set status on the span
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err // Return the error directly
	}

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
	log = log.WithField(telemetry.LogFieldProductID, productID)
	log.Infof("Handler: Received request to get product by ID %s", productID)

	var product Product
	var err error

	tracer := otel.Tracer("product-service/handler") // Get tracer directly
	// Start the span manually
	ctx, span := tracer.Start(ctx, "handler.GetProductByID")
	defer span.End() // Ensure the span is ended

	// Set attributes now that the span exists
	span.SetAttributes(telemetry.AppProductIDKey.String(productID))

	// Call the service method with the span context
	product, err = h.service.GetByID(ctx, productID)

	// --- Metric Recording ---
	lookupAttrs := []attribute.KeyValue{telemetry.AppProductIDKey.String(productID)}
	success := err == nil
	lookupAttrs = append(lookupAttrs, telemetry.AppLookupSuccessKey.Bool(success))
	productLookupsCounter.Add(ctx, 1, metric.WithAttributes(lookupAttrs...)) // Use package-level counter
	// --- End Metric Recording ---

	// Handle error, log, set span status, and return if necessary
	if err != nil {
		log.WithError(err).Errorf("Handler: Error calling service.GetByID for ID %s", productID)
		// Manually record error and set status on the span
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err // Return the error directly
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
	log = log.WithField(telemetry.LogFieldProductID, productID)
	log.Infof("Handler: Received request to get product stock for ID %s", productID)

	var stock int
	var err error

	tracer := otel.Tracer("product-service/handler") // Get tracer directly
	// Start the span manually
	ctx, span := tracer.Start(ctx, "handler.GetProductStock")
	defer span.End() // Ensure the span is ended

	// Set attributes now that the span exists
	span.SetAttributes(telemetry.AppProductIDKey.String(productID))

	// Call the service method with the span context
	stock, err = h.service.GetStock(ctx, productID)

	// --- Metric Recording ---
	stockCheckAttrs := []attribute.KeyValue{telemetry.AppProductIDKey.String(productID)}
	success := err == nil
	stockCheckAttrs = append(stockCheckAttrs, telemetry.AppStockCheckSuccessKey.Bool(success))
	if success {
		// Set stock attribute on span ONLY on success
		span.SetAttributes(telemetry.AppProductStockKey.Int(stock))
		stockCheckAttrs = append(stockCheckAttrs, telemetry.AppProductStockKey.Int(stock))
	}
	productStockCheckCounter.Add(ctx, 1, metric.WithAttributes(stockCheckAttrs...)) // Use package-level counter
	// --- End Metric Recording ---

	// Handle error, log, set span status, and return if necessary
	if err != nil {
		log.WithError(err).Errorf("Handler: Error calling service.GetStock for ID %s", productID)
		// Manually record error and set status on the span
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err // Return the error directly
	}

	// If successful, log and return OK response
	response := fiber.Map{
		JSONFieldProductID: productID,
		JSONFieldStock:     stock,
	}
	log.Infof("Handler: Responding with product stock %d for ID %s", stock, productID)
	return c.Status(http.StatusOK).JSON(response)
}

// --- Helper Functions ---

// validatePathParam extracts and validates a required path parameter.
// Returns the parameter value or a wrapped commonErrors.ValidationError if validation fails.
func (h *ProductHandler) validatePathParam(ctx context.Context, c *fiber.Ctx, paramName string) (string, error) {
	paramValue := c.Params(paramName)
	if paramValue == "" {
		msg := fmt.Sprintf("%s parameter is required", paramName)
		logrus.WithContext(ctx).Warnf("Handler: Missing %s parameter", paramName)
		// Return a distinct error that can be checked by the error handler
		return "", &commonErrors.ValidationError{Field: paramName, Message: msg}
	}
	// Add more validation if needed (e.g., regex, length)
	return paramValue, nil
}
