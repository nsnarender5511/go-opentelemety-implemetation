package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/narender/common/debugutils"
	"github.com/narender/common/filedb"
	commonmetric "github.com/narender/common/telemetry/metric"
	"github.com/narender/common/utils"
	"go.opentelemetry.io/otel/attribute"

	"github.com/narender/common/globals"
)

// ProductRepository defines the interface for accessing product data.
type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
}

type productRepository struct {
	db     *filedb.FileDatabase
	logger *slog.Logger
}

// NewProductRepository creates a new repository instance loading data from a JSON file.
func NewProductRepository() ProductRepository {
	repo := &productRepository{
		db:     filedb.NewFileDatabase(globals.Cfg().DATA_FILE_PATH),
		logger: globals.Logger(),
	}

	return repo
}

func (r *productRepository) GetAll(ctx context.Context) (products []Product, opErr error) {
	operationName := utils.GetCallerFunctionName(2)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Repository: GetAll called", slog.String("operation", operationName))

	opErr = r.db.Read(ctx, &products)
	if opErr != nil {
		r.logger.ErrorContext(ctx, "Repository: Failed to read products from file", slog.Any("error", opErr))
		return nil, opErr
	}

	r.logger.InfoContext(ctx, "Repository: GetAll returning products", slog.Int("count", len(products)), slog.String("operation", operationName))
	return products, nil
}

func (r *productRepository) GetByID(ctx context.Context, id string) (product Product, opErr error) {
	operationName := utils.GetCallerFunctionName(2)
	productIdAttr := attribute.String("product.id", id)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, productIdAttr)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Repository: GetByID called", slog.String("product_id", id), slog.String("operation", operationName))

	var allProducts []Product
	opErr = r.db.Read(ctx, &allProducts)
	if opErr != nil {
		r.logger.ErrorContext(ctx, "Repository: Failed to read products from file for GetByID", slog.Any("error", opErr))
		return Product{}, opErr
	}

	found := false
	for _, p := range allProducts {
		if p.ProductID == id {
			product = p
			found = true
			break
		}
	}

	if !found {
		opErr = fmt.Errorf("product with ID '%s' not found", id)
		r.logger.WarnContext(ctx, "Repository: Product not found", slog.String("product_id", id), slog.Any("error", opErr))
		return Product{}, opErr
	}

	r.logger.InfoContext(ctx, "Repository: GetByID found product", slog.String("product_id", id), slog.String("operation", operationName))
	return product, nil
}
