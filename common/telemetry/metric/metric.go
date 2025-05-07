package metric

import (
	"context"
	"log/slog"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// productStockDetail holds the stock level and associated attributes for a product.
// This is used as the value in the latestProductStock map.
type productStockDetail struct {
	StockLevel      int64
	ProductName     string
	ProductCategory string
}

var (
	meter           = otel.Meter("common/telemetry/metric")
	counters        = make(map[string]metric.Int64Counter)
	float64Counters = make(map[string]metric.Float64Counter)
	histograms      = make(map[string]metric.Float64Histogram)
	gauges          = make(map[string]metric.Int64ObservableGauge)

	// Storage for latest product stock levels for the observable gauge
	// Key is productName
	latestProductStock      = make(map[string]productStockDetail)
	latestProductStockMutex sync.RWMutex
)

// --- Initialization ---

func init() {
	for name, cfg := range metricDefinitions { // metricDefinitions is defined in custom_metrics.go
		switch cfg.Type {
		case counterType: // counterType is defined in custom_metrics.go
			counter := createInt64Counter(name, cfg.Description, cfg.Unit)
			if counter != nil {
				counters[name] = counter
			}
		case histogramType: // histogramType is defined in custom_metrics.go
			histogram := createFloat64Histogram(name, cfg.Description, cfg.Unit)
			if histogram != nil {
				histograms[name] = histogram
			}
		case observableGaugeType: // observableGaugeType is defined in custom_metrics.go
			gauge := createInt64ObservableGauge(name, cfg.Description, cfg.Unit)
			if gauge != nil {
				gauges[name] = gauge
				if name == ProductStockCountMetric {
					_, err := meter.RegisterCallback(observeProductStock, gauge)
					if err != nil {
						slog.Error("Failed to register callback for gauge", slog.String("metric", name), slog.Any("error", err))
					}
				}
			}
		case floatCounterType: // New case
			counter := createFloat64Counter(name, cfg.Description, cfg.Unit)
			if counter != nil {
				float64Counters[name] = counter
			}
		default:
			slog.Warn("Unknown metric type in configuration", slog.String("metric", name), slog.String("type", string(cfg.Type)))
		}
	}
}

// --- Public Functions / Constructors ---

// --- Helper Functions ---

func createInt64Counter(name, description, unit string) metric.Int64Counter {
	counter, err := meter.Int64Counter(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		slog.Error("Failed to initialize counter", slog.String("metric", name), slog.Any("error", err))
	}
	return counter
}

func createFloat64Histogram(name, description, unit string) metric.Float64Histogram {
	histogram, err := meter.Float64Histogram(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		slog.Error("Failed to initialize histogram", slog.String("metric", name), slog.Any("error", err))
	}
	return histogram
}

func createInt64ObservableGauge(name, description, unit string) metric.Int64ObservableGauge {
	gauge, err := meter.Int64ObservableGauge(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		slog.Error("Failed to initialize observable gauge", slog.String("metric", name), slog.Any("error", err))
	}
	return gauge
}

func createFloat64Counter(name, description, unit string) metric.Float64Counter {
	counter, err := meter.Float64Counter(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		slog.Error("Failed to initialize float64 counter", slog.String("metric", name), slog.Any("error", err))
	}
	return counter
}

// --- Callback Functions ---

// observeProductStock is the callback function for the product inventory gauge.
// It reads the latest stock levels and reports them to OpenTelemetry.
func observeProductStock(ctx context.Context, observer metric.Observer) error {
	latestProductStockMutex.RLock()
	defer latestProductStockMutex.RUnlock()

	gauge, ok := gauges[ProductStockCountMetric]
	if !ok {
		slog.ErrorContext(ctx, "Failed to find gauge instrument in callback", slog.String("metric", ProductStockCountMetric))
		return nil
	}

	for productNameKey, detail := range latestProductStock {
		// Observe the current stock level for this product ID
		attrs := attribute.NewSet(
			attribute.String("product.name", productNameKey),
			attribute.String("product.category", detail.ProductCategory),
			attribute.String("custom.metric", "true"),
		)
		observer.ObserveInt64(gauge, detail.StockLevel, metric.WithAttributeSet(attrs))
	}
	return nil
}

// UpdateProductStockLevels updates the in-memory store of product stock levels.
// This function is called when new stock data is available.
// productName is the map key and also stored in the detail struct.
func UpdateProductStockLevels(productName, productCategory string, stockLevel int64) {
	latestProductStockMutex.Lock()
	defer latestProductStockMutex.Unlock()
	latestProductStock[productName] = productStockDetail{
		StockLevel:      stockLevel,
		ProductName:     productName,
		ProductCategory: productCategory,
		
	}
	slog.Debug("Updated product stock level",
		slog.String("product.name", productName),
		slog.String("product.category", productCategory),
		slog.Int64("stock.level", stockLevel),
		
	)
}

func IncrementRevenueTotal(ctx context.Context, revenue float64, productName, productCategory, currencyCode string) {
	counter, ok := float64Counters[AppRevenueTotalMetric]
	if !ok {
		slog.WarnContext(ctx, "Failed to find counter", slog.String("metric", AppRevenueTotalMetric))
		return
	}
	attrs := attribute.NewSet(
		attribute.String("product.bill.amount", strconv.FormatFloat(revenue, 'f', -1, 64)),
		attribute.String("product.name", productName),
		attribute.String("product.category", productCategory),
		attribute.String("custom.metric", "true"),
	)
	counter.Add(ctx, revenue, metric.WithAttributeSet(attrs))
}

func IncrementItemsSoldCount(ctx context.Context, quantity int64, productName, productCategory string) {
	counter, ok := counters[AppItemsSoldCountMetric]
	if !ok {
		slog.WarnContext(ctx, "Failed to find counter", slog.String("metric", AppItemsSoldCountMetric))
		return
	}
	attrs := attribute.NewSet(
		attribute.String("product.name", productName),
		attribute.String("product.category", productCategory),
		attribute.String("custom.metric", "true"),
	)
	counter.Add(ctx, quantity, metric.WithAttributeSet(attrs))
}
