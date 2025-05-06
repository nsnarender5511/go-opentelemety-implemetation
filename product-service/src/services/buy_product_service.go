package services

import (
	"context"
	"fmt"
	"log/slog"

	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	apierrors "github.com/narender/common/apierrors"
)

func (s *productService) BuyProduct(ctx context.Context, name string, quantity int) (remainingStock int, appErr *apierrors.AppError) {
	newCtx, span := commontrace.StartSpan(ctx,
		attribute.String("product.name", name),
		attribute.Int("product.purchase_quantity", quantity),
	)
	ctx = newCtx // Update ctx
	defer func() {
		var telemetryErr error
		if appErr != nil {
			telemetryErr = appErr
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	s.logger.InfoContext(ctx, "Shop Manager: Processing purchase request", slog.String("product_name", name), slog.Int("quantity", quantity))

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker for current stock", slog.String("product_name", name))
	product, repoGetErr := s.repo.GetByName(ctx, name)
	if repoGetErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker couldn't find product for purchase check", slog.String("product_name", name), slog.String("error", repoGetErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoGetErr.Message)
		}
		appErr = repoGetErr
		return 0, appErr
	}
	s.logger.DebugContext(ctx, "Shop Manager: Current stock check", slog.String("product_name", product.Name), slog.Int("stock", product.Stock))

	if product.Stock < quantity {
		errMsg := fmt.Sprintf("Insufficient stock for product '%s'. Available: %d, Requested: %d", name, product.Stock, quantity)
		s.logger.WarnContext(ctx, "Shop Manager: Purchase blocked - insufficient stock",
			slog.String("product_name", name),
			slog.Int("requested", quantity),
			slog.Int("available", product.Stock),
		)
		if span != nil {
			span.SetStatus(codes.Error, "Insufficient stock") // Specific message for span
		}
		appErr = apierrors.NewAppError(apierrors.ErrCodeInsufficientStock, errMsg, nil)
		return product.Stock, appErr // Return current stock with the error
	}
	s.logger.DebugContext(ctx, "Shop Manager: Stock available for purchase")

	newStock := product.Stock - quantity
	s.logger.DebugContext(ctx, "Shop Manager: Calculated new stock", slog.String("product_name", product.Name), slog.Int("new_stock", newStock))

	s.logger.DebugContext(ctx, "Shop Manager: Asking stock room worker to update inventory", slog.String("product_name", product.Name), slog.Int("new_stock", newStock))
	repoUpdateErr := s.repo.UpdateStock(ctx, name, newStock)
	if repoUpdateErr != nil {
		s.logger.ErrorContext(ctx, "Shop Manager: Stock room worker failed to update stock during purchase", slog.String("product_name", name), slog.String("error", repoUpdateErr.Error()))
		if span != nil {
			span.SetStatus(codes.Error, repoUpdateErr.Message)
		}
		appErr = repoUpdateErr
		return product.Stock, appErr // Return pre-update stock if update fails
	}

	remainingStock = newStock
	span.SetAttributes(attribute.Int("product.remaining_stock", remainingStock))
	s.logger.InfoContext(ctx, "Shop Manager: Purchase processed successfully", slog.String("product_name", name), slog.Int("remaining_stock", remainingStock))

	return remainingStock, appErr
}
