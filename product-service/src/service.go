package main

import (
	"context"
	"fmt"

	commonErrors "github.com/narender/common-module/errors"
	"github.com/narender/common-module/telemetry"

	"github.com/sirupsen/logrus"
)

// ProductService defines the interface for product business logic
type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, productID string) (Product, error)
	GetStock(ctx context.Context, productID string) (int, error)
}

// productService implements the ProductService interface
type productService struct {
	repo ProductRepository
}

// NewProductService creates a new product service
func NewProductService(repo ProductRepository) ProductService {
	return &productService{
		repo: repo,
	}
}

// GetAll handles fetching all products
func (s *productService) GetAll(ctx context.Context) ([]Product, error) {
	log := logrus.WithContext(ctx)
	log.Info("Service: Fetching all products")

	ctx, span := telemetry.StartSpan(ctx, "product-service", "service.GetAll")
	defer span.End()

	products, err := s.repo.GetAll(ctx)
	if err != nil {
		telemetry.RecordError(span, err, "failed to fetch all products from repo")
		log.WithError(err).Error("Service: Failed to fetch all products from repo")
		return nil, fmt.Errorf("failed to find all products: %w", err)
	}

	telemetry.AddAttribute(span, "db.result.count", len(products))

	return products, nil
}

// GetByID handles fetching a product by its ID
func (s *productService) GetByID(ctx context.Context, productID string) (Product, error) {
	log := logrus.WithContext(ctx).WithField("product_id", productID)
	log.Info("Service: Fetching product by ID")

	ctx, span := telemetry.StartSpan(ctx, "product-service", "service.GetByID")
	defer span.End()

	telemetry.AddAttribute(span, "product.id", productID)

	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		telemetry.RecordError(span, err, "failed to find product by ID in repo")
		log.WithError(err).Error("Service: Failed to find product by ID in repo")
		if commonErrors.Is(err, commonErrors.ErrProductNotFound) {
			return Product{}, commonErrors.ErrProductNotFound
		} else {
			return Product{}, fmt.Errorf("failed to find product by id '%s': %w", productID, err)
		}
	}

	return product, nil
}

// GetStock handles fetching stock for a product
func (s *productService) GetStock(ctx context.Context, productID string) (int, error) {
	log := logrus.WithContext(ctx).WithField("product_id", productID)
	log.Info("Service: Checking stock for product ID")

	ctx, span := telemetry.StartSpan(ctx, "product-service", "service.GetStock")
	defer span.End()

	telemetry.AddAttribute(span, "product.id", productID)

	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		telemetry.RecordError(span, err, "failed to get product for stock check from repo")
		log.WithError(err).Error("Service: Failed to get product for stock check from repo")
		if commonErrors.Is(err, commonErrors.ErrProductNotFound) {
			return 0, commonErrors.ErrProductNotFound
		} else {
			return 0, fmt.Errorf("failed to get product for stock check (id '%s'): %w", productID, err)
		}
	}

	stock := product.Stock
	telemetry.AddAttribute(span, "product.stock", stock)

	return stock, nil
}

// --- Helper Functions ---

// Removed local logServiceError function
