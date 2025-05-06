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
	h.logger.DebugContext(c.UserContext(), "Shop open/close status requested")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (err error) {
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Front_Desk: Customer asking for all products list !!!")

	newCtx, span := commontrace.StartSpan(ctx)
	ctx = newCtx
	defer func() {
		var telemetryErr error
		if err != nil {
			telemetryErr = err
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		err = simAppErr
		return
	}

	h.logger.DebugContext(ctx, "Front Desk: waiting for all products list from shop manager")
	products, appErr := h.service.GetAll(ctx)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}
		err = appErr
		return
	}
	productCount := len(products)
	h.logger.InfoContext(ctx, "Front Desk: total "+strconv.Itoa(productCount)+" products list received from shop manager")
	span.SetAttributes(attribute.Int("products.count", productCount))
	h.logger.InfoContext(ctx, "Front Desk: Returning "+strconv.Itoa(productCount)+" products list to customer")
	err = c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(products))
	return
}

func (h *ProductHandler) GetProductsByCategory(c *fiber.Ctx) (err error) {
	ctx := c.UserContext()
	category := c.Query("category")
	h.logger.InfoContext(ctx, "Front_Desk: Customer asking for products in category: "+category)

	if category == "" {
		h.logger.WarnContext(ctx, "Front_Desk: Customer didn't specify a category")
		err = apierrors.NewAppError(apierrors.ErrCodeValidation, "Missing 'category' query parameter", nil)
		return
	}

	categoryAttr := attribute.String("product.category", category)
	newCtx, span := commontrace.StartSpan(ctx, categoryAttr)
	ctx = newCtx
	defer func() {
		var telemetryErr error
		if err != nil {
			telemetryErr = err
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		err = simAppErr
		return
	}

	h.logger.DebugContext(ctx, "Front Desk: waiting for category products list from shop manager for "+category)

	products, appErr := h.service.GetByCategory(ctx, category)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}
		err = appErr
		return
	}

	productCount := len(products)
	h.logger.InfoContext(ctx, "Front Desk: total "+strconv.Itoa(productCount)+" products received for category "+category)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))
	h.logger.InfoContext(ctx, "Front Desk: Returning "+strconv.Itoa(productCount)+" products in category "+category+" to customer")
	err = c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(products))
	return
}

func (h *ProductHandler) GetProductByName(c *fiber.Ctx) (err error) {
	ctx := c.UserContext()
	var req apirequests.GetByNameRequest
	if parseErr := c.BodyParser(&req); parseErr != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Invalid get product details request format", slog.String("error", parseErr.Error()))
		err = apierrors.NewAppError(apierrors.ErrCodeValidation, "Invalid request body format", parseErr)
		return
	}

	if validatorErr := validator.ValidateRequest(&req); validatorErr != nil {
		h.logger.WarnContext(ctx, "Front_Desk: Invalid request data", slog.String("validator_error", validatorErr.Message))
		err = validatorErr
		return
	}

	productName := req.Name
	h.logger.InfoContext(ctx, "Front_Desk: Customer asking for product details", slog.String("product_name", productName))
	productNameAttr := attribute.String("product.name", productName)

	newCtx, span := commontrace.StartSpan(ctx, productNameAttr)
	ctx = newCtx
	defer func() {
		var telemetryErr error
		if err != nil {
			telemetryErr = err
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		err = simAppErr
		return
	}

	h.logger.DebugContext(ctx, "Front Desk: waiting for product details from shop manager", slog.String("product_name", productName))

	product, appErr := h.service.GetByName(ctx, productName)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}
		err = appErr
		return
	}
	h.logger.InfoContext(ctx, "Front Desk: product details received", slog.String("product_name", productName))
	h.logger.InfoContext(ctx, "Front Desk: Returning product details to customer", slog.String("product_name", productName))
	err = c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(product))
	return
}

func (h *ProductHandler) UpdateProductStock(c *fiber.Ctx) (err error) {
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Front_Desk: Manager requesting stock update")

	var req apirequests.UpdateStockRequest
	if parseErr := c.BodyParser(&req); parseErr != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Invalid stock update request format", slog.String("error", parseErr.Error()))
		err = apierrors.NewAppError(apierrors.ErrCodeValidation, "Invalid request body format", parseErr)
		return
	}

	if validatorErr := validator.ValidateRequest(&req); validatorErr != nil {
		h.logger.WarnContext(ctx, "Front_Desk: Invalid stock update data", slog.String("validator_error", validatorErr.Message))
		err = validatorErr
		return
	}

	productName := req.Name
	newStock := req.Stock
	h.logger.DebugContext(ctx, "Front Desk: Manager wants to update stock", slog.String("product_name", productName), slog.Int("new_stock", newStock))

	newCtx, span := commontrace.StartSpan(c.UserContext(),
		attribute.String("product.name", productName),
		attribute.Int("product.update_stock_to", newStock))
	ctx = newCtx
	defer func() {
		var telemetryErr error
		if err != nil {
			telemetryErr = err
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		err = simAppErr
		return
	}

	h.logger.InfoContext(ctx, "Front Desk: Sending stock update request to shop manager")

	appErr := h.service.UpdateStock(ctx, productName, newStock)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}
		err = appErr
		return
	}

	h.logger.InfoContext(ctx, "Front Desk: Stock successfully updated", slog.String("product_name", productName), slog.Int("new_stock", newStock))
	err = c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(apiresponses.ActionConfirmation{Message: "Stock updated successfully"}))
	return
}

func (h *ProductHandler) BuyProduct(c *fiber.Ctx) (err error) {
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Front_Desk: Customer wants to buy a product")

	var req apirequests.ProductBuyRequest
	if parseErr := c.BodyParser(&req); parseErr != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Invalid purchase request format", slog.String("error", parseErr.Error()))
		err = apierrors.NewAppError(apierrors.ErrCodeValidation, "Invalid request body format", parseErr)
		return
	}

	if validatorErr := validator.ValidateRequest(&req); validatorErr != nil {
		h.logger.WarnContext(ctx, "Front Desk: Invalid purchase request data", slog.String("validator_error", validatorErr.Message))
		err = validatorErr
		return
	}

	productName := req.Name
	quantity := req.Quantity
	h.logger.DebugContext(ctx, "Front Desk: Customer wants to buy product", slog.String("product_name", productName), slog.Int("quantity", quantity))

	newCtx, span := commontrace.StartSpan(c.UserContext(),
		attribute.String("product.name", productName),
		attribute.Int("product.purchase_quantity", quantity))
	ctx = newCtx
	defer func() {
		var telemetryErr error
		if err != nil {
			telemetryErr = err
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		err = simAppErr
		return
	}

	h.logger.InfoContext(ctx, "Front Desk: Asking shop manager to process purchase", slog.String("product_name", productName), slog.Int("quantity", quantity))
	remainingStock, appErr := h.service.BuyProduct(ctx, productName, quantity)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}
		err = appErr
		return
	}

	h.logger.InfoContext(ctx, "Front Desk: Purchase successful!",
		slog.String("product_name", productName),
		slog.Int("quantity_bought", quantity),
		slog.Int("remaining_stock", remainingStock),
	)

	span.SetAttributes(attribute.Int("product.remaining_stock", remainingStock))
	err = c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(fiber.Map{
		"productName":    productName,
		"quantity":       quantity,
		"remainingStock": remainingStock,
	}))
	return
}
