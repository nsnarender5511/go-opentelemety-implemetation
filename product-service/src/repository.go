package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"sync"

	commonconst "github.com/narender/common/constants"
	"github.com/narender/common/debugutils"
	commonerrors "github.com/narender/common/errors"
	commonlog "github.com/narender/common/log"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const repositoryScopeName = "github.com/narender/product-service/repository"

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
}

func NewProductRepository(dataFilePath string) (ProductRepository, error) {
	logger := commonlog.L
	logger.Info("Repository: Initializing", slog.String("file_path", dataFilePath))

	repo := &productRepository{
		products: make(map[string]Product),
		filePath: dataFilePath,
	}

	initCtx := context.Background()
	if _, statErr := os.Stat(repo.filePath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			logger.Warn("Repository: Data file not found, starting empty", slog.String("file_path", dataFilePath))
		} else {
			loadErr := fmt.Errorf("repository: failed to stat data file %s: %w", dataFilePath, statErr)
			logger.Error("Repository: Failed to stat data file", slog.Any("error", loadErr))

			return nil, loadErr
		}
	} else {
		if loadErr := repo.loadData(initCtx); loadErr != nil {
			logger.Error("Repository: Failed to initialize from file",
				slog.String("file_path", dataFilePath),
				slog.Any("error", loadErr),
			)
			return nil, fmt.Errorf("failed to initialize product repository from %s: %w", dataFilePath, loadErr)
		}
	}
	logger.Info("Repository: Initialized successfully", slog.Int("loaded_count", len(repo.products)))
	return repo, nil
}

func (r *productRepository) loadData(ctx context.Context) (opErr error) {
	const operation = "loadData"
	logger := commonlog.L

	mc := commonmetric.StartMetricsTimer(commonconst.RepositoryLayer, operation)
	fileAttr := attribute.String("file.path", r.filePath)
	defer mc.End(ctx, &opErr, fileAttr)

	ctx, spanner := commontrace.StartSpan(ctx, repositoryScopeName, operation, commonconst.RepositoryLayer,
		semconv.DBSystemKey.String("file"),
		semconv.DBOperationKey.String("READ"),
		fileAttr,
	)
	customMapper := func(err error) codes.Code {
		if err == nil {
			return codes.Ok
		}
		if errors.Is(err, os.ErrNotExist) {
			return codes.Ok
		}
		return codes.Error
	}
	defer spanner.End(&opErr, customMapper)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.RepositoryLayer))
		}
	}()

	spanner.AddEvent("Acquiring lock for loadData")
	r.mu.Lock()
	defer r.mu.Unlock()
	spanner.AddEvent("Lock acquired for loadData")

	spanner.AddEvent("Starting file read", trace.WithAttributes(fileAttr))
	data, err := os.ReadFile(r.filePath)
	readErrEventAttrs := []attribute.KeyValue{fileAttr}
	if err != nil {
		readErrEventAttrs = append(readErrEventAttrs, attribute.Bool("read.error", true), attribute.String("error.message", err.Error()))
	} else {
		readErrEventAttrs = append(readErrEventAttrs, attribute.Bool("read.error", false), attribute.Int("read.size", len(data)))
	}
	spanner.AddEvent("File read finished", trace.WithAttributes(readErrEventAttrs...))

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("WARN: Data file not found during load, initializing empty map", slog.String("file_path", r.filePath))
			r.products = make(map[string]Product)
			return nil
		}
		opErr = fmt.Errorf("failed to read data file '%s': %w", r.filePath, err)
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.RepositoryLayer, operation, fileAttr)
		return opErr
	}

	if len(data) == 0 {
		logger.Warn("WARN: Data file is empty, initializing empty product map", slog.String("file_path", r.filePath))
		r.products = make(map[string]Product)
		return nil
	}

	spanner.AddEvent("Starting JSON unmarshal")
	var productsMap map[string]Product
	if err := json.Unmarshal(data, &productsMap); err != nil {
		opErr = fmt.Errorf("failed to unmarshal product data from '%s': %w", r.filePath, err)
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.RepositoryLayer, operation, fileAttr, attribute.Int("data.size", len(data)))
		spanner.AddEvent("JSON unmarshal failed")
		return opErr
	}
	spanner.AddEvent("JSON unmarshal successful", trace.WithAttributes(attribute.Int("data.size", len(data))))

	newProductsMap := make(map[string]Product, len(productsMap))
	newProductsSlice := make([]Product, 0, len(productsMap))
	for key, p := range productsMap {
		newProductsMap[key] = p
		newProductsSlice = append(newProductsSlice, p)
	}
	r.products = newProductsMap
	r.productsSlice = newProductsSlice

	productCount := len(r.products)
	logger.Debug("Repository: Successfully loaded products", slog.Int("count", productCount))
	spanner.SetAttributes(attribute.Int("products.loaded.count", productCount))
	return nil
}

func (r *productRepository) GetAll(ctx context.Context) (products []Product, opErr error) {
	const operation = "GetAllProducts"
	logger := commonlog.L

	mc := commonmetric.StartMetricsTimer(commonconst.RepositoryLayer, operation)
	defer mc.End(ctx, &opErr)

	ctx, spanner := commontrace.StartSpan(ctx, repositoryScopeName, operation, commonconst.RepositoryLayer,
		semconv.DBSystemKey.String("memory"),
		semconv.DBOperationKey.String("READ_ALL"),
	)
	defer spanner.End(&opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.RepositoryLayer))
		}
	}()

	logger.Info("Repository: GetAll called")

	spanner.AddEvent("Acquiring read lock for GetAll")
	r.mu.RLock()
	defer r.mu.RUnlock()
	spanner.AddEvent("Read lock acquired for GetAll")

	products = r.productsSlice
	if len(products) == 0 {
		logger.Warn("Repository: GetAll called but no products loaded/cached.")
		spanner.AddEvent("Product cache is empty")
	}

	spanner.SetAttributes(attribute.Int("products.returned.count", len(products)))
	logger.Info("Repository: GetAll returning products from cache", slog.Int("count", len(products)))
	return products, nil
}

func (r *productRepository) GetByID(ctx context.Context, id string) (product Product, opErr error) {
	const operation = "GetProductByID"
	logger := commonlog.L
	productIdAttr := attribute.String("product.id", id)

	mc := commonmetric.StartMetricsTimer(commonconst.RepositoryLayer, operation)
	defer mc.End(ctx, &opErr, productIdAttr)

	ctx, spanner := commontrace.StartSpan(ctx, repositoryScopeName, operation, commonconst.RepositoryLayer,
		semconv.DBSystemKey.String("memory"),
		semconv.DBOperationKey.String("READ"),
		productIdAttr,
	)
	notFoundMapper := func(err error) codes.Code {
		if err == nil {
			return codes.Ok
		}
		if errors.Is(err, commonerrors.ErrNotFound) {
			return codes.Ok
		}
		return codes.Error
	}
	defer spanner.End(&opErr, notFoundMapper)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.RepositoryLayer), productIdAttr)
		}
	}()

	logger.Info("Repository: GetByID called", slog.String("product_id", id))

	spanner.AddEvent("Acquiring read lock for GetByID")
	r.mu.RLock()
	defer r.mu.RUnlock()
	spanner.AddEvent("Read lock acquired for GetByID")

	product, exists := r.products[id]
	if !exists {
		opErr = fmt.Errorf("product with id '%s' not found: %w", id, commonerrors.ErrNotFound)
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.RepositoryLayer, operation, productIdAttr)
		spanner.AddEvent("Product not found in map")
		return Product{}, opErr
	}

	spanner.AddEvent("Product found in map")
	spanner.SetAttributes(attribute.String("product.name", product.Name))
	logger.Info("Repository: GetByID found product in cache", slog.String("product_id", id))
	return product, nil
}

func (r *productRepository) UpdateStock(ctx context.Context, productID string, newStock int) (opErr error) {
	const operation = "UpdateStock"
	logger := commonlog.L
	productIdAttr := attribute.String("product.id", productID)
	newStockAttr := attribute.Int("product.new_stock", newStock)
	attrs := []attribute.KeyValue{productIdAttr, newStockAttr}

	mc := commonmetric.StartMetricsTimer(commonconst.RepositoryLayer, operation)
	defer mc.End(ctx, &opErr, attrs...)

	initialSpanAttrs := []attribute.KeyValue{
		semconv.DBSystemKey.String("memory"),
		semconv.DBOperationKey.String("UPDATE"),
	}
	initialSpanAttrs = append(initialSpanAttrs, attrs...)
	ctx, spanner := commontrace.StartSpan(ctx, repositoryScopeName, operation, commonconst.RepositoryLayer, initialSpanAttrs...)
	defer spanner.End(&opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.RepositoryLayer), productIdAttr, newStockAttr)
		}
	}()

	logger.Info("Repository: UpdateStock called", slog.String("product_id", productID), slog.Int("new_stock", newStock))

	if newStock < 0 {
		opErr = fmt.Errorf("invalid stock value %d: %w", newStock, commonerrors.ErrValidation)
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.RepositoryLayer, operation, attrs...)
		return opErr
	}

	spanner.AddEvent("Acquiring write lock for UpdateStock")
	r.mu.Lock()
	spanner.AddEvent("Write lock acquired for UpdateStock")

	product, ok := r.products[productID]
	if !ok {
		r.mu.Unlock()
		opErr = fmt.Errorf("product with id '%s' not found for update: %w", productID, commonerrors.ErrNotFound)
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.RepositoryLayer, operation, attrs...)
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
		logger.Error("Repository internal inconsistency", slog.String("error", errMsg), slog.String("product_id", productID))
		opErr = fmt.Errorf("%s: %w", errMsg, commonerrors.ErrInternal)
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.RepositoryLayer, operation, attrs...)
		return opErr
	}

	spanner.SetAttributes(attribute.Int("product.old_stock", oldStock))
	logger.Info("Repository: Product stock updated in memory", slog.String("product_id", productID), slog.Int("old_stock", oldStock), slog.Int("new_stock", newStock))

	spanner.AddEvent("Calling saveData to persist changes")
	if saveErr := r.saveData(ctx); saveErr != nil {
		opErr = fmt.Errorf("failed to save data after updating stock for product '%s': %w", productID, saveErr)
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.RepositoryLayer, operation, attrs...)
		return opErr
	}
	spanner.AddEvent("saveData completed")

	return nil
}

func (r *productRepository) saveData(ctx context.Context) (opErr error) {
	const operation = "saveData"
	logger := commonlog.L
	fileAttr := attribute.String("file.path", r.filePath)

	mc := commonmetric.StartMetricsTimer(commonconst.RepositoryLayer, operation)
	defer mc.End(ctx, &opErr, fileAttr)

	ctx, spanner := commontrace.StartSpan(ctx, repositoryScopeName, operation, commonconst.RepositoryLayer,
		semconv.DBSystemKey.String("file"),
		semconv.DBOperationKey.String("WRITE"),
		fileAttr,
	)
	defer spanner.End(&opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.RepositoryLayer), fileAttr)
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
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.RepositoryLayer, operation, fileAttr)
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
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.RepositoryLayer, operation, fileAttr)
		return opErr
	}

	logger.Debug("Repository: Successfully saved product data", slog.String("file_path", r.filePath))
	return nil
}
