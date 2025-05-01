package main

import (
	"context"
	"fmt"

	commonErrors "github.com/narender/common-module/errors"
	"github.com/narender/common-module/telemetry"

	"github.com/sirupsen/logrus"
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
	repo ProductRepository
}

// NewProductService creates a new product service
func NewProductService(repo ProductRepository) ProductService {
	return &productService{
		repo: repo,
	}
}

// --- New OTel-Aligned Helper Function ---
// Assumes the span is passed down or retrieved from context within the calling method
func (s *productService) handleRepoError(span trace.Span, log *logrus.Entry, opDesc string, err error) error {
	if err == nil {
		return nil
	}

	// OTel Standard: Record error on the span
	span.RecordError(err)
	// OTel Standard: Set span status to Error
	span.SetStatus(codes.Error, fmt.Sprintf("Repository error during %s", opDesc))

	log.WithError(err).Errorf("Service: Repository error during %s", opDesc)

	if commonErrors.Is(err, commonErrors.ErrProductNotFound) {
		return commonErrors.ErrProductNotFound // Return the sentinel error directly
	}
	// Wrap other errors generically
	return fmt.Errorf("repository error during %s: %w", opDesc, err)
}

// GetAll handles fetching all products
func (s *productService) GetAll(ctx context.Context) ([]Product, error) {
	log := logrus.WithContext(ctx)
	log.Info("Service: Fetching all products")

	ctx, span := telemetry.StartSpan(ctx, "product-service", "service.GetAll")
	defer span.End()

	products, repoErr := s.repo.GetAll(ctx)
	// Call the OTel-aligned helper, passing the current span
	if err := s.handleRepoError(span, log, "GetAll", repoErr); err != nil {
		return nil, err
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

	product, repoErr := s.repo.GetByID(ctx, productID)
	// Call the OTel-aligned helper, passing the current span
	if err := s.handleRepoError(span, log, fmt.Sprintf("GetByID for '%s'", productID), repoErr); err != nil {
		return Product{}, err // Return zero-value Product for error cases
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

	// Note: This calls GetByID internally, which now uses handleRepoError.
	// We still need to handle the error returned *from* GetByID here.
	product, repoErr := s.repo.GetByID(ctx, productID)
	// Call the OTel-aligned helper, passing the current span
	if err := s.handleRepoError(span, log, fmt.Sprintf("GetStock (via GetByID) for '%s'", productID), repoErr); err != nil {
		return 0, err // Return zero stock for error cases
	}

	stock := product.Stock
	telemetry.AddAttribute(span, "product.stock", stock)

	return stock, nil
}

// --- Helper Functions ---
