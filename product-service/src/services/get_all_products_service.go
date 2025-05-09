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
	// Get request ID from context
	var requestID string
	if id, ok := ctx.Value("requestID").(string); ok {
		requestID = id
	}

	s.logger.DebugContext(ctx, "Processing product list request",
		slog.String("request_id", requestID),
		slog.String("event_type", "product_list_processing"))

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
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		appErr = simAppErr
		return nil, appErr
	}

	s.logger.DebugContext(ctx, "Retrieving all products from repository",
		slog.String("request_id", requestID))

	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Failed to retrieve products list",
			slog.String("error", repoErr.Error()),
			slog.String("error_code", repoErr.Code),
			slog.String("request_id", requestID),
			slog.String("event_type", "product_list_retrieval_failed"))

		if span != nil { // Check if span is valid before using
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
	s.logger.InfoContext(ctx, "Products list retrieved from repository",
		slog.String("request_id", requestID),
		slog.Int("product_count", productCount),
		slog.String("event_type", "products_retrieved"))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		appErr = simAppErr
		return nil, appErr
	}

	span.SetAttributes(attribute.Int("products.count", productCount))

	s.logger.DebugContext(ctx, "Processing products list completed",
		slog.String("request_id", requestID),
		slog.Int("product_count", productCount))

	return products, appErr
}
