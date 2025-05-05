package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/narender/common/debugutils"
	"github.com/narender/common/globals"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByName(ctx context.Context, name string) (Product, error)
	UpdateStock(ctx context.Context, name string, newStock int) error
	GetByCategory(ctx context.Context, category string) ([]Product, error)
	BuyProduct(ctx context.Context, name string, quantity int) (remainingStock int, opErr error)
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
	s.logger.DebugContext(ctx, "Shop Manager: Front desk asking for all products list")

	ctx, span := commontrace.StartSpan(ctx)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to get all products")
	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't get all products: "+repoErr.Error())
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Error())
		}
		return nil, repoErr
	}
	productCount := len(products)
	s.logger.InfoContext(ctx, "Shop Manager: Received "+strconv.Itoa(productCount)+" products from stock room worker")

	debugutils.Simulate(ctx)
	span.SetAttributes(attribute.Int("products.count", productCount))
	s.logger.DebugContext(ctx, "Shop Manager: Sending "+strconv.Itoa(productCount)+" products to front desk")
	return products, nil
}

func (s *productService) GetByName(ctx context.Context, name string) (product Product, opErr error) {
	productNameAttr := attribute.String("product.name", name)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, productNameAttr)

	ctx, span := commontrace.StartSpan(ctx, productNameAttr)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Shop Manager: Front desk asking for product details with name: '"+name+"'")

	debugutils.Simulate(ctx)
	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to find product with name: '"+name+"'")
	product, repoErr := s.repo.GetByName(ctx, name)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find product with name '"+name+"' : "+repoErr.Error())
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Error())
		}
		return Product{}, repoErr
	}

	debugutils.Simulate(ctx)
	s.logger.InfoContext(ctx, "Shop Manager: Found product '"+product.Name+"'")
	s.logger.InfoContext(ctx, "Shop Manager: Sending product details to front desk")
	return product, nil
}

func (s *productService) UpdateStock(ctx context.Context, name string, newStock int) (opErr error) {
	productNameAttr := attribute.String("product.name", name)
	newStockAttr := attribute.Int("product.new_stock", newStock)

	mc := commonmetric.StartMetricsTimer()
	defer mc.End(ctx, &opErr, productNameAttr, newStockAttr)

	ctx, span := commontrace.StartSpan(ctx, productNameAttr, newStockAttr)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Shop Manager: Front desk requesting stock update for product '"+name+"' to "+strconv.Itoa(newStock))

	debugutils.Simulate(ctx)
	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to update inventory record")

	repoErr := s.repo.UpdateStock(ctx, name, newStock)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't update stock for product '"+name+"' : "+repoErr.Error())
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Error())
		}
		opErr = repoErr
		return opErr
	}

	debugutils.Simulate(ctx)

	s.logger.InfoContext(ctx, "Shop Manager: Stock for product '"+name+"' updated to "+strconv.Itoa(newStock)+" successfully")
	s.logger.InfoContext(ctx, "Shop Manager: Confirming stock update to front desk")
	return nil
}

func (s *productService) GetByCategory(ctx context.Context, category string) (products []Product, opErr error) {
	s.logger.InfoContext(ctx, "Shop Manager: Front desk asking for products in category: "+category)

	ctx, span := commontrace.StartSpan(ctx, attribute.String("product.category", category))
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to find products in category: "+category)
	products, repoErr := s.repo.GetByCategory(ctx, category)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find products in category "+category+": "+repoErr.Error())
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Error())
		}
		return nil, repoErr
	}

	productCount := len(products)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))
	s.logger.InfoContext(ctx, "Shop Manager: Found "+strconv.Itoa(productCount)+" products in category: "+category)
	s.logger.InfoContext(ctx, "Shop Manager: Sending category products to front desk")
	return products, nil
}

func (s *productService) BuyProduct(ctx context.Context, name string, quantity int) (remainingStock int, opErr error) {
	ctx, span := commontrace.StartSpan(ctx,
		attribute.String("product.name", name),
		attribute.Int("product.purchase_quantity", quantity),
	)
	defer func() {
		commontrace.EndSpan(span, &opErr, nil)
	}()

	s.logger.InfoContext(ctx, "Shop Manager: Front desk processing purchase for product '"+name+"' , quantity "+strconv.Itoa(quantity))

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker for current stock of product '"+name+"'")
	product, err := s.repo.GetByName(ctx, name)
	if err != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find product '"+name+"' for purchase: "+err.Error())
		span.SetStatus(codes.Error, "Failed to get product for purchase check")
		opErr = fmt.Errorf("product %s not found: %w", name, err)
		return 0, opErr
	}
	s.logger.DebugContext(ctx, "Shop Manager: Current stock for "+product.Name+" is "+strconv.Itoa(product.Stock))

	if product.Stock < quantity {
		s.logger.WarnContext(ctx, "Shop Manager: Purchase blocked for product '"+name+"' - insufficient stock",
			slog.Int("requested", quantity), slog.Int("available", product.Stock))
		// span.SetStatus(codes.FailedPrecondition, "Insufficient stock")
		opErr = fiber.NewError(http.StatusBadRequest, "insufficient stock available")
		return product.Stock, opErr
	}
	s.logger.DebugContext(ctx, "Shop Manager: Stock available for purchase")

	newStock := product.Stock - quantity
	s.logger.DebugContext(ctx, "Shop Manager: Calculated new stock for "+product.Name+" will be "+strconv.Itoa(newStock))

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to update inventory for "+product.Name+" to "+strconv.Itoa(newStock))
	err = s.repo.UpdateStock(ctx, name, newStock)
	if err != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker failed to update stock during purchase for product '"+name+"' : "+err.Error())
		span.SetStatus(codes.Error, "Failed to update stock in repository")
		opErr = fmt.Errorf("failed to update stock for product %s: %w", name, err)
		return product.Stock, opErr
	}

	remainingStock = newStock
	span.SetAttributes(attribute.Int("product.remaining_stock", remainingStock))
	s.logger.InfoContext(ctx, "Shop Manager: Purchase processed successfully for product '"+name+"' . Remaining stock: "+strconv.Itoa(remainingStock))

	return remainingStock, nil
}
