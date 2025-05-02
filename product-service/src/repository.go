package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	commonErrors "github.com/narender/common/errors"
	"github.com/narender/common/otel"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ProductRepository defines the interface for accessing product data.
type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	UpdateStock(ctx context.Context, productID string, newStock int) error
	ObserveStockLevels(ctx context.Context, observer metric.Observer, stockGauge metric.Int64ObservableGauge) error
}

// productRepository implements ProductRepository using a JSON file.
type productRepository struct {
	products map[string]Product
	mu       sync.RWMutex
	filePath string
	logger   *logrus.Logger   // Injected logger
	tracer   oteltrace.Tracer // Injected tracer
}

// NewProductRepository creates a new file-based product repository.
func NewProductRepository(dataFilePath string, logger *logrus.Logger, tracer oteltrace.Tracer) (ProductRepository, error) {
	logger.Infof("Repository: Initializing with file path: %s", dataFilePath)
	repo := &productRepository{
		products: make(map[string]Product),
		filePath: dataFilePath,
		logger:   logger,
		tracer:   tracer,
	}

	ctx := context.Background() // Use background context for initialization

	// Ensure the data file exists, create if not
	if _, statErr := os.Stat(repo.filePath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			logger.Warnf("Repository: Data file %s not found, starting with empty product list.", dataFilePath)
		} else {
			// Wrap error for context
			dbErr := &commonErrors.DatabaseError{Operation: "StatFile", Err: statErr}
			return nil, fmt.Errorf("failed to stat data file '%s': %w", dataFilePath, dbErr)
		}
	}

	// Load data immediately
	if err := repo.readData(ctx); err != nil {
		// readData already logs and wraps the error
		return nil, fmt.Errorf("failed to initialize product repository from %s: %w", dataFilePath, err)
	}
	logger.Infof("Repository: Initialized successfully, loaded %d products", len(repo.products))

	// Gauge registration will be handled in main.go where the meter is available
	// // Register the stock gauge callback
	// if err := otel.RegisterObservableGauge(repo.ObserveStockLevels); err != nil { // Incorrect usage removed
	// 	logger.WithError(err).Error("Repository: Failed to register observable stock gauge")
	// 	// Decide if this is fatal - for now, log and continue
	// } else {
	// 	logger.Info("Repository: Registered observable stock gauge callback.")
	// }

	return repo, nil
}

// readData loads product data from the JSON file.
func (r *productRepository) readData(ctx context.Context) error {
	const operation = "ReadFile"
	ctx, span := r.tracer.Start(ctx, "ProductRepository.readData",
		oteltrace.WithAttributes(otel.AttrDBSystemKey.String("file"), otel.AttrDBOperationKey.String(operation)),
	)
	span.SetAttributes(attribute.String("db.file.path", r.filePath))
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		errWrapped := &commonErrors.DatabaseError{
			Operation: operation,
			Err:       fmt.Errorf("failed to read data file '%s': %w", r.filePath, err),
		}
		// Log with injected logger
		r.logger.WithContext(ctx).WithError(errWrapped).Error("Failed to read data file")
		// Record error on span using common helper
		otel.RecordSpanError(span, errWrapped)
		return errWrapped // Return the wrapped DatabaseError
	}

	const unmarshalOperation = "UnmarshalJSON"
	span.SetAttributes(otel.AttrDBOperationKey.String(unmarshalOperation)) // Update operation for this part

	var productsMap map[string]Product
	if err := json.Unmarshal(data, &productsMap); err != nil {
		errWrapped := &commonErrors.DatabaseError{
			Operation: unmarshalOperation,
			Err:       fmt.Errorf("failed to unmarshal data from file '%s': %w", r.filePath, err),
		}
		r.logger.WithContext(ctx).WithError(errWrapped).Error("Failed to unmarshal data")
		// Record error on span using common helper
		otel.RecordSpanError(span, errWrapped)
		return errWrapped // Return the wrapped DatabaseError
	}

	r.products = make(map[string]Product, len(productsMap))
	for key, p := range productsMap {
		r.products[key] = p
	}

	productCount := len(r.products)
	r.logger.WithContext(ctx).WithFields(logrus.Fields{
		"count": productCount,
		"path":  r.filePath,
	}).Debug("Successfully loaded products")

	span.SetAttributes(attribute.Int("db.rows_loaded", productCount))
	// No need to set span status OK
	return nil
}

// GetAll retrieves all products from the repository.
func (r *productRepository) GetAll(ctx context.Context) ([]Product, error) {
	const operation = "GetAll"
	ctx, span := r.tracer.Start(ctx, "ProductRepository.GetAll",
		oteltrace.WithAttributes(otel.AttrDBSystemKey.String("file"), otel.AttrDBOperationKey.String(operation)),
	)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.products) == 0 {
		// Log with injected logger
		r.logger.WithContext(ctx).Warn("GetAll called but no products loaded.")
	}

	result := make([]Product, 0, len(r.products))
	for _, p := range r.products {
		result = append(result, p)
	}

	span.SetAttributes(attribute.Int("db.rows_returned", len(result)))
	// No need to set span status OK
	return result, nil
}

// GetByID retrieves a single product by ID.
func (r *productRepository) GetByID(ctx context.Context, id string) (Product, error) {
	const operation = "GetByID"
	ctx, span := r.tracer.Start(ctx, "ProductRepository.GetByID",
		oteltrace.WithAttributes(otel.AttrDBSystemKey.String("file"), otel.AttrDBOperationKey.String(operation)),
	)
	// Use attribute key consistent with handler/service if possible (e.g., defined in main or common)
	span.SetAttributes(attribute.String("app.product.id", id))
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		// Log with injected logger
		r.logger.WithContext(ctx).WithField("product.id", id).Warn("Product not found")
		// Record error on span using common helper, passing the standard error
		otel.RecordSpanError(span, commonErrors.ErrNotFound, attribute.String("app.product.id", id))
		// Return the standard ErrNotFound; service layer will wrap it into an AppError
		return Product{}, commonErrors.ErrNotFound
	}

	// No need to set span status OK
	return product, nil
}

// UpdateStock updates the stock for a given product ID.
func (r *productRepository) UpdateStock(ctx context.Context, productID string, newStock int) error {
	const operation = "UpdateStock"
	ctx, span := r.tracer.Start(ctx, "ProductRepository.UpdateStock",
		oteltrace.WithAttributes(otel.AttrDBSystemKey.String("file"), otel.AttrDBOperationKey.String(operation)),
	)
	span.SetAttributes(
		attribute.String("db.key", productID),
		attribute.Int("product.new_stock", newStock),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	product, ok := r.products[productID]
	if !ok {
		r.logger.WithContext(ctx).WithField("product.id", productID).Warn("Product not found")
		return commonErrors.Wrap(commonErrors.ErrNotFound, http.StatusNotFound, fmt.Sprintf("product with ID %s not found", productID))
	}

	r.logger.WithContext(ctx).WithFields(logrus.Fields{
		"product.id":            productID,
		"product.current_stock": product.Stock,
		"product.new_stock":     newStock,
	}).Info("Updating stock")

	product.Stock = newStock
	r.products[productID] = product

	if err := r.saveData(); err != nil {
		r.logger.WithError(err).Error("Failed to save updated stock to file")
		return commonErrors.Wrap(err, http.StatusInternalServerError, "failed to save updated stock to persistent storage")
	}

	r.logger.WithContext(ctx).WithField("product.id", productID).Info("Successfully updated stock")
	return nil
}

// ObserveStockLevels is the function passed as a callback to the meter registration.
// It observes the current stock levels for all products.
func (r *productRepository) ObserveStockLevels(ctx context.Context, observer metric.Observer, stockGauge metric.Int64ObservableGauge) error {
	r.logger.Debug("Repository: ObserveStockLevels callback triggered")
	r.mu.RLock()
	defer r.mu.RUnlock()

	for id, product := range r.products {
		attrs := attribute.NewSet(
			attribute.String("product.id", id),
			attribute.String("product.name", product.Name),
			attribute.String("product.category", product.Category),
		)
		// Use the gauge passed into the callback
		observer.ObserveInt64(stockGauge, int64(product.Stock), metric.WithAttributeSet(attrs))
		r.logger.Tracef("Repository: Observed stock %d for product %s (%s)", product.Stock, id, product.Name) // Trace level for per-product obs
	}
	r.logger.Debug("Repository: ObserveStockLevels callback finished")
	return nil
}

// saveData writes the current products map back to the JSON file.
func (r *productRepository) saveData() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert map back to list for JSON structure
	productList := make([]Product, 0, len(r.products))
	for _, p := range r.products {
		productList = append(productList, p)
	}

	data, err := json.MarshalIndent(productList, "", "  ")
	if err != nil {
		r.logger.WithError(err).Error("Failed to marshal products to JSON")
		return fmt.Errorf("failed to marshal products: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		r.logger.WithError(err).Error("Failed to write data file")
		return fmt.Errorf("failed to write data file: %w", err)
	}
	r.logger.Debugf("Repository: Successfully saved %d products to %s", len(productList), r.filePath)
	return nil
}
