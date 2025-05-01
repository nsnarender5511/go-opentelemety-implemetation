package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"signoz-common/errors"
	"signoz-common/telemetry"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const dataFilePath = "data.json"

// ProductRepository defines the interface for product data access
type ProductRepository interface {
	FindAll(ctx context.Context) ([]Product, error)
	FindByProductID(ctx context.Context, productID string) (Product, error)
	FindStockByProductID(ctx context.Context, productID string) (int, error)
}

// productRepository implements ProductRepository
type productRepository struct {
	mu       sync.RWMutex
	filePath string
	tracer   trace.Tracer
}

// NewProductRepository creates a new product repository
func NewProductRepository() ProductRepository {
	r := &productRepository{
		filePath: dataFilePath,
		tracer:   telemetry.GetTracer("product-service/repository"),
	}
	// Ensure the data file exists, create if not
	if _, err := os.Stat(r.filePath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(r.filePath, []byte("{\n}"), 0644); err != nil {
			panic(fmt.Sprintf("Failed to create initial data file '%s': %v", r.filePath, err))
		}
	} else if err != nil {
		panic(fmt.Sprintf("Failed to stat data file '%s': %v", r.filePath, err))
	}

	return r
}

// FindAll fetches all products from the JSON file
func (r *productRepository) FindAll(ctx context.Context) ([]Product, error) {
	log := logrus.WithContext(ctx)
	log.Info("Repo: Fetching all products from file")

	ctx, span := r.tracer.Start(ctx, "FindAllRepo")
	defer span.End()

	data, err := r.readData(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read data file")
		return nil, err
	}

	products := make([]Product, 0, len(data))
	for _, p := range data {
		products = append(products, p)
	}

	span.SetAttributes(attribute.Int("db.result.count", len(products)))
	log.Info("Repo: Found products in file", "count", len(products))
	return products, nil
}

// FindByProductID finds a single product by its ID from the JSON file
func (r *productRepository) FindByProductID(ctx context.Context, productID string) (Product, error) {
	log := logrus.WithContext(ctx)
	log.Info("Repo: Fetching product by ProductID from file", "productID", productID)

	ctx, span := r.tracer.Start(ctx, "FindByProductIDRepo",
		trace.WithAttributes(attribute.String("db.query.parameter.product_id", productID)),
	)
	defer span.End()

	data, err := r.readData(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read data file")
		return Product{}, err
	}

	product, ok := data[productID]
	if !ok {
		log.Warn("Repo: Product not found in file", "productID", productID)
		span.SetAttributes(attribute.Bool("db.result.found", false))
		span.SetStatus(codes.Error, errors.ErrProductNotFound.Error())
		return Product{}, errors.ErrProductNotFound
	}

	span.SetAttributes(attribute.Bool("db.result.found", true))
	log.Info("Repo: Found product in file", "productID", productID)
	return product, nil
}

// FindStockByProductID finds stock for a product by its ID from the JSON file
func (r *productRepository) FindStockByProductID(ctx context.Context, productID string) (int, error) {
	log := logrus.WithContext(ctx)
	log.Info("Repo: Checking stock for ProductID in file", "productID", productID)

	ctx, span := r.tracer.Start(ctx, "FindStockByProductIDRepo",
		trace.WithAttributes(attribute.String("db.query.parameter.product_id", productID)),
	)
	defer span.End()

	data, err := r.readData(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read data file")
		return 0, err
	}

	product, ok := data[productID]
	if !ok {
		log.Warn("Repo: Product not found in file for stock check", "productID", productID)
		span.SetAttributes(attribute.Bool("db.result.found", false))
		span.SetStatus(codes.Error, errors.ErrProductNotFound.Error())
		return 0, errors.ErrProductNotFound
	}

	span.SetAttributes(
		attribute.Bool("db.result.found", true),
		attribute.Int("product.stock", product.Stock),
	)
	log.Info("Repo: Found stock for ProductID in file", "stock", product.Stock, "productID", productID)
	return product.Stock, nil
}

// readData reads the JSON file and returns the product map.
func (r *productRepository) readData(ctx context.Context) (map[string]Product, error) {
	ctx, span := r.tracer.Start(ctx, "readDataFile",
		trace.WithAttributes(attribute.String("db.system", "jsonfile"), attribute.String("db.operation", "read"), attribute.String("file.path", r.filePath)),
	)
	defer span.End()

	log := logrus.WithContext(ctx)
	r.mu.RLock()
	defer r.mu.RUnlock()

	bytes, err := os.ReadFile(r.filePath)
	if err != nil {
		log.WithError(err).Error("Repo: Failed to read data file", "filePath", r.filePath)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to read file")
		return nil, errors.ErrDatabaseOperation
	}

	var data map[string]Product
	if err := json.Unmarshal(bytes, &data); err != nil {
		log.WithError(err).Error("Repo: Failed to unmarshal JSON", "filePath", r.filePath)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to unmarshal json")
		return nil, errors.ErrDatabaseOperation
	}
	return data, nil
}
