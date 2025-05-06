package services

import (
	"context"
	"log/slog"

	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	apierrors "github.com/narender/common/apierrors"
)

func (s *productService) UpdateStock(ctx context.Context, name string, newStock int) (appErr *apierrors.AppError) {
	productNameAttr := attribute.String("product.name", name)
	newStockAttr := attribute.Int("product.new_stock", newStock)

	newCtx, span := commontrace.StartSpan(ctx, productNameAttr, newStockAttr)
	ctx = newCtx // Update ctx
	defer func() {
		var telemetryErr error
		if appErr != nil {
			telemetryErr = appErr
		}
		commontrace.EndSpan(span, &telemetryErr, nil) // Pass address
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return appErr
	}

	s.logger.InfoContext(ctx, "Shop Manager: Front desk requesting stock update", slog.String("product_name", name), slog.Int("new_stock", newStock))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return appErr
	}

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to update inventory record")

	repoErr := s.repo.UpdateStock(ctx, name, newStock)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't update stock", slog.String("product_name", name), slog.String("error", repoErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}
		appErr = repoErr
		return appErr
	}

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return appErr
	}

	s.logger.InfoContext(ctx, "Shop Manager: Stock updated successfully", slog.String("product_name", name), slog.Int("new_stock", newStock))
	s.logger.InfoContext(ctx, "Shop Manager: Confirming stock update to front desk")
	return appErr
}
