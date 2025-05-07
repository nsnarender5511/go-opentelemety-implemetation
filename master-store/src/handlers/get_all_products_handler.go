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

func (h *MasterStoreHandler) GetAllProducts(c *fiber.Ctx) (err error) {
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Master Store: Customer requesting all products list")

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

	h.logger.DebugContext(ctx, "Master Store: retrieving complete product inventory")
	products, appErr := h.service.GetAll(ctx)
	if appErr != nil {
		if span != nil {
			span.SetStatus(codes.Error, appErr.Error())
		}
		err = appErr
		return
	}
	productCount := len(products)
	h.logger.InfoContext(ctx, "Master Store: retrieved "+strconv.Itoa(productCount)+" products from inventory")
	span.SetAttributes(attribute.Int("products.count", productCount))
	h.logger.InfoContext(ctx, "Master Store: Returning "+strconv.Itoa(productCount)+" products to customer")
	err = c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(products))
	return
}
