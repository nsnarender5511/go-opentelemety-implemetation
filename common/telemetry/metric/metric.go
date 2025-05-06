package metric

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type MetricsController interface {
	End(ctx context.Context, err *error, additionalAttrs ...attribute.KeyValue)
	IncrementProductCreated(ctx context.Context)
	IncrementProductUpdated(ctx context.Context)
}

type metricsControllerImpl struct {
	startTime time.Time
}

// --- Global Variables ---

var (
	meter      = otel.Meter("common/telemetry/metric")
	counters   = make(map[string]metric.Int64Counter)
	histograms = make(map[string]metric.Float64Histogram)
	gauges     = make(map[string]metric.Int64ObservableGauge)

	// Storage for latest product stock levels for the observable gauge
	latestProductStock      = make(map[string]int64)
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
				if name == ProductInventoryCountMetric {
					_, err := meter.RegisterCallback(observeProductStock, gauge)
					if err != nil {
						slog.Error("Failed to register callback for gauge", slog.String("metric", name), slog.Any("error", err))
					}
				}
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

// --- Callback Functions ---

// observeProductStock is the callback function for the product inventory gauge.
// It reads the latest stock levels and reports them to OpenTelemetry.
func observeProductStock(ctx context.Context, observer metric.Observer) error {
	latestProductStockMutex.RLock()
	defer latestProductStockMutex.RUnlock()

	gauge, ok := gauges[ProductInventoryCountMetric]
	if !ok {
		slog.ErrorContext(ctx, "Failed to find gauge instrument in callback", slog.String("metric", ProductInventoryCountMetric))
		// Returning an error might stop further callbacks depending on the SDK implementation,
		// so we log and return nil to be safe.
		return nil
	}

	for productID, stock := range latestProductStock {
		// Observe the current stock level for this product ID
		productIDAttribute := attribute.String("product.id", productID)
		observer.ObserveInt64(gauge, stock, metric.WithAttributes(productIDAttribute))
	}
	return nil
}

// UpdateProductStockLevels updates the in-memory store of product stock levels.
// This function is called when new stock data is available.
func UpdateProductStockLevels(productID string, stockLevel int64) {
	latestProductStockMutex.Lock()
	defer latestProductStockMutex.Unlock()
	latestProductStock[productID] = stockLevel
	slog.Debug("Updated product stock level", slog.String("product.id", productID), slog.Int64("stock.level", stockLevel))
}
