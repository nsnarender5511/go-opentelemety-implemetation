package repositories

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/product-service/src/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	apierrors "github.com/narender/common/apierrors"
)

func (r *productRepository) GetByName(ctx context.Context, name string) (product models.Product, appErr *apierrors.AppError) {
	// Get request ID from context
	var requestID string
	if id, ok := ctx.Value("requestID").(string); ok {
		requestID = id
	}

	productNameAttr := attribute.String("product.name", name)
	newCtx, span := commontrace.StartSpan(ctx, productNameAttr)
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
		return models.Product{}, appErr
	}

	r.logger.InfoContext(ctx, "Looking up product by name",
		slog.String("product_name", name),
		slog.String("request_id", requestID),
		slog.String("operation", "get_by_name"))

	r.logger.DebugContext(ctx, "Accessing product database",
		slog.String("product_name", name),
		slog.String("request_id", requestID))

	var productsMap map[string]models.Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		errMsg := "Failed to read product data from database"
		r.logger.ErrorContext(ctx, "Database access error during product lookup",
			slog.String("error", err.Error()),
			slog.String("product_name", name),
			slog.String("error_code", apierrors.ErrCodeDatabaseAccess),
			slog.String("request_id", requestID),
			slog.String("operation", "get_by_name"))

		if span != nil {
			span.SetStatus(codes.Error, errMsg)
		}

		appErr = apierrors.NewApplicationError(apierrors.ErrCodeDatabaseAccess, errMsg, err).WithRequestID(requestID)
		return models.Product{}, appErr
	}

	r.logger.DebugContext(ctx, "Searching for product in database",
		slog.String("product_name", name),
		slog.String("request_id", requestID))

	product, exists := productsMap[name]
	if !exists {
		errMsg := fmt.Sprintf("Product with name '%s' not found", name)

		r.logger.WarnContext(ctx, "Product not found in database",
			slog.String("product_name", name),
			slog.String("error_code", apierrors.ErrCodeProductNotFound),
			slog.String("request_id", requestID),
			slog.String("operation", "get_by_name"))

		if span != nil {
			span.SetStatus(codes.Error, errMsg)
		}

		appErr = apierrors.NewBusinessError(
			apierrors.ErrCodeProductNotFound,
			errMsg,
			nil,
		).WithRequestID(requestID).WithContext("operation", "get_by_name")

		return models.Product{}, appErr
	}

	span.SetAttributes(attribute.String("product.category_found", product.Category))

	r.logger.InfoContext(ctx, "Product found successfully",
		slog.String("product_name", product.Name),
		slog.String("category", product.Category),
		slog.String("request_id", requestID))

	r.logger.DebugContext(ctx, "Retrieved product details",
		slog.String("product_name", product.Name),
		slog.Int("stock", product.Stock),
		slog.Float64("price", product.Price),
		slog.String("request_id", requestID))

	return product, appErr // appErr is nil here if successful
}
