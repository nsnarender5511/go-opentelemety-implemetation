package metric

// --- Metric Configuration Constants ---

type metricType string

const (
	counterType         metricType = "counter"
	histogramType       metricType = "histogram"
	observableGaugeType metricType = "observable_gauge"

	// Define metric names as constants for type safety and easier refactoring
	TotalOperationsMetric       = "APP_OPERATIONS_TOTAL"
	DurationMsMetric            = "APP_OPERATIONS_DURATION_MILLISECONDS"
	ErrorsTotalMetric           = "APP_OPERATIONS_ERRORS_TOTAL"
	TotalProductCreationsMetric = "PRODUCT_CREATION_TOTAL"
	TotalProductUpdatesMetric   = "PRODUCT_UPDATES_TOTAL"
	ProductInventoryCountMetric = "product.inventory.count"
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
	TotalOperationsMetric: {
		Description: "Total number of operations executed",
		Unit:        "{operation}",
		Type:        counterType,
	},
	DurationMsMetric: {
		Description: "Duration of operations in milliseconds",
		Unit:        "ms",
		Type:        histogramType,
	},
	ErrorsTotalMetric: {
		Description: "Total number of operations that resulted in an error",
		Unit:        "{error}",
		Type:        counterType,
	},
	TotalProductCreationsMetric: {
		Description: "Total number of products created.",
		Unit:        "{product}",
		Type:        counterType,
	},
	TotalProductUpdatesMetric: {
		Description: "Total number of products updated.",
		Unit:        "{product}",
		Type:        counterType,
	},
	ProductInventoryCountMetric: {
		Description: "Current count of items in inventory for each product",
		Unit:        "{item}",
		Type:        observableGaugeType,
	},
}
