package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/debugutils"
	commonerrors "github.com/narender/common/errors"
	"github.com/narender/common/globals"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type ProductHandler struct {
	service ProductService
	logger  *slog.Logger
}

func NewProductHandler(svc ProductService) *ProductHandler {
	return &ProductHandler{
		service: svc,
		logger:  globals.Logger(),
	}
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()

	ctx, span := commontrace.StartSpan(ctx)
	span.SetAttributes(semconv.HTTPRouteKey.String(c.Route().Path))
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %v", rec)
			h.logger.Error("Panic recovered", slog.Any("panic", rec))
		}
	}()

	h.logger.InfoContext(ctx, "Handler: Received request for GetAllProducts")

	debugutils.Simulate(ctx)
	h.logger.InfoContext(ctx, "Handler: Calling service GetAll")
	span.AddEvent("Calling service GetAll")
	products, err := h.service.GetAll(ctx)
	if err != nil {
		opErr = err
		span.AddEvent("Service GetAll failed")
		h.logger.WarnContext(ctx, "Service GetAll failed", slog.Any("error", opErr))
		return opErr
	}
	productCount := len(products)
	span.AddEvent("Service GetAll successful", trace.WithAttributes(attribute.Int("products.count", productCount)))
	h.logger.InfoContext(ctx, "Successfully retrieved all products", slog.Int("productCount", productCount))
	span.SetAttributes(attribute.Int("products.count", productCount))
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()
	productID := c.Params("productId")
	productIdAttr := attribute.String("product.id", productID)

	ctx, span := commontrace.StartSpan(ctx, productIdAttr)
	span.SetAttributes(semconv.HTTPRouteKey.String(c.Route().Path))
	notFoundMapper := func(err error) codes.Code {
		if err == nil {
			return codes.Ok
		}
		if errors.Is(err, commonerrors.ErrNotFound) {
			return codes.Ok
		}
		return codes.Error
	}
	defer commontrace.EndSpan(span, &opErr, notFoundMapper)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %v", rec)
			h.logger.Error("Panic recovered", slog.Any("panic", rec), productIdAttr)
		}
	}()

	h.logger.InfoContext(ctx, "Handler: Received request for GetProductByID", slog.String("product_id", productID))

	debugutils.Simulate(ctx)
	h.logger.InfoContext(ctx, "Handler: Calling service GetByID", slog.String("product_id", productID))
	span.AddEvent("Calling service GetByID", trace.WithAttributes(productIdAttr))
	product, err := h.service.GetByID(ctx, productID)
	if err != nil {
		opErr = err
		span.AddEvent("Service GetByID failed", trace.WithAttributes(attribute.String("error.message", opErr.Error())))
		logLevel := slog.LevelWarn
		if errors.Is(opErr, commonerrors.ErrNotFound) {
			logLevel = slog.LevelInfo
		}
		h.logger.Log(ctx, logLevel, "Service GetByID failed", slog.Any("error", opErr), slog.String("product_id", productID))
		return opErr
	}
	span.AddEvent("Service GetByID successful")
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
