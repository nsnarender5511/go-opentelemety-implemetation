package attributes

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)


var (
	DBSystemKey         = semconv.DBSystemKey
	DBOperationKey      = semconv.DBOperationKey
	DBStatementKey      = semconv.DBStatementKey
	NetPeerNameKey      = semconv.NetPeerNameKey
	HTTPMethodKey       = semconv.HTTPMethodKey
	HTTPRouteKey        = semconv.HTTPRouteKey
	HTTPStatusCodeKey   = semconv.HTTPStatusCodeKey
	NetHostNameKey      = semconv.NetHostNameKey
	NetHostPortKey      = semconv.NetHostPortKey
	URLPathKey          = semconv.URLPathKey
	URLSchemeKey        = semconv.URLSchemeKey
	ExceptionMessageKey = semconv.ExceptionMessageKey
	ExceptionTypeKey    = semconv.ExceptionTypeKey

	
	AttrDBFilePathKey      = attribute.Key("db.file.path")
	AttrAppProductIDKey    = attribute.Key("app.product.id")
	AttrProductNewStockKey = attribute.Key("product.new_stock")
	AttrAppProductCount    = attribute.Key("app.products.count")
)
