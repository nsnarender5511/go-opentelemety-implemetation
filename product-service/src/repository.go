package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"sync"

	commonconst "github.com/narender/common/constants"
	"github.com/narender/common/debugutils"
	commonerrors "github.com/narender/common/errors"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/common/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/narender/common/globals"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const repositoryScopeName = "github.com/narender/product-service/repository"

// ProductRepository defines the interface for accessing product data.
type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	UpdateStock(ctx context.Context, productID string, newStock int) error
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
	const operation = "NewProductRepository"
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

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operationName, rec)
			r.logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operationName), slog.String("layer", commonconst.RepositoryLayer))
		}
	}()

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

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operationName, rec)
			r.logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operationName), slog.String("layer", commonconst.RepositoryLayer), productIdAttr)
		}
	}()

	r.logger.InfoContext(ctx, "Repository: GetByID called", slog.String("product_id", id), slog.String("operation", operationName))

	spanner.AddEvent("Acquiring read lock for GetByID")
	r.mu.RLock()
	defer r.mu.RUnlock()
	spanner.AddEvent("Read lock acquired for GetByID")

	product, exists := r.products[id]
	if !exists {
		opErr = fmt.Errorf("product with id '%s' not found: %w", id, commonerrors.ErrNotFound)
		r.logger.Warn("Product not found", slog.String("product_id", id))
		spanner.RecordError(opErr, trace.WithAttributes(productIdAttr))
		spanner.SetStatus(codes.Ok, "Product not found")
		spanner.AddEvent("Product not found in map")
		return Product{}, opErr
	}

	spanner.AddEvent("Product found in map")
	spanner.SetAttributes(attribute.String("product.name", product.Name))
	r.logger.InfoContext(ctx, "Repository: GetByID found product in cache", slog.String("product_id", id), slog.String("operation", operationName))
	return product, nil
}

func (r *productRepository) UpdateStock(ctx context.Context, productID string, newStock int) (opErr error) {
	operationName := utils.GetCallerFunctionName(2)
	productIdAttr := attribute.String("product.id", productID)
	newStockAttr := attribute.Int("product.new_stock", newStock)
	attrs := []attribute.KeyValue{productIdAttr, newStockAttr}

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, attrs...)

	initialSpanAttrs := []attribute.KeyValue{
		semconv.DBSystemKey.String("memory"),
		semconv.DBOperationKey.String("UPDATE"),
	}
	initialSpanAttrs = append(initialSpanAttrs, attrs...)
	ctx, spanner := commontrace.StartSpan(ctx, initialSpanAttrs...)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operationName, rec)
			r.logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operationName), slog.String("layer", commonconst.RepositoryLayer), productIdAttr, newStockAttr)
		}
	}()

	r.logger.InfoContext(ctx, "Repository: UpdateStock called", slog.String("product_id", productID), slog.Int("new_stock", newStock), slog.String("operation", operationName))

	if newStock < 0 {
		opErr = fmt.Errorf("invalid stock value %d: %w", newStock, commonerrors.ErrValidation)
		commonerrors.HandleLayerError(ctx, r.logger, spanner, opErr, commonconst.RepositoryLayer, operationName, attrs...)
		return opErr
	}

	spanner.AddEvent("Acquiring write lock for UpdateStock")
	r.mu.Lock()
	spanner.AddEvent("Write lock acquired for UpdateStock")

	product, ok := r.products[productID]
	if !ok {
		r.mu.Unlock()
		spanner.AddEvent("Write lock released (product not found)")
		opErr = fmt.Errorf("product with id '%s' not found for update: %w", productID, commonerrors.ErrNotFound)
		commonerrors.HandleLayerError(ctx, r.logger, spanner, opErr, commonconst.RepositoryLayer, operationName, attrs...)
		return opErr
	}

	oldStock := product.Stock
	product.Stock = newStock
	r.products[productID] = product

	foundInSlice := false
	for i := range r.productsSlice {
		if r.productsSlice[i].ProductID == productID {
			r.productsSlice[i].Stock = newStock
			foundInSlice = true
			break
		}
	}
	r.mu.Unlock()
	spanner.AddEvent("Write lock released after UpdateStock")

	if !foundInSlice {
		errMsg := "product found in map but not in slice during UpdateStock"
		r.logger.Error("Repository internal inconsistency", slog.String("error", errMsg), slog.String("product_id", productID))
		opErr = fmt.Errorf("%s: %w", errMsg, commonerrors.ErrInternal)
		commonerrors.HandleLayerError(ctx, r.logger, spanner, opErr, commonconst.RepositoryLayer, operationName, attrs...)
		return opErr
	}

	spanner.SetAttributes(attribute.Int("product.old_stock", oldStock))
	r.logger.InfoContext(ctx, "Repository: Product stock updated in memory", slog.String("product_id", productID), slog.Int("old_stock", oldStock), slog.Int("new_stock", newStock), slog.String("operation", operationName))

	spanner.AddEvent("Calling saveData to persist changes")
	if saveErr := r.saveData(ctx); saveErr != nil {
		opErr = fmt.Errorf("failed persistence after stock update for '%s': %w", productID, saveErr)
		r.logger.Error("Failed to persist stock update", slog.String("product_id", productID), slog.Any("error", saveErr))
		return opErr
	}
	spanner.AddEvent("saveData completed successfully")

	return nil
}

func (r *productRepository) saveData(ctx context.Context) (opErr error) {
	operationName := utils.GetCallerFunctionName(2)
	fileAttr := attribute.String("file.path", r.filePath)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, fileAttr)

	ctx, spanner := commontrace.StartSpan(ctx,
		semconv.DBSystemKey.String("file"),
		semconv.DBOperationKey.String("WRITE"),
		fileAttr,
	)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operationName, rec)
			r.logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operationName), slog.String("layer", commonconst.RepositoryLayer), fileAttr)
		}
	}()

	spanner.AddEvent("Acquiring read lock for saveData (to marshal)")
	r.mu.RLock()
	spanner.AddEvent("Read lock acquired")
	data, err := json.MarshalIndent(r.products, "", "  ")
	r.mu.RUnlock()
	spanner.AddEvent("Read lock released after marshalling")

	if err != nil {
		opErr = fmt.Errorf("failed to marshal product data for saving: %w", err)
		commonerrors.HandleLayerError(ctx, r.logger, spanner, opErr, commonconst.RepositoryLayer, operationName, fileAttr)
		return opErr
	}

	spanner.AddEvent("Starting file write", trace.WithAttributes(fileAttr, attribute.Int("data.size", len(data))))
	err = os.WriteFile(r.filePath, data, 0644)
	writeErrEventAttrs := []attribute.KeyValue{fileAttr}
	if err != nil {
		writeErrEventAttrs = append(writeErrEventAttrs, attribute.Bool("write.error", true), attribute.String("error.message", err.Error()))
	} else {
		writeErrEventAttrs = append(writeErrEventAttrs, attribute.Bool("write.error", false))
	}
	spanner.AddEvent("File write finished", trace.WithAttributes(writeErrEventAttrs...))

	if err != nil {
		opErr = fmt.Errorf("failed to write data file '%s': %w", r.filePath, err)
		commonerrors.HandleLayerError(ctx, r.logger, spanner, opErr, commonconst.RepositoryLayer, operationName, fileAttr)
		return opErr
	}

	r.logger.InfoContext(ctx, "Product data saved successfully", slog.String("file_path", r.filePath), slog.String("operation", operationName))
	return nil
}
