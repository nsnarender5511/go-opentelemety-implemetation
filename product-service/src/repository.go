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
)

// Updated Interface
type ProductRepository interface {
	GetAll(ctx context.Context) ([]Product, error)
	GetByName(ctx context.Context, name string) (Product, error)
	UpdateStock(ctx context.Context, name string, newStock int) error
	GetByCategory(ctx context.Context, category string) ([]Product, error)
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

func (r *productRepository) GetAll(ctx context.Context) (productsSlice []Product, opErr error) {
	ctx, span := commontrace.StartSpan(ctx, attribute.String("repository.operation", "GetAll"))
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

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
			opErr = fmt.Errorf("failed to read products for GetAll using FileDatabase: %w", err)
			r.logger.ErrorContext(ctx, "Stock Room Worker: *Distressed* I can't make sense of this inventory ledger! It's all smudged and unreadable! Error: "+opErr.Error())
			span.SetStatus(codes.Error, opErr.Error())
			return nil, opErr
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
func (r *productRepository) GetByName(ctx context.Context, name string) (product Product, opErr error) {
	productNameAttr := attribute.String("product.name", name)
	ctx, span := commontrace.StartSpan(ctx, productNameAttr)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Stock Room Worker: *Perks up* Shop manager needs me to find product with name: '"+name+"'")
	r.logger.DebugContext(ctx, "Stock Room Worker: *Grabs inventory clipboard* Let me check if we have '"+name+"' in stock")

	var productsMap map[string]Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		opErr = fmt.Errorf("failed to read products for GetByName using FileDatabase: %w", err)
		r.logger.ErrorContext(ctx, "Stock Room Worker: *Frustrated* The inventory ledger pages are stuck together! Can't read anything! Error: "+opErr.Error())
		span.SetStatus(codes.Error, opErr.Error())
		return Product{}, opErr
	}

	r.logger.DebugContext(ctx, "Stock Room Worker: *Runs finger down inventory list* Looking for product name '"+name+"'...")

	product, exists := productsMap[name]
	if !exists {
		opErr = fmt.Errorf("product with name '%s' not found", name)
		r.logger.ErrorContext(ctx, "Stock Room Worker: *Scratches head* Hmm, that's strange. We don't have any product named '"+name+"' anywhere in our stockroom!")
		r.logger.DebugContext(ctx, "Stock Room Worker: *Double-checks shelves* Nope, definitely not here. Must be discontinued or never existed")
		return Product{}, opErr
	}

	span.SetAttributes(attribute.String("product.category_found", product.Category))
	r.logger.InfoContext(ctx, "Stock Room Worker: *Excited* Found it! Product '"+product.Name+"' is right here on shelf "+product.Category)
	r.logger.DebugContext(ctx, "Stock Room Worker: *Checks quantity* We have "+strconv.Itoa(product.Stock)+" units at $"+fmt.Sprintf("%.2f", product.Price)+" each")
	r.logger.InfoContext(ctx, "Stock Room Worker: *Hurries back* Here are the details for product '"+name+"' that shop manager asked for")

	return product, nil
}

// Updated signature: uses name instead of productID
func (r *productRepository) UpdateStock(ctx context.Context, name string, newStock int) (opErr error) {
	productNameAttr := attribute.String("product.name", name)
	newStockAttr := attribute.Int("product.new_stock", newStock)
	attrs := []attribute.KeyValue{productNameAttr, newStockAttr}

	ctx, span := commontrace.StartSpan(ctx, attrs...)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Stock Room Worker: *Nods* Shop manager wants me to update stock for product '"+name+"' to exactly "+strconv.Itoa(newStock)+" units")
	r.logger.DebugContext(ctx, "Stock Room Worker: *Rolls up sleeves* Time to adjust our physical inventory")

	var productsMap map[string]Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		opErr = fmt.Errorf("failed to read products for UpdateStock using FileDatabase: %w", err)
		r.logger.ErrorContext(ctx, "Stock Room Worker: *Panics* I dropped the inventory ledger in a puddle! Can't read anything! Error: "+opErr.Error())
		span.SetStatus(codes.Error, opErr.Error())
		return opErr
	}

	r.logger.DebugContext(ctx, "Stock Room Worker: *Searches inventory* Let me find product '"+name+"' first...")

	product, ok := productsMap[name]
	if !ok {
		opErr = fmt.Errorf("product with name '%s' not found for update", name)
		r.logger.ErrorContext(ctx, "Stock Room Worker: *Confused* That's weird. I've checked every shelf and box but we don't have any product named '"+name+"' in our inventory!")
		r.logger.DebugContext(ctx, "Stock Room Worker: *Shrugs* Can't update stock for a product we don't carry")
		span.AddEvent("product_not_found_in_map_for_update", trace.WithAttributes(attrs...))
		return opErr
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
		opErr = fmt.Errorf("failed to write products for UpdateStock using FileDatabase: %w", writeErr)
		r.logger.ErrorContext(ctx, "Stock Room Worker: *Spills coffee* No! I just spilled coffee all over the inventory ledger while updating it! Error: "+opErr.Error())
		span.SetStatus(codes.Error, opErr.Error())
		return opErr
	}

	r.logger.InfoContext(ctx, "Stock Room Worker: *Satisfied* Successfully updated the stock for "+product.Name+" from "+strconv.Itoa(oldStock)+" to "+strconv.Itoa(newStock))
	r.logger.DebugContext(ctx, "Stock Room Worker: *Closes ledger* Also updated our inventory records to match the physical stock")
	r.logger.InfoContext(ctx, "Stock Room Worker: *Returns to counter* All done! Stock update completed for product '"+name+"'")

	return nil
}

// No signature change needed, but ensure internal logic is compatible
func (r *productRepository) GetByCategory(ctx context.Context, category string) (filteredProducts []Product, opErr error) {
	categoryAttr := attribute.String("product.category", category)
	ctx, span := commontrace.StartSpan(ctx, categoryAttr)
	defer commontrace.EndSpan(span, &opErr, nil)

	debugutils.Simulate(ctx)

	r.logger.InfoContext(ctx, "Stock Room Worker: *Adjusts name tag* Shop manager needs all products from the '"+category+"' category")
	r.logger.DebugContext(ctx, "Stock Room Worker: *Walks to section "+category+"* Let me check what we have on these shelves")

	var productsMap map[string]Product
	err := r.database.Read(ctx, &productsMap)
	if err != nil {
		if os.IsNotExist(err) {
			r.logger.WarnContext(ctx, "Stock Room Worker: *Worried* Strange, our inventory ledger is missing! I better tell shop manager we have no products in category '"+category+"'")
			span.AddEvent("FileDatabase.Read indicated file not found, returning empty.", trace.WithAttributes(attribute.String("error.message", err.Error())))
			return []Product{}, nil
		} else {
			opErr = fmt.Errorf("failed to read products for GetByCategory using FileDatabase: %w", err)
			r.logger.ErrorContext(ctx, "Stock Room Worker: *Squints at pages* The inventory ledger is too faded to read! Error: "+opErr.Error())
			span.SetStatus(codes.Error, opErr.Error())
			return nil, opErr
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
