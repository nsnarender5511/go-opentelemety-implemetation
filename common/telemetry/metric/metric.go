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

func StartMetricsTimer() MetricsController {
	return &metricsControllerImpl{
		startTime: time.Now(),
	}
}

// UpdateProductStockLevels provides the latest snapshot of product stock levels.
// This should be called periodically or after operations that change stock levels
// (e.g., after fetching all products).
// The provided map should contain product ID -> current stock count.
func UpdateProductStockLevels(newStockLevels map[string]int64) {
	latestProductStockMutex.Lock()
	defer latestProductStockMutex.Unlock()

	// Clear the old map before populating with new data
	clear(latestProductStock)

	// Populate with the new stock levels
	for id, stock := range newStockLevels {
		latestProductStock[id] = stock
	}
	// Optional: Log that levels were updated
	slog.Debug("Updated product stock levels for metrics", slog.Int("product_count", len(latestProductStock)))
}

// --- Methods for metricsControllerImpl ---

func (mc *metricsControllerImpl) End(ctx context.Context, errPtr *error, additionalAttrs ...attribute.KeyValue) {
	duration := time.Since(mc.startTime)
	durationMs := float64(duration.Microseconds()) / 1000.0

	isError := errPtr != nil && *errPtr != nil

	baseAttrs := []attribute.KeyValue{
		attribute.Bool("app.error", isError),
	}
	attrs := append(baseAttrs, additionalAttrs...)

	opt := metric.WithAttributes(attrs...)

	if operationsTotal, ok := counters[TotalOperationsMetric]; ok { // TotalOperationsMetric is defined in custom_metrics.go
		operationsTotal.Add(ctx, 1, opt)
	} else {
		slog.WarnContext(ctx, "Metric '"+TotalOperationsMetric+"' not found or initialized")
	}

	if durationMillis, ok := histograms[DurationMsMetric]; ok { // DurationMsMetric is defined in custom_metrics.go
		durationMillis.Record(ctx, durationMs, opt)
	} else {
		slog.WarnContext(ctx, "Metric '"+DurationMsMetric+"' not found or initialized")
	}

	if isError {
		if errorsTotal, ok := counters[ErrorsTotalMetric]; ok { // ErrorsTotalMetric is defined in custom_metrics.go
			errorsTotal.Add(ctx, 1, opt)
		} else {
			slog.WarnContext(ctx, "Metric '"+ErrorsTotalMetric+"' not found or initialized")
		}
	}
}

func (mc *metricsControllerImpl) IncrementProductCreated(ctx context.Context) {
	if productCreationCounter, ok := counters[TotalProductCreationsMetric]; ok { // TotalProductCreationsMetric is defined in custom_metrics.go
		productCreationCounter.Add(ctx, 1)
	} else {
		slog.WarnContext(ctx, "Metric '"+TotalProductCreationsMetric+"' not found or initialized")
	}
}

func (mc *metricsControllerImpl) IncrementProductUpdated(ctx context.Context) {
	if productUpdateCounter, ok := counters[TotalProductUpdatesMetric]; ok { // TotalProductUpdatesMetric is defined in custom_metrics.go
		productUpdateCounter.Add(ctx, 1)
	} else {
		slog.WarnContext(ctx, "Metric '"+TotalProductUpdatesMetric+"' not found or initialized")
	}
}

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
		productAttribute := attribute.String("product.id", productID)
		observer.ObserveInt64(gauge, stock, metric.WithAttributes(productAttribute))
	}

	// slog.DebugContext(ctx, "observeProductStock called - placeholder implementation") // Placeholder log removed
	return nil
}
