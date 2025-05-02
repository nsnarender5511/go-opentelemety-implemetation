package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os" 

	
	"sync"
	"time"

	"github.com/narender/common/logging" 
	"github.com/narender/common/telemetry/manager"
	"github.com/narender/common/telemetry/metric" 
	"github.com/narender/common/telemetry/trace"  
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	
	oteMetric "go.opentelemetry.io/otel/metric" 
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace" 

	
	"go.uber.org/zap" 
)


const repositoryScopeName = "github.com/narender/product-service/repository"
const repoLayerName = "repository"

var ErrNotFound = errors.New("product not found")



type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	UpdateStock(ctx context.Context, productID string, newStock int) error
}

type productRepository struct {
	products map[string]Product
	mu       sync.RWMutex
	filePath string
}

func NewProductRepository(dataFilePath string) (ProductRepository, error) {
	logger := logging.GetBaseLogger() 
	logger.Info("Repository: Initializing", zap.String("file_path", dataFilePath))

	repo := &productRepository{
		products: make(map[string]Product),
		filePath: dataFilePath,
	}

	
	meter := manager.GetMeter(repositoryScopeName)
	productCountGaugeCallback := func(ctx context.Context, observer oteMetric.Int64Observer) error { 
		repo.mu.RLock()
		count := len(repo.products)
		repo.mu.RUnlock()
		observer.Observe(int64(count))
		return nil
	}
	_, err := meter.Int64ObservableGauge(
		"product.repository.count",                             
		oteMetric.WithInt64Callback(productCountGaugeCallback), 
		oteMetric.WithDescription("Measures the number of products currently loaded in the repository"), 
		oteMetric.WithUnit("{products}"), 
	)
	if err != nil {
		logger.Error("Failed to create product.repository.count observable gauge", zap.Error(err))
	}

	
	initCtx := context.Background()
	if _, statErr := os.Stat(repo.filePath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			logger.Warn("Repository: Data file not found, starting empty", zap.String("file_path", dataFilePath))
		} else {
			loadErr := fmt.Errorf("repository: failed to stat data file %s: %w", dataFilePath, statErr)
			logger.Error("Repository: Failed to stat data file", zap.Error(loadErr))
			
			return nil, loadErr
		}
	} else {
		if loadErr := repo.loadData(initCtx); loadErr != nil {
			logger.Error("Repository: Failed to initialize from file",
				zap.String("file_path", dataFilePath),
				zap.Error(loadErr),
			)
			return nil, fmt.Errorf("failed to initialize product repository from %s: %w", dataFilePath, loadErr)
		}
	}
	logger.Info("Repository: Initialized successfully", zap.Int("loaded_count", len(repo.products)))
	return repo, nil
}


func (r *productRepository) loadData(ctx context.Context) (opErr error) {
	const operation = "loadData"
	startTime := time.Now()
	defer func() {
		
		metric.RecordOperationMetrics(ctx, repoLayerName, operation, startTime, opErr)
	}()

	simulateDelayIfEnabled()
	logger := logging.LoggerFromContext(ctx)
	
	ctx, span := trace.StartSpan(ctx, repositoryScopeName, "ProductRepository."+operation,
		semconv.DBSystemKey.String("file"),
		semconv.DBOperationKey.String("READ"),
		attribute.String("file.path", r.filePath),
	)
	defer span.End()

	fileOpStartTime := time.Now() 
	logger.Info("Repository: Loading data", zap.String("file_path", r.filePath))

	span.AddEvent("Acquiring lock for loadData")
	r.mu.Lock()
	defer r.mu.Unlock()
	span.AddEvent("Lock acquired for loadData")

	span.AddEvent("Starting file read", oteltrace.WithAttributes(attribute.String("file.path", r.filePath)))
	data, err := os.ReadFile(r.filePath)
	fileIODuration := time.Since(fileOpStartTime).Seconds()
	
	
	span.AddEvent("File read finished", oteltrace.WithAttributes(
		attribute.Float64("duration_sec", fileIODuration),
		attribute.Bool("read_error", err != nil),
	))

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("WARN: Data file not found during load, initializing empty map", zap.String("file_path", r.filePath))
			r.products = make(map[string]Product)
			span.SetStatus(codes.Ok, "File not found, initialized empty map")
			
			opErr = nil
			return opErr 
		}
		opErr = fmt.Errorf("failed to read data file '%s': %w", r.filePath, err)
		logger.Error("Repository: Failed to read data file", zap.Error(opErr))
		span.RecordError(opErr)
		span.SetStatus(codes.Error, "failed to read file")
		return opErr 
	}

	if len(data) == 0 {
		logger.Warn("WARN: Data file is empty, initializing empty product map", zap.String("file_path", r.filePath))
		r.products = make(map[string]Product)
		span.SetStatus(codes.Ok, "File empty, initialized empty map")
		opErr = nil  
		return opErr 
	}

	span.AddEvent("Starting JSON unmarshal")
	var productsMap map[string]Product
	if err := json.Unmarshal(data, &productsMap); err != nil {
		opErr = fmt.Errorf("failed to unmarshal product data from '%s': %w", r.filePath, err)
		logger.Error("Repository: Failed to unmarshal JSON data", zap.Error(opErr))
		span.RecordError(opErr)
		span.SetStatus(codes.Error, "failed to unmarshal JSON")
		span.AddEvent("JSON unmarshal failed")
		return opErr
	}
	span.AddEvent("JSON unmarshal successful", oteltrace.WithAttributes(attribute.Int("data.size", len(data))))

	r.products = make(map[string]Product, len(productsMap))
	for key, p := range productsMap {
		r.products[key] = p
	}

	productCount := len(r.products)
	logger.Debug("Repository: Successfully loaded products", zap.Int("count", productCount))
	span.SetAttributes(attribute.Int("products.loaded.count", productCount))
	span.SetStatus(codes.Ok, "")
	return nil 
}


func (r *productRepository) GetAll(ctx context.Context) (products []Product, opErr error) {
	const operation = "GetAll"
	startTime := time.Now()
	defer func() {
		
		metric.RecordOperationMetrics(ctx, repoLayerName, operation, startTime, opErr)
	}()

	simulateDelayIfEnabled()
	logger := logging.LoggerFromContext(ctx)
	
	ctx, span := trace.StartSpan(ctx, repositoryScopeName, "ProductRepository."+operation,
		semconv.DBSystemKey.String("memory"),
		semconv.DBOperationKey.String("READ_ALL"),
	)
	defer span.End()

	logger.Info("Repository: GetAll called")

	span.AddEvent("Acquiring read lock for GetAll")
	r.mu.RLock()
	defer r.mu.RUnlock()
	span.AddEvent("Read lock acquired for GetAll")

	if len(r.products) == 0 {
		logger.Warn("Repository: GetAll called but no products loaded.")
		span.AddEvent("Product map is empty")
	}
	products = make([]Product, 0, len(r.products)) 
	for _, p := range r.products {
		products = append(products, p)
	}
	span.SetAttributes(attribute.Int("products.returned.count", len(products)))
	span.SetStatus(codes.Ok, "")
	logger.Info("Repository: GetAll returning products", zap.Int("count", len(products)))
	return products, nil 
}


func (r *productRepository) GetByID(ctx context.Context, id string) (product Product, opErr error) {
	const operation = "GetByID"
	startTime := time.Now()
	productIdAttr := attribute.String("product.id", id)
	defer func() {
		
		metric.RecordOperationMetrics(ctx, repoLayerName, operation, startTime, opErr, productIdAttr)
	}()

	simulateDelayIfEnabled()
	logger := logging.LoggerFromContext(ctx)
	
	ctx, span := trace.StartSpan(ctx, repositoryScopeName, "ProductRepository."+operation,
		semconv.DBSystemKey.String("memory"),
		semconv.DBOperationKey.String("READ"),
		productIdAttr,
	)
	defer span.End()

	logger.Info("Repository: GetByID called", zap.String("product.id", id))

	span.AddEvent("Acquiring read lock for GetByID")
	r.mu.RLock()
	defer r.mu.RUnlock()
	span.AddEvent("Read lock acquired for GetByID")

	product, exists := r.products[id] 
	if !exists {
		opErr = ErrNotFound 
		logger.Warn("Repository: Product not found", zap.String("product.id", id))
		span.AddEvent("Product not found in map", oteltrace.WithAttributes(productIdAttr))
		span.RecordError(opErr, oteltrace.WithAttributes(attribute.String("product.lookup.id", id)))
		span.SetStatus(codes.Error, opErr.Error())
		return Product{}, opErr 
	}

	logger.Info("Repository: Product found", zap.String("product.id", id))
	span.SetStatus(codes.Ok, "")
	return product, nil 
}


func (r *productRepository) UpdateStock(ctx context.Context, productID string, newStock int) (opErr error) {
	const operation = "UpdateStock"
	startTime := time.Now()
	productIdAttr := attribute.String("product.id", productID)
	defer func() {
		
		metric.RecordOperationMetrics(ctx, repoLayerName, operation, startTime, opErr, productIdAttr)
	}()

	simulateDelayIfEnabled()
	logger := logging.LoggerFromContext(ctx)
	
	ctx, span := trace.StartSpan(ctx, repositoryScopeName, "ProductRepository."+operation,
		semconv.DBSystemKey.String("memory"),
		semconv.DBOperationKey.String("UPDATE"),
		productIdAttr,
		attribute.Int("product.new_stock", newStock),
	)
	defer span.End()

	logger.Info("Repository: UpdateStock called",
		zap.String("product.id", productID),
		zap.Int("new.stock", newStock),
	)

	span.AddEvent("Acquiring write lock for UpdateStock")
	r.mu.Lock()
	defer r.mu.Unlock()
	span.AddEvent("Write lock acquired for UpdateStock")

	product, exists := r.products[productID]
	if !exists {
		opErr = ErrNotFound 
		logger.Warn("Repository: Product not found for update", zap.String("product.id", productID))
		span.AddEvent("Product not found for update", oteltrace.WithAttributes(productIdAttr))
		span.RecordError(opErr, oteltrace.WithAttributes(attribute.String("product.lookup.id", productID)))
		span.SetStatus(codes.Error, opErr.Error())
		return opErr
	}

	oldStock := product.Stock
	span.AddEvent("Updating stock in memory", oteltrace.WithAttributes(
		attribute.Int("product.old_stock", oldStock),
	))
	product.Stock = newStock
	r.products[productID] = product

	
	
	
	
	
	
	

	span.AddEvent("Calling saveData to persist stock update")
	if err := r.saveData(ctx); err != nil {
		opErr = err 
		
		span.SetStatus(codes.Error, "failed to save updated data")
		
		return opErr
	}

	logger.Info("Repository: Stock updated and data saved successfully", zap.String("product.id", productID))
	span.SetStatus(codes.Ok, "")
	return nil 
}


func (r *productRepository) saveData(ctx context.Context) (opErr error) {
	const operation = "saveData"
	startTime := time.Now()
	defer func() {
		
		metric.RecordOperationMetrics(ctx, repoLayerName, operation, startTime, opErr)
	}()

	simulateDelayIfEnabled()
	logger := logging.LoggerFromContext(ctx)
	
	ctx, span := trace.StartSpan(ctx, repositoryScopeName, "ProductRepository."+operation,
		semconv.DBSystemKey.String("file"),
		semconv.DBOperationKey.String("WRITE"),
		attribute.String("file.path", r.filePath),
		attribute.Int("products.to_save.count", len(r.products)),
	)
	defer span.End()

	logger.Info("Repository: Saving data", zap.String("file_path", r.filePath))
	fileOpStartTime := time.Now() 

	span.AddEvent("Starting JSON marshal")
	data, err := json.MarshalIndent(r.products, "", "  ")
	if err != nil {
		opErr = fmt.Errorf("failed to marshal product data: %w", err)
		logger.Error("Repository: Failed to marshal data for saving", zap.Error(opErr))
		span.RecordError(opErr)
		span.SetStatus(codes.Error, "failed to marshal JSON")
		span.AddEvent("JSON marshal failed")
		return opErr
	}
	span.AddEvent("JSON marshal successful", oteltrace.WithAttributes(attribute.Int("data.size", len(data))))

	span.AddEvent("Starting file write", oteltrace.WithAttributes(attribute.String("file.path", r.filePath)))
	err = os.WriteFile(r.filePath, data, 0644)
	fileIODuration := time.Since(fileOpStartTime).Seconds()
	
	span.AddEvent("File write finished", oteltrace.WithAttributes(
		attribute.Float64("duration_sec", fileIODuration),
		attribute.Bool("write_error", err != nil),
	))

	if err != nil {
		opErr = fmt.Errorf("failed to write data file '%s': %w", r.filePath, err)
		logger.Error("Repository: Failed to write data file", zap.Error(opErr))
		span.RecordError(opErr)
		span.SetStatus(codes.Error, "failed to write file")
		return opErr
	}

	logger.Debug("Repository: Data saved successfully", zap.String("file_path", r.filePath))
	span.SetStatus(codes.Ok, "")
	return nil 
}
