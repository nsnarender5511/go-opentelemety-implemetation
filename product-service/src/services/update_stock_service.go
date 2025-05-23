package services

import (
	"context"
	"log/slog"

	"github.com/narender/common/debugutils"
	"github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	apierrors "github.com/narender/common/apierrors"
)

func (s *productService) UpdateStock(ctx context.Context, name string, newStock int) (appErr *apierrors.AppError) {
	productNameAttr := attribute.String(metric.AttrProductName, name)
	newStockAttr := attribute.Int("product.new_stock", newStock)

	newCtx, span := commontrace.StartSpan(ctx, "product_service", "update_stock", productNameAttr, newStockAttr)
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
		// Track error metrics
		metric.IncrementErrorCount(ctx, simAppErr.Code, "update_stock", "service")
		return appErr
	}

	s.logger.InfoContext(ctx, "Processing stock update request",
		slog.String("component", "product_service"),
		slog.String("product_name", name),
		slog.Int("new_stock", newStock),
		slog.String("operation", "update_stock"))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		// Track error metrics
		metric.IncrementErrorCount(ctx, simAppErr.Code, "update_stock", "service")
		return appErr
	}

	s.logger.DebugContext(ctx, "Updating product stock in repository",
		slog.String("component", "product_service"),
		slog.String("product_name", name),
		slog.Int("new_stock", newStock),
		slog.String("operation", "repository_update_stock"))

	repoErr := s.repo.UpdateStock(ctx, name, newStock)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Failed to update product stock",
			slog.String("component", "product_service"),
			slog.String("product_name", name),
			slog.String("error", repoErr.Error()),
			slog.String("error_code", repoErr.Code),
			slog.String("operation", "update_stock"))

		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}

		appErr = repoErr
		// Track error metrics
		metric.IncrementErrorCount(ctx, repoErr.Code, "update_stock", "service")
		return appErr
	}

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		// Track error metrics
		metric.IncrementErrorCount(ctx, simAppErr.Code, "update_stock", "service")
		return appErr
	}

	s.logger.InfoContext(ctx, "Product stock updated successfully",
		slog.String("component", "product_service"),
		slog.String("product_name", name),
		slog.Int("new_stock", newStock),
		slog.String("operation", "update_stock"))

	return appErr
}
