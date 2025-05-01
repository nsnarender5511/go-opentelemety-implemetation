package main

import (
	"context"
	"errors"
	"fmt"

	commonErrors "github.com/narender/common/errors"
	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute" // Import attribute
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ProductService defines the interface for product business logic
type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, productID string) (Product, error)
	GetStock(ctx context.Context, productID string) (int, error)
}

// productService implements the ProductService interface
type productService struct {
	repo   ProductRepository
	tracer trace.Tracer // Add tracer field
}

// NewProductService creates a new product service
func NewProductService(repo ProductRepository) ProductService {
	return &productService{
		repo:   repo,
		tracer: otel.Tracer("product-service"), // Initialize tracer
	}
}

// handleRepoError helper function now uses global logger
func (s *productService) handleRepoError(ctx context.Context, span trace.Span, opDesc string, err error) error {
	if err == nil {
		return nil
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, fmt.Sprintf("Repository error during %s", opDesc))

	logrus.WithContext(ctx).WithError(err).Errorf("Service: Repository error during %s", opDesc)

	// Check if the underlying error is ErrNotFound
	if errors.Is(err, commonErrors.ErrNotFound) {
		// Return the original error (which is likely an AppError)
		// Or potentially return a new commonErrors.NotFound if we want to reset context?
		// Let's return the original error for now.
		return err
	}
	// Wrap other repository errors
	return fmt.Errorf("repository error during %s: %w", opDesc, err)
}

// GetAll handles fetching all products
func (s *productService) GetAll(ctx context.Context) ([]Product, error) {
	logrus.WithContext(ctx).Info("Service: Fetching all products")

	ctx, span := s.tracer.Start(ctx, "service.GetAll")
	defer span.End()

	products, repoErr := s.repo.GetAll(ctx)
	if err := s.handleRepoError(ctx, span, "GetAll", repoErr); err != nil {
		return nil, err
	}

	span.SetAttributes(attribute.Int("db.result.count", len(products)))
	span.SetStatus(codes.Ok, "")

	return products, nil
}

// GetByID handles fetching a product by its ID
func (s *productService) GetByID(ctx context.Context, productID string) (Product, error) {
	logrus.WithContext(ctx).WithField("product.id", productID).Info("Service: Fetching product by ID")

	ctx, span := s.tracer.Start(ctx, "service.GetByID")
	defer span.End()

	span.SetAttributes(attribute.String("app.product.id", productID))

	product, repoErr := s.repo.GetByID(ctx, productID)
	if err := s.handleRepoError(ctx, span, fmt.Sprintf("GetByID for '%s'", productID), repoErr); err != nil {
		return Product{}, err
	}
	span.SetStatus(codes.Ok, "")

	return product, nil
}

// GetStock handles fetching stock for a product
func (s *productService) GetStock(ctx context.Context, productID string) (int, error) {
	logrus.WithContext(ctx).WithField("product.id", productID).Info("Service: Checking stock for product ID")

	ctx, span := s.tracer.Start(ctx, "service.GetStock")
	defer span.End()

	span.SetAttributes(attribute.String("app.product.id", productID))

	product, err := s.repo.GetByID(ctx, productID)
	if err != nil {
		return 0, s.handleRepoError(ctx, span, fmt.Sprintf("GetStock (GetByID stage) for '%s'", productID), err)
	}

	stock := product.Stock
	span.SetAttributes(attribute.Int("app.product.stock", stock))
	span.SetStatus(codes.Ok, "")

	return stock, nil
}
