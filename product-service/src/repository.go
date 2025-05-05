package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	db "github.com/narender/common/db"
	"github.com/narender/common/debugutils"
	commonmetric "github.com/narender/common/telemetry/metric"
	commontrace "github.com/narender/common/telemetry/trace"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/narender/common/globals"
	"go.opentelemetry.io/otel/trace"

	// Import common errors package
	apierrors "github.com/narender/common/apierrors"
)

// Updated Interface
type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, *apierrors.AppError)
	GetByName(ctx context.Context, name string) (Product, *apierrors.AppError)
	UpdateStock(ctx context.Context, name string, newStock int) *apierrors.AppError
	GetByCategory(ctx context.Context, category string) ([]Product, *apierrors.AppError)
}

type productRepository struct {
	database *db.FileDatabase
	logger   *slog.Logger
}

// NewProductRepository creates a new repository instance loading data from a JSON file.
func NewProductRepository() ProductRepository {
	repo := &productRepository{
		database: db.NewFileDatabase(),
		logger:   globals.Logger(),
	}
	return repo
}

func (r *productRepository) GetAll(ctx context.Context) (productsSlice []Product, appErr *apierrors.AppError) {
	var opErr error
	ctx, span := commontrace.StartSpan(ctx, attribute.String("repository.operation", "GetAll"))

	defer func() {
		if appErr != nil && opErr == nil {
			opErr = appErr
		}
		commontrace.EndSpan(span, &opErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return nil, simAppErr
	}

	r.logger.InfoContext(ctx, "Stock Room Worker: *Adjusts uniform* Shop manager asked me to fetch ALL products from our inventory")
	r.logger.DebugContext(ctx, "Stock Room Worker: *Walks to the back room* Let me check our master inventory ledger")

	var productsMap map[string]Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		if os.IsNotExist(err) {
			r.logger.WarnContext(ctx, "Stock Room Worker: *Panics* Oh no! The inventory ledger is missing! *Takes deep breath* I better tell the shop manager our shelves are empty")
			span.AddEvent("FileDatabase.Read indicated file not found, returning empty.", trace.WithAttributes(attribute.String("error.message", err.Error())))
			return []Product{}, nil
		} else {
			errMsg := "Failed to read inventory ledger"
			r.logger.ErrorContext(ctx, "Stock Room Worker: *Distressed* Cannot read inventory ledger!", slog.String("error", err.Error()))
			span.SetStatus(codes.Error, errMsg)
			appErr = apierrors.NewAppError(apierrors.ErrCodeDatabase, errMsg, err)
			return nil, appErr
		}
	}

	r.logger.DebugContext(ctx, "Stock Room Worker: *Flips through pages* Ah! Here's our complete inventory. Let me count everything...")

	productsSlice = make([]Product, 0, len(productsMap))
	for _, p := range productsMap {
		productsSlice = append(productsSlice, p)
		r.logger.DebugContext(ctx, "Stock Room Worker: *Checks shelf* Product "+p.Name+" - we have "+strconv.Itoa(p.Stock)+" in stock")
	}

	stockLevels := make(map[string]int64, len(productsSlice))
	for _, p := range productsSlice {
		stockLevels[p.Name] = int64(p.Stock)
	}
	commonmetric.UpdateProductStockLevels(stockLevels)
	r.logger.DebugContext(ctx, "Stock Room Worker: *Updates big inventory board* Just updated our stock level display board with current numbers")

	productCount := len(productsSlice)
	span.SetAttributes(attribute.Int("products.returned.count", productCount))
	r.logger.InfoContext(ctx, "Stock Room Worker: *Wipes brow* Phew! Counted all "+strconv.Itoa(productCount)+" products in our actual stock")
	r.logger.InfoContext(ctx, "Stock Room Worker: *Walks back to counter* Here's the complete inventory list for the shop manager")

	return productsSlice, nil
}

// Renamed from GetByID
func (r *productRepository) GetByName(ctx context.Context, name string) (product Product, appErr *apierrors.AppError) {
	productNameAttr := attribute.String("product.name", name)
	ctx, span := commontrace.StartSpan(ctx, productNameAttr)
	var opErr error
	defer func() {
		if appErr != nil && opErr == nil {
			opErr = appErr
		}
		commontrace.EndSpan(span, &opErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return Product{}, simAppErr
	}

	r.logger.InfoContext(ctx, "Stock Room Worker: *Perks up* Shop manager needs me to find product with name: '"+name+"'")
	r.logger.DebugContext(ctx, "Stock Room Worker: *Grabs inventory clipboard* Let me check if we have '"+name+"' in stock")

	var productsMap map[string]Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		errMsg := "Failed to read inventory data"
		r.logger.ErrorContext(ctx, "Stock Room Worker: *Frustrated* Cannot read inventory ledger!", slog.String("error", err.Error()))
		span.SetStatus(codes.Error, errMsg)
		appErr = apierrors.NewAppError(apierrors.ErrCodeDatabase, errMsg, err)
		return Product{}, appErr
	}

	r.logger.DebugContext(ctx, "Stock Room Worker: *Runs finger down inventory list* Looking for product name '"+name+"'...")

	product, exists := productsMap[name]
	if !exists {
		errMsg := fmt.Sprintf("Product with name '%s' not found", name)
		r.logger.WarnContext(ctx, "Stock Room Worker: Product not found", slog.String("product_name", name))
		span.SetStatus(codes.Error, errMsg)
		appErr = apierrors.NewAppError(apierrors.ErrCodeNotFound, errMsg, nil)
		return Product{}, appErr
	}

	span.SetAttributes(attribute.String("product.category_found", product.Category))
	r.logger.InfoContext(ctx, "Stock Room Worker: *Excited* Found it! Product '"+product.Name+"' is right here on shelf "+product.Category)
	r.logger.DebugContext(ctx, "Stock Room Worker: *Checks quantity* We have "+strconv.Itoa(product.Stock)+" units at $"+fmt.Sprintf("%.2f", product.Price)+" each")
	r.logger.InfoContext(ctx, "Stock Room Worker: *Hurries back* Here are the details for product '"+name+"' that shop manager asked for")

	return product, nil
}

// Updated signature: uses name instead of productID
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

	var productsMap map[string]Product
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

	r.logger.InfoContext(ctx, "Stock Room Worker: *Satisfied* Successfully updated the stock for "+product.Name+" from "+strconv.Itoa(oldStock)+" to "+strconv.Itoa(newStock))
	r.logger.DebugContext(ctx, "Stock Room Worker: *Closes ledger* Also updated our inventory records to match the physical stock")
	r.logger.InfoContext(ctx, "Stock Room Worker: *Returns to counter* All done! Stock update completed for product '"+name+"'")

	return nil
}

// No signature change needed, but ensure internal logic is compatible
func (r *productRepository) GetByCategory(ctx context.Context, category string) (filteredProducts []Product, appErr *apierrors.AppError) {
	categoryAttr := attribute.String("product.category", category)
	ctx, span := commontrace.StartSpan(ctx, categoryAttr)
	var opErr error
	defer func() {
		if appErr != nil && opErr == nil {
			opErr = appErr
		}
		commontrace.EndSpan(span, &opErr, nil)
	}()

	if simAppErr := debugutils.Simulate(ctx); simAppErr != nil {
		return nil, simAppErr
	}

	r.logger.InfoContext(ctx, "Stock Room Worker: *Adjusts name tag* Shop manager needs all products from the '"+category+"' category")
	r.logger.DebugContext(ctx, "Stock Room Worker: *Walks to section "+category+"* Let me check what we have on these shelves")

	var productsMap map[string]Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		if os.IsNotExist(err) {
			r.logger.WarnContext(ctx, "Stock Room Worker: *Worried* Strange, our inventory ledger is missing! I better tell shop manager we have no products in category '"+category+"'", slog.String("category", category))
			span.AddEvent("FileDatabase.Read indicated file not found, returning empty.", trace.WithAttributes(attribute.String("error.message", err.Error())))
			return []Product{}, nil
		} else {
			errMsg := "Failed to read inventory data for category lookup"
			r.logger.ErrorContext(ctx, "Stock Room Worker: *Squints* Cannot read inventory ledger!", slog.String("error", err.Error()))
			span.SetStatus(codes.Error, errMsg)
			appErr = apierrors.NewAppError(apierrors.ErrCodeDatabase, errMsg, err)
			return nil, appErr
		}
	}

	r.logger.DebugContext(ctx, "Stock Room Worker: *Searching shelves* Looking through our '"+category+"' section...")

	filteredProducts = make([]Product, 0)
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
	return filteredProducts, nil
}
