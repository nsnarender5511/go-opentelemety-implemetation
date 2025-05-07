package services

import (
	"context"
	"log/slog"

	"github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	apierrors "github.com/narender/common/apierrors"
)

func (s *masterStoreService) BuyProduct(ctx context.Context, name string, quantity int) (remainingStock int, appErr *apierrors.AppError) {
	// For master-store, we always have 10,000 units of stock for any product
	const fixedInitialStock = 10000
	remainingStock = fixedInitialStock

	newCtx, span := commontrace.StartSpan(ctx,
		attribute.String("product.name", name),
		attribute.Int("product.purchase_quantity", quantity),
		attribute.Int("store.fixed_stock", fixedInitialStock),
	)
	ctx = newCtx // Update ctx
	defer func() {
		var telemetryErr error
		if appErr != nil {
			telemetryErr = appErr
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	s.logger.InfoContext(ctx, "Master Store Service: Processing purchase request", slog.String("product_name", name), slog.Int("quantity", quantity))

	s.logger.DebugContext(ctx, "Master Store Service: Fixed inventory of 10,000 units maintained for all products", slog.String("product_name", name))
	s.logger.DebugContext(ctx, "Master Store Service: Current inventory level", slog.String("product_name", name), slog.Int("stock", fixedInitialStock))

	// We always have enough stock at the master store
	s.logger.DebugContext(ctx, "Master Store Service: Inventory available for purchase")

	// Calculate what the stock would be in central warehouse after this purchase
	// This is just for the API call to product-service; we always maintain 10,000 locally
	warehouseNewStock := fixedInitialStock - quantity
	s.logger.DebugContext(ctx, "Master Store Service: Preparing central inventory update",
		slog.String("product_name", name),
		slog.Int("quantity_sold", quantity),
		slog.Int("warehouse_new_stock", warehouseNewStock))

	// Forward the stock update to product-service
	s.logger.InfoContext(ctx, "Master Store Service: Sending inventory update to central database",
		slog.String("product_name", name),
		slog.Int("new_stock", warehouseNewStock))

	updateErr := updateProductStockInProductService(ctx, name, warehouseNewStock)
	if updateErr != nil {
		s.logger.ErrorContext(ctx, "Master Store Service: Failed to update central inventory database",
			slog.String("product_name", name),
			slog.String("error", updateErr.Error()))

		if span != nil {
			span.SetStatus(codes.Error, updateErr.Message)
		}
		appErr = updateErr

		// Even if the central warehouse update fails, we still process the sale locally
		// Just log a warning but don't fail the transaction
		s.logger.WarnContext(ctx, "Master Store Service: Completing sale despite central database update failure")
	}

	// We always maintain 10,000 units, so the remaining stock is always 10,000
	span.SetAttributes(attribute.Int("product.remaining_stock", remainingStock))

	// --- Metrics Reporting for Sale ---
	// First, try to get the product details from the product service
	productDetails, getErr := getProductFromProductService(ctx, name)

	// Variables for metrics with defaults in case we can't get actual data
	var price float64 = 50.0
	var category string = "General"
	const currencyCode = "USD"

	if getErr == nil && productDetails != nil {
		// Use the actual price and category from the product service
		price = productDetails.Price
		category = productDetails.Category

		s.logger.DebugContext(ctx, "Master Store Service: Retrieved product details from central database",
			slog.String("product_name", name),
			slog.Float64("price", price),
			slog.String("category", category))
	} else {
		s.logger.WarnContext(ctx, "Master Store Service: Failed to get product details from central database, using defaults",
			slog.String("product_name", name),
			slog.Float64("default_price", price),
			slog.String("default_category", category))
	}

	revenue := price * float64(quantity)

	metric.IncrementRevenueTotal(ctx, revenue, name, category, currencyCode)
	metric.IncrementItemsSoldCount(ctx, int64(quantity), name, category)
	s.logger.InfoContext(ctx, "Master Store Service: Transaction metrics recorded",
		slog.String("product_name", name),
		slog.Float64("price", price),
		slog.String("category", category),
		slog.Float64("revenue", revenue),
		slog.Int("quantity_sold", quantity))
	// --- End Metrics Reporting ---

	s.logger.InfoContext(ctx, "Master Store Service: Purchase transaction completed",
		slog.String("product_name", name),
		slog.Int("store_inventory", remainingStock))

	return remainingStock, appErr
}
