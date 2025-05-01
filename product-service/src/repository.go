package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	commonErrors "github.com/narender/common/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ProductRepository defines the interface for product data access
type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
}

// productRepository implements ProductRepository
type productRepository struct {
	products map[string]Product
	mu       sync.RWMutex
	filePath string
	tracer   trace.Tracer
}

// NewProductRepository creates a new product repository, accepting the data file path
func NewProductRepository(dataFilePath string) (ProductRepository, error) {
	if dataFilePath == "" {
		return nil, fmt.Errorf("data file path cannot be empty")
	}

	repo := &productRepository{
		products: make(map[string]Product),
		filePath: dataFilePath,                      // Use passed argument
		tracer:   otel.Tracer("product-repository"), // Get tracer instance
	}

	// Use background context and global logger for initialization messages
	ctx := context.Background()

	// Ensure the data file exists, create if not
	if _, statErr := os.Stat(repo.filePath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			// Use global logger
			logrus.WithField("path", repo.filePath).Info("Data file not found, creating empty file")
			if writeErr := os.WriteFile(repo.filePath, []byte("{\n}"), 0644); writeErr != nil {
				return nil, fmt.Errorf("failed to create initial data file '%s': %w", repo.filePath, writeErr)
			}
		} else {
			return nil, fmt.Errorf("failed to stat data file '%s': %w", repo.filePath, statErr)
		}
	}

	// Load data immediately using the configured path
	if err := repo.readData(ctx); err != nil { // Pass context to readData
		// Use global logger
		logrus.WithError(err).WithField("path", repo.filePath).Error("Failed to read initial data")
		return nil, fmt.Errorf("failed to initialize product repository from %s: %w", repo.filePath, err)
	}
	// Use global logger
	logrus.WithField("path", repo.filePath).Info("Initialized product repository")
	return repo, nil
}

// readData method loads product data from JSON file
// Now accepts context for logging.
func (r *productRepository) readData(ctx context.Context) error {
	// Use global logger with context
	log := logrus.WithContext(ctx)

	// Start span using the incoming context and the repo's tracer
	ctx, span := r.tracer.Start(ctx, "ProductRepository.readData")
	span.SetAttributes(
		attribute.String("db.system", "file"),
		attribute.String("db.file.path", r.filePath),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		errWrapped := &commonErrors.DatabaseError{
			Operation: "ReadFile",
			Err:       fmt.Errorf("failed to read data file '%s': %w", r.filePath, err),
		}
		// Use logger with context
		log.WithError(errWrapped).Error("Failed to read data file")
		span.RecordError(errWrapped)
		span.SetStatus(codes.Error, errWrapped.Error())
		return errWrapped
	}

	var productsMap map[string]Product
	if err := json.Unmarshal(data, &productsMap); err != nil {
		errWrapped := &commonErrors.DatabaseError{
			Operation: "UnmarshalJSON",
			Err:       fmt.Errorf("failed to unmarshal data from file '%s': %w", r.filePath, err),
		}
		// Use logger with context
		log.WithError(errWrapped).Error("Failed to unmarshal data")
		span.RecordError(errWrapped)
		span.SetStatus(codes.Error, errWrapped.Error())
		return errWrapped
	}

	r.products = make(map[string]Product, len(productsMap))
	for key, p := range productsMap {
		r.products[key] = p
	}

	productCount := len(r.products)
	// Use logger with context
	log.WithFields(logrus.Fields{
		"count": productCount,
		"path":  r.filePath,
	}).Debug("Successfully loaded products")

	span.SetAttributes(attribute.Int("db.rows_loaded", productCount))
	return nil
}

// GetAll method returns all products
func (r *productRepository) GetAll(ctx context.Context) ([]Product, error) {
	// Use global logger with context
	log := logrus.WithContext(ctx)
	ctx, span := r.tracer.Start(ctx, "ProductRepository.GetAll")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.products) == 0 {
		log.Warn("GetAll called but no products loaded.")
	}

	var result []Product
	for _, p := range r.products {
		result = append(result, p)
	}
	span.SetAttributes(attribute.Int("db.rows_returned", len(result)))
	span.SetStatus(codes.Ok, "")
	return result, nil
}

// GetByID method retrieves a product by ID
func (r *productRepository) GetByID(ctx context.Context, id string) (Product, error) {
	// Use global logger with context
	log := logrus.WithContext(ctx)
	ctx, span := r.tracer.Start(ctx, "ProductRepository.GetByID")
	span.SetAttributes(attribute.String("app.product.id", id))
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		// Use the NotFound constructor from commonErrors
		errNotFound := commonErrors.NotFound(fmt.Sprintf("product with id '%s' not found", id))
		// Use logger with context
		log.WithField("product.id", id).Warn("Product not found")
		span.RecordError(errNotFound)
		span.SetStatus(codes.Error, errNotFound.Error())
		return Product{}, errNotFound
	}

	span.SetStatus(codes.Ok, "")
	return product, nil
}
