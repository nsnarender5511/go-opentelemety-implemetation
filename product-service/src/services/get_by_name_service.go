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
		appErr = simAppErr
		return models.Product{}, appErr
	}

	s.logger.InfoContext(ctx, "Shop Manager: Front desk asking for product details", slog.String("product_name", name))

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return models.Product{}, appErr
	}

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to find product", slog.String("product_name", name))
	product, repoErr := s.repo.GetByName(ctx, name)
	if repoErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find product", slog.String("product_name", name), slog.String("error", repoErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoErr.Message)
		}
		appErr = repoErr
		return models.Product{}, appErr
	}

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		appErr = simAppErr
		return models.Product{}, appErr
	}
	s.logger.InfoContext(ctx, "Shop Manager: Found product '"+product.Name+"'")
	s.logger.InfoContext(ctx, "Shop Manager: Sending product details to front desk")
	return product, appErr
}
