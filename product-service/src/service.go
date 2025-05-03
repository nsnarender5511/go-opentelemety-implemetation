package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	commonconst "github.com/narender/common/constants"
	"github.com/narender/common/debugutils"
	commonerrors "github.com/narender/common/errors"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"github.com/narender/common/globals"
)

const serviceScopeName = "github.com/narender/product-service/service"

type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
	UpdateStock(ctx context.Context, productID string, newStock int) error
}

type productService struct {
	repo ProductRepository
	logger *slog.Logger
}

func NewProductService(repo ProductRepository) ProductService {
	return &productService{
		repo:   repo,
		logger: globals.Logger(),
	}
}

func (s *productService) GetAll(ctx context.Context) (products []Product, opErr error) {
	const operation = "GetAll"

	mc := commonmetric.StartMetricsTimer(commonconst.ServiceLayer, operation)
	defer mc.End(ctx, &opErr)

	ctx, spanner := commontrace.StartSpan(ctx, serviceScopeName, operation, commonconst.ServiceLayer)
	defer spanner.End(&opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			s.logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.ServiceLayer))
		}
	}()

	s.logger.Info("Service: GetAll called")

	debugutils.Simulate(ctx)
	spanner.AddEvent("Calling repository GetAll")
	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		opErr = repoErr
		commonerrors.HandleLayerError(ctx, s.logger, spanner, opErr, commonconst.ServiceLayer, operation)
		spanner.AddEvent("Repository GetAll failed")
		return nil, opErr
	}
	productCount := len(products)
	spanner.AddEvent("Repository GetAll successful", trace.WithAttributes(attribute.Int("products.count", productCount)))

	debugutils.Simulate(ctx)
	spanner.SetAttributes(attribute.Int("products.count", productCount))
	s.logger.Info("Service: GetAll completed successfully", slog.Int("count", productCount))
	return products, nil
}

func (s *productService) GetByID(ctx context.Context, productID string) (product Product, opErr error) {
	const operation = "GetByID"
	productIdAttr := attribute.String("product.id", productID)

	mc := commonmetric.StartMetricsTimer(commonconst.ServiceLayer, operation)
	defer mc.End(ctx, &opErr, productIdAttr)

	ctx, spanner := commontrace.StartSpan(ctx, serviceScopeName, operation, commonconst.ServiceLayer, productIdAttr)
	notFoundMapper := func(err error) codes.Code {
		if err == nil {
			return codes.Ok
		}
		if errors.Is(err, commonerrors.ErrNotFound) {
			return codes.Ok
		}
		return codes.Error
	}
	defer spanner.End(&opErr, notFoundMapper)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			s.logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.ServiceLayer), productIdAttr)
		}
	}()

	s.logger.Info("Service: GetByID called", slog.String("product_id", productID))

	debugutils.Simulate(ctx)
	spanner.AddEvent("Calling repository GetByID", trace.WithAttributes(productIdAttr))
	product, repoErr := s.repo.GetByID(ctx, productID)
	if repoErr != nil {
		opErr = repoErr
		commonerrors.HandleLayerError(ctx, s.logger, spanner, opErr, commonconst.ServiceLayer, operation, productIdAttr)
		spanner.AddEvent("Repository GetByID failed", trace.WithAttributes(attribute.String("error.message", opErr.Error())))
		return Product{}, opErr
	}
	spanner.AddEvent("Repository GetByID successful")

	debugutils.Simulate(ctx)
	s.logger.Info("Service: GetByID completed successfully", slog.String("product_id", productID))
	return product, nil
}

func (s *productService) UpdateStock(ctx context.Context, productID string, newStock int) (opErr error) {
	const operation = "UpdateStock"
	productIdAttr := attribute.String("product.id", productID)
	newStockAttr := attribute.Int("product.new_stock", newStock)
	attrs := []attribute.KeyValue{productIdAttr, newStockAttr}

	mc := commonmetric.StartMetricsTimer(commonconst.ServiceLayer, operation)
	defer mc.End(ctx, &opErr, attrs...)

	ctx, spanner := commontrace.StartSpan(ctx, serviceScopeName, operation, commonconst.ServiceLayer, attrs...)
	defer spanner.End(&opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			s.logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.ServiceLayer), attrs)
		}
	}()

	s.logger.Info("Service: UpdateStock called", slog.String("product_id", productID), slog.Int("new_stock", newStock))

	if newStock < 0 {
		opErr = fmt.Errorf("invalid stock value %d: %w", newStock, commonerrors.ErrValidation)
		commonerrors.HandleLayerError(ctx, s.logger, spanner, opErr, commonconst.ServiceLayer, operation, attrs...)
		return opErr
	}

	debugutils.Simulate(ctx)
	spanner.AddEvent("Calling repository UpdateStock", trace.WithAttributes(attrs...))
	repoErr := s.repo.UpdateStock(ctx, productID, newStock)
	if repoErr != nil {
		opErr = repoErr
		commonerrors.HandleLayerError(ctx, s.logger, spanner, opErr, commonconst.ServiceLayer, operation, attrs...)
		spanner.AddEvent("Repository UpdateStock failed", trace.WithAttributes(attribute.String("error.message", opErr.Error())))
		return opErr
	}
	spanner.AddEvent("Repository UpdateStock successful")

	debugutils.Simulate(ctx)
	s.logger.Info("Service: UpdateStock completed successfully", slog.String("product_id", productID), slog.Int("new_stock", newStock))
	return nil
}
