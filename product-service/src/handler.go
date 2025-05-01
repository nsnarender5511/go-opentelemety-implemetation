package main

import (
	"context"
	"errors"
	"net/http"

	commonErrors "github.com/narender/common/errors"
	"github.com/narender/common/telemetry"
	"github.com/sirupsen/logrus"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ProductHandler handles HTTP requests for products
type ProductHandler struct {
	service ProductService
	logger  *logrus.Logger
}

// NewProductHandler creates a new product handler
func NewProductHandler(service ProductService) *ProductHandler {
	return &ProductHandler{
		service: service,
		logger:  logrus.StandardLogger(),
	}
}

// GetAllProducts handles GET /products
func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)

	h.logger.WithContext(ctx).Debug("Handler: Received request for GetAllProducts")

	products, err := h.service.GetAll(ctx)
	if err != nil {
		return err
	}

	span.SetAttributes(telemetry.AttrProductCount.Int(len(products)))
	span.SetStatus(codes.Ok, "")

	h.logger.WithContext(ctx).WithField("count", len(products)).Debug("Handler: Responding successfully for GetAllProducts")
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

	span.SetAttributes(telemetry.AttrAppProductID.String(productID))

	h.logger.WithContext(ctx).WithField(string(telemetry.AttrAppProductID), productID).Debug("Handler: Received request for GetProductByID")

	product, err := h.service.GetByID(ctx, productID)

	if err != nil {
		return err
	}

	span.SetStatus(codes.Ok, "")

	h.logger.WithContext(ctx).WithField(string(telemetry.AttrAppProductID), productID).Debug("Handler: Responding successfully for GetProductByID")
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

	span.SetAttributes(telemetry.AttrAppProductID.String(productID))

	h.logger.WithContext(ctx).WithField(string(telemetry.AttrAppProductID), productID).Debug("Handler: Received request for GetProductStock")

	stock, err := h.service.GetStock(ctx, productID)

	if err != nil {
		return err
	}

	span.SetAttributes(telemetry.AttrAppProductStock.Int(stock))
	span.SetStatus(codes.Ok, "")

	h.logger.WithContext(ctx).WithFields(logrus.Fields{
		string(telemetry.AttrAppProductID):    productID,
		string(telemetry.AttrAppProductStock): stock,
	}).Debug("Handler: Responding successfully for GetProductStock")

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
		h.logger.WithContext(ctx).WithError(err).Warn("Path parameter validation failed")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	return id, nil
}

// MapErrorToResponse maps application errors to HTTP responses and sets span status
// Renamed to be exported.
func (h *ProductHandler) MapErrorToResponse(c *fiber.Ctx, err error) error {
	ctx := c.UserContext()
	span := trace.SpanFromContext(ctx)

	// Default error response
	code := http.StatusInternalServerError
	httpErrMessage := "An unexpected internal server error occurred"
	spanStatus := codes.Error
	spanMessage := httpErrMessage
	logLevel := logrus.ErrorLevel // Default log level for errors

	var validationErr *commonErrors.ValidationError
	var dbErr *commonErrors.DatabaseError

	if errors.As(err, &validationErr) {
		code = http.StatusBadRequest
		httpErrMessage = validationErr.Error()
		spanMessage = httpErrMessage
		logLevel = logrus.WarnLevel
	} else if errors.Is(err, commonErrors.ErrProductNotFound) {
		code = http.StatusNotFound
		httpErrMessage = commonErrors.ErrProductNotFound.Error()
		spanMessage = httpErrMessage
		logLevel = logrus.WarnLevel
	} else if errors.As(err, &dbErr) {
		// Keep 500 for database errors, but log details
		httpErrMessage = "An internal database error occurred"
		spanMessage = httpErrMessage
		// Log underlying DB error detail at Error level
		h.logger.WithContext(ctx).WithFields(logrus.Fields{"operation": dbErr.Operation}).WithError(dbErr.Err).Error("Database error occurred")
	} else {
		// Log any other unexpected error
		h.logger.WithContext(ctx).WithError(err).Error("Unhandled internal server error")
	}

	// Record error on span regardless of type
	span.RecordError(err)
	span.SetStatus(spanStatus, spanMessage)

	// Log the mapped error with the determined level
	h.logger.WithContext(ctx).WithError(err).Logf(logLevel, "Request error mapped to HTTP %d", code)

	// Return the JSON error response
	return c.Status(code).JSON(fiber.Map{"error": httpErrMessage})
}
