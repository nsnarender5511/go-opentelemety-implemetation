package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"

	apiresponses "github.com/narender/common/apiresponses"
	"go.opentelemetry.io/otel/codes"
)

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (err error) {
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Front_Desk: Customer asking for all products list !!!")

	newCtx, span := commontrace.StartSpan(ctx)
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

	h.logger.DebugContext(ctx, "Front Desk: waiting for all products list from shop manager")
	products, appErr := h.service.GetAll(ctx)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}
		err = appErr
		return
	}
	productCount := len(products)
	h.logger.InfoContext(ctx, "Front Desk: total "+strconv.Itoa(productCount)+" products list received from shop manager")
	span.SetAttributes(attribute.Int("products.count", productCount))
	h.logger.InfoContext(ctx, "Front Desk: Returning "+strconv.Itoa(productCount)+" products list to customer")
	err = c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(products))
	return
}
