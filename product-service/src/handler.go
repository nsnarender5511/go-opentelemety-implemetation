package main

import (
	"log/slog"
	"net/http"

	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/debugutils"
	"github.com/narender/common/globals"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"

	// Import common packages
	apierrors "github.com/narender/common/apierrors"
	apiresponses "github.com/narender/common/apiresponses"
	validator "github.com/narender/common/validator"

	// Import common requests
	apirequests "github.com/narender/common/apirequests"
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
	h.logger.InfoContext(c.UserContext(), "Shop open/close status requested")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Front_Desk: Customer asking for all products list !!!")

	ctx, span := commontrace.StartSpan(ctx)
	defer commontrace.EndSpan(span, nil, nil)

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return simAppErr
	}

	h.logger.DebugContext(ctx, "Front Desk: waiting for all products list from shop manager")
	products, appErr := h.service.GetAll(ctx)
	if appErr != nil {
		return appErr
	}
	productCount := len(products)
	h.logger.InfoContext(ctx, "Front Desk: total "+strconv.Itoa(productCount)+" products list received from shop manager")
	span.SetAttributes(attribute.Int("products.count", productCount))
	h.logger.InfoContext(ctx, "Front Desk: Returning "+strconv.Itoa(productCount)+" products list to customer")
	return c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(products))
}

func (h *ProductHandler) GetProductsByCategory(c *fiber.Ctx) error {
	ctx := c.UserContext()
	category := c.Query("category")
	h.logger.InfoContext(ctx, "Front_Desk: Customer asking for products in category: "+category)

	if category == "" {
		h.logger.WarnContext(ctx, "Front_Desk: Customer didn't specify a category")
		return apierrors.NewAppError(apierrors.ErrCodeValidation, "Missing 'category' query parameter", nil)
	}

	categoryAttr := attribute.String("product.category", category)
	ctx, span := commontrace.StartSpan(ctx, categoryAttr)
	defer commontrace.EndSpan(span, nil, nil)

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return simAppErr
	}

	h.logger.DebugContext(ctx, "Front Desk: waiting for category products list from shop manager for "+category)

	products, appErr := h.service.GetByCategory(ctx, category)
	if appErr != nil {
		return appErr
	}

	productCount := len(products)
	h.logger.InfoContext(ctx, "Front Desk: total "+strconv.Itoa(productCount)+" products received for category "+category)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))
	h.logger.InfoContext(ctx, "Front Desk: Returning "+strconv.Itoa(productCount)+" products in category "+category+" to customer")
	return c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(products))
}

func (h *ProductHandler) GetProductByName(c *fiber.Ctx) error {
	ctx := c.UserContext()
	ctx, span := commontrace.StartSpan(ctx)
	defer commontrace.EndSpan(span, nil, nil)

	var req apirequests.GetByNameRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Invalid get product details request format", slog.String("error", err.Error()))
		return apierrors.NewAppError(apierrors.ErrCodeValidation, "Invalid request body format", err)
	}

	if validatorErr := validator.ValidateRequest(&req); validatorErr != nil {
		h.logger.WarnContext(ctx, "Front_Desk: Invalid request data", slog.String("validator_error", validatorErr.Message))
		return validatorErr
	}

	productName := req.Name
	h.logger.InfoContext(ctx, "Front_Desk: Customer asking for product details", slog.String("product_name", productName))
	productNameAttr := attribute.String("product.name", productName)
	span.SetAttributes(productNameAttr)

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return simAppErr
	}

	h.logger.DebugContext(ctx, "Front Desk: waiting for product details from shop manager", slog.String("product_name", productName))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return simAppErr
	}

	product, appErr := h.service.GetByName(ctx, productName)
	if appErr != nil {
		return appErr
	}
	h.logger.InfoContext(ctx, "Front Desk: product details received", slog.String("product_name", productName))
	h.logger.InfoContext(ctx, "Front Desk: Returning product details to customer", slog.String("product_name", productName))
	return c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(product))
}

func (h *ProductHandler) UpdateProductStock(c *fiber.Ctx) error {
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Front_Desk: Manager requesting stock update")

	ctx, span := commontrace.StartSpan(c.UserContext())
	defer commontrace.EndSpan(span, nil, nil)

	var req apirequests.UpdateStockRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Invalid stock update request format", slog.String("error", err.Error()))
		return apierrors.NewAppError(apierrors.ErrCodeValidation, "Invalid request body format", err)
	}

	if validatorErr := validator.ValidateRequest(&req); validatorErr != nil {
		h.logger.WarnContext(ctx, "Front_Desk: Invalid stock update data", slog.String("validator_error", validatorErr.Message))
		return validatorErr
	}

	productName := req.Name
	newStock := req.Stock

	h.logger.DebugContext(ctx, "Front Desk: Manager wants to update stock", slog.String("product_name", productName), slog.Int("new_stock", newStock))
	span.SetAttributes(attribute.String("product.name", productName))
	span.SetAttributes(attribute.Int("product.update_stock_to", newStock))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return simAppErr
	}

	h.logger.InfoContext(ctx, "Front Desk: Sending stock update request to shop manager")

	appErr := h.service.UpdateStock(ctx, productName, newStock)
	if appErr != nil {
		return appErr
	}

	h.logger.InfoContext(ctx, "Front Desk: Stock successfully updated", slog.String("product_name", productName), slog.Int("new_stock", newStock))
	return c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(apiresponses.ActionConfirmation{Message: "Stock updated successfully"}))
}

func (h *ProductHandler) BuyProduct(c *fiber.Ctx) error {
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Front_Desk: Customer wants to buy a product")

	ctx, span := commontrace.StartSpan(c.UserContext())
	defer commontrace.EndSpan(span, nil, nil)

	var req apirequests.ProductBuyRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Invalid purchase request format", slog.String("error", err.Error()))
		return apierrors.NewAppError(apierrors.ErrCodeValidation, "Invalid request body format", err)
	}

	if validatorErr := validator.ValidateRequest(&req); validatorErr != nil {
		h.logger.WarnContext(ctx, "Front Desk: Invalid purchase request data", slog.String("validator_error", validatorErr.Message))
		return validatorErr
	}

	productName := req.Name
	quantity := req.Quantity

	h.logger.DebugContext(ctx, "Front Desk: Customer wants to buy product", slog.String("product_name", productName), slog.Int("quantity", quantity))
	span.SetAttributes(attribute.String("product.name", productName))
	span.SetAttributes(attribute.Int("product.purchase_quantity", quantity))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return simAppErr
	}

	h.logger.InfoContext(ctx, "Front Desk: Asking shop manager to process purchase", slog.String("product_name", productName), slog.Int("quantity", quantity))
	remainingStock, appErr := h.service.BuyProduct(ctx, productName, quantity)
	if appErr != nil {
		return appErr
	}

	h.logger.InfoContext(ctx, "Front Desk: Purchase successful!",
		slog.String("product_name", productName),
		slog.Int("quantity_bought", quantity),
		slog.Int("remaining_stock", remainingStock),
	)

	span.SetAttributes(attribute.Int("product.remaining_stock", remainingStock))
	return c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(fiber.Map{
		"productName":    productName,
		"quantity":       quantity,
		"remainingStock": remainingStock,
	}))
}
