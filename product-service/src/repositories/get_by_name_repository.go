package repositories

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/narender/common/debugutils"
	commontrace "github.com/narender/common/telemetry/trace"
	"github.com/narender/product-service/src/models" // Corrected path
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"strconv"

	apierrors "github.com/narender/common/apierrors"
)

func (r *productRepository) GetByName(ctx context.Context, name string) (product models.Product, appErr *apierrors.AppError) {
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

	r.logger.InfoContext(ctx, "Stock Room Worker: *Perks up* Shop manager needs me to find product with name: '"+name+"'")
	r.logger.DebugContext(ctx, "Stock Room Worker: *Grabs inventory clipboard* Let me check if we have '"+name+"'")

	var productsMap map[string]models.Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		errMsg := "Failed to read inventory data"
		r.logger.ErrorContext(ctx, "Stock Room Worker: *Frustrated* Cannot read inventory ledger!", slog.String("error", err.Error()))
		if span != nil {
			span.SetStatus(codes.Error, errMsg)
		}
		appErr = apierrors.NewAppError(apierrors.ErrCodeDatabase, errMsg, err)
		return models.Product{}, appErr
	}

	r.logger.DebugContext(ctx, "Stock Room Worker: *Runs finger down inventory list* Looking for product name '"+name+"'")

	product, exists := productsMap[name]
	if !exists {
		errMsg := fmt.Sprintf("Product with name '%s' not found", name)
		r.logger.WarnContext(ctx, "Stock Room Worker: Product not found", slog.String("product_name", name))
		if span != nil {
			span.SetStatus(codes.Error, errMsg)
		}
		appErr = apierrors.NewAppError(apierrors.ErrCodeNotFound, errMsg, nil)
		return models.Product{}, appErr
	}

	span.SetAttributes(attribute.String("product.category_found", product.Category))
	r.logger.InfoContext(ctx, "Stock Room Worker: *Excited* Found it! Product '"+product.Name+"' is right here on shelf "+product.Category)
	r.logger.DebugContext(ctx, "Stock Room Worker: *Checks quantity* We have "+strconv.Itoa(product.Stock)+" units at $"+fmt.Sprintf("%.2f", product.Price)+" each")
	r.logger.InfoContext(ctx, "Stock Room Worker: *Hurries back* Here are the details for product '"+name+"' that shop manager asked for")

	return product, appErr // appErr is nil here if successful
}
