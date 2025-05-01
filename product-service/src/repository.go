package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	// Use correct module path for common packages
	"github.com/narender/common-module/config"
	commonErrors "github.com/narender/common-module/errors" // Alias this import
	"github.com/narender/common-module/telemetry"

	// Model is in the same package, no need to import it separately.

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	codes "go.opentelemetry.io/otel/codes" // Ensure this import is present
	"go.opentelemetry.io/otel/trace"
)

var log = logrus.New()

// ProductRepository defines the interface for product data access
type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, error)           // Use local Product type
	GetByID(ctx context.Context, id string) (Product, error) // Use local Product type
}

// productRepository implements ProductRepository
type productRepository struct {
	products map[string]Product // Use local Product type
	mu       sync.RWMutex
	filePath string
}

// NewProductRepository creates a new product repository
func NewProductRepository() (ProductRepository, error) {
	repo := &productRepository{
		products: make(map[string]Product), // Use local Product type
		filePath: config.DATA_FILE_PATH,    // Use config from imported package
	}
	// Ensure the data file exists, create if not
	if _, statErr := os.Stat(repo.filePath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			log.Infof("Data file not found, creating empty file at %s", repo.filePath)
			if writeErr := os.WriteFile(repo.filePath, []byte("{\n}"), 0644); writeErr != nil {
				return nil, fmt.Errorf("failed to create initial data file '%s': %w", repo.filePath, writeErr)
			}
		} else {
			return nil, fmt.Errorf("failed to stat data file '%s': %w", repo.filePath, statErr)
		}
	}

	// Load data immediately using the configured path
	if err := repo.readData(); err != nil {
		log.WithError(err).Errorf("Failed to read initial data from %s", repo.filePath)
		return nil, fmt.Errorf("failed to initialize product repository from %s: %w", repo.filePath, err)
	}
	log.Printf("Initialized product repository with data from: %s", repo.filePath)
	return repo, nil
}

// getTracer is a helper to get the tracer instance consistently
func (r *productRepository) getTracer() trace.Tracer {
	return otel.Tracer("product-service/repository")
}

// Updated readData method - this is the primary definition
func (r *productRepository) readData() error {
	tr := r.getTracer()
	// Consider if a context is needed here, maybe from NewProductRepository?
	// For now, using background context for the trace span.
	ctx := context.Background()
	_, span := tr.Start(ctx, "ProductRepository.readData")
	span.SetAttributes(telemetry.DBFilePathKey.String(r.filePath))
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		errWrapped := &commonErrors.DatabaseError{
			Operation: "ReadFile",
			Err:       fmt.Errorf("failed to read data file '%s': %w", r.filePath, err),
		}
		span.RecordError(errWrapped)
		span.SetStatus(codes.Error, "Failed to read data file")
		return errWrapped
	}

	var products []Product // Use local Product type
	if err := json.Unmarshal(data, &products); err != nil {
		errWrapped := &commonErrors.DatabaseError{
			Operation: "UnmarshalJSON",
			Err:       fmt.Errorf("failed to unmarshal data from file '%s': %w", r.filePath, err),
		}
		span.RecordError(errWrapped)
		span.SetStatus(codes.Error, "Failed to unmarshal data")
		return errWrapped
	}

	r.products = make(map[string]Product)
	for _, p := range products {
		// Convert uint ID to string for map key
		r.products[fmt.Sprintf("%d", p.ID)] = p
	}
	log.Debugf("Successfully loaded %d products from %s", len(products), r.filePath)
	span.SetAttributes(attribute.Int("db.rows_loaded", len(products)))
	return nil
}

// GetAll method
func (r *productRepository) GetAll(ctx context.Context) ([]Product, error) {
	tr := r.getTracer()
	ctx, span := tr.Start(ctx, "ProductRepository.GetAll")
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.products) == 0 {
		log.Warn("GetAll called but no products loaded.")
		// Optional: Attempt to reload data if empty?
		// err := r.readData()
		// if err != nil { log.WithError(err).Error("Failed to reload data in GetAll") }
	}

	var result []Product // Use local Product type
	for _, p := range r.products {
		result = append(result, p)
	}
	span.SetAttributes(attribute.Int("db.rows_returned", len(result)))
	return result, nil
}

// GetByID method
func (r *productRepository) GetByID(ctx context.Context, id string) (Product, error) {
	tr := r.getTracer()
	ctx, span := tr.Start(ctx, "ProductRepository.GetByID")
	span.SetAttributes(attribute.String("product.id", id))
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		span.RecordError(commonErrors.ErrProductNotFound)
		// Use codes.NotFound which should be available via import
		// span.SetStatus(codes.NotFound, commonErrors.ErrProductNotFound.Error())
		return Product{}, commonErrors.ErrProductNotFound
	}

	return product, nil
}
