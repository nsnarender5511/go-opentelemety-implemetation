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

