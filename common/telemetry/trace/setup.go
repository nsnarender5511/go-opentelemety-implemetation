package trace

import (
	"context"

	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/manager"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

func NewSampler(cfg *config.Config) sdktrace.Sampler {
	log := manager.GetLogger()
	samplerType := cfg.OtelSamplerType
	ratio := cfg.OtelSampleRatio

	log.Info("Configuring OTel sampler", zap.String("type", samplerType), zap.Float64("ratio", ratio))

	switch samplerType {
	case "always_on":
		log.Info("Using AlwaysSample sampler.")
		return sdktrace.AlwaysSample()
	case "always_off":
		log.Info("Using NeverSample sampler.")
		return sdktrace.NeverSample()
	case "traceidratio":
		log.Info("Using TraceIDRatioBased sampler", zap.Float64("ratio", ratio))
		return sdktrace.TraceIDRatioBased(ratio)
	case "parentbased_traceidratio":
		log.Info("Using ParentBased(TraceIDRatioBased) sampler", zap.Float64("ratio", ratio))
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	default:
		log.Warn("Invalid sampler type received, defaulting",
			zap.String("invalid_type", samplerType),
			zap.String("default_type", "parentbased_traceidratio"),
			zap.Float64("ratio", ratio),
		)
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
