package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	commonErrors "github.com/narender/common/errors"
	"github.com/narender/common/otel"
	"github.com/sirupsen/logrus"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	AttrAppProductID    = attribute.Key("app.product.id")
	AttrAppProductStock = attribute.Key("app.product.stock")
)

type ProductHandler struct {
	service ProductService

	// Metrics
	requestCounter    metric.Int64Counter
	stockGauge        metric.Int64ObservableGauge
	durationHistogram metric.Int64Histogram

	// For gauge observer registration
	stockObservable metric.Registration
}

func NewProductHandler(service ProductService) *ProductHandler {
	// Create metrics
	meter := otel.GetMeter("product-service")

	// Request counter - for tracking API calls
	requestCounter, _ := meter.Int64Counter(
		"product_requests_total",
		metric.WithDescription("Total number of product API requests"),
	)

	// Stock level gauge - for current stock level by product
	stockGauge, _ := meter.Int64ObservableGauge(
		"product_stock_level",
		metric.WithDescription("Current stock level by product"),
	)

	// Duration histogram - for API performance tracking
	durationHistogram, _ := meter.Int64Histogram(
		"product_request_duration_ms",
		metric.WithDescription("Duration of product API requests in milliseconds"),
		metric.WithUnit("ms"),
	)

	handler := &ProductHandler{
		service:           service,
		requestCounter:    requestCounter,
		stockGauge:        stockGauge,
		durationHistogram: durationHistogram,
	}

	// Register callback for stock gauge (will call refreshStockGauge)
	stockObservable, _ := meter.RegisterCallback(
		handler.refreshStockGauge,
		stockGauge,
	)
	handler.stockObservable = stockObservable

	return handler
}

func (h *ProductHandler) refreshStockGauge(ctx context.Context, observer metric.Observer) error {
	// Get all products to update stock levels
	products, err := h.service.GetAll(ctx)
	if err != nil {
		logrus.WithContext(ctx).WithError(err).Error("Failed to refresh stock gauge")
		return err
	}

	// Update gauge for each product
	for _, product := range products {
		observer.ObserveInt64(
			h.stockGauge,
			int64(product.Stock),
			metric.WithAttributes(
				attribute.String("product.id", product.ProductID),
				attribute.String("product.name", product.Name),
				attribute.String("product.category", product.Category),
			),
		)
	}

	return nil
}

func (h *ProductHandler) logRequestStart(ctx context.Context, operation string, fields ...logrus.Fields) {
	logger := logrus.WithContext(ctx)
	if len(fields) > 0 && len(fields[0]) > 0 {
		logger = logger.WithFields(fields[0])
	}
	logger.Debugf("Handler: Received request for %s", operation)
}

func (h *ProductHandler) logRequestEnd(ctx context.Context, operation string, fields logrus.Fields) {
	logrus.WithContext(ctx).WithFields(fields).Debugf("Handler: Responding successfully for %s", operation)
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)
	startTime := time.Now()
	endpoint := "/products"

	// Increment request counter with endpoint attribution
	h.requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("method", "GET"),
	))

	h.logRequestStart(ctx, "GetAllProducts")

	products, err := h.service.GetAll(ctx)
	if err != nil {
		// Record duration even on error
		duration := time.Since(startTime).Milliseconds()
		h.durationHistogram.Record(ctx, duration, metric.WithAttributes(
			attribute.String("endpoint", endpoint),
			attribute.String("status", "error"),
		))
		return err
	}

	// Calculate request duration and record it
	duration := time.Since(startTime).Milliseconds()
	h.durationHistogram.Record(ctx, duration, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("status", "success"),
	))

	// Add span attributes for better trace analysis
	span.SetAttributes(
		attribute.Int("app.product.count", len(products)),
		attribute.String("app.endpoint", endpoint),
		attribute.Int64("app.request.duration_ms", duration),
	)
	span.SetStatus(codes.Ok, "")

	h.logRequestEnd(ctx, "GetAllProducts", logrus.Fields{
		"count":       len(products),
		"duration_ms": duration,
	})
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)
	startTime := time.Now()
	endpoint := "/products/:productId"

	productID, validationErr := h.validatePathParam(ctx, c, JSONFieldProductID)
	if validationErr != nil {
		return validationErr
	}

	// Increment request counter with endpoint and product ID attribution
	h.requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("method", "GET"),
		attribute.String("product.id", productID),
	))

	span.SetAttributes(AttrAppProductID.String(productID))

	h.logRequestStart(ctx, "GetProductByID", logrus.Fields{string(AttrAppProductID): productID})

	product, err := h.service.GetByID(ctx, productID)

	if err != nil {
		// Record duration even on error
		duration := time.Since(startTime).Milliseconds()
		h.durationHistogram.Record(ctx, duration, metric.WithAttributes(
			attribute.String("endpoint", endpoint),
			attribute.String("product.id", productID),
			attribute.String("status", "error"),
		))
		return err
	}

	// Calculate request duration and record it
	duration := time.Since(startTime).Milliseconds()
	h.durationHistogram.Record(ctx, duration, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("product.id", productID),
		attribute.String("product.category", product.Category),
		attribute.String("status", "success"),
	))

	// Add span attributes for better trace analysis
	span.SetAttributes(
		attribute.String("app.endpoint", endpoint),
		attribute.String("product.category", product.Category),
		attribute.Int64("app.request.duration_ms", duration),
	)
	span.SetStatus(codes.Ok, "")

	h.logRequestEnd(ctx, "GetProductByID", logrus.Fields{
		string(AttrAppProductID): productID,
		"duration_ms":            duration,
		"product.category":       product.Category,
	})
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) GetProductStock(c *fiber.Ctx) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)
	startTime := time.Now()
	endpoint := "/products/:productId/stock"

	productID, validationErr := h.validatePathParam(ctx, c, JSONFieldProductID)
	if validationErr != nil {
		return validationErr
	}

	// Increment request counter with endpoint and product ID attribution
	h.requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("method", "GET"),
		attribute.String("product.id", productID),
	))

	span.SetAttributes(AttrAppProductID.String(productID))

	h.logRequestStart(ctx, "GetProductStock", logrus.Fields{string(AttrAppProductID): productID})

	stock, err := h.service.GetStock(ctx, productID)

	if err != nil {
		// Record duration even on error
		duration := time.Since(startTime).Milliseconds()
		h.durationHistogram.Record(ctx, duration, metric.WithAttributes(
			attribute.String("endpoint", endpoint),
			attribute.String("product.id", productID),
			attribute.String("status", "error"),
		))
		return err
	}

	// Calculate request duration and record it
	duration := time.Since(startTime).Milliseconds()
	h.durationHistogram.Record(ctx, duration, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("product.id", productID),
		attribute.String("status", "success"),
	))

	// Add span attributes for better trace analysis
	span.SetAttributes(
		AttrAppProductStock.Int(stock),
		attribute.String("app.endpoint", endpoint),
		attribute.Int64("app.request.duration_ms", duration),
	)
	span.SetStatus(codes.Ok, "")

	logFields := logrus.Fields{
		string(AttrAppProductID):    productID,
		string(AttrAppProductStock): stock,
		"duration_ms":               duration,
	}
	h.logRequestEnd(ctx, "GetProductStock", logFields)

	return c.Status(http.StatusOK).JSON(fiber.Map{
		JSONFieldProductID: productID,
		JSONFieldStock:     stock,
	})
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)
	startTime := time.Now()
	endpoint := "/healthz"

	// Increment request counter with endpoint attribution
	h.requestCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("method", "GET"),
	))

	h.logRequestStart(ctx, "HealthCheck")

	// Check if the product service is healthy by querying the repository
	healthy := true
	healthDetails := map[string]interface{}{
		"service":   "product-service",
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Try to get products from repository to ensure data layer is working
	_, err := h.service.GetAll(ctx)
	if err != nil {
		healthy = false
		healthDetails["status"] = "error"
		healthDetails["error"] = "Data layer not accessible"

		// Record error in span
		span.RecordError(err)
		span.SetStatus(codes.Error, "Health check failed: data layer issue")
	} else {
		span.SetStatus(codes.Ok, "")
	}

	// Calculate request duration and record it
	duration := time.Since(startTime).Milliseconds()
	h.durationHistogram.Record(ctx, duration, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("status", healthDetails["status"].(string)),
	))

	// Add span attributes for better trace analysis
	span.SetAttributes(
		attribute.String("app.endpoint", endpoint),
		attribute.Int64("app.request.duration_ms", duration),
		attribute.Bool("app.health.status", healthy),
	)

	h.logRequestEnd(ctx, "HealthCheck", logrus.Fields{
		"duration_ms": duration,
		"healthy":     healthy,
	})

	// Return status code based on health
	statusCode := http.StatusOK
	if !healthy {
		statusCode = http.StatusServiceUnavailable
	}

	return c.Status(statusCode).JSON(healthDetails)
}

func (h *ProductHandler) validatePathParam(ctx context.Context, c *fiber.Ctx, paramName string) (string, error) {
	id := c.Params(paramName)
	if id == "" {
		span := trace.SpanFromContext(ctx)
		err := &commonErrors.ValidationError{
			Field:   paramName,
			Message: "must not be empty",
		}
		logrus.WithContext(ctx).WithError(err).Warn("Path parameter validation failed")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	return id, nil
}

func (h *ProductHandler) MapErrorToResponse(c *fiber.Ctx, err error) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)
	logger := logrus.WithContext(ctx)
	path := c.Path()

	// Define error handling configuration
	type errorConfig struct {
		statusCode  int
		message     string
		logLevel    logrus.Level
		errorType   string
		customLogFn func(logger *logrus.Entry)
	}

	// Default config for unknown errors
	config := errorConfig{
		statusCode: http.StatusInternalServerError,
		message:    "An unexpected internal server error occurred",
		logLevel:   logrus.ErrorLevel,
		errorType:  "internal_error",
		customLogFn: func(logger *logrus.Entry) {
			logger.WithError(err).Error("Unhandled internal server error")
		},
	}

	var validationErr *commonErrors.ValidationError
	var appErr *commonErrors.AppError
	var dbErr *commonErrors.DatabaseError

	// Determine the appropriate error config based on error type
	switch {
	case errors.As(err, &validationErr):
		config = errorConfig{
			statusCode: http.StatusBadRequest,
			message:    validationErr.Error(),
			logLevel:   logrus.WarnLevel,
			errorType:  "validation_error",
		}
	case errors.As(err, &appErr):
		config = errorConfig{
			statusCode: appErr.StatusCode,
			message:    appErr.Error(),
			logLevel:   logrus.ErrorLevel,
			errorType:  "application_error",
		}
		if config.statusCode < 500 {
			config.logLevel = logrus.WarnLevel
		}
	case errors.As(err, &dbErr):
		config = errorConfig{
			statusCode: http.StatusInternalServerError,
			message:    "An internal database error occurred",
			logLevel:   logrus.ErrorLevel,
			errorType:  "database_error",
			customLogFn: func(logger *logrus.Entry) {
				logger.WithFields(logrus.Fields{"operation": dbErr.Operation}).
					WithError(dbErr.Err).Error("Database error occurred")
			},
		}
	}

	// Apply the custom log function if provided
	if config.customLogFn != nil {
		config.customLogFn(logger)
	}

	// Record error metrics
	meter := otel.GetMeter("product-service")
	errorCounter, _ := meter.Int64Counter(
		"product_error_total",
		metric.WithDescription("Total number of errors by type and status code"),
	)

	errorCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("endpoint", path),
		attribute.String("error.type", config.errorType),
		attribute.Int("http.status_code", config.statusCode),
	))

	// Record error details on span using the modern pattern
	span.SetStatus(codes.Error, config.message) // Set status first
	span.RecordError(err, trace.WithAttributes(
		attribute.Int("http.status_code", config.statusCode),
		attribute.String("error.type", config.errorType),
		attribute.String("error.message", config.message),
	))

	// Log the mapped error
	logger.WithFields(logrus.Fields{
		"error.type": config.errorType,
		"path":       path,
	}).Logf(config.logLevel, "Request error mapped to HTTP %d: %s", config.statusCode, config.message)

	// Return the standardized JSON error response
	return c.Status(config.statusCode).JSON(fiber.Map{
		"error":      config.message,
		"error_type": config.errorType,
		"status":     config.statusCode,
	})
}
