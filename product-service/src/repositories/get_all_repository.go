package repositories

import (
	"context"
	"log/slog"
	"os"

	"github.com/narender/common/debugutils"
	"github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/product-service/src/models" // Corrected path
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/trace"

	apierrors "github.com/narender/common/apierrors"
)

func (r *productRepository) GetAll(ctx context.Context) (productsSlice []models.Product, appErr *apierrors.AppError) {
	// Get request ID from context
	var requestID string
	if id, ok := ctx.Value("requestID").(string); ok {
		requestID = id
	}

	newCtx, span := commontrace.StartSpan(ctx, attribute.String("repository.operation", "GetAll"))
	ctx = newCtx // Update ctx if StartSpan modifies it
	defer func() {
		var telemetryErr error
		if appErr != nil {
			telemetryErr = appErr
		}
		commontrace.EndSpan(span, &telemetryErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		// Ensure request ID is set
		if simAppErr.RequestID == "" {
			simAppErr.RequestID = requestID
		}
		appErr = simAppErr
		return nil, appErr
	}

	r.logger.InfoContext(ctx, "Retrieving all products from database",
		slog.String("request_id", requestID),
		slog.String("operation", "get_all_products"))

	r.logger.DebugContext(ctx, "Accessing product database",
		slog.String("request_id", requestID))

	var productsMap map[string]models.Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		if os.IsNotExist(err) {
			r.logger.WarnContext(ctx, "No products found in database",
				slog.String("error_code", apierrors.ErrCodeDatabaseAccess),
				slog.String("request_id", requestID),
				slog.String("operation", "get_all_products"),
				slog.String("error", err.Error()))

			span.AddEvent("FileDatabase.Read indicated file not found, returning empty.", trace.WithAttributes(attribute.String("error.message", err.Error())))
			return []models.Product{}, nil
		} else {
			errMsg := "Failed to read product data from database"
			r.logger.ErrorContext(ctx, "Database access error",
				slog.String("error", err.Error()),
				slog.String("error_code", apierrors.ErrCodeDatabaseAccess),
				slog.String("request_id", requestID),
				slog.String("operation", "get_all_products"))

			if span != nil {
				span.SetStatus(codes.Error, errMsg)
			}

			appErr = apierrors.NewApplicationError(
				apierrors.ErrCodeDatabaseAccess,
				errMsg,
				err,
			).WithRequestID(requestID)

			return nil, appErr
		}
	}

	r.logger.DebugContext(ctx, "Processing product inventory data",
		slog.String("request_id", requestID),
		slog.Int("product_count", len(productsMap)))

	productsSlice = make([]models.Product, 0, len(productsMap))
	for _, p := range productsMap {
		productsSlice = append(productsSlice, p)
		r.logger.DebugContext(ctx, "Processing product data",
			slog.String("product_name", p.Name),
			slog.Int("stock", p.Stock),
			slog.String("request_id", requestID))
	}

	// Update product stock levels for telemetry
	for _, p := range productsSlice {
		metric.UpdateProductStockLevels(ctx, p.Name, p.Category, int64(p.Stock))
	}

	productCount := len(productsSlice)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))

	r.logger.InfoContext(ctx, "Products retrieval completed",
		slog.String("request_id", requestID),
		slog.Int("product_count", productCount),
		slog.String("operation", "get_all_products"),
		slog.String("event_type", "products_retrieved"))

	return productsSlice, appErr // appErr is nil here if successful
}
