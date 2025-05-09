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

	// Get request ID
	requestID := c.Locals("requestID").(string)

	h.logger.InfoContext(ctx, "Purchase request received",
		slog.String("request_id", requestID),
		slog.String("path", c.Path()),
		slog.String("method", c.Method()),
		slog.String("event_type", "purchase_initiated"))

	var req apirequests.ProductBuyRequest
	if parseErr := c.BodyParser(&req); parseErr != nil {
		h.logger.WarnContext(ctx, "Request rejected: invalid request format",
			slog.String("error", parseErr.Error()),
			slog.String("error_code", apierrors.ErrCodeRequestValidation),
			slog.String("request_id", requestID),
			slog.String("path", c.Path()),
		)

		err = apierrors.NewApplicationError(
			apierrors.ErrCodeRequestValidation,
			"Invalid request body format",
			parseErr,
		).WithRequestID(requestID)

		return
	}

	if validatorErr := validator.ValidateRequest(&req); validatorErr != nil {
		// Ensure request ID is set on the validator error
		if validatorErr.RequestID == "" {
			validatorErr.RequestID = requestID
		}

		h.logger.WarnContext(ctx, "Request validation failed",
			slog.String("validator_error", validatorErr.Message),
			slog.String("error_code", validatorErr.Code),
			slog.String("request_id", requestID),
			slog.String("path", c.Path()),
			slog.String("event_type", "request_validation_failed"),
		)

		err = validatorErr
		return
	}

	productName := req.Name
	quantity := req.Quantity

	h.logger.DebugContext(ctx, "Processing purchase details",
		slog.String("product_name", productName),
		slog.Int("quantity", quantity),
		slog.String("request_id", requestID))

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
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		err = simAppErr
		return
	}

	h.logger.InfoContext(ctx, "Processing purchase request",
		slog.String("product_name", productName),
		slog.Int("quantity", quantity),
		slog.String("request_id", requestID),
		slog.String("event_type", "purchase_processing"))

	revenue, appErr := h.service.BuyProduct(ctx, productName, quantity)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}

		// Ensure request ID is set
		if appErr.RequestID == "" {
			appErr.RequestID = requestID
		}

		err = appErr
		return
	}

	h.logger.InfoContext(ctx, "Purchase completed successfully",
		slog.String("product_name", productName),
		slog.Int("quantity", quantity),
		slog.Float64("revenue", revenue),
		slog.String("request_id", requestID),
		slog.String("event_type", "purchase_completed"),
	)

	span.SetAttributes(attribute.Float64("product.revenue", revenue))

	// Create response with request ID
	response := apiresponses.NewSuccessResponse(fiber.Map{
		"productName": productName,
		"quantity":    quantity,
		"revenue":     revenue,
	}).WithRequestID(requestID)

	err = c.Status(http.StatusOK).JSON(response)
	return
}
