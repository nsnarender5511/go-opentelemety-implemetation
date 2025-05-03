package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	commonconst "github.com/narender/common/constants"
	"github.com/narender/common/debugutils"
	commonerrors "github.com/narender/common/errors"
	commonlog "github.com/narender/common/log"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const handlerScopeName = "github.com/narender/product-service/handler"

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

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (opErr error) {
	const operation = "GetAllProducts"
	ctx := c.UserContext()
	logger := commonlog.L
	logger.DebugContext(ctx, "Entering GetAllProducts handler", slog.String("operation", operation))

	mc := commonmetric.StartMetricsTimer(commonconst.HandlerLayer, operation)
	defer mc.End(ctx, &opErr)

	ctx, spanner := commontrace.StartSpan(ctx, handlerScopeName, operation, commonconst.HandlerLayer)
	defer spanner.End(&opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.HandlerLayer))
		}
	}()

	logger.InfoContext(ctx, "Handler: Received request for GetAllProducts", slog.String("operation", operation))

	debugutils.Simulate(ctx)
	logger.InfoContext(ctx, "Handler: Calling service GetAll", slog.String("operation", operation))
	spanner.AddEvent("Calling service GetAll")
	products, err := h.service.GetAll(ctx)
	if err != nil {
		opErr = err
		spanner.AddEvent("Service GetAll failed")
		return opErr
	}
	productCount := len(products)
	spanner.AddEvent("Service GetAll successful", trace.WithAttributes(attribute.Int("products.count", productCount)))
	logger.InfoContext(ctx, "Handler: Service GetAll returned successfully", slog.Int("productCount", productCount), slog.String("operation", operation))

	debugutils.Simulate(ctx)
	logger.InfoContext(ctx, "Handler: Successfully retrieved all products", slog.Int("productCount", productCount))
	spanner.SetAttributes(attribute.Int("products.count", productCount))
	logger.DebugContext(ctx, "Handler: Preparing successful response", slog.String("operation", operation), slog.Int("productCount", productCount))
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) (opErr error) {
	const operation = "GetProductByID"
	ctx := c.UserContext()
	logger := commonlog.L
	productID := c.Params("productId")
	productIdAttr := attribute.String("product.id", productID)
	logger.DebugContext(ctx, "Entering GetProductByID handler", slog.String("operation", operation), productIdAttr)

	mc := commonmetric.StartMetricsTimer(commonconst.HandlerLayer, operation)
	defer mc.End(ctx, &opErr, productIdAttr)

	ctx, spanner := commontrace.StartSpan(ctx, handlerScopeName, operation, commonconst.HandlerLayer, productIdAttr)
	notFoundMapper := func(err error) codes.Code {
		if err == nil {
			return codes.Ok
		}
		if errors.Is(err, commonerrors.ErrNotFound) {
			return codes.Ok
		}
		return codes.Error
	}
	defer spanner.End(&opErr, notFoundMapper)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.HandlerLayer), productIdAttr)
		}
	}()

	logger.InfoContext(ctx, "Handler: Received request for GetProductByID", slog.String("product_id", productID), slog.String("operation", operation))

	debugutils.Simulate(ctx)
	logger.InfoContext(ctx, "Handler: Calling service GetByID", slog.String("product_id", productID), slog.String("operation", operation))
	spanner.AddEvent("Calling service GetByID", trace.WithAttributes(productIdAttr))
	product, err := h.service.GetByID(ctx, productID)
	if err != nil {
		opErr = err
		spanner.AddEvent("Service GetByID failed", trace.WithAttributes(attribute.String("error.message", opErr.Error())))
		return opErr
	}
	spanner.AddEvent("Service GetByID successful")
	logger.InfoContext(ctx, "Handler: Service GetByID returned successfully", slog.String("product_id", productID), slog.String("operation", operation))

	debugutils.Simulate(ctx)
	logger.InfoContext(ctx, "Handler: Successfully retrieved product by ID", slog.String("product_id", productID))
	logger.DebugContext(ctx, "Handler: Preparing successful response", slog.String("operation", operation), productIdAttr)
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger := commonlog.L
	const operation = "HealthCheckHandler"
	logger.DebugContext(ctx, "Entering HealthCheck handler", slog.String("operation", operation))
	logger.InfoContext(ctx, "Handler: Health check requested", slog.String("operation", operation))
	logger.DebugContext(ctx, "Handler: Preparing health check response", slog.String("operation", operation))
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

type UpdateStockRequest struct {
	NewStock int `json:"new_stock"`
}

func (h *ProductHandler) UpdateStock(c *fiber.Ctx) (opErr error) {
	const operation = "UpdateStockHandler"
	ctx := c.UserContext()
	logger := commonlog.L
	productID := c.Params("productId")
	productIdAttr := attribute.String("product.id", productID)
	logger.DebugContext(ctx, "Entering UpdateStock handler", slog.String("operation", operation), productIdAttr)

	mc := commonmetric.StartMetricsTimer(commonconst.HandlerLayer, operation)
	defer mc.End(ctx, &opErr, productIdAttr)

	ctx, spanner := commontrace.StartSpan(ctx, handlerScopeName, operation, commonconst.HandlerLayer, productIdAttr)
	// Custom mapper to treat validation/not found as OK for span status, but still return error to client
	updateStockSpanMapper := func(err error) codes.Code {
		if err == nil {
			return codes.Ok
		}
		if errors.Is(err, commonerrors.ErrNotFound) || errors.Is(err, commonerrors.ErrValidation) {
			return codes.Ok
		}
		return codes.Error
	}
	defer spanner.End(&opErr, updateStockSpanMapper)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.HandlerLayer), productIdAttr)
		}
	}()

	logger.InfoContext(ctx, "Handler: Received request for UpdateStock", slog.String("product_id", productID))

	// Parse request body
	var req UpdateStockRequest
	if err := c.BodyParser(&req); err != nil {
		// Use ErrValidation for bad request body
		opErr = fmt.Errorf("%w: failed to parse request body: %v", commonerrors.ErrValidation, err)
		logger.WarnContext(ctx, "Failed to parse update stock request body", slog.Any("error", err), productIdAttr)
		spanner.SetAttributes(attribute.Bool("request.body.parse.error", true))
		return opErr // Return validation error
	}
	spanner.SetAttributes(attribute.Int("request.new_stock", req.NewStock))
	newStockAttr := attribute.Int("product.new_stock", req.NewStock)

	logger.InfoContext(ctx, "Handler: Calling service UpdateStock", slog.String("product_id", productID), slog.Int("new_stock", req.NewStock))
	spanner.AddEvent("Calling service UpdateStock", trace.WithAttributes(productIdAttr, newStockAttr))
	err := h.service.UpdateStock(ctx, productID, req.NewStock)
	if err != nil {
		opErr = err // Assign service error to opErr to be handled by middleware
		spanner.AddEvent("Service UpdateStock failed", trace.WithAttributes(attribute.String("error.message", opErr.Error())))
		logger.WarnContext(ctx, "Service UpdateStock failed", slog.Any("error", err), productIdAttr, newStockAttr)
		return opErr // Return the specific error (NotFound, Validation, Internal)
	}
	spanner.AddEvent("Service UpdateStock successful")
	logger.InfoContext(ctx, "Handler: Service UpdateStock completed successfully", slog.String("product_id", productID), slog.Int("new_stock", req.NewStock))

	debugutils.Simulate(ctx)
	logger.InfoContext(ctx, "Handler: Successfully updated product stock", slog.String("product_id", productID), slog.Int("new_stock", req.NewStock))
	logger.DebugContext(ctx, "Handler: Preparing successful empty response", slog.String("operation", operation), productIdAttr)
	return c.SendStatus(http.StatusOK)
}
