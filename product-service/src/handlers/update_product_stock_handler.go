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

	h.logger.InfoContext(ctx, "Stock update request received",
		slog.String("component", "product_handler"),
		slog.String("path", c.Path()),
		slog.String("method", c.Method()),
		slog.String("operation", "update_product_stock"),
		slog.String("event_type", "stock_update_initiated"))

	var req apirequests.UpdateStockRequest
	if parseErr := c.BodyParser(&req); parseErr != nil {
		h.logger.WarnContext(ctx, "Request rejected: invalid request format",
			slog.String("component", "product_handler"),
			slog.String("error", parseErr.Error()),
			slog.String("error_code", apierrors.ErrCodeRequestValidation),
			slog.String("path", c.Path()),
			slog.String("operation", "update_product_stock"))

		err = apierrors.NewApplicationError(
			apierrors.ErrCodeRequestValidation,
			"Invalid request body format",
			parseErr)
		return
	}

	if validatorErr := validator.ValidateRequest(&req); validatorErr != nil {
		h.logger.WarnContext(ctx, "Request validation failed",
			slog.String("component", "product_handler"),
			slog.String("validator_error", validatorErr.Message),
			slog.String("error_code", validatorErr.Code),
			slog.String("path", c.Path()),
			slog.String("operation", "update_product_stock"),
			slog.String("event_type", "request_validation_failed"))

		err = validatorErr
		return
	}

	productName := req.Name
	newStock := req.Stock

	h.logger.DebugContext(ctx, "Processing stock update request",
		slog.String("component", "product_handler"),
		slog.String("product_name", productName),
		slog.Int("new_stock", newStock),
		slog.String("operation", "update_product_stock"))

	newCtx, span := commontrace.StartSpan(c.UserContext(), "product_handler", "update_product_stock",
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

	h.logger.InfoContext(ctx, "Updating product stock",
		slog.String("component", "product_handler"),
		slog.String("product_name", productName),
		slog.Int("new_stock", newStock),
		slog.String("operation", "update_product_stock"),
		slog.String("event_type", "stock_update_processing"))

	appErr := h.service.UpdateStock(ctx, productName, newStock)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}

		err = appErr
		return
	}

	h.logger.InfoContext(ctx, "Stock update completed successfully",
		slog.String("component", "product_handler"),
		slog.String("product_name", productName),
		slog.Int("new_stock", newStock),
		slog.String("operation", "update_product_stock"),
		slog.String("status", "success"),
		slog.String("event_type", "stock_update_completed"))

	// Create response without RequestID
	response := apiresponses.NewSuccessResponse(
		apiresponses.ActionConfirmation{Message: "Stock updated successfully"},
	)

	err = c.Status(http.StatusOK).JSON(response)
	return
}
