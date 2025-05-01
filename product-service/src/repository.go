package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	// Use correct module path for common packages
	"example.com/product-service/common/errors"
	"example.com/product-service/common/telemetry"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const dataFilePath = "../data.json"

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
}

// NewProductRepository creates a new product repository
func NewProductRepository() (ProductRepository, error) {
	r := &productRepository{
		filePath: dataFilePath,
	}
	// Ensure the data file exists, create if not
	if _, statErr := os.Stat(r.filePath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			logrus.Infof("Data file not found, creating empty file at %s", r.filePath)
			if writeErr := os.WriteFile(r.filePath, []byte("{\n}"), 0644); writeErr != nil {
				return nil, fmt.Errorf("failed to create initial data file '%s': %w", r.filePath, writeErr)
			}
		} else {
			return nil, fmt.Errorf("failed to stat data file '%s': %w", r.filePath, statErr)
		}
	}

	return r, nil
}

// getTracer is a helper to get the tracer instance consistently
func (r *productRepository) getTracer() trace.Tracer {
	return otel.Tracer("product-service/repository")
}

// FindAll fetches all products from the JSON file
func (r *productRepository) FindAll(ctx context.Context) ([]Product, error) {
	log := logrus.WithContext(ctx)
	log.Info("Repo: Fetching all products from file")

	var products []Product
	var data map[string]Product
	var err error

	tracer := r.getTracer()
	// Start span manually
	ctx, span := tracer.Start(ctx, "repository.FindAll")
	defer span.End()

	// Call readData with the new context
	data, err = r.readData(ctx)
	if err != nil {
		// Manually record error and set status
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Repo: Failed during readData in FindAll")
		return nil, err // Propagate error
	}

	products = make([]Product, 0, len(data))
	for _, p := range data {
		products = append(products, p)
	}
	span.SetAttributes(telemetry.DBResultCountKey.Int(len(products)))

	log.Info("Repo: Found products in file", telemetry.LogFieldCount, len(products))
	return products, nil
}

// FindByProductID finds a single product by its ID from the JSON file
func (r *productRepository) FindByProductID(ctx context.Context, productID string) (Product, error) {
	log := logrus.WithContext(ctx).WithField(telemetry.LogFieldProductID, productID)
	log.Info("Repo: Fetching product by ProductID from file")

	var product Product
	var err error

	tracer := r.getTracer()
	// Start span manually
	ctx, span := tracer.Start(ctx, "repository.FindByProductID")
	defer span.End()

	span.SetAttributes(telemetry.DBQueryParamProductIDKey.String(productID))

	// Call readData with the new context
	data, err := r.readData(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Repo: Failed during readData in FindByProductID")
		return Product{}, err
	}

	foundProduct, ok := data[productID]
	if !ok {
		log.Warn("Repo: Product not found in file")
		span.SetAttributes(telemetry.DBResultFoundKey.Bool(false))
		err = errors.ErrProductNotFound
		span.RecordError(err)                    // Record the specific not found error
		span.SetStatus(codes.Error, err.Error()) // Set status for not found
		return Product{}, err
	}

	product = foundProduct
	span.SetAttributes(telemetry.DBResultFoundKey.Bool(true))

	log.Info("Repo: Found product in file")
	return product, nil
}

// FindStockByProductID finds stock for a product by its ID from the JSON file
func (r *productRepository) FindStockByProductID(ctx context.Context, productID string) (int, error) {
	log := logrus.WithContext(ctx).WithField(telemetry.LogFieldProductID, productID)
	log.Info("Repo: Checking stock for ProductID in file")

	var stock int
	var err error

	tracer := r.getTracer()
	// Start span manually
	ctx, span := tracer.Start(ctx, "repository.FindStockByProductID")
	defer span.End()

	span.SetAttributes(telemetry.DBQueryParamProductIDKey.String(productID))

	// Call readData with the new context
	data, err := r.readData(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Repo: Failed during readData in FindStockByProductID")
		return 0, err
	}

	product, ok := data[productID]
	if !ok {
		log.Warn("Repo: Product not found in file for stock check")
		span.SetAttributes(telemetry.DBResultFoundKey.Bool(false))
		err = errors.ErrProductNotFound
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}

	stock = product.Stock
	span.SetAttributes(
		telemetry.DBResultFoundKey.Bool(true),
		telemetry.AppProductStockKey.Int(stock),
	)

	log.Info("Repo: Found stock for ProductID in file", telemetry.LogFieldStock, stock)
	return stock, nil
}

// readData reads the JSON file and returns the product map.
func (r *productRepository) readData(ctx context.Context) (map[string]Product, error) {
	var data map[string]Product
	var bytes []byte
	var err error
	var fileReadErr error
	var unmarshalErr error

	tracer := r.getTracer()

	// Span for file read
	ctxRead, spanRead := tracer.Start(ctx, "repository.readDataFile")
	spanRead.SetAttributes(telemetry.DBSystemJSONFile, telemetry.DBOperationRead, telemetry.FilePathKey.String(r.filePath))

	r.mu.RLock()
	fileBytes, fileReadErr := os.ReadFile(r.filePath)
	r.mu.RUnlock()
	if fileReadErr != nil {
		logrus.WithContext(ctxRead).WithError(fileReadErr).Error("Repo: Failed to read data file", "filePath", r.filePath)
		// Wrap error
		err = fmt.Errorf("%w: %w", errors.ErrDatabaseOperation, fileReadErr)
		spanRead.RecordError(err)
		spanRead.SetStatus(codes.Error, err.Error())
		spanRead.End() // End this span before returning
		return nil, err
	}
	bytes = fileBytes
	spanRead.End() // End file read span successfully

	if err != nil {
		// This block should technically be unreachable now if fileReadErr causes return
		logrus.WithContext(ctx).WithError(err).Error("Repo: Error during file read operation")
		return nil, err
	}

	// Span for unmarshal
	ctxUnmarshal, spanUnmarshal := tracer.Start(ctx, "repository.unmarshalData")
	unmarshalErr = json.Unmarshal(bytes, &data)
	if unmarshalErr != nil {
		logrus.WithContext(ctxUnmarshal).WithError(unmarshalErr).Error("Repo: Failed to unmarshal JSON", "filePath", r.filePath)
		// Wrap error
		err = fmt.Errorf("%w: %w", errors.ErrDatabaseOperation, unmarshalErr)
		spanUnmarshal.RecordError(err)
		spanUnmarshal.SetStatus(codes.Error, err.Error())
		spanUnmarshal.End() // End this span before returning
		return nil, err
	}
	spanUnmarshal.End() // End unmarshal span successfully

	if err != nil {
		// This block should technically be unreachable now if unmarshalErr causes return
		logrus.WithContext(ctx).WithError(err).Error("Repo: Error during JSON unmarshal operation")
		return nil, err
	}

	return data, nil
}
