package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	commonErrors "github.com/narender/common/errors"
	"github.com/narender/common/otel"
	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/attribute"
	// "go.opentelemetry.io/otel/codes" // No longer needed directly
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ProductService defines the interface for product operations.
type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, productID string) (Product, error)
	GetStock(ctx context.Context, productID string) (int, error)
	// Ping(ctx context.Context) error // Example: Add Ping if health check needs it
}

// productService implements the ProductService interface.
type productService struct {
	repo   ProductRepository
	logger *logrus.Logger   // Add logger field
	tracer oteltrace.Tracer // Use imported trace type
}

// NewProductService creates a new product service.
func NewProductService(repo ProductRepository, logger *logrus.Logger, tracer oteltrace.Tracer) ProductService {
	return &productService{
		repo:   repo,
		logger: logger,
		tracer: tracer,
	}
}

// handleRepoError logs the repository error, records it on the span, and wraps it into an AppError.
func (s *productService) handleRepoError(ctx context.Context, span oteltrace.Span, operation string, err error) error {
	if err == nil {
		return nil
	}

	// Log the error with context
	s.logger.WithContext(ctx).WithError(err).Errorf("Service: Repository error during %s", operation)

	// Record the error on the span using the common helper
	otel.RecordSpanError(span, err, attribute.String("app.operation", operation))

	// Wrap the error into an appropriate AppError for the handler/middleware
	var dbErr *commonErrors.DatabaseError // Check if repo returns this type
	switch {
	case errors.Is(err, commonErrors.ErrNotFound):
		// Wrap NotFound error with status code
		return commonErrors.Wrap(err, http.StatusNotFound, fmt.Sprintf("%s failed: resource not found", operation)).
			WithUserMessage("The requested product could not be found.")

	case errors.As(err, &dbErr):
		// Wrap DatabaseError as internal server error
		appErr := commonErrors.Wrap(err, http.StatusInternalServerError, fmt.Sprintf("%s failed: database error", operation)).
			WithUserMessage("An internal error occurred while accessing data.")
		// Add database operation context if available
		if dbErr.Operation != "" {
			appErr = appErr.WithContext("db.operation", dbErr.Operation)
		}
		return appErr

	default:
		// Wrap other unexpected errors as internal server errors
		return commonErrors.Wrap(err, http.StatusInternalServerError, fmt.Sprintf("%s failed: internal error", operation)).
			WithUserMessage("An unexpected internal server error occurred.")
	}
}

// GetAll retrieves all products.
func (s *productService) GetAll(ctx context.Context) ([]Product, error) {
	// Logging is handled by middleware
	ctx, span := s.tracer.Start(ctx, "service.GetAll")
	defer span.End()
	s.logger.Info("Service: GetAll called")

	products, repoErr := s.repo.GetAll(ctx)
	if err := s.handleRepoError(ctx, span, "GetAllProducts", repoErr); err != nil {
		s.logger.Warnf("Service: GetAll finished with error: %v", err) // Log before returning error
		return nil, err                                                // Return the wrapped AppError
	}

	// Set specific attributes if needed
	span.SetAttributes(attribute.Int("db.result.count", len(products)))
	// Span status OK is set by default, no need for SetStatus(codes.Ok, ...)
	s.logger.Infof("Service: GetAll completed successfully, returning %d products", len(products))
	return products, nil
}

// GetByID retrieves a product by its ID.
func (s *productService) GetByID(ctx context.Context, productID string) (Product, error) {
	// Logging handled by middleware
	ctx, span := s.tracer.Start(ctx, "service.GetByID")
	defer span.End()
	s.logger.Infof("Service: GetByID called for productID: %s", productID)

	// Add attributes early
	span.SetAttributes(attribute.String("app.product.id", productID))

	product, repoErr := s.repo.GetByID(ctx, productID)
	if err := s.handleRepoError(ctx, span, "GetProductByID", repoErr); err != nil {
		s.logger.Warnf("Service: GetByID finished for productID %s with error: %v", productID, err) // Log before returning error
		return Product{}, err                                                                       // Return the wrapped AppError
	}

	// Span status OK is default
	s.logger.Infof("Service: GetByID completed successfully for productID: %s", productID)
	return product, nil
}

// GetStock retrieves the stock for a product by its ID.
func (s *productService) GetStock(ctx context.Context, productID string) (int, error) {
	// Logging handled by middleware
	ctx, span := s.tracer.Start(ctx, "service.GetStock")
	defer span.End()
	s.logger.Infof("Service: GetStock called for productID: %s", productID)

	// Add attributes early
	span.SetAttributes(attribute.String("app.product.id", productID))

	// This service method currently calls GetByID in the repository.
	// Consider if a dedicated GetStock repository method is better.
	product, repoErr := s.repo.GetByID(ctx, productID)
	if err := s.handleRepoError(ctx, span, "GetStock (via GetByID)", repoErr); err != nil {
		s.logger.Warnf("Service: GetStock finished for productID %s with error: %v", productID, err) // Log before returning error
		return 0, err                                                                                // Return the wrapped AppError
	}

	// Span status OK is default
	s.logger.Infof("Service: GetStock completed successfully for productID: %s, stock: %d", productID, product.Stock)
	return product.Stock, nil
}
