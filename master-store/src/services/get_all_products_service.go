package services

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/master-store/src/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	apierrors "github.com/narender/common/apierrors"
)

func (s *masterStoreService) GetAll(ctx context.Context) (products []models.Product, appErr *apierrors.AppError) {
	s.logger.DebugContext(ctx, "Master Store Service: Retrieving product catalog")

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
		appErr = simAppErr
		return nil, appErr
	}

	s.logger.DebugContext(ctx, "Master Store Service: Requesting product data from repository")
	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Master Store Service: Failed to retrieve products from repository", slog.String("error", repoErr.Error()))
		if span != nil { // Check if span is valid before using
			span.SetStatus(codes.Error, repoErr.Message)
		}
		appErr = repoErr
		return nil, appErr
	}
	productCount := len(products)
	s.logger.InfoContext(ctx, "Master Store Service: Retrieved "+strconv.Itoa(productCount)+" products from repository")

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return nil, appErr
	}
	span.SetAttributes(attribute.Int("products.count", productCount))
	s.logger.DebugContext(ctx, "Master Store Service: Returning "+strconv.Itoa(productCount)+" products to handler")
	return products, appErr
}
