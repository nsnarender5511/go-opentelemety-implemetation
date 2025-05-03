package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"sync"

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

	// Attempt to load initial data
	if err := repo.loadData(context.Background()); err != nil { // Use context.Background() for startup
		// Log the error but potentially continue with an empty repository
		repo.logger.Error("Failed to load initial product data", slog.String("filePath", dataFilePath), slog.Any("error", err))
	}

	return repo
}

// loadData reads the JSON file and populates the in-memory store.
func (r *productRepository) loadData(ctx context.Context) (opErr error) {
	operationName := utils.GetCallerFunctionName(2)
	fileAttr := attribute.String("file.path", r.filePath)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, fileAttr)

	ctx, spanner := commontrace.StartSpan(ctx,
		semconv.DBSystemKey.String("file"),
		semconv.DBOperationKey.String("READ"),
		fileAttr,
	)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Repository: Loading data from file", slog.String("file_path", r.filePath), slog.String("operation", operationName))
	spanner.AddEvent("Reading data file", trace.WithAttributes(fileAttr))
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, log a warning but maybe it's okay to start empty?
			r.logger.WarnContext(ctx, "Product data file does not exist, starting with empty repository", slog.String("file_path", r.filePath))
			// Depending on requirements, you might want to return an error here
			// opErr = fmt.Errorf("product data file not found '%s': %w", r.filePath, err)
			return nil // Allow starting empty
		} else {
			opErr = fmt.Errorf("failed to read products file '%s': %w", r.filePath, err)
			logLevel := slog.LevelError
			eventName := "file_read_error"
			r.logger.Log(ctx, logLevel, "Failed to read products file",
				slog.String("layer", "repository"),
				slog.String("operation", operationName),
				slog.String("error", opErr.Error()),
				slog.String("file.path", r.filePath),
			)
			if spanner != nil {
				spanAttrs := []attribute.KeyValue{
					attribute.String("layer", "repository"),
					attribute.String("operation", operationName),
					attribute.String("error.message", opErr.Error()),
					fileAttr,
				}
				spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
				spanner.SetStatus(codes.Error, opErr.Error())
			}
			return opErr
		}
	}
	spanner.AddEvent("Data read successfully, unmarshalling...")

	// Acquire write lock to modify internal state
	spanner.AddEvent("Acquiring write lock for loadData")
	r.mu.Lock()
	defer r.mu.Unlock()
	spanner.AddEvent("Write lock acquired")

	if err := json.Unmarshal(data, &r.products); err != nil {
		opErr = fmt.Errorf("failed to unmarshal products data from '%s': %w", r.filePath, err)
		logLevel := slog.LevelError
		eventName := "unmarshal_error"
		r.logger.Log(ctx, logLevel, "Failed to unmarshal products data",
			slog.String("layer", "repository"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
			slog.String("file.path", r.filePath),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "repository"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
				fileAttr,
			}
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
			spanner.SetStatus(codes.Error, opErr.Error())
		}
		return opErr
	}

	// Populate the slice for GetAll
	r.productsSlice = make([]Product, 0, len(r.products))
	for _, p := range r.products {
		r.productsSlice = append(r.productsSlice, p)
	}
	spanner.AddEvent("Data unmarshalled and cached successfully")
	r.logger.InfoContext(ctx, "Repository: Data loaded successfully", slog.String("file_path", r.filePath), slog.Int("product_count", len(r.products)), slog.String("operation", operationName))
	spanner.SetAttributes(attribute.Int("products.loaded.count", len(r.products)))

	return nil
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
	notFoundMapper := func(err error) codes.Code {
		if err == nil {
			return codes.Ok
		}
		if errors.Is(err, commonerrors.ErrNotFound) {
			return codes.Ok
		}
		return codes.Error
	}
	defer commontrace.EndSpan(spanner, &opErr, notFoundMapper)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Repository: GetByID called", slog.String("product_id", id), slog.String("operation", operationName))

	spanner.AddEvent("Acquiring read lock for GetByID")
	r.mu.RLock()
	defer r.mu.RUnlock()
	spanner.AddEvent("Read lock acquired for GetByID")

	product, exists := r.products[id]
	if !exists {
		opErr = fmt.Errorf("product with id '%s' not found: %w", id, commonerrors.ErrNotFound)
		logLevel := slog.LevelWarn
		eventName := "resource_not_found"
		r.logger.Log(ctx, logLevel, "Product not found",
			slog.String("layer", "repository"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
			slog.String("product_id", id),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "repository"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
				productIdAttr,
				attribute.Bool("error.expected", true),
			}
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
		}
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
	notFoundMapper := func(err error) codes.Code {
		if err == nil {
			return codes.Ok
		}
		if errors.Is(err, commonerrors.ErrNotFound) {
			return codes.Ok
		}
		return codes.Error
	}
	defer commontrace.EndSpan(spanner, &opErr, notFoundMapper)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Repository: UpdateStock called", slog.String("product_id", productID), slog.Int("new_stock", newStock), slog.String("operation", operationName))

	spanner.AddEvent("Acquiring write lock for UpdateStock")
	r.mu.Lock()
	spanner.AddEvent("Write lock acquired for UpdateStock")

	product, ok := r.products[productID]
	if !ok {
		r.mu.Unlock()
		spanner.AddEvent("Write lock released (product not found)")
		opErr = fmt.Errorf("product with id '%s' not found for update: %w", productID, commonerrors.ErrNotFound)
		logLevel := slog.LevelWarn
		eventName := "resource_not_found"
		r.logger.Log(ctx, logLevel, "Product not found for update",
			slog.String("layer", "repository"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
			slog.String("product_id", productID),
			slog.Int("new_stock", newStock),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "repository"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
				attribute.Bool("error.expected", true),
			}
			spanAttrs = append(spanAttrs, attrs...)
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
		}
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
		opErr = fmt.Errorf("%s: %w", errMsg, commonerrors.ErrInternal)
		logLevel := slog.LevelError
		eventName := "internal_consistency_error"
		r.logger.Log(ctx, logLevel, "Repository internal inconsistency",
			slog.String("layer", "repository"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
			slog.String("product_id", productID),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "repository"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
			}
			spanAttrs = append(spanAttrs, attrs...)
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
			spanner.SetStatus(codes.Error, opErr.Error())
		}
		return opErr
	}

	spanner.SetAttributes(attribute.Int("product.old_stock", oldStock))
	r.logger.InfoContext(ctx, "Repository: Product stock updated in memory", slog.String("product_id", productID), slog.Int("old_stock", oldStock), slog.Int("new_stock", newStock), slog.String("operation", operationName))

	spanner.AddEvent("Calling saveData to persist changes")
	if saveErr := r.saveData(ctx); saveErr != nil {
		opErr = fmt.Errorf("failed persistence after stock update for '%s': %w", productID, saveErr)
		logLevel := slog.LevelError
		eventName := "save_data_error"
		r.logger.Log(ctx, logLevel, "Failed to persist stock update",
			slog.String("layer", "repository"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
			slog.String("product_id", productID),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "repository"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
				attribute.String("underlying_error", saveErr.Error()),
			}
			spanAttrs = append(spanAttrs, attrs...)
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
			spanner.SetStatus(codes.Error, opErr.Error())
		}
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

	spanner.AddEvent("Acquiring read lock for saveData (to marshal)")
	r.mu.RLock()
	spanner.AddEvent("Read lock acquired")
	data, err := json.MarshalIndent(r.products, "", "  ")
	r.mu.RUnlock()
	spanner.AddEvent("Read lock released after marshal")
	if err != nil {
		opErr = fmt.Errorf("failed to marshal products for saving: %w", err)
		logLevel := slog.LevelError
		eventName := "marshal_error"
		r.logger.Log(ctx, logLevel, "Failed to marshal products",
			slog.String("layer", "repository"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
			slog.String("file.path", r.filePath),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "repository"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
				fileAttr,
			}
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
			spanner.SetStatus(codes.Error, opErr.Error())
		}
		return opErr
	}

	spanner.AddEvent("Writing data to file", trace.WithAttributes(fileAttr))
	if writeErr := os.WriteFile(r.filePath, data, 0644); writeErr != nil {
		opErr = fmt.Errorf("failed to write products file '%s': %w", r.filePath, writeErr)
		logLevel := slog.LevelError
		eventName := "file_write_error"
		r.logger.Log(ctx, logLevel, "Failed to write products file",
			slog.String("layer", "repository"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
			slog.String("file.path", r.filePath),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "repository"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
				fileAttr,
			}
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
			spanner.SetStatus(codes.Error, opErr.Error())
		}
		return opErr
	}
	spanner.AddEvent("Data written successfully")
	r.logger.InfoContext(ctx, "Repository: Data saved successfully", slog.String("file_path", r.filePath), slog.String("operation", operationName))

	return nil
}
