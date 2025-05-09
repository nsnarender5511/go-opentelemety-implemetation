package repositories

import (
	"context"
	"log/slog"
	"os"
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

func (r *productRepository) GetAll(ctx context.Context) (productsSlice []models.Product, appErr *apierrors.AppError) {
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
		appErr = simAppErr
		return nil, appErr
	}

	r.logger.InfoContext(ctx, "Stock Room Worker: *Adjusts uniform* Shop manager asked me to fetch ALL products from our inventory")
	r.logger.DebugContext(ctx, "Stock Room Worker: *Walks to the back room* Let me check our master inventory ledger")

	var productsMap map[string]models.Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		if os.IsNotExist(err) {
			r.logger.WarnContext(ctx, "Stock Room Worker: *Panics* Oh no! The inventory ledger is missing! *Takes deep breath* I better tell the shop manager our shelves are empty")
			span.AddEvent("FileDatabase.Read indicated file not found, returning empty.", trace.WithAttributes(attribute.String("error.message", err.Error())))
			return []models.Product{}, nil
		} else {
			errMsg := "Failed to read inventory ledger"
			r.logger.ErrorContext(ctx, "Stock Room Worker: *Distressed* Cannot read inventory ledger!", slog.String("error", err.Error()))
			if span != nil {
				span.SetStatus(codes.Error, errMsg)
			}
			appErr = apierrors.NewAppError(apierrors.ErrCodeDatabase, errMsg, err)
			return nil, appErr
		}
	}

	r.logger.DebugContext(ctx, "Stock Room Worker: *Flips through pages* Ah! Here's our complete inventory. Let me count everything...")

	productsSlice = make([]models.Product, 0, len(productsMap))
	// var productID string // Removed unused productID variable and its assignment loop
	for _, p := range productsMap { // Iterate once to populate productsSlice
		productsSlice = append(productsSlice, p)
		r.logger.DebugContext(ctx, "Stock Room Worker: *Checks shelf* Product "+p.Name+" - we have "+strconv.Itoa(p.Stock)+" in stock")
	}

	// Update product stock levels for telemetry
	// const storeID = "default-store" // Removed storeID
	for _, p := range productsSlice { // Iterate productsSlice, p.Name is the identifier
		metric.UpdateProductStockLevels(ctx, p.Name, p.Category, int64(p.Stock))
	}

	productCount := len(productsSlice)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))
	r.logger.InfoContext(ctx, "Stock Room Worker: *Wipes brow* Phew! Counted all "+strconv.Itoa(productCount)+" products in our actual stock")
	r.logger.InfoContext(ctx, "Stock Room Worker: *Walks back to counter* Here's the complete inventory list for the shop manager")

	return productsSlice, appErr // appErr is nil here if successful
}
