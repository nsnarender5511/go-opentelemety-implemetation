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

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

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
}

// NewProductRepository creates a new product repository
func NewProductRepository() (ProductRepository, error) {
	repo := &productRepository{
		products: make(map[string]Product),
		filePath: config.DATA_FILE_PATH,
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

// readData method loads product data from JSON file
func (r *productRepository) readData() error {
	// Create a background context for telemetry span
	ctx := context.Background()
	_, span := telemetry.StartSpan(ctx, "product-service", "ProductRepository.readData")
	telemetry.AddAttribute(span, "db.file_path", r.filePath)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		errWrapped := &commonErrors.DatabaseError{
			Operation: "ReadFile",
			Err:       fmt.Errorf("failed to read data file '%s': %w", r.filePath, err),
		}
		telemetry.RecordError(span, errWrapped, "Failed to read data file")
		return errWrapped
	}

	// Changed: Unmarshal into a map[string]Product first
	var productsMap map[string]Product
	if err := json.Unmarshal(data, &productsMap); err != nil {
		errWrapped := &commonErrors.DatabaseError{
			Operation: "UnmarshalJSON",
			Err:       fmt.Errorf("failed to unmarshal data from file '%s': %w", r.filePath, err),
		}
		telemetry.RecordError(span, errWrapped, "Failed to unmarshal data")
		return errWrapped
	}

	// Changed: Populate the repository's map directly from the unmarshaled map
	r.products = make(map[string]Product, len(productsMap))
	for key, p := range productsMap {
		// Ensure the productID in the object matches the key, if desired, or trust the key
		// For simplicity, we'll use the map key as the identifier in the repository map
		r.products[key] = p
	}

	log.Debugf("Successfully loaded %d products from %s", len(r.products), r.filePath)
	telemetry.AddAttribute(span, "db.rows_loaded", len(r.products))
	return nil
}

// GetAll method returns all products
func (r *productRepository) GetAll(ctx context.Context) ([]Product, error) {
	ctx, span := telemetry.StartSpan(ctx, "product-service", "ProductRepository.GetAll")
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
	telemetry.AddAttribute(span, "db.rows_returned", len(result))
	return result, nil
}

// GetByID method retrieves a product by ID
func (r *productRepository) GetByID(ctx context.Context, id string) (Product, error) {
	ctx, span := telemetry.StartSpan(ctx, "product-service", "ProductRepository.GetByID")
	telemetry.AddAttribute(span, "product.id", id)
	defer span.End()

	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		telemetry.RecordError(span, commonErrors.ErrProductNotFound, "Product not found")
		return Product{}, commonErrors.ErrProductNotFound
	}

	return product, nil
}
