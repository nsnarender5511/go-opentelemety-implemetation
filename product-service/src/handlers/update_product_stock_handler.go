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
