package services

import (
	"context"
	"log/slog"

	apierrors "github.com/narender/common/apierrors"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
)

func (s *masterStoreService) UpdateStock(ctx context.Context, name string, newStock int) *apierrors.AppError {
	productNameAttr := attribute.String("product.name", name)
	newStockAttr := attribute.Int("product.new_stock", newStock)

	_, span := commontrace.StartSpan(ctx, productNameAttr, newStockAttr)
	defer func() {
		commontrace.EndSpan(span, nil, nil)
	}()

	s.logger.InfoContext(ctx, "Master Store Service: Processing inventory update request",
		slog.String("product_name", name),
		slog.Int("new_stock", newStock))

	// In master-store we don't directly update the database,
	// this could be replaced with an API call to product-service
	// or direct communication to handle the actual update

	s.logger.InfoContext(ctx, "Master Store Service: Inventory update applied",
		slog.String("product_name", name),
		slog.Int("new_stock", newStock))

	return nil
}
