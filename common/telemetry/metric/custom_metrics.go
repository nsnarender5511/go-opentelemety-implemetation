package metric

// --- Metric Configuration Constants ---

type metricType string

const (
	counterType         metricType = "counter"
	histogramType       metricType = "histogram"
	observableGaugeType metricType = "observable_gauge"

	// Define metric names as constants for type safety and easier refactoring
	ProductInventoryCountMetric = "product.stock.count"
)

// --- Metric Configuration Types ---

type metricConfig struct {
	Description string
	Unit        string
	Type        metricType
}

// --- Metric Definitions Map ---

// Centralized definition for all metrics using constants
var metricDefinitions = map[string]metricConfig{
	ProductInventoryCountMetric: {
		Description: "Current count of items in inventory for each product",
		Unit:        "{item}",
		Type:        observableGaugeType,
	},
}
