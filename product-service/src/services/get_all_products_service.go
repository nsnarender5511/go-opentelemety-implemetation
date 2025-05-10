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

func (s *productService) GetAll(ctx context.Context) (products []models.Product, appErr *apierrors.AppError) {
	s.logger.DebugContext(ctx, "Initializing service layer processing for complete product catalog retrieval",
		slog.String("component", "product_service"),
		slog.String("operation", "get_all_products"))

	newCtx, span := commontrace.StartSpan(ctx, "product_service", "get_all_products")
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

	s.logger.DebugContext(ctx, "Delegating complete product catalog query to repository layer",
		slog.String("component", "product_service"),
		slog.String("operation", "repository_fetch_all"))

	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Repository layer encountered error during complete product catalog retrieval",
			slog.String("error", repoErr.Error()),
			slog.String("error_code", repoErr.Code),
			slog.String("component", "product_service"),
			slog.String("operation", "get_all_products"))
		if span != nil { // Check if span is valid before using
			span.SetStatus(codes.Error, repoErr.Message)
		}

		appErr = repoErr
		return nil, appErr
	}

	productCount := len(products)
	s.logger.InfoContext(ctx, "Repository layer successfully returned complete product catalog",
		slog.Int("product_count", productCount),
		slog.String("component", "product_service"),
		slog.String("operation", "get_all_products"),
		slog.String("status", "success"))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return nil, appErr
	}

	span.SetAttributes(attribute.Int("products.count", productCount))

	s.logger.DebugContext(ctx, "Service layer has completed processing of product catalog retrieval request",
		slog.Int("product_count", productCount),
		slog.String("component", "product_service"),
		slog.String("operation", "get_all_products"),
		slog.String("status", "completed"))

	return products, appErr
}
