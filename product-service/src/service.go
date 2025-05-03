package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/narender/common/debugutils"
	commonerrors "github.com/narender/common/errors"
	"github.com/narender/common/globals"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/common/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	operationName := utils.GetCallerFunctionName(2)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr)

	ctx, spanner := commontrace.StartSpan(ctx)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Service: GetAll called", slog.String("operation", operationName))

	debugutils.Simulate(ctx)
	spanner.AddEvent("Calling repository GetAll")
	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		opErr = repoErr
		logLevel := slog.LevelError
		eventName := "error"
		if errors.Is(opErr, commonerrors.ErrNotFound) {
			logLevel = slog.LevelWarn
			eventName = "resource_not_found"
		}
		s.logger.Log(ctx, logLevel, "Repository GetAll failed",
			slog.String("layer", "service"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "service"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
			}
			if errors.Is(opErr, commonerrors.ErrNotFound) {
				spanAttrs = append(spanAttrs, attribute.Bool("error.expected", true))
			}
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
			if !errors.Is(opErr, commonerrors.ErrNotFound) {
				spanner.SetStatus(codes.Error, opErr.Error())
			}
		}
		return nil, opErr
	}
	productCount := len(products)
	spanner.AddEvent("Repository GetAll successful", trace.WithAttributes(attribute.Int("products.count", productCount)))

	debugutils.Simulate(ctx)
	spanner.SetAttributes(attribute.Int("products.count", productCount))
	return products, nil
}

func (s *productService) GetByID(ctx context.Context, productID string) (product Product, opErr error) {
	operationName := utils.GetCallerFunctionName(2)
	productIdAttr := attribute.String("product.id", productID)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, productIdAttr)

	ctx, spanner := commontrace.StartSpan(ctx, productIdAttr)
	notFoundMapper := func(err error) codes.Code {
		if err == nil {
			return codes.Ok
		}
		if errors.Is(err, commonerrors.ErrNotFound) {
			return codes.Ok
		}
		return codes.Error
	}
	defer commontrace.EndSpan(spanner, &opErr, notFoundMapper)

	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Service: GetByID called", slog.String("product_id", productID), slog.String("operation", operationName))

	debugutils.Simulate(ctx)
	spanner.AddEvent("Calling repository GetByID", trace.WithAttributes(productIdAttr))
	product, repoErr := s.repo.GetByID(ctx, productID)
	if repoErr != nil {
		opErr = repoErr
		logLevel := slog.LevelError
		eventName := "error"
		if errors.Is(opErr, commonerrors.ErrNotFound) {
			logLevel = slog.LevelWarn
			eventName = "resource_not_found"
		}
		s.logger.Log(ctx, logLevel, "Repository GetByID failed",
			slog.String("layer", "service"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
			slog.String("product_id", productID),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "service"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
				productIdAttr,
			}
			if errors.Is(opErr, commonerrors.ErrNotFound) {
				spanAttrs = append(spanAttrs, attribute.Bool("error.expected", true))
			}
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
			if !errors.Is(opErr, commonerrors.ErrNotFound) {
				spanner.SetStatus(codes.Error, opErr.Error())
			}
		}
		return Product{}, opErr
	}
	spanner.AddEvent("Repository GetByID successful")

	debugutils.Simulate(ctx)
	s.logger.InfoContext(ctx, "Service: GetByID completed successfully", slog.String("product_id", productID), slog.String("operation", operationName))
	return product, nil
}

func (s *productService) UpdateStock(ctx context.Context, productID string, newStock int) (opErr error) {
	operationName := utils.GetCallerFunctionName(2)
	productIdAttr := attribute.String("product.id", productID)
	newStockAttr := attribute.Int("product.new_stock", newStock)
	attrs := []attribute.KeyValue{productIdAttr, newStockAttr}

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, attrs...)

	ctx, spanner := commontrace.StartSpan(ctx, attrs...)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Service: UpdateStock called", slog.String("product_id", productID), slog.Int("new_stock", newStock), slog.String("operation", operationName))

	if newStock < 0 {
		opErr = commonerrors.NewValidationError(
			map[string]string{
				"userMessage":   "Invalid stock value provided",
				"product_id":    productID,
				"invalid_stock": fmt.Sprintf("%d", newStock),
			},
			commonerrors.ErrValidation,
		)
		logLevel := slog.LevelWarn
		eventName := "validation_error"
		s.logger.Log(ctx, logLevel, "Validation failed",
			slog.String("layer", "service"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "service"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
				attribute.Bool("error.expected", true),
			}
			spanAttrs = append(spanAttrs, attrs...)
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
		}
		return opErr
	}

	debugutils.Simulate(ctx)
	spanner.AddEvent("Calling repository UpdateStock", trace.WithAttributes(attrs...))
	repoErr := s.repo.UpdateStock(ctx, productID, newStock)
	if repoErr != nil {
		opErr = repoErr
		logLevel := slog.LevelError
		eventName := "error"
		if errors.Is(opErr, commonerrors.ErrNotFound) {
			logLevel = slog.LevelWarn
			eventName = "resource_not_found"
		}
		s.logger.Log(ctx, logLevel, "Repository UpdateStock failed",
			slog.String("layer", "service"),
			slog.String("operation", operationName),
			slog.String("error", opErr.Error()),
			slog.String("product_id", productID),
			slog.Int("new_stock", newStock),
		)
		if spanner != nil {
			spanAttrs := []attribute.KeyValue{
				attribute.String("layer", "service"),
				attribute.String("operation", operationName),
				attribute.String("error.message", opErr.Error()),
			}
			spanAttrs = append(spanAttrs, attrs...)
			if errors.Is(opErr, commonerrors.ErrNotFound) {
				spanAttrs = append(spanAttrs, attribute.Bool("error.expected", true))
			}
			spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
			if !errors.Is(opErr, commonerrors.ErrNotFound) {
				spanner.SetStatus(codes.Error, opErr.Error())
			}
		}
		return opErr
	}
	spanner.AddEvent("Repository UpdateStock successful")

	debugutils.Simulate(ctx)
	s.logger.InfoContext(ctx, "Service: UpdateStock completed successfully", slog.String("product_id", productID), slog.Int("new_stock", newStock), slog.String("operation", operationName))
	return nil
}
