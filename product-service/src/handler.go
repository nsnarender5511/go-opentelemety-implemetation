package main

import (
	"errors"
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
	const operation = "GetAllProductsHandler"

	ctx, span := commontrace.StartSpan(ctx)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	h.logger.Info("Handler: Received request for GetAllProducts", slog.String("operation", operation))

	debugutils.Simulate(ctx)
	h.logger.Info( "Handler: Calling service GetAll", slog.String("operation", operation))
	span.AddEvent("Calling service GetAll")
	products, err := h.service.GetAll(ctx)
	if err != nil {
		opErr = err
		logLevel := slog.LevelError
		eventName := "error"
		if errors.Is(opErr, commonerrors.ErrNotFound) {
			logLevel = slog.LevelWarn
			eventName = "resource_not_found"
		}
		h.logger.Log(ctx, logLevel, "Service GetAll failed",
			slog.String("layer", "handler"),
			slog.String("operation", operation),
			slog.String("error", opErr.Error()),
		)
		if span != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "handler"),
				attribute.String("operation", operation),
				attribute.String("error.message", opErr.Error()),
			}
			if errors.Is(opErr, commonerrors.ErrNotFound) {
				spanAttrs = append(spanAttrs, attribute.Bool("error.expected", true))
			}
			span.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
			if !errors.Is(opErr, commonerrors.ErrNotFound) {
				span.SetStatus(codes.Error, opErr.Error())
			}
		}
		return opErr
	}
	productCount := len(products)
	span.AddEvent("Service GetAll successful", trace.WithAttributes(attribute.Int("products.count", productCount)))
	h.logger.Info( "Successfully retrieved all products", slog.Int("productCount", productCount), slog.String("operation", operation))
	span.SetAttributes(attribute.Int("products.count", productCount))
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()
	productID := c.Params("productId")
	productIdAttr := attribute.String("product.id", productID)
	const operation = "GetProductByIDHandler"

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

	h.logger.Info( "Handler: Received request for GetProductByID", slog.String("product_id", productID), slog.String("operation", operation))

	debugutils.Simulate(ctx)
	h.logger.Info( "Handler: Calling service GetByID", slog.String("product_id", productID), slog.String("operation", operation))
	span.AddEvent("Calling service GetByID", trace.WithAttributes(productIdAttr))
	product, err := h.service.GetByID(ctx, productID)
	if err != nil {
		opErr = err
		logLevel := slog.LevelError
		eventName := "error"
		if errors.Is(opErr, commonerrors.ErrNotFound) {
			logLevel = slog.LevelWarn
			eventName = "resource_not_found"
		}
		h.logger.Log(ctx, logLevel, "Service GetByID failed",
			slog.String("layer", "handler"),
			slog.String("operation", operation),
			slog.String("error", opErr.Error()),
			slog.String("product_id", productID),
		)
		if span != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "handler"),
				attribute.String("operation", operation),
				attribute.String("error.message", opErr.Error()),
				productIdAttr,
			}
			if errors.Is(opErr, commonerrors.ErrNotFound) {
				spanAttrs = append(spanAttrs, attribute.Bool("error.expected", true))
			}
			span.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
			if !errors.Is(opErr, commonerrors.ErrNotFound) {
				span.SetStatus(codes.Error, opErr.Error())
			}
		}
		return opErr
	}
	span.AddEvent("Service GetByID successful")
	h.logger.Info( "Successfully retrieved product by ID", slog.String("product_id", productID), slog.String("operation", operation))
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	const operation = "HealthCheckHandler"
	h.logger.Info( "Health check requested", slog.String("operation", operation))
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}
