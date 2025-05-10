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
	// Remove request ID extraction from context

	productNameAttr := attribute.String("product.name", name)
	newCtx, span := commontrace.StartSpan(ctx, "product_repository", "get_by_name", productNameAttr)
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
		slog.String("component", "product_repository"),
		slog.String("product_name", name),
		slog.String("operation", "get_by_name"))

	r.logger.DebugContext(ctx, "Accessing product database",
		slog.String("component", "product_repository"),
		slog.String("operation", "access_database"),
		slog.String("product_name", name))

	var productsMap map[string]models.Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		errMsg := "Failed to read product data from database"
		r.logger.ErrorContext(ctx, "Database access error during product lookup",
			slog.String("component", "product_repository"),
			slog.String("operation", "database_access_error"),
			slog.String("error", err.Error()),
			slog.String("product_name", name),
			slog.String("error_code", apierrors.ErrCodeDatabaseAccess),
			slog.String("operation", "get_by_name"))

		if span != nil {
			span.SetStatus(codes.Error, errMsg)
		}

		appErr = apierrors.NewApplicationError(apierrors.ErrCodeDatabaseAccess, errMsg, err)
		return models.Product{}, appErr
	}

	r.logger.DebugContext(ctx, "Searching for product in database",
		slog.String("component", "product_repository"),
		slog.String("operation", "search_for_product"),
		slog.String("product_name", name))

	product, exists := productsMap[name]
	if !exists {
		errMsg := fmt.Sprintf("Product with name '%s' not found", name)

		r.logger.WarnContext(ctx, "Product not found in database",
			slog.String("component", "product_repository"),
			slog.String("operation", "product_not_found"),
			slog.String("product_name", name),
			slog.String("error_code", apierrors.ErrCodeProductNotFound),
			slog.String("operation", "get_by_name"))

		if span != nil {
			span.SetStatus(codes.Error, errMsg)
		}

		appErr = apierrors.NewBusinessError(
			apierrors.ErrCodeProductNotFound,
			errMsg,
			nil,
		).WithContext("operation", "get_by_name")

		return models.Product{}, appErr
	}

	span.SetAttributes(attribute.String("product.category_found", product.Category))

	r.logger.InfoContext(ctx, "Product found successfully",
		slog.String("component", "product_repository"),
		slog.String("operation", "product_found"),
		slog.String("product_name", product.Name),
		slog.String("category", product.Category))

	r.logger.DebugContext(ctx, "Retrieved product details",
		slog.String("component", "product_repository"),
		slog.String("operation", "retrieve_product_details"),
		slog.String("product_name", product.Name),
		slog.Int("stock", product.Stock),
		slog.Float64("price", product.Price))

	return product, appErr // appErr is nil here if successful
}
