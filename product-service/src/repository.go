package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	common_errors "github.com/narender/common/errors"
	otel "github.com/narender/common/otel"
	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	UpdateStock(ctx context.Context, productID string, newStock int) error
	GetCurrentStockLevels(ctx context.Context) (map[string]int, error)
}

type productRepository struct {
	products map[string]Product
	mu       sync.RWMutex
	filePath string
}

func NewProductRepository(dataFilePath string) (ProductRepository, error) {
	logger := otel.GetLogger()
	logger.Infof("Repository: Initializing with file path: %s", dataFilePath)

	repo := &productRepository{
		products: make(map[string]Product),
		filePath: dataFilePath,
	}

	initCtx := context.Background()
	if _, statErr := os.Stat(repo.filePath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			logger.Warnf("Repository: Data file %s not found, starting with empty product list.", dataFilePath)
		} else {
			dbErr := &common_errors.DatabaseError{Operation: "StatFile", Err: statErr}
			logger.WithError(dbErr).Errorf("Repository: Failed to stat data file %s", dataFilePath)
			return nil, dbErr
		}
	} else {
		if err := repo.readData(initCtx); err != nil {
			logger.WithError(err).Errorf("Repository: Failed to initialize from %s", dataFilePath)
			return nil, fmt.Errorf("failed to initialize product repository from %s: %w", dataFilePath, err)
		}
	}
	logger.Infof("Repository: Initialized successfully, loaded %d products", len(repo.products))
	return repo, nil
}

const repoInstrumentationName = "product-service/repository"

func (r *productRepository) readData(ctx context.Context) error {
	logger := otel.GetLogger()
	tracer := otel.GetTracer(repoInstrumentationName)
	ctx, span := tracer.Start(ctx, "ProductRepository.readData",
		oteltrace.WithAttributes(otel.DBSystemKey.String("file"), otel.DBOperationKey.String("ReadFile")),
	)
	span.SetAttributes(otel.AttrDBFilePathKey.String(r.filePath))
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		dbErr := &common_errors.DatabaseError{
			Operation: "ReadFile",
			Err:       err,
		}
		logger.WithContext(ctx).WithError(dbErr).Error("Failed to read data file")
		span.RecordError(dbErr)
		span.SetStatus(codes.Error, "Failed to read data file")
		return dbErr
	}

	if len(data) == 0 {
		logger.WithContext(ctx).Warnf("Data file %s is empty, initializing empty product map.", r.filePath)
		r.products = make(map[string]Product)
		return nil
	}

	const unmarshalOperation = "UnmarshalJSON"
	span.SetAttributes(otel.DBOperationKey.String(unmarshalOperation))
	var productsMap map[string]Product
	if err := json.Unmarshal(data, &productsMap); err != nil {
		dbErr := &common_errors.DatabaseError{
			Operation: unmarshalOperation,
			Err:       fmt.Errorf("failed to unmarshal product data: %w", err),
		}
		logger.WithContext(ctx).WithError(dbErr).Error("Failed to unmarshal data")
		span.RecordError(dbErr)
		span.SetStatus(codes.Error, "Failed to unmarshal data")
		return dbErr
	}

	r.products = make(map[string]Product, len(productsMap))
	for key, p := range productsMap {
		r.products[key] = p
	}

	productCount := len(r.products)
	logger.WithContext(ctx).WithFields(logrus.Fields{
		"count": productCount,
		"path":  r.filePath,
	}).Debug("Successfully loaded products")
	span.SetAttributes(otel.AttrAppProductCount.Int(productCount))
	return nil
}

func (r *productRepository) GetAll(ctx context.Context) ([]Product, error) {
	logger := otel.GetLogger()
	tracer := otel.GetTracer(repoInstrumentationName)
	ctx, span := tracer.Start(ctx, "ProductRepository.GetAll",
		oteltrace.WithAttributes(otel.DBSystemKey.String("file"), otel.DBOperationKey.String("GetAll")),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.products) == 0 {
		logger.WithContext(ctx).Warn("GetAll called but no products loaded.")
	}
	result := make([]Product, 0, len(r.products))
	for _, p := range r.products {
		result = append(result, p)
	}
	span.SetAttributes(attribute.Int("db.rows_returned", len(result)))
	return result, nil
}

func (r *productRepository) GetByID(ctx context.Context, id string) (Product, error) {
	logger := otel.GetLogger()
	tracer := otel.GetTracer(repoInstrumentationName)
	ctx, span := tracer.Start(ctx, "ProductRepository.GetByID",
		oteltrace.WithAttributes(otel.DBSystemKey.String("file"), otel.DBOperationKey.String("GetByID")),
	)
	span.SetAttributes(otel.AttrAppProductIDKey.String(id))
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		msg := fmt.Sprintf("product with ID %s not found", id)
		logger.WithContext(ctx).WithField("product.id", id).Warn(msg)
		errNotFound := common_errors.ErrNotFound
		span.RecordError(errNotFound)
		span.SetStatus(codes.Unset, msg)
		return Product{}, errNotFound
	}
	return product, nil
}

func (r *productRepository) UpdateStock(ctx context.Context, productID string, newStock int) error {
	logger := otel.GetLogger()
	tracer := otel.GetTracer(repoInstrumentationName)
	ctx, span := tracer.Start(ctx, "ProductRepository.UpdateStock",
		oteltrace.WithAttributes(otel.DBSystemKey.String("file"), otel.DBOperationKey.String("UpdateStock")),
	)
	span.SetAttributes(
		otel.AttrAppProductIDKey.String(productID),
		otel.AttrProductNewStockKey.Int(newStock),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[productID]; !exists {
		logger.WithField("productID", productID).Warn("Attempted to update stock for non-existent product")
		span.SetStatus(codes.Error, "Product not found for stock update")
		return common_errors.ErrNotFound
	}

	// Retrieve the product, modify it, and put it back in the map
	p := r.products[productID]
	p.Stock = newStock
	r.products[productID] = p // Put the modified struct back

	if err := r.saveData(ctx); err != nil {
		span.SetStatus(codes.Error, "Failed to save updated stock data")
		return err
	}

	logger.WithContext(ctx).WithField("product.id", productID).Info("Successfully updated stock")
	return nil
}

func (r *productRepository) GetCurrentStockLevels(ctx context.Context) (map[string]int, error) {
	logger := otel.GetLogger()
	tracer := otel.GetTracer(repoInstrumentationName)
	ctx, span := tracer.Start(ctx, "ProductRepository.GetCurrentStockLevels")
	defer span.End()

	logger.Debug("Repository: GetCurrentStockLevels called")
	r.mu.RLock()
	defer r.mu.RUnlock()
	stockLevels := make(map[string]int, len(r.products))
	for id, product := range r.products {
		stockLevels[id] = product.Stock
	}
	span.SetAttributes(otel.AttrAppProductCount.Int(len(stockLevels)))
	return stockLevels, nil
}

func (r *productRepository) saveData(ctx context.Context) error {
	logger := otel.GetLogger()
	tracer := otel.GetTracer(repoInstrumentationName)
	ctx, span := tracer.Start(ctx, "ProductRepository.saveData",
		oteltrace.WithAttributes(otel.DBSystemKey.String("file"), otel.DBOperationKey.String("WriteFile")),
	)
	span.SetAttributes(otel.AttrDBFilePathKey.String(r.filePath))
	defer span.End()

	r.mu.RLock()
	productsToSave := make(map[string]Product, len(r.products))
	for k, v := range r.products {
		productsToSave[k] = v
	}
	r.mu.RUnlock()

	const marshalOperation = "MarshalJSON"
	span.SetAttributes(otel.DBOperationKey.String(marshalOperation))
	data, err := json.MarshalIndent(productsToSave, "", "  ")
	if err != nil {
		dbErr := &common_errors.DatabaseError{
			Operation: marshalOperation,
			Err:       fmt.Errorf("failed to marshal product data: %w", err),
		}
		logger.WithContext(ctx).WithError(dbErr).Error("Failed to marshal data for saving")
		span.RecordError(dbErr)
		span.SetStatus(codes.Error, "Failed to marshal data")
		return dbErr
	}

	span.SetAttributes(otel.DBOperationKey.String("WriteFile"))
	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		dbErr := &common_errors.DatabaseError{
			Operation: "WriteFile",
			Err:       fmt.Errorf("failed to write product data to file '%s': %w", r.filePath, err),
		}
		logger.WithContext(ctx).WithError(dbErr).Error("Failed to write data file")
		span.RecordError(dbErr)
		span.SetStatus(codes.Error, "Failed to write data file")
		return dbErr
	}

	logger.WithContext(ctx).Debug("Successfully saved data")
	return nil
}
