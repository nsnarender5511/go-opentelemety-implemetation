package telemetry

import "go.opentelemetry.io/otel/attribute"

// Standard Application Attribute Keys
// Consider reviewing OpenTelemetry Semantic Conventions for standard keys
// (e.g., https://opentelemetry.io/docs/specs/semconv/)
// before adding purely custom keys.
const (
	AttrAppProductID     = attribute.Key("app.product.id")
	AttrAppProductStock  = attribute.Key("app.product.stock")
	AttrAppLookupSuccess = attribute.Key("app.lookup.success")
	AttrAppStockCheck    = attribute.Key("app.stock.check.success")
	AttrProductCount     = attribute.Key("product.count")
	// Add other common application-specific keys here
)
