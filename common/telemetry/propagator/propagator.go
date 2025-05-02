package propagator

import (
	"github.com/narender/common/telemetry/manager" // For logging
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// SetupPropagators configures the global OTel propagators.
func SetupPropagators() {
	logger := manager.GetLogger()
	prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(prop)
	logger.Debug("Global TextMapPropagator configured.")
}
