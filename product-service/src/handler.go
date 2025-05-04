package main

import (
	"log/slog"
	"net/http"

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

func NewProductHandler(svc ProductService) *ProductHandler {
	return &ProductHandler{
		service: svc,
		logger:  globals.Logger(),
	}
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()

	ctx, span := commontrace.StartSpan(ctx)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	h.logger.Info("Handler: Received request for GetAllProducts")

	h.logger.Info("Handler: Calling service GetAll")
	products, err := h.service.GetAll(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "Service GetAll failed", slog.String("error", err.Error()))
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
	productCount := len(products)
	h.logger.Info("Successfully retrieved all products", slog.Int("productCount", productCount))
	span.SetAttributes(attribute.Int("products.count", productCount))
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) (opErr error) {
	ctx := c.UserContext()
	productID := c.Params("productId")
	productIdAttr := attribute.String("product.id", productID)
	const operation = "GetProductByIDHandler"

	ctx, span := commontrace.StartSpan(ctx, productIdAttr)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	h.logger.Info("Handler: Received request for GetProductByID", slog.String("product_id", productID), slog.String("operation", operation))

	debugutils.Simulate(ctx)
	h.logger.Info("Handler: Calling service GetByID", slog.String("product_id", productID), slog.String("operation", operation))
	product, err := h.service.GetByID(ctx, productID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Service GetByID failed", slog.String("error", err.Error()))
		if span != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
	h.logger.Info("Successfully retrieved product by ID", slog.String("product_id", productID), slog.String("operation", operation))
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	const operation = "HealthCheckHandler"
	h.logger.Info("Health check requested", slog.String("operation", operation))
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

// createProductPayload defines the structure for the create product request body.
type createProductPayload struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

// updateStockPayload defines the expected structure for the update stock request body.
type updateStockPayload struct {
	Stock int `json:"stock"`
}

func (h *ProductHandler) UpdateProductStock(c *fiber.Ctx) error {
	// Declare opErr early for the defer statement
	var opErr error
	h.logger.Info("Handler: Received request for UpdateProductStock")

	ctx, span := commontrace.StartSpan(c.UserContext())
	// Use named return for opErr in defer to capture the final error state
	defer func() {
		commontrace.EndSpan(span, &opErr, nil)
	}()

	// Extract productID from path parameters
	productID := c.Params("productID")
	if productID == "" {
		h.logger.WarnContext(ctx, "Handler: Missing productID in path parameters")
		if span != nil {
			span.SetStatus(codes.Error, "missing productID in path parameters")
		}
		return opErr
	}
	h.logger.DebugContext(ctx, "Handler: Extracted productID", slog.String("productID", productID))

	// Parse request body for newStock
	var payload updateStockPayload
	if err := c.BodyParser(&payload); err != nil {
		h.logger.WarnContext(ctx, "Handler: Failed to parse request body", slog.String("error", err.Error()))
		// Use fiber.NewError for standard Bad Request
		opErr = fiber.NewError(http.StatusBadRequest, "invalid request body: "+err.Error())
		// Span status is set correctly in defer using the assigned opErr
		return opErr // Return the Fiber error
	}
	newStock := payload.Stock // Extracted new stock value
	h.logger.DebugContext(ctx, "Handler: Parsed newStock value", slog.Int("newStock", newStock))

	debugutils.Simulate(ctx)

	h.logger.Info("Handler: Calling service UpdateStock", slog.String("productID", productID), slog.Int("newStock", newStock))
	opErr = h.service.UpdateStock(ctx, productID, newStock) // Assign error to opErr
	if opErr != nil {
		h.logger.ErrorContext(ctx, "Service UpdateStock failed", slog.String("productID", productID), slog.String("error", opErr.Error()))
		// Span status is set within the defer based on opErr
		// Return the error for the framework's error handler
		return opErr
	}

	h.logger.Info("Handler: UpdateStock completed successfully", slog.String("productID", productID))
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

// CreateProduct handles requests to create a new product.
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	var opErr error
	ctx := c.UserContext()
	h.logger.Info("Handler: Received request for CreateProduct")

	// Parse request body
	payload := new(createProductPayload)
	if err := c.BodyParser(payload); err != nil {
		h.logger.WarnContext(ctx, "Handler: Failed to parse request body for create", slog.String("error", err.Error()))
		// Use fiber.NewError for standard Bad Request
		opErr = fiber.NewError(http.StatusBadRequest, "invalid request body: "+err.Error())
		// Span status is set correctly in defer using the assigned opErr
		return opErr // Return the Fiber error
	}

	h.logger.DebugContext(ctx, "Handler: Parsed create payload", slog.String("name", payload.Name))
	debugutils.Simulate(ctx)

	// Call the service method
	ctx, span := commontrace.StartSpan(c.UserContext())
	defer commontrace.EndSpan(span, &opErr, nil)

	createdProduct, err := h.service.Create(ctx, *payload)
	if err != nil {
		h.logger.ErrorContext(ctx, "Service Create failed", slog.String("error", err.Error()))
		opErr = err  // Assign service error to opErr for defer/return
		return opErr // Return error for framework handler (might be 400, 500, 409 etc.)
	}

	debugutils.Simulate(ctx)
	span.SetAttributes(attribute.String("product.id", createdProduct.ProductID))
	h.logger.InfoContext(ctx, "Handler: Product created successfully", slog.String("productID", createdProduct.ProductID))
	return c.Status(http.StatusCreated).JSON(createdProduct)
}
