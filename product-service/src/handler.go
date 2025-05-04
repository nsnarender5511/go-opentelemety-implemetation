package main

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/debugutils"
	"github.com/narender/common/globals"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	h.logger.Info("Handler: Received request for GetAllProducts")

	h.logger.Info( "Handler: Calling service GetAll")
	products, err := h.service.GetAll(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "Service GetAll failed", slog.String("error", err.Error()))
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
	productCount := len(products)
	h.logger.Info( "Successfully retrieved all products", slog.Int("productCount", productCount))
	span.SetAttributes(attribute.Int("products.count", productCount))
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()
	productID := c.Params("productId")
	productIdAttr := attribute.String("product.id", productID)
	const operation = "GetProductByIDHandler"

	ctx, span := commontrace.StartSpan(ctx, productIdAttr)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	h.logger.Info( "Handler: Received request for GetProductByID", slog.String("product_id", productID), slog.String("operation", operation))

	debugutils.Simulate(ctx)
	h.logger.Info( "Handler: Calling service GetByID", slog.String("product_id", productID), slog.String("operation", operation))
	product, err := h.service.GetByID(ctx, productID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Service GetByID failed", slog.String("error", err.Error()))
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
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
