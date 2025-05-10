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

	h.logger.InfoContext(ctx, "Purchase request received",
		slog.String("component", "product_handler"),
		slog.String("operation", "buy_product"),
		slog.String("user_agent", c.Get("User-Agent")))

	var req apirequests.ProductBuyRequest
	if parseErr := c.BodyParser(&req); parseErr != nil {
		h.logger.WarnContext(ctx, "Request rejected: invalid request format",
			slog.String("component", "product_handler"),
			slog.String("error", parseErr.Error()),
			slog.String("operation", "buy_product"))
		
		

		err = apierrors.NewApplicationError(
			apierrors.ErrCodeRequestValidation,
			"Invalid request body format",
			parseErr)
		return
	}

	if validatorErr := validator.ValidateRequest(&req); validatorErr != nil {
		h.logger.WarnContext(ctx, "Request validation failed",
			slog.String("component", "product_handler"),
			slog.String("operation", "buy_product"),
			slog.String("error", validatorErr.Error()))
		err = validatorErr
		return
	}

	productName := req.Name
	quantity := req.Quantity

	h.logger.DebugContext(ctx, "Processing purchase details",
		slog.String("component", "product_handler"),
		slog.String("operation", "buy_product"),
		slog.String("product_name", productName),
		slog.Int("quantity", quantity))

	newCtx, span := commontrace.StartSpan(ctx, "product_handler", "buy_product",
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

	h.logger.InfoContext(ctx, "Processing purchase request",
		slog.String("component", "product_handler"),
		slog.String("product_name", productName),
		slog.Int("quantity", quantity),
		slog.String("operation", "buy_product"))

	revenue, appErr := h.service.BuyProduct(ctx, productName, quantity)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}

		err = appErr
		return
	}

	h.logger.InfoContext(ctx, "Purchase completed successfully",
		slog.String("component", "product_handler"),
		slog.String("product_name", productName),
		slog.Int("quantity", quantity),
		slog.Float64("revenue", revenue),
		slog.String("operation", "buy_product"),
		slog.String("status", "success"))

	span.SetAttributes(attribute.Float64("product.revenue", revenue))

	response := apiresponses.NewSuccessResponse(fiber.Map{
		"productName": productName,
		"quantity":    quantity,
		"revenue":     revenue,
	})

	err = c.Status(http.StatusOK).JSON(response)
	return
}
