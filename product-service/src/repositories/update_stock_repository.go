package repositories

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

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
	productNameAttr := attribute.String("product.name", name)
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
		return simAppErr
	}

	r.logger.InfoContext(ctx, "Stock Room Worker: *Nods* Shop manager wants me to update stock for product '"+name+"' to exactly "+strconv.Itoa(newStock)+" units")
	r.logger.DebugContext(ctx, "Stock Room Worker: *Rolls up sleeves* Time to adjust our physical inventory")

	var productsMap map[string]models.Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		errMsg := "Failed to read inventory data for update"
		r.logger.ErrorContext(ctx, "Stock Room Worker: *Panics* Cannot read inventory ledger for update!", slog.String("error", err.Error()))
		span.SetStatus(codes.Error, errMsg)
		appErr = apierrors.NewAppError(apierrors.ErrCodeDatabase, errMsg, err)
		return appErr
	}

	r.logger.DebugContext(ctx, "Stock Room Worker: *Searches inventory* Let me find product '"+name+"' first...")

	product, ok := productsMap[name]
	if !ok {
		errMsg := fmt.Sprintf("Product with name '%s' not found for stock update", name)
		r.logger.WarnContext(ctx, "Stock Room Worker: Product not found for update", slog.String("product_name", name))
		span.AddEvent("product_not_found_in_map_for_update", trace.WithAttributes(attrs...))
		span.SetStatus(codes.Error, errMsg)
		appErr = apierrors.NewAppError(apierrors.ErrCodeNotFound, errMsg, nil)
		return appErr
	}

	oldStock := product.Stock
	product.Stock = newStock
	productsMap[name] = product

	span.SetAttributes(attribute.Int("product.old_stock", oldStock))

	if newStock > oldStock {
		added := newStock - oldStock
		r.logger.InfoContext(ctx, "Stock Room Worker: *Unloading boxes* Adding "+strconv.Itoa(added)+" units of "+product.Name+" to the shelf")
		r.logger.DebugContext(ctx, "Stock Room Worker: *Arranges products* Making sure they're displayed nicely")
	} else if newStock < oldStock {
		removed := oldStock - newStock
		r.logger.InfoContext(ctx, "Stock Room Worker: *Counts carefully* Removing "+strconv.Itoa(removed)+" units of "+product.Name+" from the shelf")
		r.logger.DebugContext(ctx, "Stock Room Worker: *Updates display* Making sure the remaining "+strconv.Itoa(newStock)+" units look presentable")
	} else {
		r.logger.DebugContext(ctx, "Stock Room Worker: *Puzzled* Stock count for "+product.Name+" is unchanged at "+strconv.Itoa(newStock)+". Just double-checking everything looks good")
	}

	if writeErr := r.database.Write(ctx, productsMap); writeErr != nil {
		errMsg := "Failed to write updated inventory data"
		r.logger.ErrorContext(ctx, "Stock Room Worker: *Spills coffee* Cannot write updated inventory ledger!", slog.String("error", writeErr.Error()))
		span.SetStatus(codes.Error, errMsg)
		appErr = apierrors.NewAppError(apierrors.ErrCodeDatabase, errMsg, writeErr)
		return appErr
	}

	// Update product stock level for telemetry
	metric.UpdateProductStockLevels(product.Name, int64(newStock))

	r.logger.InfoContext(ctx, "Stock Room Worker: *Satisfied* Successfully updated the stock for "+product.Name+" from "+strconv.Itoa(oldStock)+" to "+strconv.Itoa(newStock))
	r.logger.DebugContext(ctx, "Stock Room Worker: *Closes ledger* Also updated our inventory records to match the physical stock")
	r.logger.InfoContext(ctx, "Stock Room Worker: *Returns to counter* All done! Stock update completed for product '"+name+"'")

	return nil
}
