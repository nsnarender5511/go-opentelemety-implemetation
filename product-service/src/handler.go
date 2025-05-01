package main

import (
	"context"
	"errors"
	"net/http"

	commonErrors "github.com/narender/common/errors"
	"github.com/sirupsen/logrus"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Define custom attribute keys specific to this service
var (
	AttrAppProductID    = attribute.Key("app.product.id")
	AttrAppProductStock = attribute.Key("app.product.stock")
)

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

// logRequestStart logs the start of a request handler
func (h *ProductHandler) logRequestStart(ctx context.Context, operation string, fields ...logrus.Fields) {
	logger := logrus.WithContext(ctx)
	if len(fields) > 0 && len(fields[0]) > 0 {
		logger = logger.WithFields(fields[0])
	}
	logger.Debugf("Handler: Received request for %s", operation)
}

// logRequestEnd logs the successful completion of a request handler
func (h *ProductHandler) logRequestEnd(ctx context.Context, operation string, fields logrus.Fields) {
	logrus.WithContext(ctx).WithFields(fields).Debugf("Handler: Responding successfully for %s", operation)
}

// GetAllProducts handles GET /products
func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)

	h.logRequestStart(ctx, "GetAllProducts")

	products, err := h.service.GetAll(ctx)
	if err != nil {
		return err
	}

	span.SetAttributes(attribute.Int("app.product.count", len(products)))
	span.SetStatus(codes.Ok, "")

	h.logRequestEnd(ctx, "GetAllProducts", logrus.Fields{"count": len(products)})
	return c.Status(http.StatusOK).JSON(products)
}

// GetProductByID handles GET /products/:productId
func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)

	productID, validationErr := h.validatePathParam(ctx, c, JSONFieldProductID)
	if validationErr != nil {
		return validationErr
	}

	span.SetAttributes(AttrAppProductID.String(productID))

	h.logRequestStart(ctx, "GetProductByID", logrus.Fields{string(AttrAppProductID): productID})

	product, err := h.service.GetByID(ctx, productID)

	if err != nil {
		return err
	}

	span.SetStatus(codes.Ok, "")

	h.logRequestEnd(ctx, "GetProductByID", logrus.Fields{string(AttrAppProductID): productID})
	return c.Status(http.StatusOK).JSON(product)
}

// GetProductStock handles GET /products/:productId/stock
func (h *ProductHandler) GetProductStock(c *fiber.Ctx) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)

	productID, validationErr := h.validatePathParam(ctx, c, JSONFieldProductID)
	if validationErr != nil {
		return validationErr
	}

	span.SetAttributes(AttrAppProductID.String(productID))

	h.logRequestStart(ctx, "GetProductStock", logrus.Fields{string(AttrAppProductID): productID})

	stock, err := h.service.GetStock(ctx, productID)

	if err != nil {
		return err
	}

	span.SetAttributes(AttrAppProductStock.Int(stock))
	span.SetStatus(codes.Ok, "")

	logFields := logrus.Fields{
		string(AttrAppProductID):    productID,
		string(AttrAppProductStock): stock,
	}
	h.logRequestEnd(ctx, "GetProductStock", logFields)

	return c.Status(http.StatusOK).JSON(fiber.Map{
		JSONFieldProductID: productID,
		JSONFieldStock:     stock,
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

// validatePathParam handles path parameter validation
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

// MapErrorToResponse maps application errors to HTTP responses and sets span status.
func (h *ProductHandler) MapErrorToResponse(c *fiber.Ctx, err error) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)
	logger := logrus.WithContext(ctx)

	// Define error handling configuration
	type errorConfig struct {
		statusCode  int
		message     string
		logLevel    logrus.Level
		customLogFn func(logger *logrus.Entry)
	}

	// Default config for unknown errors
	config := errorConfig{
		statusCode: http.StatusInternalServerError,
		message:    "An unexpected internal server error occurred",
		logLevel:   logrus.ErrorLevel,
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
		}
	case errors.As(err, &appErr):
		config = errorConfig{
			statusCode: appErr.StatusCode,
			message:    appErr.Error(),
			logLevel:   logrus.ErrorLevel,
		}
		if config.statusCode < 500 {
			config.logLevel = logrus.WarnLevel
		}
	case errors.As(err, &dbErr):
		config = errorConfig{
			statusCode: http.StatusInternalServerError,
			message:    "An internal database error occurred",
			logLevel:   logrus.ErrorLevel,
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

	// Record error details on span using the modern pattern
	span.SetStatus(codes.Error, config.message) // Set status first
	span.RecordError(err, trace.WithAttributes(
		attribute.Int("http.status_code", config.statusCode),
	))

	// Log the mapped error
	logger.Logf(config.logLevel, "Request error mapped to HTTP %d: %s", config.statusCode, config.message)

	// Return the standardized JSON error response
	return c.Status(config.statusCode).JSON(fiber.Map{"error": config.message})
}
