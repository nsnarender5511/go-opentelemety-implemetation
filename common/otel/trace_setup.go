package otel

import (
	"context"
	"net/http"

	"github.com/narender/common/config"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func newSampler(cfg *config.Config) sdktrace.Sampler {
	log := GetLogger()
	samplerType := cfg.OtelSamplerType
	ratio := cfg.OtelSampleRatio

	log.Infof("Configuring OTel sampler type: %s", samplerType)

	switch samplerType {
	case "always_on":
		log.Info("Using AlwaysSample sampler.")
		return sdktrace.AlwaysSample()
	case "always_off":
		log.Info("Using NeverSample sampler.")
		return sdktrace.NeverSample()
	case "traceidratio":
		log.Infof("Using TraceIDRatioBased sampler with ratio: %.2f", ratio)
		return sdktrace.TraceIDRatioBased(ratio)
	case "parentbased_traceidratio":
		log.Infof("Using ParentBased(TraceIDRatioBased) sampler with ratio: %.2f", ratio)
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	default:
		log.Warnf("Invalid sampler type '%s' received, defaulting to parentbased_traceidratio with ratio: %.2f", samplerType, ratio)
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	}
}

func newTraceProvider(res *resource.Resource, exporter sdktrace.SpanExporter, sampler sdktrace.Sampler) (*sdktrace.TracerProvider, func(context.Context) error) {

	bspOpts := []sdktrace.BatchSpanProcessorOption{}
	bsp := sdktrace.NewBatchSpanProcessor(exporter, bspOpts...)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	GetLogger().Info("Tracer provider configured.")

	return tp, bsp.Shutdown
}

// NewHTTPHandler wraps an http.Handler with OpenTelemetry instrumentation.
// Moved from autoinstrument.go for better cohesion.
func NewHTTPHandler(handler http.Handler, operationName string) http.Handler {
	// Consider adding specific otelhttp options if needed later
	return otelhttp.NewHandler(handler, operationName)
}
