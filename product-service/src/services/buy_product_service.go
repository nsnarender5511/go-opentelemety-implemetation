package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	apierrors "github.com/narender/common/apierrors"
)

func (s *productService) BuyProduct(ctx context.Context, name string, quantity int) (revenue float64, appErr *apierrors.AppError) {
	// Get request ID from context
	var requestID string
	if id, ok := ctx.Value("requestID").(string); ok {
		requestID = id
	}

	newCtx, span := commontrace.StartSpan(ctx,
		attribute.String(metric.AttrProductName, name),
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

	s.logger.InfoContext(ctx, "Processing purchase request",
		slog.String("product_name", name),
		slog.Int("quantity", quantity),
		slog.String("request_id", requestID),
		slog.String("event_type", "purchase_initiated"))

	s.logger.DebugContext(ctx, "Retrieving product stock information",
		slog.String("product_name", name),
		slog.String("request_id", requestID))

	product, repoGetErr := s.repo.GetByName(ctx, name)
	if repoGetErr != nil {
		s.logger.ErrorContext(ctx, "Failed to retrieve product information",
			slog.String("product_name", name),
			slog.String("error", repoGetErr.Error()),
			slog.String("error_code", repoGetErr.Code),
			slog.String("request_id", requestID),
			slog.String("event_type", "product_lookup_failed"))

		if span != nil {
			span.SetStatus(codes.Error, repoGetErr.Message)
		}

		// Track error metrics
		metric.IncrementErrorCount(ctx, repoGetErr.Code, "buy_product", "service")
		return 0, repoGetErr
	}

	s.logger.DebugContext(ctx, "Product stock verification",
		slog.String("product_name", product.Name),
		slog.Int("stock", product.Stock),
		slog.String("request_id", requestID))

	if product.Stock < quantity {
		errMsg := fmt.Sprintf("Insufficient stock for product '%s'. Available: %d, Requested: %d", name, product.Stock, quantity)

		s.logger.WarnContext(ctx, "Purchase rejected: insufficient stock",
			slog.String("product_name", name),
			slog.Int("requested", quantity),
			slog.Int("available", product.Stock),
			slog.String("error_code", apierrors.ErrCodeInsufficientStock),
			slog.String("request_id", requestID),
			slog.String("event_type", "purchase_rejected"))

		if span != nil {
			span.SetStatus(codes.Error, "Insufficient stock")
		}

		// Create business error with request ID
		appErr = apierrors.NewBusinessError(
			apierrors.ErrCodeInsufficientStock,
			errMsg,
			nil,
		).WithRequestID(requestID)

		// Track error metrics
		metric.IncrementErrorCount(ctx, apierrors.ErrCodeInsufficientStock, "buy_product", "service")
		return 0, appErr
	}

	s.logger.DebugContext(ctx, "Stock verification completed: sufficient stock available",
		slog.String("product_name", name),
		slog.Int("available", product.Stock),
		slog.Int("requested", quantity),
		slog.String("event_type", "stock_verified"),
		slog.String("request_id", requestID))

	newStock := product.Stock - quantity
	s.logger.DebugContext(ctx, "Calculating inventory update",
		slog.String("product_name", product.Name),
		slog.Int("new_stock", newStock),
		slog.String("request_id", requestID))

	s.logger.DebugContext(ctx, "Updating product inventory",
		slog.String("product_name", product.Name),
		slog.Int("new_stock", newStock),
		slog.String("request_id", requestID))

	repoUpdateErr := s.repo.UpdateStock(ctx, name, newStock)
	if repoUpdateErr != nil {
		s.logger.ErrorContext(ctx, "Failed to update inventory during purchase",
			slog.String("product_name", name),
			slog.String("error", repoUpdateErr.Error()),
			slog.String("error_code", repoUpdateErr.Code),
			slog.String("request_id", requestID),
			slog.String("event_type", "inventory_update_failed"))

		if span != nil {
			span.SetStatus(codes.Error, repoUpdateErr.Message)
		}

		// Ensure error has request ID
		if repoUpdateErr.RequestID == "" {
			repoUpdateErr.RequestID = requestID
		}

		appErr = repoUpdateErr
		// Track error metrics
		metric.IncrementErrorCount(ctx, repoUpdateErr.Code, "buy_product", "service")
		return 0, appErr // Return zero revenue if update fails
	}

	// Calculate revenue for the purchase
	revenue = product.Price * float64(quantity)
	span.SetAttributes(attribute.Float64("product.revenue", revenue))
	span.SetAttributes(attribute.Int("product.remaining_stock", newStock))

	// --- Metrics Reporting for Sale ---
	metric.IncrementRevenueTotal(ctx, revenue, product.Name, product.Category)
	metric.IncrementItemsSoldCount(ctx, int64(quantity), product.Name, product.Category)
	s.logger.InfoContext(ctx, "Sales metrics recorded",
		slog.String("product_name", product.Name),
		slog.Float64("revenue", revenue),
		slog.Int("quantity_sold", quantity),
		slog.String("request_id", requestID))
	// --- End Metrics Reporting ---

	s.logger.InfoContext(ctx, "Purchase completed successfully",
		slog.String("product_name", name),
		slog.Float64("revenue", revenue),
		slog.Int("remaining_stock", newStock),
		slog.String("request_id", requestID),
		slog.String("event_type", "purchase_completed"))

	return revenue, appErr
}
