package repositories

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/narender/common/debugutils"
	"github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/product-service/src/models" // Corrected path
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/trace"

	apierrors "github.com/narender/common/apierrors"
)

func (r *productRepository) UpdateStock(ctx context.Context, name string, newStock int) (appErr *apierrors.AppError) {
	// Get request ID from context
	var requestID string
	if id, ok := ctx.Value("requestID").(string); ok {
		requestID = id
	}

	productNameAttr := attribute.String(metric.AttrProductName, name)
	newStockAttr := attribute.Int("product.new_stock", newStock)
	attrs := []attribute.KeyValue{productNameAttr, newStockAttr}

	ctx, span := commontrace.StartSpan(ctx, attrs...)
	var opErr error
	defer func() {
		if appErr != nil && opErr == nil {
			opErr = appErr
		}
		commontrace.EndSpan(span, &opErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		return simAppErr
	}

	r.logger.InfoContext(ctx, "Updating product stock",
		slog.String("product_name", name),
		slog.Int("new_stock", newStock),
		slog.String("request_id", requestID),
		slog.String("operation", "update_stock"))

	r.logger.DebugContext(ctx, "Accessing product database",
		slog.String("request_id", requestID),
		slog.String("product_name", name))

	var productsMap map[string]models.Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		errMsg := "Failed to read product data from database"
		r.logger.ErrorContext(ctx, "Database access error",
			slog.String("error", err.Error()),
			slog.String("error_code", apierrors.ErrCodeDatabaseAccess),
			slog.String("request_id", requestID),
			slog.String("operation", "update_stock"))
			
		span.SetStatus(codes.Error, errMsg)
		
		appErr = apierrors.NewApplicationError(
			apierrors.ErrCodeDatabaseAccess,
			errMsg,
			err).WithRequestID(requestID)
			
		// Track error metrics
		metric.IncrementErrorCount(ctx, apierrors.ErrCodeDatabaseAccess, "update_stock", "repository")
		return appErr
	}

	r.logger.DebugContext(ctx, "Verifying product exists",
		slog.String("product_name", name),
		slog.String("request_id", requestID))

	product, ok := productsMap[name]
	if !ok {
		errMsg := fmt.Sprintf("Product with name '%s' not found for stock update", name)
		r.logger.WarnContext(ctx, "Product not found",
			slog.String("product_name", name),
			slog.String("error_code", apierrors.ErrCodeProductNotFound),
			slog.String("request_id", requestID),
			slog.String("operation", "update_stock"))
			
		span.AddEvent("product_not_found_in_map_for_update", trace.WithAttributes(attrs...))
		span.SetStatus(codes.Error, errMsg)
		
		appErr = apierrors.NewBusinessError(
			apierrors.ErrCodeProductNotFound,
			errMsg,
			nil).WithRequestID(requestID)
			
		// Track error metrics
		metric.IncrementErrorCount(ctx, apierrors.ErrCodeProductNotFound, "update_stock", "repository")
		return appErr
	}

	oldStock := product.Stock
	product.Stock = newStock
	productsMap[name] = product

	span.SetAttributes(attribute.Int("product.old_stock", oldStock))

	stockDiff := newStock - oldStock
	stockChangeType := "unchanged"
	if stockDiff > 0 {
		stockChangeType = "increased"
	} else if stockDiff < 0 {
		stockChangeType = "decreased"
	}
	
	r.logger.InfoContext(ctx, "Updating product stock level",
		slog.String("product_name", product.Name),
		slog.Int("old_stock", oldStock),
		slog.Int("new_stock", newStock),
		slog.Int("stock_change", stockDiff),
		slog.String("stock_change_type", stockChangeType),
		slog.String("request_id", requestID))

	if writeErr := r.database.Write(ctx, productsMap); writeErr != nil {
		errMsg := "Failed to write updated product data"
		r.logger.ErrorContext(ctx, "Database write error",
			slog.String("error", writeErr.Error()),
			slog.String("error_code", apierrors.ErrCodeDatabaseAccess),
			slog.String("product_name", name),
			slog.String("request_id", requestID),
			slog.String("operation", "update_stock"))
			
		span.SetStatus(codes.Error, errMsg)
		
		appErr = apierrors.NewApplicationError(
			apierrors.ErrCodeDatabaseAccess,
			errMsg,
			writeErr).WithRequestID(requestID)
			
		// Track error metrics
		metric.IncrementErrorCount(ctx, apierrors.ErrCodeDatabaseAccess, "update_stock", "repository")
		return appErr
	}

	// Update product stock level for telemetry
	metric.UpdateProductStockLevels(ctx, product.Name, product.Category, int64(newStock))

	r.logger.InfoContext(ctx, "Product stock update completed",
		slog.String("product_name", product.Name),
		slog.Int("old_stock", oldStock),
		slog.Int("new_stock", newStock),
		slog.String("request_id", requestID),
		slog.String("operation", "update_stock"),
		slog.String("event_type", "stock_update_completed"))

	return nil
}
