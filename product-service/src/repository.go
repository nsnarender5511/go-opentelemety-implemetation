package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)


var ErrNotFound = errors.New("product not found")

type ProductRepository interface {
	GetAll() ([]Product, error)
	GetByID(id string) (Product, error)
	UpdateStock(productID string, newStock int) error
}

type productRepository struct {
	products map[string]Product
	mu       sync.RWMutex
	filePath string
}

func NewProductRepository(dataFilePath string) (ProductRepository, error) {
	fmt.Printf("Repository: Initializing with file path: %s\n", dataFilePath)

	repo := &productRepository{
		products: make(map[string]Product),
		filePath: dataFilePath,
	}

	if _, statErr := os.Stat(repo.filePath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			fmt.Printf("WARN: Data file %s not found, starting with empty product list.\n", dataFilePath)
		} else {
			err := fmt.Errorf("repository: failed to stat data file %s: %w", dataFilePath, statErr)
			log.Printf("ERROR: %v", err)
			return nil, err
		}
	} else {
		if err := repo.loadData(); err != nil {
			log.Printf("ERROR: Repository: Failed to initialize from %s: %v", dataFilePath, err)
			return nil, fmt.Errorf("failed to initialize product repository from %s: %w", dataFilePath, err)
		}
	}
	fmt.Printf("Repository: Initialized successfully, loaded %d products\n", len(repo.products))
	return repo, nil
}

func (r *productRepository) loadData() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("WARN: Data file %s not found during load, initializing empty map.\n", r.filePath)
			r.products = make(map[string]Product)
			return nil
		}
		return fmt.Errorf("failed to read data file '%s': %w", r.filePath, err)
	}

	if len(data) == 0 {
		fmt.Printf("WARN: Data file %s is empty, initializing empty product map.\n", r.filePath)
		r.products = make(map[string]Product)
		return nil
	}

	var productsMap map[string]Product
	if err := json.Unmarshal(data, &productsMap); err != nil {
		return fmt.Errorf("failed to unmarshal product data from '%s': %w", r.filePath, err)
	}

	r.products = make(map[string]Product, len(productsMap))
	for key, p := range productsMap {
		r.products[key] = p
	}

	productCount := len(r.products)
	fmt.Printf("DEBUG: Successfully loaded %d products from %s\n", productCount, r.filePath)
	return nil
}

func (r *productRepository) GetAll() ([]Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.products) == 0 {
		fmt.Println("WARN: GetAll called but no products loaded.")
	}
	result := make([]Product, 0, len(r.products))
	for _, p := range r.products {
		result = append(result, p)
	}
	return result, nil
}

func (r *productRepository) GetByID(id string) (Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	product, ok := r.products[id]
	if !ok {
		return Product{}, ErrNotFound
	}
	return product, nil
}

func (r *productRepository) UpdateStock(productID string, newStock int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.products[productID]; !exists {
		fmt.Printf("WARN: Attempted to update stock for non-existent product ID: %s\n", productID)
		return ErrNotFound
	}

	p := r.products[productID]
	p.Stock = newStock
	r.products[productID] = p

	if err := r.saveData(); err != nil {
		log.Printf("ERROR: Failed to save updated stock data for product %s: %v", productID, err)
		return err
	}

	fmt.Printf("INFO: Successfully updated stock for product ID: %s\n", productID)
	return nil
}

func (r *productRepository) saveData() error {
	r.mu.RLock()
	productsToSave := make(map[string]Product, len(r.products))
	for k, v := range r.products {
		productsToSave[k] = v
	}
	r.mu.RUnlock()

	data, err := json.MarshalIndent(productsToSave, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal product data for saving: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write data file '%s': %w", r.filePath, err)
	}

	fmt.Printf("DEBUG: Successfully saved %d products to %s\n", len(productsToSave), r.filePath)
	return nil
}
