package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

// Standard Span Attribute Keys
// Using constants improves consistency and reduces typos.
// Prefixes align with semantic conventions (db, app, code, etc.) where possible.
var (
	// General Application
	AppOperationKey = attribute.Key("app.operation") // e.g., "json_unmarshal"

	// Product specific
	AppProductIDKey    = attribute.Key("app.product.id")    // Application-level product identifier
	AppProductStockKey = attribute.Key("app.product.stock") // Application-level stock value
	AppProductCountKey = attribute.Key("app.product.count") // Application-level count of products

	// DB/Repo related (using Semantic Conventions where applicable)
	DBSystemKey              = semconv.DBSystemKey                      // Value: "json_file"
	DBOperationKey           = semconv.DBOperationKey                   // Value: "read"
	DBStatementKey           = semconv.DBStatementKey                   // Value: "FindAll", "FindByProductID", etc.
	DBFilePathKey            = attribute.Key("db.file.path")            // Specific key for file path in DB context
	DBResultCountKey         = attribute.Key("db.result.count")         // Count of results (e.g., from FindAll)
	DBResultFoundKey         = attribute.Key("db.result.found")         // Boolean indicating if FindBy* found data
	DBQueryParamProductIDKey = attribute.Key("db.parameter.product_id") // Parameter used in query

	// File operations (using Semantic Conventions)
	CodeFilepathKey = semconv.CodeFilepathKey // General file path (used if not DB specific)

	// Operation status (application level)
	AppLookupSuccessKey     = attribute.Key("app.lookup.success")    // Success status for product lookup
	AppStockCheckSuccessKey = attribute.Key("app.check.success")     // Success status for stock check
	AppOperationSuccessKey  = attribute.Key("app.operation.success") // General operation success boolean

	// Error related (using Semantic Conventions)
	ErrorTypeKey = semconv.ExceptionTypeKey // Type of error encountered
)

// --- Log Field Keys ---
// Define constants for common structured logging field keys.
const (
	LogFieldProductID = "product_id" // Standardized log field key
	LogFieldCount     = "count"      // Standardized log field key
	LogFieldStock     = "stock"      // Standardized log field key
)
