package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/logging"
	"github.com/narender/common/telemetry/metric"
	"github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.uber.org/zap"
)

const handlerScopeName = "github.com/narender/product-service/handler"
const handlerLayerName = "handler"

type ProductHandler struct {
	service ProductService
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (opErr error) {
	const operation = "GetAllProducts"
	startTime := time.Now()
	ctx := c.UserContext()
	defer func() {
		metric.RecordOperationMetrics(ctx, handlerLayerName, operation, startTime, opErr)
	}()

	simulateDelayIfEnabled()
	logger := logging.LoggerFromContext(ctx)
	ctx, span := trace.StartSpan(ctx, handlerScopeName, "ProductHandler."+operation,
		semconv.HTTPRouteKey.String(c.Route().Path),
	)
	defer span.End()

	logger.Info("Handler: Received request for GetAllProducts")

	simulateDelayIfEnabled()
	products, err := h.service.GetAll(ctx)
	if err != nil {
		opErr = err
		simulateDelayIfEnabled()
		logger.Error("Handler: Failed to get all products from service", zap.Error(opErr))
		span.RecordError(opErr)
		span.SetStatus(codes.Error, "service layer error")
		return opErr
	}

	simulateDelayIfEnabled()
	logger.Info("Handler: Successfully retrieved all products")
	span.SetStatus(codes.Ok, "")
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) (opErr error) {
	const operation = "GetProductByID"
	startTime := time.Now()
	ctx := c.UserContext()
	productID := c.Params("productId")
	productIdAttr := attribute.String("product.id", productID)
	defer func() {
		metric.RecordOperationMetrics(ctx, handlerLayerName, operation, startTime, opErr, productIdAttr)
	}()

	simulateDelayIfEnabled()
	logger := logging.LoggerFromContext(ctx)
	ctx, span := trace.StartSpan(ctx, handlerScopeName, "ProductHandler."+operation,
		semconv.HTTPRouteKey.String(c.Route().Path),
		productIdAttr,
	)
	defer span.End()

	logger.Info("Handler: Received request for GetProductByID", zap.String("product_id", productID))

	simulateDelayIfEnabled()
	product, err := h.service.GetByID(ctx, productID)
	if err != nil {
		opErr = err
		simulateDelayIfEnabled()
		span.RecordError(opErr)
		if errors.Is(opErr, ErrNotFound) {
			logger.Warn("Handler: Product not found", zap.String("product_id", productID))
			span.SetStatus(codes.Error, opErr.Error())
		} else {
			logger.Error("Handler: Failed to get product by ID from service",
				zap.String("product_id", productID),
				zap.Error(opErr),
			)
			span.SetStatus(codes.Error, "service layer error")
		}
		return opErr
	}

	simulateDelayIfEnabled()
	logger.Info("Handler: Successfully retrieved product by ID", zap.String("product_id", productID))
	span.SetStatus(codes.Ok, "")
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) (opErr error) {
	const operation = "HealthCheck"
	startTime := time.Now()
	ctx := c.UserContext()
	defer func() {
		metric.RecordOperationMetrics(ctx, handlerLayerName, operation, startTime, opErr)
	}()

	simulateDelayIfEnabled()
	logger := logging.LoggerFromContext(ctx)
	logger.Info("Handler: Health check requested")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}
