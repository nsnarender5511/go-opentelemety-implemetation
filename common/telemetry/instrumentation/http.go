package instrumentation

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)


func NewHTTPHandler(handler http.Handler, operationName string) http.Handler {
	
	return otelhttp.NewHandler(handler, operationName)
}
