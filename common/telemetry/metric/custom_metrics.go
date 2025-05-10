package metric

// --- Metric Configuration Constants ---

type metricType string

const (
	counterType         metricType = "counter"
	histogramType       metricType = "histogram"
	observableGaugeType metricType = "observable_gauge"
	floatCounterType    metricType = "float_counter"

	// Define metric names as constants for type safety and easier refactoring
	ProductStockCountMetric = "app.product.stock.count"
	AppRevenueTotalMetric   = "app.revenue.total"
	AppItemsSoldCountMetric = "app.items.sold.count"
	AppErrorCountMetric     = "app.error.count"

	// Standard attribute names
	AttrProductName     = "product.name"
	AttrProductCategory = "product.category"
	AttrStockLevel      = "product.stock.level"
	AttrRevenue         = "transaction.revenue"
	AttrQuantity        = "transaction.quantity"
	AttrErrorType       = "error.type"
	AttrOperation       = "operation"
	AttrComponent       = "component"
	AttrCustomMetric    = "custom.metric"
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
	ProductStockCountMetric: {
		Description: "Current count of items in inventory for each product. Attributes: product.name, product.category",
		Unit:        "{item}",
		Type:        observableGaugeType,
	},
	AppRevenueTotalMetric: {
		Description: "Total revenue generated from product sales. Attributes: product.name, product.category, currency_code",
		Unit:        "1",
		Type:        floatCounterType,
	},
	AppItemsSoldCountMetric: {
		Description: "Total number of items sold. Attributes: product.name, product.category",
		Unit:        "{item}",
		Type:        counterType,
	},
	AppErrorCountMetric: {
		Description: "Count of errors by error type, operation, and component",
		Unit:        "{error}",
		Type:        counterType,
	},
}
