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

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	h.logger.InfoContext(c.UserContext(), "Health check requested")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()

	ctx, span := commontrace.StartSpan(ctx)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	h.logger.InfoContext(ctx, "Handler: Received request for GetAllProducts")

	h.logger.InfoContext(ctx, "Handler: Calling service GetAll")
	products, err := h.service.GetAll(ctx)
	if err != nil {
		opErr = err
		h.logger.ErrorContext(ctx, "Service GetAll failed",
			slog.String("error", opErr.Error()),
		)
		span.SetStatus(codes.Error, opErr.Error())
		return opErr
	}
	productCount := len(products)
	h.logger.InfoContext(ctx, "Successfully retrieved all products", slog.Int("productCount", productCount))
	span.SetAttributes(attribute.Int("products.count", productCount))
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()
	productID := c.Params("productId")

	ctx, span := commontrace.StartSpan(ctx,
		attribute.String("product.id", productID),
	)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	h.logger.InfoContext(ctx, "Handler: Received request for GetProductByID", slog.String("product_id", productID))

	product, err := h.service.GetByID(ctx, productID)
	if err != nil {
		opErr = err
		h.logger.ErrorContext(ctx, "Service GetByID failed",
			slog.String("error", opErr.Error()),
			slog.String("product_id", productID),
		)
		span.SetStatus(codes.Error, opErr.Error())
		return opErr
	}
	span.AddEvent("Service GetByID successful")
	h.logger.InfoContext(ctx, "Successfully retrieved product by ID", slog.String("product_id", productID))
	return c.Status(http.StatusOK).JSON(product)
}
