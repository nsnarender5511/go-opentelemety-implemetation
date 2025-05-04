package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	db "github.com/narender/common/db"
	"github.com/narender/common/debugutils"
	commonerrors "github.com/narender/common/errors"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/narender/common/globals"
	"go.opentelemetry.io/otel/trace"
)

// ProductRepository defines the interface for accessing product data.
type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	UpdateStock(ctx context.Context, productID string, newStock int) error
	Create(ctx context.Context, product Product) error
}

type productRepository struct {
	database *db.FileDatabase
	logger   *slog.Logger
}

// NewProductRepository creates a new repository instance loading data from a JSON file.
func NewProductRepository() ProductRepository {
	repo := &productRepository{
		database: db.NewFileDatabase(),
		logger:   globals.Logger(),
	}
	return repo
}

func (r *productRepository) GetAll(ctx context.Context) (productsSlice []Product, opErr error) {

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr)

	ctx, spanner := commontrace.StartSpan(ctx,
		attribute.String("repository.operation", "GetAll"),
	)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Repository: GetAll called - reading from FileDatabase")

	var productsMap map[string]Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		if os.IsNotExist(err) {
			r.logger.WarnContext(ctx, "Product data file not found during GetAll, returning empty list")
			spanner.AddEvent("FileDatabase.Read indicated file not found, returning empty.", trace.WithAttributes(attribute.String("error.message", err.Error())))
			return []Product{}, nil // Return empty slice, not an error
		} else {
			opErr = fmt.Errorf("failed to read products for GetAll using FileDatabase: %w", err)
			r.logger.ErrorContext(ctx, "Failed to read products for GetAll", slog.String("error", opErr.Error()))
			spanner.SetStatus(codes.Error, opErr.Error())
			return nil, opErr
		}
	}

	productsSlice = make([]Product, 0, len(productsMap))
	for _, p := range productsMap {
		productsSlice = append(productsSlice, p)
	}

	spanner.SetAttributes(attribute.Int("products.returned.count", len(productsSlice)))
	r.logger.InfoContext(ctx, "Repository: GetAll returning products read from file", slog.Int("count", len(productsSlice)))
	return productsSlice, nil
}

func (r *productRepository) GetByID(ctx context.Context, id string) (product Product, opErr error) {
	productIdAttr := attribute.String("product.id", id)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, productIdAttr)

	ctx, spanner := commontrace.StartSpan(ctx,
		productIdAttr,
	)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Repository: GetByID called - reading from FileDatabase", slog.String("product_id", id))

	var productsMap map[string]Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		opErr = fmt.Errorf("failed to read products for GetByID using FileDatabase: %w", err)
		r.logger.ErrorContext(ctx, "Failed to read products for GetByID", slog.String("product_id", id), slog.String("error", opErr.Error()))
		spanner.SetStatus(codes.Error, opErr.Error())
		return Product{}, opErr
	}

	product, exists := productsMap[id]
	if !exists {
		opErr = fmt.Errorf("product with id '%s' not found in file data: %w", id, commonerrors.ErrNotFound)
		r.logger.ErrorContext(ctx, "Product not found in file data", slog.String("error", opErr.Error()), slog.String("product_id", id))
		return Product{}, opErr
	}

	spanner.SetAttributes(attribute.String("product.name", product.Name))
	r.logger.InfoContext(ctx, "Repository: GetByID found product in file data", slog.String("product_id", id))
	return product, nil
}

func (r *productRepository) UpdateStock(ctx context.Context, productID string, newStock int) (opErr error) {
	productIdAttr := attribute.String("product.id", productID)
	newStockAttr := attribute.Int("product.new_stock", newStock)
	attrs := []attribute.KeyValue{productIdAttr, newStockAttr}

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, attrs...)

	ctx, spanner := commontrace.StartSpan(ctx, attrs...)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Repository: UpdateStock called - requires read-modify-write on FileDatabase", slog.String("product_id", productID), slog.Int("new_stock", newStock))

	// 1. Read current data
	var productsMap map[string]Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		opErr = fmt.Errorf("failed to read products for UpdateStock using FileDatabase: %w", err)
		r.logger.ErrorContext(ctx, "Failed to read products for UpdateStock", slog.String("product_id", productID), slog.String("error", opErr.Error()))
		spanner.SetStatus(codes.Error, opErr.Error())
		return opErr
	}

	// 2. Modify data
	product, ok := productsMap[productID]
	if !ok {
		opErr = fmt.Errorf("product with id '%s' not found in file data for update: %w", productID, commonerrors.ErrNotFound)
		r.logger.ErrorContext(ctx, "Product not found in file data for update", slog.String("error", opErr.Error()), slog.String("product_id", productID))
		spanner.AddEvent("product_not_found_in_map_for_update", trace.WithAttributes(attrs...))
		return opErr
	}

	oldStock := product.Stock
	product.Stock = newStock
	productsMap[productID] = product // Update the map

	spanner.SetAttributes(attribute.Int("product.old_stock", oldStock))
	r.logger.InfoContext(ctx, "Repository: Product stock updated in memory map (pre-save)", slog.String("product_id", productID), slog.Int("old_stock", oldStock), slog.Int("new_stock", newStock))

	// 3. Write modified data back
	if writeErr := r.database.Write(ctx, productsMap); writeErr != nil {
		r.logger.ErrorContext(ctx, "Failed to persist stock update via FileDatabase", slog.String("error", opErr.Error()), slog.String("product_id", productID))
		spanner.SetStatus(codes.Error, opErr.Error())
		return opErr
	}

	r.logger.InfoContext(ctx, "Repository: Product stock updated and saved via FileDatabase", slog.String("product_id", productID), slog.Int("new_stock", newStock))

	return nil
}

// Create adds a new product to the JSON file.
func (repo *productRepository) Create(ctx context.Context, product Product) (opErr error) {
	mc := commonmetric.StartMetricsTimer()
	productIdAttr := attribute.String("product.id", product.ProductID)
	defer mc.End(ctx, &opErr, productIdAttr)

	ctx, spanner := commontrace.StartSpan(ctx, productIdAttr)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	// NOTE: Assuming db.FileDatabase handles concurrency, so no explicit repo mutex needed here.

	repo.logger.InfoContext(ctx, "Repository: Create called", slog.String("productID", product.ProductID))

	// Load existing products using FileDatabase
	var productsMap map[string]Product
	err := repo.database.Read(ctx, &productsMap)
	// Handle file not existing initially - create an empty map
	if err != nil && os.IsNotExist(err) {
		repo.logger.InfoContext(ctx, "Repository: data file not found during create, initializing empty map", slog.String("productID", product.ProductID))
		productsMap = make(map[string]Product)
		err = nil // Clear the IsNotExist error
	} else if err != nil {
		repo.logger.ErrorContext(ctx, "Repository: Failed to read products during create", slog.String("error", err.Error()))
		if spanner != nil {
			spanner.SetStatus(codes.Error, "failed to read products: "+err.Error())
		}
		opErr = fmt.Errorf("failed to read products during create: %w", err)
		return opErr
	}

	// Check if product ID already exists in the map
	if _, exists := productsMap[product.ProductID]; exists {
		repo.logger.WarnContext(ctx, "Repository: Product ID already exists", slog.String("productID", product.ProductID))
		// Assign a conflict error
		if spanner != nil {
			spanner.SetStatus(codes.Error, opErr.Error())
		}
		return opErr
	}

	// Add the new product to the map
	productsMap[product.ProductID] = product

	// Save updated products map using FileDatabase
	err = repo.database.Write(ctx, productsMap)
	if err != nil {
		repo.logger.ErrorContext(ctx, "Repository: Failed to save products after create", slog.String("productID", product.ProductID), slog.String("error", err.Error()))
		if spanner != nil {
			spanner.SetStatus(codes.Error, "failed to save products: "+err.Error())
		}
		opErr = fmt.Errorf("failed to save products after create: %w", err)
		return opErr
	}

	repo.logger.InfoContext(ctx, "Repository: Product created successfully", slog.String("productID", product.ProductID))
	return nil // Success
}
