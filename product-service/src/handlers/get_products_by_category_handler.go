package handlers

import (
	"net/http"
	"strconv"

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
