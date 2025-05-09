package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"

	apiresponses "github.com/narender/common/apiresponses"
	"go.opentelemetry.io/otel/codes"
)

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (err error) {
	ctx := c.UserContext()

	h.logger.InfoContext(ctx, "Initiating request processing for retrieving all products",
		slog.String("path", c.Path()),
		slog.String("method", c.Method()),
		slog.String("operation", "get_all_products"),
		slog.String("event_type", "products_list_requested"),
		slog.String("client_ip", c.IP()),
		slog.String("user_agent", c.Get("User-Agent")))

	newCtx, span := commontrace.StartSpan(ctx, "product_handler", "get_all_products")
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

	h.logger.DebugContext(ctx, "Executing database query to retrieve complete product catalog",
		slog.String("operation", "fetch_all_products"),
		slog.String("component", "product_handler"))

	products, appErr := h.service.GetAll(ctx)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}
		err = appErr
		return
	}

	productCount := len(products)
	h.logger.InfoContext(ctx, "Product catalog retrieval operation completed successfully",
		slog.Int("product_count", productCount),
		slog.String("operation", "get_all_products"),
		slog.String("event_type", "products_retrieved"),
		slog.String("status", "success"))

	span.SetAttributes(attribute.Int("products.count", productCount))

	// Create response without request ID
	response := apiresponses.NewSuccessResponse(products)

	err = c.Status(http.StatusOK).JSON(response)
	return
}
