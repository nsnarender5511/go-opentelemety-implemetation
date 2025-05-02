package main

import (
	"context"
	"errors"
	"fmt"

	commonErrors "github.com/narender/common/errors"
	otel "github.com/narender/common/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, productID string) (Product, error)
	GetStock(ctx context.Context, productID string) (int, error)
}

type productService struct {
	repo ProductRepository
}

func NewProductService(repo ProductRepository) ProductService {
	return &productService{
		repo: repo,
	}
}

func (s *productService) handleRepoError(ctx context.Context, span oteltrace.Span, operation string, err error) error {
	if err == nil {
		return nil
	}
	otel.GetLogger().WithContext(ctx).WithError(err).Errorf("Service: Repository error during %s", operation)
	span.RecordError(err, oteltrace.WithAttributes(attribute.String("app.operation", operation)))
	span.SetStatus(codes.Error, fmt.Sprintf("Repository error during %s", operation))

	var dbErr *commonErrors.DatabaseError
	switch {
	case errors.Is(err, commonErrors.ErrNotFound):
		return commonErrors.Wrap(err, commonErrors.TypeNotFound, fmt.Sprintf("%s failed: resource not found", operation)).
			WithUserMessage("The requested product could not be found.")
	case errors.As(err, &dbErr):
		appErr := commonErrors.Wrap(err, commonErrors.TypeDatabase, fmt.Sprintf("%s failed: database error", operation)).
			WithUserMessage("An internal error occurred while accessing data.")
		if dbErr.Operation != "" {
			appErr = appErr.WithContext("db.operation", dbErr.Operation)
		}
		return appErr
	default:
		return commonErrors.Wrap(err, commonErrors.TypeInternalServer, fmt.Sprintf("%s failed: internal error", operation)).
			WithUserMessage("An unexpected internal server error occurred.")
	}
}

const serviceInstrumentationName = "product-service/service"

func (s *productService) GetAll(ctx context.Context) ([]Product, error) {
	otel.GetLogger().WithContext(ctx).Info("Service: GetAll called")
	tracer := otel.GetTracer(serviceInstrumentationName)
	ctx, span := tracer.Start(ctx, "service.GetAll")
	defer span.End()

	products, repoErr := s.repo.GetAll(ctx)
	if err := s.handleRepoError(ctx, span, "GetAllProducts", repoErr); err != nil {
		otel.GetLogger().WithContext(ctx).Warnf("Service: GetAll finished with error: %v", err)
		return nil, err
	}

	span.SetAttributes(otel.AttrAppProductCount.Int(len(products)))
	otel.GetLogger().WithContext(ctx).Infof("Service: GetAll completed successfully, returning %d products", len(products))
	return products, nil
}

func (s *productService) GetByID(ctx context.Context, productID string) (Product, error) {
	otel.GetLogger().WithField("productID", productID).Info("Service: GetByID called")
	tracer := otel.GetTracer(serviceInstrumentationName)
	ctx, span := tracer.Start(ctx, "service.GetByID",
		oteltrace.WithAttributes(otel.AttrAppProductIDKey.String(productID)),
	)
	defer span.End()

	product, repoErr := s.repo.GetByID(ctx, productID)
	if err := s.handleRepoError(ctx, span, "GetProductByID", repoErr); err != nil {
		otel.GetLogger().WithContext(ctx).Warnf("Service: GetByID finished for productID %s with error: %v", productID, err)
		return Product{}, err
	}

	otel.GetLogger().WithContext(ctx).Infof("Service: GetByID completed successfully for productID: %s", productID)
	return product, nil
}

func (s *productService) GetStock(ctx context.Context, productID string) (int, error) {
	otel.GetLogger().WithField("productID", productID).Info("Service: GetStock called")
	tracer := otel.GetTracer(serviceInstrumentationName)
	ctx, span := tracer.Start(ctx, "service.GetStock",
		oteltrace.WithAttributes(otel.AttrAppProductIDKey.String(productID)),
	)
	defer span.End()

	product, err := s.GetByID(ctx, productID)
	if err != nil {
		return 0, err
	}

	otel.GetLogger().WithContext(ctx).Infof("Service: GetStock completed successfully for productID: %s, stock: %d", productID, product.Stock)
	span.SetAttributes(attribute.Int("app.product.stock", product.Stock))
	return product.Stock, nil
}
