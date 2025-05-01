package main

import (
	"context"
	"fmt"
	"net/http"

	// Use correct module path for common telemetry
	"example.com/product-service/common/telemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// --- Custom Error for Validation --- //
var ErrValidation = fmt.Errorf("validation failed")

// ProductHandler handles HTTP requests for products
type ProductHandler struct {
	service                  ProductService
	productLookupsCounter    metric.Int64Counter
	productStockCheckCounter metric.Int64Counter
}

// NewProductHandler creates a new product handler
func NewProductHandler(service ProductService) *ProductHandler {
	// Get meter instance directly using otel global provider
	meter := otel.Meter("product-service/handler")

	// Initialize instruments using the obtained meter
	lookupsCounter, err := meter.Int64Counter(
		"app.product.lookups",
		metric.WithDescription("Counts product lookup attempts"),
		metric.WithUnit("{lookup}"),
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create productLookupsCounter")
	}

	stockChecksCounter, err := meter.Int64Counter(
		"app.product.stock_checks",
		metric.WithDescription("Counts product stock check attempts"),
		metric.WithUnit("{check}"),
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create productStockCheckCounter")
	}

	return &ProductHandler{
		service:                  service,
		productLookupsCounter:    lookupsCounter,
		productStockCheckCounter: stockChecksCounter,
	}
}

// getTracer is a helper to get the tracer instance consistently
func (h *ProductHandler) getTracer() trace.Tracer {
	return otel.Tracer("product-service/handler")
}

// GetAllProducts handles GET /products
func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	ctx := c.UserContext()
	log := logrus.WithContext(ctx)
	log.Info("Handler: Received request to get all products")

	var products []Product
	var err error

	tracer := h.getTracer()
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
		return fmt.Errorf("%w: %w", ErrValidation, validationErr)
	}
	log = log.WithField(telemetry.LogFieldProductID, productID)
	log.Infof("Handler: Received request to get product by ID %s", productID)

	var product Product
	var err error

	tracer := h.getTracer()
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
	if !success {
		// Optionally add error type attribute if available
		// lookupAttrs = append(lookupAttrs, attribute.String("app.error.type", commonErrors.GetType(err)))
	}
	h.productLookupsCounter.Add(ctx, 1, metric.WithAttributes(lookupAttrs...))
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
		return fmt.Errorf("%w: %w", ErrValidation, validationErr)
	}
	log = log.WithField(telemetry.LogFieldProductID, productID)
	log.Infof("Handler: Received request to get product stock for ID %s", productID)

	var stock int
	var err error

	tracer := h.getTracer()
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
	} else {
		// Optionally add error type attribute if available
		// stockCheckAttrs = append(stockCheckAttrs, attribute.String("app.error.type", commonErrors.GetType(err)))
	}
	h.productStockCheckCounter.Add(ctx, 1, metric.WithAttributes(stockCheckAttrs...))
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
// Returns the parameter value or a wrapped error if validation fails.
func (h *ProductHandler) validatePathParam(ctx context.Context, c *fiber.Ctx, paramName string) (string, error) {
	paramValue := c.Params(paramName)
	if paramValue == "" {
		msg := fmt.Sprintf("%s parameter is required", paramName)
		logrus.WithContext(ctx).Warnf("Handler: Missing %s parameter", paramName)
		// Return a distinct error that can be checked by the error handler
		return "", fmt.Errorf(msg) // Return standard error
	}
	return paramValue, nil
}
