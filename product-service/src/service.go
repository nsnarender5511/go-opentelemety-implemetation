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
)

type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	UpdateStock(ctx context.Context, productID string, newStock int) error
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
	productIdAttr := attribute.String("product.id", productID)
	newStockAttr := attribute.Int("product.new_stock", newStock)
	attrs := []attribute.KeyValue{productIdAttr, newStockAttr}

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, attrs...)

	ctx, spanner := commontrace.StartSpan(ctx, attrs...)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Service: UpdateStock called", slog.String("product_id", productID), slog.Int("new_stock", newStock))

	debugutils.Simulate(ctx)
	repoErr := s.repo.UpdateStock(ctx, productID, newStock)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Repository UpdateStock failed", slog.String("error", repoErr.Error()))
		if spanner != nil {
			spanner.SetStatus(codes.Error, repoErr.Error())
		}
		return repoErr // Propagate the specific error
	}

	debugutils.Simulate(ctx)
	s.logger.InfoContext(ctx, "Service: UpdateStock completed successfully", slog.String("product_id", productID), slog.Int("new_stock", newStock))
	return nil
}
