package services

import (
	"context"
	"log/slog"

	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/product-service/src/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	apierrors "github.com/narender/common/apierrors"
)

func (s *productService) GetByCategory(ctx context.Context, category string) (products []models.Product, appErr *apierrors.AppError) {
	s.logger.InfoContext(ctx, "Initializing service layer processing for category-based product filtering",
		slog.String("category", category),
		slog.String("component", "product_service"),
		slog.String("operation", "get_products_by_category"),
		slog.String("event_type", "category_products_processing"))

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

	s.logger.DebugContext(ctx, "Delegating category-based product query to repository layer",
		slog.String("category", category),
		slog.String("component", "product_service"),
		slog.String("operation", "repository_fetch_by_category"))

	products, repoErr := s.repo.GetByCategory(ctx, category)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Repository layer encountered error during category-based product retrieval",
			slog.String("category", category),
			slog.String("error", repoErr.Error()),
			slog.String("error_code", repoErr.Code),
			slog.String("component", "product_service"),
			slog.String("operation", "get_products_by_category"),
			slog.String("event_type", "category_products_retrieval_failed"))

		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}

		appErr = repoErr
		return nil, appErr
	}

	productCount := len(products)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))

	s.logger.InfoContext(ctx, "Service layer successfully processed category-based product retrieval",
		slog.String("category", category),
		slog.Int("product_count", productCount),
		slog.String("component", "product_service"),
		slog.String("operation", "get_products_by_category"),
		slog.String("status", "success"),
		slog.String("event_type", "category_products_retrieved"))

	return products, appErr
}
