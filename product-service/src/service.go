package main

import (
	"context"
	"fmt"

	commonErrors "github.com/narender/common-module/errors"
	"github.com/narender/common-module/telemetry"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
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

// getTracer is a helper to get the tracer instance consistently
func (s *productService) getTracer() trace.Tracer {
	return otel.Tracer("product-service/service")
}

// GetAll handles fetching all products
func (s *productService) GetAll(ctx context.Context) ([]Product, error) {
	log := logrus.WithContext(ctx)
	log.Info("Service: Fetching all products")

	var products []Product
	var err error

	tracer := s.getTracer()
	ctx, span := tracer.Start(ctx, "service.GetAll")
	defer span.End()

	products, err = s.repo.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Service: Failed to fetch all products from repo")
		return nil, fmt.Errorf("failed to find all products: %w", err)
	}

	span.SetAttributes(telemetry.DBResultCountKey.Int(len(products)))

	return products, nil
}

// GetByID handles fetching a product by its ID
func (s *productService) GetByID(ctx context.Context, productID string) (Product, error) {
	log := logrus.WithContext(ctx).WithField(telemetry.LogFieldProductID, productID)
	log.Info("Service: Fetching product by ID")

	var product Product
	var err error

	tracer := s.getTracer()
	ctx, span := tracer.Start(ctx, "service.GetByID")
	defer span.End()

	span.SetAttributes(telemetry.AppProductIDKey.String(productID))

	product, err = s.repo.FindByProductID(ctx, productID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
	log := logrus.WithContext(ctx).WithField(telemetry.LogFieldProductID, productID)
	log.Info("Service: Checking stock for product ID")

	var stock int
	var err error

	tracer := s.getTracer()
	ctx, span := tracer.Start(ctx, "service.GetStock")
	defer span.End()

	span.SetAttributes(telemetry.AppProductIDKey.String(productID))

	stock, err = s.repo.FindStockByProductID(ctx, productID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Service: Failed to get stock for product ID from repo")
		if commonErrors.Is(err, commonErrors.ErrProductNotFound) {
			return 0, commonErrors.ErrProductNotFound
		} else {
			return 0, fmt.Errorf("failed to get stock for product id '%s': %w", productID, err)
		}
	}

	span.SetAttributes(telemetry.AppProductStockKey.Int(stock))

	return stock, nil
}

// --- Helper Functions ---

// Removed local logServiceError function
