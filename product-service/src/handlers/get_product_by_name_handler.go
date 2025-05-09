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
		h.logger.WarnContext(ctx, "Request rejected: invalid request format",
			slog.String("component", "product_handler"),
			slog.String("error", parseErr.Error()),
			slog.String("error_code", apierrors.ErrCodeRequestValidation),
			slog.String("path", c.Path()),
			slog.String("operation", "get_product_by_name"))

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
			slog.String("operation", "get_product_by_name"),
			slog.String("event_type", "request_validation_failed"))

		err = validatorErr
		return
	}

	productName := req.Name

	h.logger.InfoContext(ctx, "Product details request received",
		slog.String("component", "product_handler"),
		slog.String("product_name", productName),
		slog.String("path", c.Path()),
		slog.String("method", c.Method()),
		slog.String("operation", "get_product_by_name"),
		slog.String("event_type", "product_details_requested"))

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

	h.logger.DebugContext(ctx, "Fetching product details",
		slog.String("component", "product_handler"),
		slog.String("product_name", productName),
		slog.String("operation", "service_get_by_name"))

	product, appErr := h.service.GetByName(ctx, productName)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}

		err = appErr
		return
	}

	h.logger.InfoContext(ctx, "Product details retrieved successfully",
		slog.String("component", "product_handler"),
		slog.String("product_name", productName),
		slog.String("operation", "get_product_by_name"),
		slog.String("status", "success"),
		slog.String("event_type", "product_details_retrieved"))

	// Create response without RequestID
	response := apiresponses.NewSuccessResponse(product)

	err = c.Status(http.StatusOK).JSON(response)
	return
}
