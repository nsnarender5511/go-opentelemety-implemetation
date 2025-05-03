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
	"github.com/narender/common/globals"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const handlerScopeName = "github.com/narender/product-service/handler"

type ProductHandler struct {
	service ProductService
	logger  *slog.Logger
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func NewProductHandler(svc ProductService) *ProductHandler {
	return &ProductHandler{
		service: svc,
		logger:  globals.Logger(),
	}
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (opErr error) {
	const operation = "GetAllProducts"
	ctx := c.UserContext()

	mc := commonmetric.StartMetricsTimer(commonconst.HandlerLayer, operation)
	defer mc.End(ctx, &opErr)

	ctx, spanner := commontrace.StartSpan(ctx, handlerScopeName, operation, commonconst.HandlerLayer)
	defer spanner.End(&opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			h.logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.HandlerLayer))
		}
	}()

	h.logger.InfoContext(ctx, "Handler: Received request for GetAllProducts", slog.String("operation", operation))

	debugutils.Simulate(ctx)
	h.logger.InfoContext(ctx, "Handler: Calling service GetAll", slog.String("operation", operation))
	spanner.AddEvent("Calling service GetAll")
	products, err := h.service.GetAll(ctx)
	if err != nil {
		opErr = err
		spanner.AddEvent("Service GetAll failed")
		h.logger.WarnContext(ctx, "Service GetAll failed", slog.Any("error", opErr), slog.String("operation", operation))
		return opErr
	}
	productCount := len(products)
	spanner.AddEvent("Service GetAll successful", trace.WithAttributes(attribute.Int("products.count", productCount)))
	h.logger.InfoContext(ctx, "Successfully retrieved all products", slog.Int("productCount", productCount))
	spanner.SetAttributes(attribute.Int("products.count", productCount))
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) (opErr error) {
	const operation = "GetProductByID"
	ctx := c.UserContext()
	productID := c.Params("productId")
	productIdAttr := attribute.String("product.id", productID)

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
			h.logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.HandlerLayer), productIdAttr)
		}
	}()

	h.logger.InfoContext(ctx, "Handler: Received request for GetProductByID", slog.String("product_id", productID), slog.String("operation", operation))

	debugutils.Simulate(ctx)
	h.logger.InfoContext(ctx, "Handler: Calling service GetByID", slog.String("product_id", productID), slog.String("operation", operation))
	spanner.AddEvent("Calling service GetByID", trace.WithAttributes(productIdAttr))
	product, err := h.service.GetByID(ctx, productID)
	if err != nil {
		opErr = err
		spanner.AddEvent("Service GetByID failed", trace.WithAttributes(attribute.String("error.message", opErr.Error())))
		logLevel := slog.LevelWarn
		if errors.Is(opErr, commonerrors.ErrNotFound) {
			logLevel = slog.LevelInfo
		}
		h.logger.Log(ctx, logLevel, "Service GetByID failed", slog.Any("error", opErr), slog.String("product_id", productID))
		return opErr
	}
	spanner.AddEvent("Service GetByID successful")
	h.logger.InfoContext(ctx, "Successfully retrieved product by ID", slog.String("product_id", productID))
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	ctx := c.UserContext()
	const operation = "HealthCheckHandler"
	h.logger.InfoContext(ctx, "Health check requested", slog.String("operation", operation))
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}
