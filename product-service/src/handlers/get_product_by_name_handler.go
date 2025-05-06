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
