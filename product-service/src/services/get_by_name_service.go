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

func (s *productService) GetByName(ctx context.Context, name string) (product models.Product, appErr *apierrors.AppError) {
	// Get request ID from context
	var requestID string
	if id, ok := ctx.Value("requestID").(string); ok {
		requestID = id
	}

	productNameAttr := attribute.String("product.name", name)

	newCtx, span := commontrace.StartSpan(ctx, productNameAttr)
	ctx = newCtx // Update ctx
	defer func() {
		var telemetryErr error
		if appErr != nil {
			telemetryErr = appErr
		}
		commontrace.EndSpan(span, &telemetryErr, nil) // Pass address of telemetryErr
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		appErr = simAppErr
		return models.Product{}, appErr
	}

	s.logger.InfoContext(ctx, "Processing product details request",
		slog.String("product_name", name),
		slog.String("request_id", requestID),
		slog.String("event_type", "product_details_processing"))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		appErr = simAppErr
		return models.Product{}, appErr
	}

	s.logger.DebugContext(ctx, "Retrieving product from repository",
		slog.String("product_name", name),
		slog.String("request_id", requestID))

	product, repoErr := s.repo.GetByName(ctx, name)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Failed to retrieve product details",
			slog.String("product_name", name),
			slog.String("error", repoErr.Error()),
			slog.String("error_code", repoErr.Code),
			slog.String("request_id", requestID),
			slog.String("event_type", "product_lookup_failed"))

		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}

		// Ensure request ID is set
		if repoErr.RequestID == "" {
			repoErr.RequestID = requestID
		}

		appErr = repoErr
		return models.Product{}, appErr
	}

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		appErr = simAppErr
		return models.Product{}, appErr
	}

	s.logger.InfoContext(ctx, "Product details retrieved successfully",
		slog.String("product_name", product.Name),
		slog.String("request_id", requestID),
		slog.String("event_type", "product_details_retrieved"))

	return product, appErr
}
