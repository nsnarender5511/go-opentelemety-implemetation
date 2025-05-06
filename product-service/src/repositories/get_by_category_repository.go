package repositories

import (
	"context"
	"log/slog"
	"os"
	"strconv"

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
	newCtx, span := commontrace.StartSpan(ctx, categoryAttr)
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

	r.logger.InfoContext(ctx, "Stock Room Worker: *Adjusts name tag* Shop manager needs all products from the '"+category+"' category")
	r.logger.DebugContext(ctx, "Stock Room Worker: *Walks to section "+category+"* Let me check what we have on these shelves")

	var productsMap map[string]models.Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		if os.IsNotExist(err) {
			r.logger.WarnContext(ctx, "Stock Room Worker: *Worried* Strange, our inventory ledger is missing! I better tell shop manager we have no products in category '"+category+"'", slog.String("category", category))
			span.AddEvent("FileDatabase.Read indicated file not found, returning empty.", trace.WithAttributes(attribute.String("error.message", err.Error())))
			return []models.Product{}, nil
		} else {
			errMsg := "Failed to read inventory data for category lookup"
			r.logger.ErrorContext(ctx, "Stock Room Worker: *Squints* Cannot read inventory ledger!", slog.String("error", err.Error()))
			if span != nil {
				span.SetStatus(codes.Error, errMsg)
			}
			appErr = apierrors.NewAppError(apierrors.ErrCodeDatabase, errMsg, err)
			return nil, appErr
		}
	}

	r.logger.DebugContext(ctx, "Stock Room Worker: *Searching shelves* Looking through our '"+category+"' section...")

	filteredProducts = make([]models.Product, 0)
	for _, p := range productsMap {
		if p.Category == category {
			filteredProducts = append(filteredProducts, p)
			r.logger.DebugContext(ctx, "Stock Room Worker: *Picks up item* Found "+p.Name+" in the '"+category+"' section, we have "+strconv.Itoa(p.Stock)+" in stock")
		}
	}

	productCount := len(filteredProducts)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))

	if productCount == 0 {
		r.logger.InfoContext(ctx, "Stock Room Worker: *Shrugs* We don't have any products in the '"+category+"' category right now")
	} else {
		r.logger.InfoContext(ctx, "Stock Room Worker: *Counts items* We have "+strconv.Itoa(productCount)+" different products in the '"+category+"' category")
	}

	r.logger.InfoContext(ctx, "Stock Room Worker: *Returns to counter* Here's the list of all our '"+category+"' products for the shop manager")
	return filteredProducts, appErr // appErr is nil here if successful
}
