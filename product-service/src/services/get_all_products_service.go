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

	s.logger.DebugContext(ctx, "Initializing service layer processing for complete product catalog retrieval",
		slog.String("request_id", requestID),
		slog.String("component", "product_service"),
		slog.String("operation", "get_all_products"),
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

	s.logger.DebugContext(ctx, "Delegating complete product catalog query to repository layer",
		slog.String("component", "product_service"),
		slog.String("operation", "repository_fetch_all"),
		slog.String("request_id", requestID))

	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Repository layer encountered error during complete product catalog retrieval",
			slog.String("error", repoErr.Error()),
			slog.String("error_code", repoErr.Code),
			slog.String("component", "product_service"),
			slog.String("operation", "get_all_products"),
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
	s.logger.InfoContext(ctx, "Repository layer successfully returned complete product catalog",
		slog.String("request_id", requestID),
		slog.Int("product_count", productCount),
		slog.String("component", "product_service"),
		slog.String("operation", "get_all_products"),
		slog.String("event_type", "products_retrieved"),
		slog.String("status", "success"))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		appErr = simAppErr
		return nil, appErr
	}

	span.SetAttributes(attribute.Int("products.count", productCount))

	s.logger.DebugContext(ctx, "Service layer has completed processing of product catalog retrieval request",
		slog.String("request_id", requestID),
		slog.Int("product_count", productCount),
		slog.String("component", "product_service"),
		slog.String("operation", "get_all_products"),
		slog.String("status", "completed"))

	return products, appErr
}
