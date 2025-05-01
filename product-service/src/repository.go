package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	// Use correct module path for common packages
	"github.com/narender/common-module/errors"
	"github.com/narender/common-module/telemetry"

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
	// Add DB statement attribute
	span.SetAttributes(telemetry.DBStatementKey.String("FindAll"))

	// Call readData with the new context
	data, err = r.readData(ctx)
	if err != nil {
		// Manually record error and set status
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Repo: Failed during readData in FindAll")
		// Wrap the error before returning
		return nil, fmt.Errorf("failed to read data for finding all products: %w", err)
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

	span.SetAttributes(
		telemetry.DBStatementKey.String("FindByProductID"),
		telemetry.DBQueryParamProductIDKey.String(productID),
	)

	// Call readData with the new context
	data, err := r.readData(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Repo: Failed during readData in FindByProductID")
		// Wrap the error from readData before returning
		return Product{}, fmt.Errorf("failed to read data for product lookup: %w", err)
	}

	foundProduct, ok := data[productID]
	if !ok {
		log.Warn("Repo: Product not found in file")
		span.SetAttributes(telemetry.DBResultFoundKey.Bool(false))
		err = errors.ErrProductNotFound // Use the sentinel error directly here
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return Product{}, err // Return the specific sentinel
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

	span.SetAttributes(
		telemetry.DBStatementKey.String("FindStockByProductID"),
		telemetry.DBQueryParamProductIDKey.String(productID),
	)

	// Call readData with the new context
	data, err := r.readData(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Repo: Failed during readData in FindStockByProductID")
		// Wrap the error from readData before returning
		return 0, fmt.Errorf("failed to read data for stock check: %w", err)
	}

	product, ok := data[productID]
	if !ok {
		log.Warn("Repo: Product not found in file for stock check")
		span.SetAttributes(telemetry.DBResultFoundKey.Bool(false))
		err = errors.ErrProductNotFound // Use the sentinel error directly here
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err // Return the specific sentinel
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
// It now uses manual span management internally.
func (r *productRepository) readData(ctx context.Context) (map[string]Product, error) {
	var data map[string]Product
	var bytes []byte
	var err error // Consolidated error variable

	tracer := r.getTracer() // Use package-level tracer helper

	// --- File Read Span ---
	// Start the span using the incoming context
	ctxRead, spanRead := tracer.Start(ctx, "repository.readDataFile")
	spanRead.SetAttributes(
		telemetry.DBSystemKey.String("json_file"), // Use constant key, provide value
		telemetry.DBOperationKey.String("read"),   // Use constant key, provide value
		telemetry.DBFilePathKey.String(r.filePath),
	)

	r.mu.RLock()
	fileBytes, fileReadErr := os.ReadFile(r.filePath)
	r.mu.RUnlock()

	if fileReadErr != nil {
		// Wrap the underlying error in a DatabaseError
		err = &errors.DatabaseError{Operation: "read file", Err: fileReadErr}
		logrus.WithContext(ctxRead).WithError(err).Error("Repo: Failed to read data file") // Log with span context
		spanRead.RecordError(err)
		spanRead.SetStatus(codes.Error, err.Error())
		spanRead.End() // End span *before* returning
		return nil, err
	}
	spanRead.End()    // End span successfully
	bytes = fileBytes // Assign bytes only after successful read and span end

	// --- Unmarshal Span ---
	// Start this span using the context possibly updated by the previous span
	ctxUnmarshal, spanUnmarshal := tracer.Start(ctxRead, "repository.unmarshalData")
	spanUnmarshal.SetAttributes(
		telemetry.AppOperationKey.String("json_unmarshal"),
		telemetry.DBFilePathKey.String(r.filePath),
	)

	unmarshalErr := json.Unmarshal(bytes, &data)
	if unmarshalErr != nil {
		// Wrap the underlying error in a DatabaseError
		err = &errors.DatabaseError{Operation: "unmarshal json", Err: unmarshalErr}
		logrus.WithContext(ctxUnmarshal).WithError(err).Error("Repo: Failed to unmarshal JSON data") // Log with span context
		spanUnmarshal.RecordError(err)
		spanUnmarshal.SetStatus(codes.Error, err.Error())
		spanUnmarshal.End() // End span *before* returning
		return nil, err
	}
	spanUnmarshal.End() // End span successfully

	// If we reached here, both operations were successful
	return data, nil
}
