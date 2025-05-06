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

	newCtx, span := commontrace.StartSpan(ctx)
	ctx = newCtx // Update ctx if StartSpan modifies it
	defer func() {
		var telemetryErr error
		if appErr != nil {
			telemetryErr = appErr
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return nil, appErr
	}

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to get all products")
	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't get all products", slog.String("error", repoErr.Error()))
		if span != nil { // Check if span is valid before using
			span.SetStatus(codes.Error, repoErr.Message)
		}
		appErr = repoErr
		return nil, appErr
	}
	productCount := len(products)
	s.logger.InfoContext(ctx, "Shop Manager: Received "+strconv.Itoa(productCount)+" products from stock room worker")

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return nil, appErr
	}
	span.SetAttributes(attribute.Int("products.count", productCount))
	s.logger.DebugContext(ctx, "Shop Manager: Sending "+strconv.Itoa(productCount)+" products to front desk")
	// appErr is already nil or set by previous errors
	return products, appErr
}

func (s *productService) GetByName(ctx context.Context, name string) (product Product, appErr *apierrors.AppError) {
	productNameAttr := attribute.String("product.name", name)

	mc := commonmetric.StartMetricsTimer() // Assuming StartMetricsTimer doesn't modify context or return a new one
	newCtx, span := commontrace.StartSpan(ctx, productNameAttr)
	ctx = newCtx // Update ctx
	defer func() {
		var telemetryErr error
		if appErr != nil {
			telemetryErr = appErr
		}
		mc.End(ctx, &telemetryErr, productNameAttr)   // Pass address of telemetryErr
		commontrace.EndSpan(span, &telemetryErr, nil) // Pass address of telemetryErr
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return Product{}, appErr
	}

	s.logger.InfoContext(ctx, "Shop Manager: Front desk asking for product details", slog.String("product_name", name))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return Product{}, appErr
	}

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to find product", slog.String("product_name", name))
	product, repoErr := s.repo.GetByName(ctx, name)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find product", slog.String("product_name", name), slog.String("error", repoErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}
		appErr = repoErr
		return Product{}, appErr
	}

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return Product{}, appErr
	}
	s.logger.InfoContext(ctx, "Shop Manager: Found product '"+product.Name+"'")
	s.logger.InfoContext(ctx, "Shop Manager: Sending product details to front desk")
	return product, appErr // appErr is nil here if successful
}

func (s *productService) UpdateStock(ctx context.Context, name string, newStock int) (appErr *apierrors.AppError) {
	productNameAttr := attribute.String("product.name", name)
	newStockAttr := attribute.Int("product.new_stock", newStock)

	mc := commonmetric.StartMetricsTimer()
	newCtx, span := commontrace.StartSpan(ctx, productNameAttr, newStockAttr)
	ctx = newCtx // Update ctx
	defer func() {
		var telemetryErr error
		if appErr != nil {
			telemetryErr = appErr
		}
		mc.End(ctx, &telemetryErr, productNameAttr, newStockAttr) // Pass address
		commontrace.EndSpan(span, &telemetryErr, nil)             // Pass address
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return appErr
	}

	s.logger.InfoContext(ctx, "Shop Manager: Front desk requesting stock update", slog.String("product_name", name), slog.Int("new_stock", newStock))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return appErr
	}

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to update inventory record")

	repoErr := s.repo.UpdateStock(ctx, name, newStock)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't update stock", slog.String("product_name", name), slog.String("error", repoErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}
		appErr = repoErr
		return appErr
	}

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return appErr
	}

	s.logger.InfoContext(ctx, "Shop Manager: Stock updated successfully", slog.String("product_name", name), slog.Int("new_stock", newStock))
	s.logger.InfoContext(ctx, "Shop Manager: Confirming stock update to front desk")
	return appErr // appErr is nil here if successful
}

func (s *productService) GetByCategory(ctx context.Context, category string) (products []Product, appErr *apierrors.AppError) {
	s.logger.InfoContext(ctx, "Shop Manager: Front desk asking for products by category", slog.String("category", category))

	newCtx, span := commontrace.StartSpan(ctx, attribute.String("product.category", category))
	ctx = newCtx // Update ctx
	defer func() {
		var telemetryErr error
		if appErr != nil {
			telemetryErr = appErr
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return nil, appErr
	}

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to find products", slog.String("category", category))
	products, repoErr := s.repo.GetByCategory(ctx, category)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find products", slog.String("category", category), slog.String("error", repoErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}
		appErr = repoErr
		return nil, appErr
	}

	productCount := len(products)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))
	s.logger.InfoContext(ctx, "Shop Manager: Found "+strconv.Itoa(productCount)+" products in category: "+category)
	s.logger.InfoContext(ctx, "Shop Manager: Sending category products to front desk")
	return products, appErr // appErr is nil here if successful
}

func (s *productService) BuyProduct(ctx context.Context, name string, quantity int) (remainingStock int, appErr *apierrors.AppError) {
	newCtx, span := commontrace.StartSpan(ctx,
		attribute.String("product.name", name),
		attribute.Int("product.purchase_quantity", quantity),
	)
	ctx = newCtx // Update ctx
	defer func() {
		var telemetryErr error
		if appErr != nil {
			telemetryErr = appErr
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	s.logger.InfoContext(ctx, "Shop Manager: Processing purchase request", slog.String("product_name", name), slog.Int("quantity", quantity))

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker for current stock", slog.String("product_name", name))
	product, repoGetErr := s.repo.GetByName(ctx, name)
	if repoGetErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find product for purchase check", slog.String("product_name", name), slog.String("error", repoGetErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoGetErr.Message)
		}
		appErr = repoGetErr
		return 0, appErr
	}
	s.logger.DebugContext(ctx, "Shop Manager: Current stock check", slog.String("product_name", product.Name), slog.Int("stock", product.Stock))

	if product.Stock < quantity {
		errMsg := fmt.Sprintf("Insufficient stock for product '%s'. Available: %d, Requested: %d", name, product.Stock, quantity)
		s.logger.WarnContext(ctx, "Shop Manager: Purchase blocked - insufficient stock",
			slog.String("product_name", name),
			slog.Int("requested", quantity),
			slog.Int("available", product.Stock),
		)
		if span != nil {
			span.SetStatus(codes.Error, "Insufficient stock") // Specific message for span
		}
		appErr = apierrors.NewAppError(apierrors.ErrCodeInsufficientStock, errMsg, nil)
		return product.Stock, appErr // Return current stock with the error
	}
	s.logger.DebugContext(ctx, "Shop Manager: Stock available for purchase")

	newStock := product.Stock - quantity
	s.logger.DebugContext(ctx, "Shop Manager: Calculated new stock", slog.String("product_name", product.Name), slog.Int("new_stock", newStock))

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to update inventory", slog.String("product_name", product.Name), slog.Int("new_stock", newStock))
	repoUpdateErr := s.repo.UpdateStock(ctx, name, newStock)
	if repoUpdateErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker failed to update stock during purchase", slog.String("product_name", name), slog.String("error", repoUpdateErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoUpdateErr.Message)
		}
		appErr = repoUpdateErr
		return product.Stock, appErr // Return pre-update stock if update fails
	}

	remainingStock = newStock
	span.SetAttributes(attribute.Int("product.remaining_stock", remainingStock))
	s.logger.InfoContext(ctx, "Shop Manager: Purchase processed successfully", slog.String("product_name", name), slog.Int("remaining_stock", remainingStock))

	return remainingStock, appErr // appErr is nil here if successful
}
