package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/narender/common/debugutils"
	"github.com/narender/common/globals"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	// Import common errors package
	apierrors "github.com/narender/common/apierrors"
)

type ProductService interface {
	GetAll(ctx context.Context) ([]Product, *apierrors.AppError)
	GetByName(ctx context.Context, name string) (Product, *apierrors.AppError)
	UpdateStock(ctx context.Context, name string, newStock int) *apierrors.AppError
	GetByCategory(ctx context.Context, category string) ([]Product, *apierrors.AppError)
	BuyProduct(ctx context.Context, name string, quantity int) (remainingStock int, appErr *apierrors.AppError)
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

func (s *productService) GetAll(ctx context.Context) (products []Product, appErr *apierrors.AppError) {
	s.logger.DebugContext(ctx, "Shop Manager: Front desk asking for all products list")

	var opErr error
	ctx, span := commontrace.StartSpan(ctx)
	defer func() {
		if appErr != nil && opErr == nil {
			opErr = appErr
		}
		commontrace.EndSpan(span, &opErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return nil, simAppErr
	}

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to get all products")
	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't get all products", slog.String("error", repoErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}
		return nil, repoErr
	}
	productCount := len(products)
	s.logger.InfoContext(ctx, "Shop Manager: Received "+strconv.Itoa(productCount)+" products from stock room worker")

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return nil, simAppErr
	}
	span.SetAttributes(attribute.Int("products.count", productCount))
	s.logger.DebugContext(ctx, "Shop Manager: Sending "+strconv.Itoa(productCount)+" products to front desk")
	return products, nil
}

func (s *productService) GetByName(ctx context.Context, name string) (product Product, appErr *apierrors.AppError) {
	productNameAttr := attribute.String("product.name", name)

	mc := commonmetric.StartMetricsTimer()
	var opErr error
	ctx, span := commontrace.StartSpan(ctx, productNameAttr)
	var spanOpErr error
	defer func() {
		if appErr != nil {
			if opErr == nil {
				opErr = appErr
			}
			if spanOpErr == nil {
				spanOpErr = appErr
			}
		}
		mc.End(ctx, &opErr, productNameAttr)
		commontrace.EndSpan(span, &spanOpErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return Product{}, simAppErr
	}

	s.logger.InfoContext(ctx, "Shop Manager: Front desk asking for product details", slog.String("product_name", name))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return Product{}, simAppErr
	}

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to find product", slog.String("product_name", name))
	product, repoErr := s.repo.GetByName(ctx, name)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find product", slog.String("product_name", name), slog.String("error", repoErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}
		return Product{}, repoErr
	}

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return Product{}, simAppErr
	}
	s.logger.InfoContext(ctx, "Shop Manager: Found product '"+product.Name+"'")
	s.logger.InfoContext(ctx, "Shop Manager: Sending product details to front desk")
	return product, nil
}

func (s *productService) UpdateStock(ctx context.Context, name string, newStock int) (appErr *apierrors.AppError) {
	productNameAttr := attribute.String("product.name", name)
	newStockAttr := attribute.Int("product.new_stock", newStock)

	mc := commonmetric.StartMetricsTimer()
	var opErr error
	ctx, span := commontrace.StartSpan(ctx, productNameAttr, newStockAttr)
	var spanOpErr error
	defer func() {
		if appErr != nil {
			if opErr == nil {
				opErr = appErr
			}
			if spanOpErr == nil {
				spanOpErr = appErr
			}
		}
		mc.End(ctx, &opErr, productNameAttr, newStockAttr)
		commontrace.EndSpan(span, &spanOpErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return simAppErr
	}

	s.logger.InfoContext(ctx, "Shop Manager: Front desk requesting stock update", slog.String("product_name", name), slog.Int("new_stock", newStock))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return simAppErr
	}

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to update inventory record")

	repoErr := s.repo.UpdateStock(ctx, name, newStock)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't update stock", slog.String("product_name", name), slog.String("error", repoErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}
		return repoErr
	}

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return simAppErr
	}

	s.logger.InfoContext(ctx, "Shop Manager: Stock updated successfully", slog.String("product_name", name), slog.Int("new_stock", newStock))
	s.logger.InfoContext(ctx, "Shop Manager: Confirming stock update to front desk")
	return nil
}

func (s *productService) GetByCategory(ctx context.Context, category string) (products []Product, appErr *apierrors.AppError) {
	s.logger.InfoContext(ctx, "Shop Manager: Front desk asking for products by category", slog.String("category", category))

	var opErr error
	ctx, span := commontrace.StartSpan(ctx, attribute.String("product.category", category))
	defer func() {
		if appErr != nil && opErr == nil {
			opErr = appErr
		}
		commontrace.EndSpan(span, &opErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return nil, simAppErr
	}

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to find products", slog.String("category", category))
	products, repoErr := s.repo.GetByCategory(ctx, category)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find products", slog.String("category", category), slog.String("error", repoErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}
		return nil, repoErr
	}

	productCount := len(products)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))
	s.logger.InfoContext(ctx, "Shop Manager: Found "+strconv.Itoa(productCount)+" products in category: "+category)
	s.logger.InfoContext(ctx, "Shop Manager: Sending category products to front desk")
	return products, nil
}

func (s *productService) BuyProduct(ctx context.Context, name string, quantity int) (remainingStock int, appErr *apierrors.AppError) {
	var opErr error
	ctx, span := commontrace.StartSpan(ctx,
		attribute.String("product.name", name),
		attribute.Int("product.purchase_quantity", quantity),
	)
	defer func() {
		if appErr != nil && opErr == nil {
			opErr = appErr
		}
		commontrace.EndSpan(span, &opErr, nil)
	}()

	s.logger.InfoContext(ctx, "Shop Manager: Processing purchase request", slog.String("product_name", name), slog.Int("quantity", quantity))

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker for current stock", slog.String("product_name", name))
	product, repoGetErr := s.repo.GetByName(ctx, name)
	if repoGetErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find product for purchase check", slog.String("product_name", name), slog.String("error", repoGetErr.Error()))
		span.SetStatus(codes.Error, repoGetErr.Message)
		return 0, repoGetErr
	}
	s.logger.DebugContext(ctx, "Shop Manager: Current stock check", slog.String("product_name", product.Name), slog.Int("stock", product.Stock))

	if product.Stock < quantity {
		errMsg := fmt.Sprintf("Insufficient stock for product '%s'. Available: %d, Requested: %d", name, product.Stock, quantity)
		s.logger.WarnContext(ctx, "Shop Manager: Purchase blocked - insufficient stock",
			slog.String("product_name", name),
			slog.Int("requested", quantity),
			slog.Int("available", product.Stock),
		)
		span.SetStatus(codes.Error, "Insufficient stock")
		appErr = apierrors.NewAppError(apierrors.ErrCodeInsufficientStock, errMsg, nil)
		return product.Stock, appErr
	}
	s.logger.DebugContext(ctx, "Shop Manager: Stock available for purchase")

	newStock := product.Stock - quantity
	s.logger.DebugContext(ctx, "Shop Manager: Calculated new stock", slog.String("product_name", product.Name), slog.Int("new_stock", newStock))

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to update inventory", slog.String("product_name", product.Name), slog.Int("new_stock", newStock))
	repoUpdateErr := s.repo.UpdateStock(ctx, name, newStock)
	if repoUpdateErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker failed to update stock during purchase", slog.String("product_name", name), slog.String("error", repoUpdateErr.Error()))
		span.SetStatus(codes.Error, repoUpdateErr.Message)
		return product.Stock, repoUpdateErr
	}

	remainingStock = newStock
	span.SetAttributes(attribute.Int("product.remaining_stock", remainingStock))
	s.logger.InfoContext(ctx, "Shop Manager: Purchase processed successfully", slog.String("product_name", name), slog.Int("remaining_stock", remainingStock))

	return remainingStock, nil
}
