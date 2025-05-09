package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"

	apierrors "github.com/narender/common/apierrors"
	apiresponses "github.com/narender/common/apiresponses"
	"go.opentelemetry.io/otel/codes"
)

func (h *ProductHandler) GetProductsByCategory(c *fiber.Ctx) (err error) {
	ctx := c.UserContext()

	// Get request ID
	requestID := c.Locals("requestID").(string)

	category := c.Query("category")

	h.logger.InfoContext(ctx, "Category products request received",
		slog.String("request_id", requestID),
		slog.String("category", category),
		slog.String("path", c.Path()),
		slog.String("method", c.Method()),
		slog.String("event_type", "category_products_requested"))

	if category == "" {
		h.logger.WarnContext(ctx, "Request rejected: missing category parameter",
			slog.String("error_code", apierrors.ErrCodeRequestValidation),
			slog.String("request_id", requestID),
			slog.String("path", c.Path()))

		err = apierrors.NewApplicationError(
			apierrors.ErrCodeRequestValidation,
			"Missing 'category' query parameter",
			nil).WithRequestID(requestID)
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
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		err = simAppErr
		return
	}

	h.logger.DebugContext(ctx, "Fetching products by category",
		slog.String("category", category),
		slog.String("request_id", requestID))

	products, appErr := h.service.GetByCategory(ctx, category)
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

	productCount := len(products)

	h.logger.InfoContext(ctx, "Category products retrieved successfully",
		slog.String("category", category),
		slog.Int("product_count", productCount),
		slog.String("request_id", requestID),
		slog.String("event_type", "category_products_retrieved"))

	span.SetAttributes(attribute.Int("products.returned.count", productCount))

	// Create response with request ID
	response := apiresponses.NewSuccessResponse(products).WithRequestID(requestID)

	err = c.Status(http.StatusOK).JSON(response)
	return
}
