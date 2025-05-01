package main

import (
	"context"
	"signoz-common/telemetry"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
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
	tracer trace.Tracer
}

// NewProductService creates a new product service
func NewProductService(repo ProductRepository) ProductService {
	return &productService{
		repo:   repo,
		tracer: telemetry.GetTracer("product-service/service"),
	}
}

// GetAll handles fetching all products
func (s *productService) GetAll(ctx context.Context) ([]Product, error) {
	log := logrus.WithContext(ctx)
	log.Info("Service: Fetching all products")

	ctx, span := s.tracer.Start(ctx, "GetAllService")
	defer span.End()

	products, err := s.repo.FindAll(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Service: Failed to fetch all products from repo")
		return nil, err
	}

	span.SetAttributes(attribute.Int("product.count", len(products)))
	return products, nil
}

// GetByID handles fetching a product by its ID
func (s *productService) GetByID(ctx context.Context, productID string) (Product, error) {
	log := logrus.WithContext(ctx)
	log.Info("Service: Fetching product by ID", "productID", productID)

	ctx, span := s.tracer.Start(ctx, "GetByIDService",
		trace.WithAttributes(attribute.String("product.id", productID)),
	)
	defer span.End()

	product, err := s.repo.FindByProductID(ctx, productID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Service: Failed to find product by ID in repo", "productID", productID)
		return Product{}, err
	}
	return product, nil
}

// GetStock handles fetching stock for a product
func (s *productService) GetStock(ctx context.Context, productID string) (int, error) {
	log := logrus.WithContext(ctx)
	log.Info("Service: Checking stock for product ID", "productID", productID)

	ctx, span := s.tracer.Start(ctx, "GetStockService",
		trace.WithAttributes(attribute.String("product.id", productID)),
	)
	defer span.End()

	stock, err := s.repo.FindStockByProductID(ctx, productID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.WithError(err).Error("Service: Failed to get stock for product ID from repo", "productID", productID)
		return 0, err
	}

	span.SetAttributes(attribute.Int("product.stock", stock))
	return stock, nil
}

// --- Helper Functions ---

// Removed local logServiceError function
