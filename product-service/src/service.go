package main

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	commonlog "github.com/narender/common/log"
	"github.com/narender/common/telemetry"
	"github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteMetric "go.opentelemetry.io/otel/metric"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const serviceScopeName = "github.com/narender/product-service/service"
const serviceLayerName = "service"

var (
	productServiceOpsCounter   oteMetric.Int64Counter
	productServiceDurationHist oteMetric.Float64Histogram
	productErrorsCounter       oteMetric.Int64Counter

	serviceMetricsOnce sync.Once
)

func initServiceMetrics() {
	serviceMetricsOnce.Do(func() {
		logger := commonlog.L
		meter := telemetry.GetMeter(serviceScopeName)
		var err error

		productServiceOpsCounter, err = meter.Int64Counter(
			"product.service.operations.count",
			oteMetric.WithDescription("Counts service layer operations like GetAll, GetByID"),
			oteMetric.WithUnit("{operation}"),
		)
		if err != nil {
			logger.Error("Failed to create product.service.operations.count counter", slog.Any("error", err))
		}

		productServiceDurationHist, err = meter.Float64Histogram(
			"product.service.duration",
			oteMetric.WithDescription("Measures the duration of service layer execution"),
			oteMetric.WithUnit("s"),
		)
		if err != nil {
			logger.Error("Failed to create product.service.duration histogram", slog.Any("error", err))
		}

		productErrorsCounter, err = meter.Int64Counter(
			"product.errors.count",
			oteMetric.WithDescription("Counts errors encountered, categorized by type and layer"),
			oteMetric.WithUnit("{error}"),
		)
		if err != nil {
			logger.Warn("Attempted to re-initialize product.errors.count counter from service layer", slog.Any("error", err))
		}
	})
}

func recordServiceError(ctx context.Context, err error) {
	if err == nil || productErrorsCounter == nil {
		return
	}

	errorType := "internal"
	if errors.Is(err, ErrNotFound) {
		errorType = "not_found"
	}

	productErrorsCounter.Add(ctx, 1, oteMetric.WithAttributes(
		attribute.String("layer", serviceLayerName),
		attribute.String("error_type", errorType),
	))
}

type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, productID string) (Product, error)
}

type productService struct {
	repo ProductRepository
}

func NewProductService(repo ProductRepository) ProductService {
	initServiceMetrics()
	return &productService{
		repo: repo,
	}
}

func (s *productService) GetAll(ctx context.Context) (products []Product, opErr error) {
	const operation = "GetAll"
	startTime := time.Now()
	defer func() {
		metric.RecordOperationMetrics(ctx, serviceLayerName, operation, startTime, opErr)
	}()

	simulateDelayIfEnabled()
	logger := commonlog.L

	tracer := telemetry.GetTracer(serviceScopeName)
	ctx, span := tracer.Start(ctx, "ProductService.GetAll")
	defer func() {
		if opErr != nil {
			commontrace.RecordSpanError(span, opErr)
		}
		span.End()
	}()

	logger.Info("Service: GetAll called")

	simulateDelayIfEnabled()
	span.AddEvent("Calling repository GetAll")
	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		opErr = repoErr
		logger.Error("Service: Repository error during GetAllProducts", slog.Any("error", opErr))
		span.RecordError(opErr)
		span.SetStatus(codes.Error, "repository error")
		span.AddEvent("Repository GetAll failed")
		return nil, opErr
	}
	span.AddEvent("Repository GetAll successful", oteltrace.WithAttributes(attribute.Int("products.count", len(products))))

	simulateDelayIfEnabled()
	span.SetAttributes(attribute.Int("products.count", len(products)))
	logger.Info("Service: GetAll completed successfully", slog.Int("count", len(products)))
	span.SetStatus(codes.Ok, "")
	return products, nil
}

func (s *productService) GetByID(ctx context.Context, productID string) (product Product, opErr error) {
	const operation = "GetByID"
	startTime := time.Now()
	productIdAttr := attribute.String("product.id", productID)
	defer func() {
		metric.RecordOperationMetrics(ctx, serviceLayerName, operation, startTime, opErr, productIdAttr)
	}()

	simulateDelayIfEnabled()
	logger := commonlog.L

	tracer := telemetry.GetTracer(serviceScopeName)
	ctx, span := tracer.Start(ctx, "ProductService.GetProductByID", oteltrace.WithAttributes(productIdAttr))
	defer func() {
		if opErr != nil {
			commontrace.RecordSpanError(span, opErr, productIdAttr)
		}
		span.End()
	}()

	logger.Info("Service: GetByID called", slog.String("product_id", productID))

	simulateDelayIfEnabled()
	span.AddEvent("Calling repository GetByID", oteltrace.WithAttributes(productIdAttr))
	product, repoErr := s.repo.GetByID(ctx, productID)
	if repoErr != nil {
		opErr = repoErr
		span.RecordError(opErr)
		span.AddEvent("Repository GetByID failed", oteltrace.WithAttributes(attribute.String("error", opErr.Error())))
		if errors.Is(opErr, ErrNotFound) {
			logger.Warn("Service: Product not found in repository", slog.String("product_id", productID))
			span.SetStatus(codes.Error, opErr.Error())

		} else {
			logger.Error("Service: Repository error during GetProductByID",
				slog.String("product_id", productID),
				slog.Any("error", opErr),
			)
			span.SetStatus(codes.Error, "repository error")
		}
		return Product{}, opErr
	}
	span.AddEvent("Repository GetByID successful")

	simulateDelayIfEnabled()
	logger.Info("Service: GetByID completed successfully", slog.String("product_id", productID))
	span.SetStatus(codes.Ok, "")
	return product, nil
}
