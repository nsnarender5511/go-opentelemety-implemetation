package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"

	apierrors "github.com/narender/common/apierrors"
	apirequests "github.com/narender/common/apirequests"
	apiresponses "github.com/narender/common/apiresponses"
	"github.com/narender/common/validator"
	"go.opentelemetry.io/otel/codes"
)

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
	revenue, appErr := h.service.BuyProduct(ctx, productName, quantity)
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
		slog.Float64("revenue", revenue),
	)

	span.SetAttributes(attribute.Float64("product.revenue", revenue))
	err = c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(fiber.Map{
		"productName": productName,
		"quantity":    quantity,
		"revenue":     revenue,
	}))
	return
}
