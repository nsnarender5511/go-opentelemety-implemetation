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
	// Get request ID from context
	var requestID string
	if id, ok := ctx.Value("requestID").(string); ok {
		requestID = id
	}

	s.logger.InfoContext(ctx, "Processing category products request",
		slog.String("category", category),
		slog.String("request_id", requestID),
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
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		appErr = simAppErr
		return nil, appErr
	}

	s.logger.DebugContext(ctx, "Retrieving products by category from repository",
		slog.String("category", category),
		slog.String("request_id", requestID))

	products, repoErr := s.repo.GetByCategory(ctx, category)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Failed to retrieve products by category",
			slog.String("category", category),
			slog.String("error", repoErr.Error()),
			slog.String("error_code", repoErr.Code),
			slog.String("request_id", requestID),
			slog.String("event_type", "category_products_retrieval_failed"))

		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}

		// Ensure request ID is set
		if repoErr.RequestID == "" {
			repoErr.RequestID = requestID
		}

		appErr = repoErr
		return nil, appErr
	}

	productCount := len(products)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))

	s.logger.InfoContext(ctx, "Category products retrieved successfully",
		slog.String("category", category),
		slog.Int("product_count", productCount),
		slog.String("request_id", requestID),
		slog.String("event_type", "category_products_retrieved"))

	return products, appErr
}
