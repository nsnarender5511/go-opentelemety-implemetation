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

func (s *productService) GetByCategory(ctx context.Context, category string) (products []models.Product, appErr *apierrors.AppError) {
	s.logger.InfoContext(ctx, "Shop Manager: Front desk asking for products by category", slog.String("category", category))

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

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to find products", slog.String("category", category))
	products, repoErr := s.repo.GetByCategory(ctx, category)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find products", slog.String("category", category), slog.String("error", repoErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}
		appErr = repoErr
		return nil, appErr
	}

	productCount := len(products)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))
	s.logger.InfoContext(ctx, "Shop Manager: Found "+strconv.Itoa(productCount)+" products in category: "+category)
	s.logger.InfoContext(ctx, "Shop Manager: Sending category products to front desk")
	return products, appErr
}
