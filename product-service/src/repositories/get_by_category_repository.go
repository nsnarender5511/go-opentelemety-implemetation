package repositories

import (
	"context"
	"log/slog"
	"os"

	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/product-service/src/models" // Corrected path
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/trace"

	apierrors "github.com/narender/common/apierrors"
)

func (r *productRepository) GetByCategory(ctx context.Context, category string) (filteredProducts []models.Product, appErr *apierrors.AppError) {
	categoryAttr := attribute.String("product.category", category)
	newCtx, span := commontrace.StartSpan(ctx, "product_repository", "get_by_category", categoryAttr)
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
		return nil, appErr
	}

	r.logger.InfoContext(ctx, "Initiating repository operation for category-filtered product retrieval",
		slog.String("category", category),
		slog.String("component", "product_repository"),
		slog.String("operation", "get_by_category"))

	r.logger.DebugContext(ctx, "Executing database read operation to access product data",
		slog.String("category", category),
		slog.String("component", "product_repository"),
		slog.String("operation", "read_from_database"))

	var productsMap map[string]models.Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		if os.IsNotExist(err) {
			r.logger.WarnContext(ctx, "No products found in database",
				slog.String("category", category),
				slog.String("error_code", apierrors.ErrCodeDatabaseAccess),
				slog.String("operation", "get_by_category"),
				slog.String("error", err.Error()))

			span.AddEvent("FileDatabase.Read indicated file not found, returning empty.", trace.WithAttributes(attribute.String("error.message", err.Error())))
			return []models.Product{}, nil
		} else {
			errMsg := "Failed to read product data from database"
			r.logger.ErrorContext(ctx, "Database access error",
				slog.String("error", err.Error()),
				slog.String("error_code", apierrors.ErrCodeDatabaseAccess),
				slog.String("operation", "get_by_category"))

			if span != nil {
				span.SetStatus(codes.Error, errMsg)
			}

			appErr = apierrors.NewApplicationError(
				apierrors.ErrCodeDatabaseAccess,
				errMsg,
				err)

			return nil, appErr
		}
	}

	r.logger.DebugContext(ctx, "Applying category filter to product inventory data",
		slog.String("category", category),
		slog.String("component", "product_repository"),
		slog.Int("total_products", len(productsMap)),
		slog.String("operation", "category_match"))

	filteredProducts = make([]models.Product, 0)
	for _, p := range productsMap {
		if p.Category == category {
			filteredProducts = append(filteredProducts, p)
			r.logger.DebugContext(ctx, "Product entity matches requested category criteria",
				slog.String("product_name", p.Name),
				slog.Int("stock", p.Stock),
				slog.String("product_category", p.Category),
				slog.Float64("product_price", p.Price),
				slog.String("component", "product_repository"),
				slog.String("operation", "category_filtering"))
		}
	}

	productCount := len(filteredProducts)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))

	r.logger.InfoContext(ctx, "Repository layer successfully completed category-filtered product retrieval",
		slog.String("category", category),
		slog.Int("product_count", productCount),
		slog.String("component", "product_repository"),
		slog.String("operation", "get_by_category"),
		slog.String("status", "success"))

	return filteredProducts, appErr // appErr is nil here if successful
}
