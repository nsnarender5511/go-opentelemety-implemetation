package main

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	otel "github.com/narender/common/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type ProductHandler struct {
	service ProductService
}

func NewProductHandler(service ProductService) *ProductHandler {
	handler := &ProductHandler{
		service: service,
	}
	return handler
}

const handlerInstrumentationName = "product-service/handler"

func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	tracer := otel.GetTracer(handlerInstrumentationName)
	ctx, span := tracer.Start(c.UserContext(), "GetAllProductsHandler")
	defer span.End()

	products, err := h.service.GetAll(ctx)
	if err != nil {
		return err
	}

	span.AddEvent("Retrieved all products", oteltrace.WithAttributes(otel.AttrAppProductCount.Int(len(products))))
	span.SetAttributes(otel.AttrAppProductCount.Int(len(products)))
	span.SetStatus(codes.Ok, "Products retrieved successfully")
	return c.Status(http.StatusOK).JSON(products)
}

func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	tracer := otel.GetTracer(handlerInstrumentationName)
	ctx, span := tracer.Start(c.UserContext(), "GetProductByIDHandler", oteltrace.WithAttributes(otel.AttrAppProductIDKey.String(c.Params("productId"))))
	defer span.End()

	product, err := h.service.GetByID(ctx, c.Params("productId"))
	if err != nil {
		return err
	}

	span.AddEvent("Retrieved product by ID")
	span.SetStatus(codes.Ok, "Product retrieved successfully")
	return c.Status(http.StatusOK).JSON(product)
}

func (h *ProductHandler) GetProductStock(c *fiber.Ctx) error {
	tracer := otel.GetTracer(handlerInstrumentationName)
	ctx, span := tracer.Start(c.UserContext(), "GetProductStockHandler", oteltrace.WithAttributes(otel.AttrAppProductIDKey.String(c.Params("productId"))))
	defer span.End()

	stock, err := h.service.GetStock(ctx, c.Params("productId"))
	if err != nil {
		return err
	}

	type StockResponse struct {
		ProductID string `json:"productId"`
		Stock     int    `json:"stock"`
	}

	span.AddEvent("Retrieved product stock", oteltrace.WithAttributes(attribute.Int("app.product.stock", stock)))
	span.SetAttributes(attribute.Int("app.product.stock", stock))
	span.SetStatus(codes.Ok, "Product stock retrieved successfully")
	return c.Status(http.StatusOK).JSON(StockResponse{ProductID: c.Params("productId"), Stock: stock})
}

func (h *ProductHandler) HealthCheck(c *fiber.Ctx) error {
	tracer := otel.GetTracer(handlerInstrumentationName)
	_, span := tracer.Start(c.UserContext(), "HealthCheckHandler")
	defer span.End()

	span.AddEvent("Health check successful")
	span.SetStatus(codes.Ok, "Health check ok")
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}
