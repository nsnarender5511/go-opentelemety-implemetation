package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	commonconst "github.com/narender/common/constants"
	"github.com/narender/common/debugutils"
	commonerrors "github.com/narender/common/errors"
	commonlog "github.com/narender/common/log"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const serviceScopeName = "github.com/narender/product-service/service"

type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, id string) (Product, error)
}

type JsonProductService struct {
	products map[string]Product
	logger   *slog.Logger
	mu       sync.RWMutex // Added for potential future updates
}

type productService struct {
	repo ProductRepository
}

func NewProductService(repo ProductRepository) ProductService {
	return &productService{
		repo: repo,
	}
}

func (s *productService) GetAll(ctx context.Context) (products []Product, opErr error) {
	const operation = "GetAll"
	logger := commonlog.L

	mc := commonmetric.StartMetricsTimer(commonconst.ServiceLayer, operation)
	defer mc.End(ctx, &opErr)

	ctx, spanner := commontrace.StartSpan(ctx, serviceScopeName, operation, commonconst.ServiceLayer)
	defer spanner.End(&opErr, nil)

	debugutils.Simulate(ctx)

	defer func() {
		if rec := recover(); rec != nil {
			opErr = fmt.Errorf("panic recovered in %s: %v", operation, rec)
			logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.ServiceLayer))
		}
	}()

	logger.Info("Service: GetAll called")

	debugutils.Simulate(ctx)
	spanner.AddEvent("Calling repository GetAll")
	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		opErr = repoErr
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.ServiceLayer, operation)
		spanner.AddEvent("Repository GetAll failed")
		return nil, opErr
	}
	productCount := len(products)
	spanner.AddEvent("Repository GetAll successful", trace.WithAttributes(attribute.Int("products.count", productCount)))

	debugutils.Simulate(ctx)
	spanner.SetAttributes(attribute.Int("products.count", productCount))
	logger.Info("Service: GetAll completed successfully", slog.Int("count", productCount))
	return products, nil
}

func (s *productService) GetByID(ctx context.Context, productID string) (product Product, opErr error) {
	const operation = "GetByID"
	logger := commonlog.L
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
			logger.Error("Panic recovered", slog.Any("panic", rec), slog.String("operation", operation), slog.String("layer", commonconst.ServiceLayer), productIdAttr)
		}
	}()

	logger.Info("Service: GetByID called", slog.String("product_id", productID))

	debugutils.Simulate(ctx)
	spanner.AddEvent("Calling repository GetByID", trace.WithAttributes(productIdAttr))
	product, repoErr := s.repo.GetByID(ctx, productID)
	if repoErr != nil {
		opErr = repoErr
		commonerrors.HandleLayerError(ctx, logger, spanner, opErr, commonconst.ServiceLayer, operation, productIdAttr)
		spanner.AddEvent("Repository GetByID failed", trace.WithAttributes(attribute.String("error.message", opErr.Error())))
		return Product{}, opErr
	}
	spanner.AddEvent("Repository GetByID successful")

	debugutils.Simulate(ctx)
	logger.Info("Service: GetByID completed successfully", slog.String("product_id", productID))
	return product, nil
}

func NewJsonProductService(dataPath string, logger *slog.Logger) (*JsonProductService, error) {
	const operation = "NewJsonProductService"
	logger.Info("Initializing JsonProductService", slog.String("dataPath", dataPath))

	dataBytes, err := os.ReadFile(dataPath)
	if err != nil {
		logger.Error("Failed to read product data file", slog.String("path", dataPath), slog.Any("error", err), slog.String("operation", operation))
		return nil, fmt.Errorf("failed to read data file '%s': %w", dataPath, err)
	}

	var products map[string]Product
	err = json.Unmarshal(dataBytes, &products)
	if err != nil {
		logger.Error("Failed to unmarshal product data JSON", slog.String("path", dataPath), slog.Any("error", err), slog.String("operation", operation))
		return nil, fmt.Errorf("failed to parse product data from '%s': %w", dataPath, err)
	}

	logger.Info("Successfully loaded product data", slog.Int("productCount", len(products)), slog.String("operation", operation))

	return &JsonProductService{
		products: products,
		logger:   logger,
	}, nil
}

func (s *JsonProductService) GetAll(ctx context.Context) ([]Product, error) {
	const operation = "JsonGetAll"
	s.logger.DebugContext(ctx, "Entering JsonProductService GetAll", slog.String("operation", operation))
	s.mu.RLock() // Use RLock for read operations
	defer s.mu.RUnlock()

	productList := make([]Product, 0, len(s.products))
	for _, product := range s.products {
		productList = append(productList, product)
	}
	s.logger.InfoContext(ctx, "JsonProductService returning all products", slog.Int("count", len(productList)), slog.String("operation", operation))
	return productList, nil
}

func (s *JsonProductService) GetByID(ctx context.Context, id string) (Product, error) {
	const operation = "JsonGetByID"
	s.logger.DebugContext(ctx, "Entering JsonProductService GetByID", slog.String("product_id", id), slog.String("operation", operation))
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, found := s.products[id]
	if !found {
		s.logger.WarnContext(ctx, "Product not found in JsonProductService", slog.String("product_id", id), slog.String("operation", operation))
		s.logger.DebugContext(ctx, "JsonProductService returning not found error", slog.String("product_id", id), slog.String("operation", operation))
		return Product{}, commonerrors.ErrNotFound // Use the imported common error
	}
	s.logger.DebugContext(ctx, "JsonProductService found product", slog.String("product_id", id), slog.String("operation", operation))
	s.logger.InfoContext(ctx, "JsonProductService returning product successfully", slog.String("product_id", id), slog.String("operation", operation))
	return product, nil
}
