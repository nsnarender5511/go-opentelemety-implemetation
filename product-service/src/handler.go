package main

import (
	"fmt"
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

	h.logger.InfoContext(ctx, "Handler: Received request for GetAllProducts")

	h.logger.InfoContext(ctx, "Handler: Calling service GetAll")
	products, err := h.service.GetAll(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "Service GetAll failed", slog.String("error", err.Error()))
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
	productCount := len(products)
	h.logger.InfoContext(ctx, "Successfully retrieved all products", slog.Int("productCount", productCount))
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

	h.logger.InfoContext(ctx, "Handler: Received request for GetProductByID", slog.String("product_id", productID), slog.String("operation", operation))

	debugutils.Simulate(ctx)
	h.logger.InfoContext(ctx, "Handler: Calling service GetByID", slog.String("product_id", productID), slog.String("operation", operation))
	product, err := h.service.GetByID(ctx, productID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Service GetByID failed", slog.String("error", err.Error()))
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
	h.logger.InfoContext(ctx, "Successfully retrieved product by ID", slog.String("product_id", productID), slog.String("operation", operation))
	return c.Status(http.StatusOK).JSON(product)
}

// UpdateStockRequest defines the expected structure for the update stock request body.
type UpdateStockRequest struct {
	Stock int `json:"stock"`
}

func (h *ProductHandler) UpdateProductStock(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()
	productID := c.Params("productId")
	productIdAttr := attribute.String("product.id", productID)
	const operation = "UpdateProductStockHandler"

	ctx, span := commontrace.StartSpan(ctx, productIdAttr)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	h.logger.InfoContext(ctx, "Handler: Received request for UpdateProductStock", slog.String("product_id", productID), slog.String("operation", operation))

	// Parse request body
	var req UpdateStockRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.ErrorContext(ctx, "Failed to parse request body", slog.String("error", err.Error()), slog.String("product_id", productID))
		if span != nil {
			span.SetStatus(codes.Error, "invalid_request_body")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}
		// Return 400 Bad Request for parsing errors
		return fiber.NewError(http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
	}

	// Optional: Add basic validation (e.g., non-negative stock)
	if req.Stock < 0 {
		h.logger.WarnContext(ctx, "Invalid stock value provided", slog.Int("stock", req.Stock), slog.String("product_id", productID))
		if span != nil {
			span.SetStatus(codes.Error, "invalid_stock_value")
			span.SetAttributes(attribute.Int("invalid.stock.value", req.Stock))
		}
		return fiber.NewError(http.StatusBadRequest, "Stock value cannot be negative")
	}

	newStockAttr := attribute.Int("product.new_stock", req.Stock)
	span.SetAttributes(newStockAttr)

	debugutils.Simulate(ctx)
	h.logger.InfoContext(ctx, "Handler: Calling service UpdateStock", slog.String("product_id", productID), slog.Int("new_stock", req.Stock), slog.String("operation", operation))
	err := h.service.UpdateStock(ctx, productID, req.Stock)
	if err != nil {
		// Specific error handling will be done by the global error handler
		// based on the error type returned by the service (e.g., ErrNotFound)
		h.logger.ErrorContext(ctx, "Service UpdateStock failed", slog.String("error", err.Error()), slog.String("product_id", productID), slog.Int("new_stock", req.Stock))
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		return err // Propagate error to the global error handler
	}

	h.logger.InfoContext(ctx, "Successfully updated product stock", slog.String("product_id", productID), slog.Int("new_stock", req.Stock), slog.String("operation", operation))
	return c.SendStatus(http.StatusOK) // Send 200 OK with no body on successful update
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	const operation = "HealthCheckHandler"
	h.logger.InfoContext(c.UserContext(), "Health check requested", slog.String("operation", operation))
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}
