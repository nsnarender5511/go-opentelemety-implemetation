package main

import (
	"context"
	"log/slog"

	"sync"

	"github.com/narender/common/debugutils"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/common/utils"
	"go.opentelemetry.io/otel/attribute"

	"github.com/narender/common/globals"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/codes"
)

// ProductRepository defines the interface for accessing product data.
type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
}

type productRepository struct {
	products      map[string]Product
	productsSlice []Product
	mu            sync.RWMutex
	filePath      string
	logger        *slog.Logger
}

// NewProductRepository creates a new repository instance loading data from a JSON file.
func NewProductRepository(dataFilePath string) ProductRepository {
	repo := &productRepository{
		products: make(map[string]Product),
		filePath: dataFilePath,
		logger:   globals.Logger(),
	}

	return repo
}

func (r *productRepository) GetAll(ctx context.Context) (products []Product, opErr error) {
	operationName := utils.GetCallerFunctionName(2)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr)

	ctx, spanner := commontrace.StartSpan(ctx,
		semconv.DBSystemKey.String("memory"),
		semconv.DBOperationKey.String("READ_ALL"),
	)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Repository: GetAll called", slog.String("operation", operationName))

	spanner.AddEvent("Acquiring read lock for GetAll")
	r.mu.RLock()
	defer r.mu.RUnlock()
	spanner.AddEvent("Read lock acquired for GetAll")

	products = r.productsSlice
	if len(products) == 0 {
		r.logger.Warn("Repository: GetAll called but no products loaded/cached.")
		spanner.AddEvent("Product cache is empty")
	}

	spanner.SetAttributes(attribute.Int("products.returned.count", len(products)))
	r.logger.InfoContext(ctx, "Repository: GetAll returning products from cache", slog.Int("count", len(products)), slog.String("operation", operationName))
	return products, nil
}

func (r *productRepository) GetByID(ctx context.Context, id string) (product Product, opErr error) {
	operationName := utils.GetCallerFunctionName(2)
	productIdAttr := attribute.String("product.id", id)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, productIdAttr)

	ctx, spanner := commontrace.StartSpan(ctx,
		semconv.DBSystemKey.String("memory"),
		semconv.DBOperationKey.String("READ"),
		productIdAttr,
	)

	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Repository: GetByID called", slog.String("product_id", id), slog.String("operation", operationName))

	product, exists := r.products[id]
	if !exists {
		r.logger.WarnContext(ctx, "Repository: GetByID product not found",
			slog.String("product_id", id),
		)
		spanner.SetStatus(codes.Error, "PRODUCT_NOT_FOUND")
		return Product{}, nil
	}

	spanner.SetAttributes(attribute.String("product.name", product.Name))
	r.logger.InfoContext(ctx, "Repository: GetByID found product in cache", slog.String("product_id", id), slog.String("operation", operationName))
	return product, nil
}
