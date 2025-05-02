package main

import (
	"context"
	"fmt"
	"net/http"

	commonErrors "github.com/narender/common/errors"
	"github.com/narender/common/otel"
	"github.com/sirupsen/logrus"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	AttrAppProductID    = attribute.Key("app.product.id")
	AttrAppProductCount = attribute.Key("app.product.count")
)

type ProductHandler struct {
	service ProductService
	logger  *logrus.Logger
	tracer  oteltrace.Tracer
	metrics *otel.Metrics
}

func NewProductHandler(service ProductService, logger *logrus.Logger, tracer oteltrace.Tracer, metrics *otel.Metrics) *ProductHandler {
	handler := &ProductHandler{
		service: service,
		logger:  logger,
		tracer:  tracer,
		metrics: metrics,
	}

	if err := handler.registerStockGaugeCallback(otel.GetMeterProvider()); err != nil {
		logger.WithError(err).Error("Failed to register observable stock gauge callback")
	}

	return handler
}

func (h *ProductHandler) registerStockGaugeCallback(meterProvider otelmetric.MeterProvider) error {
	meter := meterProvider.Meter(ServiceName + ".handler")

	stockGauge, err := meter.Int64ObservableGauge(
		"product.stock.level",
		otelmetric.WithDescription("Current stock level by product"),
		otelmetric.WithUnit("{items}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create product.stock.level gauge: %w", err)
	}

	refreshStockGaugeCallback := func(ctx context.Context, observer otelmetric.Observer) error {
		products, err := h.service.GetAll(context.Background())
		if err != nil {
			h.logger.WithContext(ctx).WithError(err).Error("Failed to get products for stock gauge refresh")
			return nil
		}

		for _, product := range products {
			observer.ObserveInt64(
				stockGauge,
				int64(product.Stock),
				otelmetric.WithAttributes(
					AttrAppProductID.String(product.ProductID),
				),
			)
		}
		return nil
	}

	_, err = meter.RegisterCallback(refreshStockGaugeCallback, stockGauge)
	if err != nil {
		return fmt.Errorf("failed to register callback for product.stock.level gauge: %w", err)
	}

	h.logger.Info("Registered observable stock gauge callback.")
	return nil
}

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	_, span := h.tracer.Start(c.UserContext(), "handler.GetAllProducts")
	defer span.End()
	h.logger.Info("Handler: GetAllProducts called")

	products, err := h.service.GetAll(c.UserContext())
	if err != nil {
		h.logger.WithError(err).Error("Handler: Error getting all products from service")
		otel.RecordSpanError(span, err)
		return err
	}

	h.logger.Infof("Handler: Successfully retrieved %d products", len(products))
	span.AddEvent("Retrieved all products", oteltrace.WithAttributes(attribute.Int("product.count", len(products))))
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	ctx, span := h.tracer.Start(c.UserContext(), "handler.GetProductByID")
	defer span.End()

	productIDStr := c.Params("productId")
	h.logger.Infof("Handler: GetProductByID called with productId: %s", productIDStr)
	span.SetAttributes(attribute.String("product.id", productIDStr))

	product, err := h.service.GetByID(ctx, productIDStr)
	if err != nil {
		h.logger.WithError(err).Errorf("Handler: Error getting product %s from service", productIDStr)
		otel.RecordSpanError(span, err)
		return err
	}

	h.logger.Infof("Handler: Successfully retrieved product %s", productIDStr)
	span.AddEvent("Retrieved product by ID")
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) GetProductStock(c *fiber.Ctx) error {
	ctx, span := h.tracer.Start(c.UserContext(), "handler.GetProductStock")
	defer span.End()

	productIDStr := c.Params("productId")
	h.logger.Infof("Handler: GetProductStock called with productId: %s", productIDStr)
	span.SetAttributes(attribute.String("product.id", productIDStr))

	stock, err := h.service.GetStock(ctx, productIDStr)
	if err != nil {
		h.logger.WithError(err).Errorf("Handler: Error getting stock for product %s from service", productIDStr)
		otel.RecordSpanError(span, err)
		return err
	}

	type StockResponse struct {
		ProductID string `json:"productId"`
		Stock     int    `json:"stock"`
	}

	h.logger.Infof("Handler: Successfully retrieved stock %d for product %s", stock, productIDStr)
	span.AddEvent("Retrieved product stock", oteltrace.WithAttributes(attribute.Int("product.stock", stock)))
	return c.Status(http.StatusOK).JSON(StockResponse{ProductID: productIDStr, Stock: stock})
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	_, span := h.tracer.Start(c.UserContext(), "handler.HealthCheck")
	defer span.End()
	h.logger.Info("Handler: HealthCheck called")


	h.logger.Info("Handler: HealthCheck successful")
	span.AddEvent("Health check successful")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

func (h *ProductHandler) validatePathParam(ctx context.Context, c *fiber.Ctx, paramName string) (string, error) {
	paramValue := c.Params(paramName)
	if paramValue == "" {
		h.logger.WithContext(ctx).Warnf("Missing path parameter: %s", paramName)
		return "", commonErrors.BadRequest(fmt.Sprintf("Path parameter '%s' is required", paramName))
	}
	return paramValue, nil
}
