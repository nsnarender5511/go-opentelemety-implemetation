package main

import (
	"errors"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	commonlog "github.com/narender/common/log"
	"github.com/narender/common/telemetry"
	commonmetric "github.com/narender/common/telemetry/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const handlerScopeName = "github.com/narender/product-service/handler"
const handlerLayerName = "handler"

type ProductHandler struct {
	service ProductService
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func NewProductHandler(svc ProductService) *ProductHandler {
	return &ProductHandler{
		service: svc,
	}
}

func simulateDelayIfEnabled() {
	if appConfig != nil && appConfig.SimulateDelayEnabled {
		if appConfig.SimulateDelayMinMs > 0 && appConfig.SimulateDelayMaxMs >= appConfig.SimulateDelayMinMs {
			delayRange := appConfig.SimulateDelayMaxMs - appConfig.SimulateDelayMinMs
			delay := rand.Intn(delayRange+1) + appConfig.SimulateDelayMinMs
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (opErr error) {
	const operation = "GetAllProducts"
	startTime := time.Now()
	ctx := c.UserContext()
	defer func() {
		commonmetric.RecordOperationMetrics(ctx, handlerLayerName, operation, startTime, opErr)
	}()

	simulateDelayIfEnabled()
	logger := commonlog.L
	tracer := telemetry.GetTracer(handlerScopeName)
	ctx, span := tracer.Start(ctx, "ProductHandler.GetAllProducts", oteltrace.WithSpanKind(oteltrace.SpanKindServer))
	defer span.End()

	logger.Info("Handler: Received request for GetAllProducts")

	simulateDelayIfEnabled()
	products, err := h.service.GetAll(ctx)
	if err != nil {
		opErr = err
		simulateDelayIfEnabled()
		logger.Error("Handler: Failed to get all products from service", slog.Any("error", opErr))
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
		commonmetric.RecordOperationMetrics(ctx, handlerLayerName, operation, startTime, opErr, productIdAttr)
	}()

	simulateDelayIfEnabled()
	logger := commonlog.L
	tracer := telemetry.GetTracer(handlerScopeName)
	ctx, span := tracer.Start(ctx, "ProductHandler.GetProductByID", oteltrace.WithAttributes(productIdAttr), oteltrace.WithSpanKind(oteltrace.SpanKindServer))
	defer span.End()

	logger.Info("Handler: Received request for GetProductByID", slog.String("product_id", productID))

	meter := telemetry.GetMeter(handlerScopeName)
	requestCounter, _ := meter.Int64Counter("product_service.requests", otelmetric.WithDescription("Counts requests to product service endpoints"))
	if requestCounter != nil {
		requestCounter.Add(ctx, 1, otelmetric.WithAttributes(
			attribute.String("http.route", c.Path()),
			attribute.String("product.id.param", productID),
		))
	}

	simulateDelayIfEnabled()
	product, err := h.service.GetByID(ctx, productID)
	if err != nil {
		opErr = err
		simulateDelayIfEnabled()
		span.RecordError(opErr)
		if errors.Is(opErr, ErrNotFound) {
			logger.Warn("Handler: Product not found", slog.String("product_id", productID))
			span.SetStatus(codes.Error, opErr.Error())
		} else {
			logger.Error("Handler: Failed to get product by ID from service",
				slog.String("product_id", productID),
				slog.Any("error", opErr),
			)
			span.SetStatus(codes.Error, "service layer error")
		}
		return opErr
	}

	simulateDelayIfEnabled()
	logger.Info("Handler: Successfully retrieved product by ID", slog.String("product_id", productID))
	span.SetStatus(codes.Ok, "")
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) (opErr error) {
	const operation = "HealthCheck"
	startTime := time.Now()
	ctx := c.UserContext()
	defer func() {
		commonmetric.RecordOperationMetrics(ctx, handlerLayerName, operation, startTime, opErr)
	}()

	simulateDelayIfEnabled()
	logger := commonlog.L
	logger.Info("Handler: Health check requested")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}
