package services

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/product-service/src/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	apierrors "github.com/narender/common/apierrors"
)

func (s *productService) GetAll(ctx context.Context) (products []models.Product, appErr *apierrors.AppError) {
	s.logger.DebugContext(ctx, "Shop Manager: Front desk asking for all products list")

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

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to get all products")
	products, repoErr := s.repo.GetAll(ctx)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't get all products", slog.String("error", repoErr.Error()))
		if span != nil { // Check if span is valid before using
			span.SetStatus(codes.Error, repoErr.Message)
		}
		appErr = repoErr
		return nil, appErr
	}
	productCount := len(products)
	s.logger.InfoContext(ctx, "Shop Manager: Received "+strconv.Itoa(productCount)+" products from stock room worker")

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return nil, appErr
	}
	span.SetAttributes(attribute.Int("products.count", productCount))
	s.logger.DebugContext(ctx, "Shop Manager: Sending "+strconv.Itoa(productCount)+" products to front desk")
	return products, appErr
}
