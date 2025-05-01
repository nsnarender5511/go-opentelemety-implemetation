package main

import (
	"context"
	"fmt"
	"net/http"
	"signoz-common/errors"
	"signoz-common/telemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Define common attribute keys
var (
	productIDKey         = attribute.Key("product.id")
	lookupSuccessKey     = attribute.Key("lookup.success")
	stockCheckSuccessKey = attribute.Key("check.success")
)

// ProductHandler handles HTTP requests for products
type ProductHandler struct {
	service                  ProductService
	tracer                   trace.Tracer
	meter                    metric.Meter
	productLookupsCounter    metric.Int64Counter
	productStockCheckCounter metric.Int64Counter
}

// NewProductHandler creates a new product handler
func NewProductHandler(service ProductService) *ProductHandler {
	// Initialize meter and instruments
	meter := telemetry.GetMeter("product-service/handler")

	lookupsCounter, err := meter.Int64Counter(
		"app.product.lookups",
		metric.WithDescription("Counts product lookup attempts"),
		metric.WithUnit("{lookup}"),
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create productLookupsCounter")
	}

	stockChecksCounter, err := meter.Int64Counter(
		"app.product.stock_checks",
		metric.WithDescription("Counts product stock check attempts"),
		metric.WithUnit("{check}"),
	)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create productStockCheckCounter")
	}

	return &ProductHandler{
		service:                  service,
		tracer:                   telemetry.GetTracer("product-service/handler"),
		meter:                    meter,
		productLookupsCounter:    lookupsCounter,
		productStockCheckCounter: stockChecksCounter,
	}
}

// GetAllProducts handles GET /products
func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	ctx := c.UserContext()
	log := logrus.WithContext(ctx)
	log.Info("Handler: Received request to get all products")

	ctxSpan, span := h.tracer.Start(ctx, "GetAllProductsHandler")
	defer span.End()

	products, err := h.service.GetAll(ctxSpan)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Handler: Error calling service.GetAll")
		return errors.HandleServiceError(c, err, "get all products")
	}

	span.SetAttributes(attribute.Int("product.count", len(products)))
	log.Infof("Handler: Responding with %d products", len(products))
	return c.Status(http.StatusOK).JSON(products)
}

// GetProductByID handles GET /products/:productId
func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	ctx := c.UserContext()
	log := logrus.WithContext(ctx)

	productID, errResp := h.validatePathParam(ctx, c, "productId")
	if errResp != nil {
		return errResp
	}
	log.Infof("Handler: Received request to get product by ID %s", productID)

	lookupAttrs := []attribute.KeyValue{productIDKey.String(productID)}

	ctxSpan, span := h.tracer.Start(ctx, "GetProductByIDHandler",
		trace.WithAttributes(productIDKey.String(productID)),
	)
	defer span.End()

	product, err := h.service.GetByID(ctxSpan, productID)
	if err != nil {
		lookupAttrs = append(lookupAttrs, lookupSuccessKey.Bool(false))
		h.productLookupsCounter.Add(ctxSpan, 1, metric.WithAttributes(lookupAttrs...))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Errorf("Handler: Error calling service.GetByID for ID %s", productID)
		return errors.HandleServiceError(c, err, fmt.Sprintf("get product by ID %s", productID))
	}

	lookupAttrs = append(lookupAttrs, lookupSuccessKey.Bool(true))
	h.productLookupsCounter.Add(ctxSpan, 1, metric.WithAttributes(lookupAttrs...))

	log.Infof("Handler: Responding with product data for ID %s", productID)
	return c.Status(http.StatusOK).JSON(product)
}

// GetProductStock handles GET /products/:productId/stock
func (h *ProductHandler) GetProductStock(c *fiber.Ctx) error {
	ctx := c.UserContext()
	log := logrus.WithContext(ctx)

	productID, errResp := h.validatePathParam(ctx, c, "productId")
	if errResp != nil {
		return errResp
	}
	log.Infof("Handler: Received request to get product stock for ID %s", productID)

	stockCheckAttrs := []attribute.KeyValue{productIDKey.String(productID)}

	ctxSpan, span := h.tracer.Start(ctx, "GetProductStockHandler",
		trace.WithAttributes(productIDKey.String(productID)),
	)
	defer span.End()

	stock, err := h.service.GetStock(ctxSpan, productID)
	if err != nil {
		stockCheckAttrs = append(stockCheckAttrs, stockCheckSuccessKey.Bool(false))
		h.productStockCheckCounter.Add(ctxSpan, 1, metric.WithAttributes(stockCheckAttrs...))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Errorf("Handler: Error calling service.GetStock for ID %s", productID)
		return errors.HandleServiceError(c, err, fmt.Sprintf("get stock for product ID %s", productID))
	}

	stockCheckAttrs = append(stockCheckAttrs, stockCheckSuccessKey.Bool(true))
	h.productStockCheckCounter.Add(ctxSpan, 1, metric.WithAttributes(stockCheckAttrs...))

	span.SetAttributes(attribute.Int("product.stock", stock))
	response := fiber.Map{
		"productId": productID,
		"stock":     stock,
	}
	log.Infof("Handler: Responding with product stock %d for ID %s", stock, productID)
	return c.Status(http.StatusOK).JSON(response)
}

// --- Helper Functions ---

// validatePathParam extracts and validates a required path parameter.
// Returns the parameter value or a fiber error response if validation fails.
func (h *ProductHandler) validatePathParam(ctx context.Context, c *fiber.Ctx, paramName string) (string, error) {
	paramValue := c.Params(paramName)
	if paramValue == "" {
		logrus.WithContext(ctx).Warnf("Handler: Missing %s parameter", paramName)
		return "", c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": fmt.Sprintf("%s parameter is required", paramName)})
	}
	return paramValue, nil
}
