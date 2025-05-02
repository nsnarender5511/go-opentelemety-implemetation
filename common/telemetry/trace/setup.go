package trace

import (
	"context"

	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/manager"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func NewSampler(cfg *config.Config) sdktrace.Sampler {
	log := manager.GetLogger()
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

func NewTraceProvider(res *resource.Resource, exporter sdktrace.SpanExporter, sampler sdktrace.Sampler) (*sdktrace.TracerProvider, func(context.Context) error) {

	bspOpts := []sdktrace.BatchSpanProcessorOption{}
	bsp := sdktrace.NewBatchSpanProcessor(exporter, bspOpts...)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	manager.GetLogger().Info("Tracer provider configured.")

	return tp, bsp.Shutdown
}
