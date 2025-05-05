package main

import (
	"log/slog"
	"net/http"

	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/debugutils"
	"github.com/narender/common/globals"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type ProductHandler struct {
	service ProductService
	logger  *slog.Logger
}

type getByNamePayload struct {
	Name string `json:"name"`
}

type updateStockPayload struct {
	Name  string `json:"name"`
	Stock int    `json:"stock"`
}

type productBuyPayload struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

func NewProductHandler(svc ProductService) *ProductHandler {
	return &ProductHandler{
		service: svc,
		logger:  globals.Logger(),
	}
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	h.logger.InfoContext(c.UserContext(), "Shop open/close status requested")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Front_Desk: Customer asking for all products list !!!")

	ctx, span := commontrace.StartSpan(ctx)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)
	h.logger.DebugContext(ctx, "Front Desk: waiting for all products list from shop manager")
	products, err := h.service.GetAll(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "Front Desk: error getting all products list from shop manager: "+err.Error())
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	productCount := len(products)
	h.logger.InfoContext(ctx, "Front Desk: total "+strconv.Itoa(productCount)+" products list received from shop manager")
	span.SetAttributes(attribute.Int("products.count", productCount))
	h.logger.InfoContext(ctx, "Front Desk: Returning "+strconv.Itoa(productCount)+" products list to customer")
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductsByCategory(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()
	category := c.Query("category")
	h.logger.InfoContext(ctx, "Front_Desk: Customer asking for products in category: "+category)

	if category == "" {
		h.logger.WarnContext(ctx, "Front_Desk: Customer didn't specify a category")
		return fiber.NewError(http.StatusBadRequest, "Missing 'category' query parameter")
	}

	categoryAttr := attribute.String("product.category", category)
	ctx, span := commontrace.StartSpan(ctx, categoryAttr)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)
	h.logger.DebugContext(ctx, "Front Desk: waiting for category products list from shop manager for "+category)

	products, err := h.service.GetByCategory(ctx, category)
	if err != nil {
		h.logger.ErrorContext(ctx, "Front Desk: error getting products for category "+category+": "+err.Error())
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	productCount := len(products)
	h.logger.InfoContext(ctx, "Front Desk: total "+strconv.Itoa(productCount)+" products received for category "+category)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))
	h.logger.InfoContext(ctx, "Front Desk: Returning "+strconv.Itoa(productCount)+" products in category "+category+" to customer")
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByName(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()
	// Span setup should ideally capture the operation name
	ctx, span := commontrace.StartSpan(ctx) // Consider adding operation name attribute
	defer commontrace.EndSpan(span, &opErr, nil)

	// Parse body for product name
	var payload getByNamePayload
	if err := c.BodyParser(&payload); err != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Invalid get product details request format: "+err.Error())
		opErr = fiber.NewError(http.StatusBadRequest, "invalid request body: "+err.Error())
		return opErr
	}
	productName := payload.Name
	if productName == "" {
		h.logger.WarnContext(ctx, "Front_Desk: Customer didn't specify product name in request body")
		opErr = fiber.NewError(http.StatusBadRequest, "missing 'name' in request body")
		return opErr
	}

	h.logger.InfoContext(ctx, "Front_Desk: Customer asking for product details with Name: '"+productName+"'")
	productNameAttr := attribute.String("product.name", productName)
	span.SetAttributes(productNameAttr)

	debugutils.Simulate(ctx)
	h.logger.DebugContext(ctx, "Front Desk: waiting for product details from shop manager for Name '"+productName+"'")

	debugutils.Simulate(ctx)
	// Call service method using name from payload
	product, err := h.service.GetByName(ctx, productName)
	if err != nil {
		h.logger.ErrorContext(ctx, "Front Desk: error getting product details for Name '"+productName+"' : "+err.Error())
		span.SetStatus(codes.Error, err.Error())
		opErr = err
		if ferr, ok := opErr.(*fiber.Error); ok {
			return ferr
		}
		return fiber.NewError(http.StatusInternalServerError, "Failed to get product details")
	}
	h.logger.InfoContext(ctx, "Front Desk: product details received from shop manager for Name '"+productName+"'")
	h.logger.InfoContext(ctx, "Front Desk: Returning product details to customer for Name '"+productName+"'")
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) UpdateProductStock(c *fiber.Ctx) error {
	var opErr error
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Front_Desk: Manager requesting stock update")

	ctx, span := commontrace.StartSpan(c.UserContext()) // Consider adding operation name attribute
	defer func() {
		commontrace.EndSpan(span, &opErr, nil)
	}()

	// Parse body for name and stock
	var payload updateStockPayload
	if err := c.BodyParser(&payload); err != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Invalid stock update request format: "+err.Error())
		opErr = fiber.NewError(http.StatusBadRequest, "invalid request body: "+err.Error())
		return opErr
	}
	productName := payload.Name
	newStock := payload.Stock
	if productName == "" {
		h.logger.WarnContext(ctx, "Front_Desk: Manager didn't specify product name in request body")
		opErr = fiber.NewError(http.StatusBadRequest, "missing 'name' in request body")
		return opErr
	}
	// Optional: Add validation for newStock (e.g., non-negative)
	if newStock < 0 {
		h.logger.WarnContext(ctx, "Front_Desk: Manager provided invalid negative stock value", slog.Int("stock", newStock))
		opErr = fiber.NewError(http.StatusBadRequest, "stock cannot be negative")
		return opErr
	}

	h.logger.DebugContext(ctx, "Front Desk: Manager wants to update stock for product Name '"+productName+"'")
	span.SetAttributes(attribute.String("product.name", productName))

	h.logger.DebugContext(ctx, "Front Desk: Stock to be updated to "+strconv.Itoa(newStock)+" for product Name '"+productName+"'")

	debugutils.Simulate(ctx)
	h.logger.InfoContext(ctx, "Front Desk: Sending stock update request to shop manager")

	// Call service method with name from payload
	opErr = h.service.UpdateStock(ctx, productName, newStock)
	if opErr != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Failed to update stock for product Name '"+productName+"' : "+opErr.Error())
		if ferr, ok := opErr.(*fiber.Error); ok {
			return ferr
		}
		return fiber.NewError(http.StatusInternalServerError, "Failed to update stock")
	}

	h.logger.InfoContext(ctx, "Front Desk: Stock successfully updated to "+strconv.Itoa(newStock)+" for product Name '"+productName+"'")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *ProductHandler) BuyProduct(c *fiber.Ctx) error {
	var opErr error
	ctx := c.UserContext()
	h.logger.InfoContext(ctx, "Front_Desk: Customer wants to buy a product")

	ctx, span := commontrace.StartSpan(c.UserContext()) // Consider adding operation name attribute
	defer func() {
		commontrace.EndSpan(span, &opErr, nil)
	}()

	// Parse body for name and quantity
	var payload productBuyPayload
	if err := c.BodyParser(&payload); err != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Invalid purchase request format: "+err.Error())
		opErr = fiber.NewError(http.StatusBadRequest, "invalid request body: "+err.Error())
		return opErr
	}
	productName := payload.Name
	quantity := payload.Quantity
	if productName == "" {
		h.logger.WarnContext(ctx, "Front_Desk: Customer didn't specify product name in request body")
		opErr = fiber.NewError(http.StatusBadRequest, "missing 'name' in request body")
		return opErr
	}
	if quantity <= 0 {
		h.logger.WarnContext(ctx, "Front Desk: Customer tried to buy "+strconv.Itoa(quantity)+" items, which is invalid")
		opErr = fiber.NewError(http.StatusBadRequest, "quantity must be greater than 0")
		return opErr
	}

	h.logger.DebugContext(ctx, "Front Desk: Customer wants to buy product Name '"+productName+"'")
	span.SetAttributes(attribute.String("product.name", productName))

	h.logger.DebugContext(ctx, "Front Desk: Customer wants to buy "+strconv.Itoa(quantity)+" of product Name '"+productName+"'")
	span.SetAttributes(attribute.Int("product.purchase_quantity", quantity))

	debugutils.Simulate(ctx)

	// Delegate core logic to the service layer
	h.logger.InfoContext(ctx, "Front Desk: Asking shop manager to process purchase of "+strconv.Itoa(quantity)+" items for product Name '"+productName+"'")
	// Call service method with name from payload
	remainingStock, err := h.service.BuyProduct(ctx, productName, quantity)
	if err != nil {
		h.logger.ErrorContext(ctx, "Front Desk: Shop manager reported an error processing purchase: "+err.Error())
		opErr = err
		if ferr, ok := opErr.(*fiber.Error); ok {
			return ferr
		}
		return fiber.NewError(http.StatusInternalServerError, "Failed to process purchase")
	}

	// Handle successful response
	h.logger.InfoContext(ctx, "Front Desk: Purchase successful! Customer bought "+strconv.Itoa(quantity)+
		" of product Name '"+productName+"' . Manager confirms remaining stock is "+strconv.Itoa(remainingStock))
	span.SetAttributes(attribute.Int("product.remaining_stock", remainingStock))

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status":  "ok",
		"message": "Product purchased successfully",
		"data": fiber.Map{
			"productName":    productName, // Use name from payload
			"quantity":       quantity,
			"remainingStock": remainingStock,
		},
	})
}
