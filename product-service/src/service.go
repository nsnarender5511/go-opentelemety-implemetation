package main

import (
	"context"
	"log/slog"

	"github.com/narender/common/debugutils"
	"github.com/narender/common/globals"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/google/uuid"
)

type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	UpdateStock(ctx context.Context, productID string, newStock int) error
	Create(ctx context.Context, payload createProductPayload) (Product, error)
}

type productService struct {
	repo   ProductRepository
	logger *slog.Logger
}

func NewProductService(repo ProductRepository) ProductService {
	return &productService{
		repo:   repo,
		logger: globals.Logger(),
	}
}

func (s *productService) GetAll(ctx context.Context) (products []Product, opErr error) {

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr)

	ctx, spanner := commontrace.StartSpan(ctx)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Service: GetAll called")

	debugutils.Simulate(ctx)
	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Repository GetAll failed", slog.String("error", repoErr.Error()))
		if spanner != nil {
			spanner.SetStatus(codes.Error, repoErr.Error())
		}
		return nil, repoErr
	}
	productCount := len(products)

	debugutils.Simulate(ctx)
	spanner.SetAttributes(attribute.Int("products.count", productCount))
	return products, nil
}

func (s *productService) GetByID(ctx context.Context, productID string) (product Product, opErr error) {
	productIdAttr := attribute.String("product.id", productID)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, productIdAttr)

	ctx, spanner := commontrace.StartSpan(ctx, productIdAttr)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Service: GetByID called", slog.String("product_id", productID))

	debugutils.Simulate(ctx)
	product, repoErr := s.repo.GetByID(ctx, productID)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Repository GetByID failed", slog.String("error", repoErr.Error()))
		if spanner != nil {
			spanner.SetStatus(codes.Error, repoErr.Error())
		}
		return Product{}, repoErr
	}

	debugutils.Simulate(ctx)
	s.logger.InfoContext(ctx, "Service: GetByID completed successfully", slog.String("product_id", productID))
	return product, nil
}

func (s *productService) UpdateStock(ctx context.Context, productID string, newStock int) (opErr error) {
	// Setup attributes for tracing/metrics
	productIdAttr := attribute.String("product.id", productID)
	newStockAttr := attribute.Int("product.new_stock", newStock)

	// Start metrics timer
	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, productIdAttr, newStockAttr) // End metrics timer with attributes

	// Start tracing span
	ctx, spanner := commontrace.StartSpan(ctx, productIdAttr, newStockAttr)
	defer commontrace.EndSpan(spanner, &opErr, nil) // End tracing span

	// Simulate potential delays/errors
	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Service: UpdateStock called", slog.String("productID", productID), slog.Int("newStock", newStock))

	// Simulate potential delays/errors before repository call
	debugutils.Simulate(ctx)

	// Call the repository method
	repoErr := s.repo.UpdateStock(ctx, productID, newStock)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Repository UpdateStock failed", slog.String("productID", productID), slog.String("error", repoErr.Error()))
		if spanner != nil {
			spanner.SetStatus(codes.Error, repoErr.Error())
		}
		opErr = repoErr // Assign the error to opErr for the defer handlers
		return opErr
	}

	// Simulate potential delays/errors after successful repository call
	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Service: UpdateStock completed successfully", slog.String("productID", productID))
	return nil // Return nil on success
}

// Create handles the logic for creating a new product.
func (s *productService) Create(ctx context.Context, payload createProductPayload) (createdProduct Product, opErr error) {
	// Setup attributes for tracing/metrics (payload doesn't have ID yet)
	nameAttr := attribute.String("product.name", payload.Name)

	// Start metrics timer
	mc := commonmetric.StartMetricsTimer()
	// Defer end call - note: ID attribute will be added later if successful
	defer mc.End(ctx, &opErr, nameAttr)

	// Start tracing span
	ctx, spanner := commontrace.StartSpan(ctx, nameAttr)
	defer commontrace.EndSpan(spanner, &opErr, nil) // End tracing span

	// Basic Validation
	if payload.Name == "" {
		s.logger.WarnContext(ctx, "Service: Create failed - missing product name")
		if spanner != nil {
			spanner.SetStatus(codes.Error, opErr.Error())
		}
		return Product{}, opErr
	}
	if payload.Price < 0 {
		s.logger.WarnContext(ctx, "Service: Create failed - invalid price", slog.Float64("price", payload.Price))
		if spanner != nil {
			spanner.SetStatus(codes.Error, opErr.Error())
		}
		return Product{}, opErr
	}
	if payload.Stock < 0 {
		s.logger.WarnContext(ctx, "Service: Create failed - invalid stock", slog.Int("stock", payload.Stock))
		if spanner != nil {
			spanner.SetStatus(codes.Error, opErr.Error())
		}
		return Product{}, opErr
	}

	// Generate Product ID
	productID := "prod_" + uuid.NewString()
	idAttr := attribute.String("product.id", productID)
	if spanner != nil {
		spanner.SetAttributes(idAttr)
	}

	s.logger.InfoContext(ctx, "Service: Creating new product", slog.String("generatedProductID", productID), slog.String("name", payload.Name))

	// Create Product struct
	newProduct := Product{
		ProductID:   productID,
		Name:        payload.Name,
		Description: payload.Description,
		Price:       payload.Price,
		Stock:       payload.Stock,
	}

	// Simulate potential delays/errors before repository call
	debugutils.Simulate(ctx)

	// Call Repository Create
	repoErr := s.repo.Create(ctx, newProduct)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Repository Create failed", slog.String("productID", productID), slog.String("error", repoErr.Error()))
		// Error type might be Conflict or other internal error
		opErr = repoErr // Assign repo error to opErr for defer/return
		if spanner != nil {
			spanner.SetStatus(codes.Error, opErr.Error())
		}
		return Product{}, opErr // Return empty product and the error
	}

	// Simulate potential delays/errors after successful repository call
	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Service: Product created successfully", slog.String("productID", productID))
	return newProduct, nil // Return created product and nil error
}
