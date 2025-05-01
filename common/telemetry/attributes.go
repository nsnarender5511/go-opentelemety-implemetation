package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

// Standard Span Attribute Keys
// Using constants improves consistency and reduces typos.
// Prefixed with 'App' for custom attributes to avoid clashes with semantic conventions.
var (
	// Product specific
	AppProductIDKey    = attribute.Key("app.product.id")
	AppProductStockKey = attribute.Key("app.product.stock")
	AppProductCountKey = attribute.Key("app.product.count")

	// DB/Repo related
	DBSystemJSONFile         = semconv.DBSystemKey.String("jsonfile") // Reusable value
	DBOperationRead          = semconv.DBOperationKey.String("read")  // Reusable value
	DBResultCountKey         = attribute.Key("db.result.count")
	DBResultFoundKey         = attribute.Key("db.result.found")
	DBQueryParamProductIDKey = attribute.Key("db.query.parameter.product_id")
	FilePathKey              = semconv.CodeFilepathKey // Using semconv for file path

	// Operation status
	AppLookupSuccessKey     = attribute.Key("app.lookup.success")
	AppStockCheckSuccessKey = attribute.Key("app.check.success")

	// Add other common keys as needed...
)

// --- Log Field Keys ---
// Define constants for common structured logging field keys.
const (
	LogFieldProductID = "product_id" // Standardized log field key
	LogFieldCount     = "count"      // Standardized log field key
	LogFieldStock     = "stock"      // Standardized log field key
)
