package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	commonErrors "github.com/narender/common/errors"
	"github.com/narender/common/otel"
	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByID(ctx context.Context, productID string) (Product, error)
	GetStock(ctx context.Context, productID string) (int, error)
}

type productService struct {
	repo   ProductRepository
	logger *logrus.Logger
	tracer oteltrace.Tracer
}

func NewProductService(repo ProductRepository, logger *logrus.Logger, tracer oteltrace.Tracer) ProductService {
	return &productService{
		repo:   repo,
		logger: logger,
		tracer: tracer,
	}
}

func (s *productService) handleRepoError(ctx context.Context, span oteltrace.Span, operation string, err error) error {
	if err == nil {
		return nil
	}

	s.logger.WithContext(ctx).WithError(err).Errorf("Service: Repository error during %s", operation)

	otel.RecordSpanError(span, err, attribute.String("app.operation", operation))

	var dbErr *commonErrors.DatabaseError
	switch {
	case errors.Is(err, commonErrors.ErrNotFound):
		return commonErrors.Wrap(err, http.StatusNotFound, fmt.Sprintf("%s failed: resource not found", operation)).
			WithUserMessage("The requested product could not be found.")

	case errors.As(err, &dbErr):
		appErr := commonErrors.Wrap(err, http.StatusInternalServerError, fmt.Sprintf("%s failed: database error", operation)).
			WithUserMessage("An internal error occurred while accessing data.")
		if dbErr.Operation != "" {
			appErr = appErr.WithContext("db.operation", dbErr.Operation)
		}
		return appErr

	default:
		return commonErrors.Wrap(err, http.StatusInternalServerError, fmt.Sprintf("%s failed: internal error", operation)).
			WithUserMessage("An unexpected internal server error occurred.")
	}
}

func (s *productService) GetAll(ctx context.Context) ([]Product, error) {
	ctx, span := s.tracer.Start(ctx, "service.GetAll")
	defer span.End()
	s.logger.Info("Service: GetAll called")

	products, repoErr := s.repo.GetAll(ctx)
	if err := s.handleRepoError(ctx, span, "GetAllProducts", repoErr); err != nil {
		s.logger.Warnf("Service: GetAll finished with error: %v", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("db.result.count", len(products)))
	s.logger.Infof("Service: GetAll completed successfully, returning %d products", len(products))
	return products, nil
}

func (s *productService) GetByID(ctx context.Context, productID string) (Product, error) {
	ctx, span := s.tracer.Start(ctx, "service.GetByID")
	defer span.End()
	s.logger.Infof("Service: GetByID called for productID: %s", productID)

	span.SetAttributes(attribute.String("app.product.id", productID))

	product, repoErr := s.repo.GetByID(ctx, productID)
	if err := s.handleRepoError(ctx, span, "GetProductByID", repoErr); err != nil {
		s.logger.Warnf("Service: GetByID finished for productID %s with error: %v", productID, err)
		return Product{}, err
	}

	s.logger.Infof("Service: GetByID completed successfully for productID: %s", productID)
	return product, nil
}

func (s *productService) GetStock(ctx context.Context, productID string) (int, error) {
	ctx, span := s.tracer.Start(ctx, "service.GetStock")
	defer span.End()
	s.logger.Infof("Service: GetStock called for productID: %s", productID)

	span.SetAttributes(attribute.String("app.product.id", productID))

	product, repoErr := s.repo.GetByID(ctx, productID)
	if err := s.handleRepoError(ctx, span, "GetStock (via GetByID)", repoErr); err != nil {
		s.logger.Warnf("Service: GetStock finished for productID %s with error: %v", productID, err)
		return 0, err
	}

	s.logger.Infof("Service: GetStock completed successfully for productID: %s, stock: %d", productID, product.Stock)
	return product.Stock, nil
}
