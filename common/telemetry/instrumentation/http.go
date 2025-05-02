package instrumentation

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewHTTPHandler wraps an http.Handler with OpenTelemetry instrumentation.
func NewHTTPHandler(handler http.Handler, operationName string) http.Handler {
	// Consider adding specific otelhttp options if needed later
	return otelhttp.NewHandler(handler, operationName)
}
