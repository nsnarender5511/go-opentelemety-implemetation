package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/resource"
	"go.uber.org/zap"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
	tempLogger := zap.NewNop()

	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var shutdownErr error
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			shutdownErr = errors.Join(shutdownErr, shutdownFuncs[i](ctx))
		}
		shutdownFuncs = nil
		tempLogger.Debug("OpenTelemetry resources shutdown sequence completed.")
		return shutdownErr
	}

	defer func() {
		if err != nil {
			tempLogger.Error("OpenTelemetry SDK initialization failed", zap.Error(err))
			if shutdownErr := shutdown(context.Background()); shutdownErr != nil {
				tempLogger.Error("Error during OTel cleanup after setup failure", zap.Error(shutdownErr))
			}
		}
	}()

	res, err := resource.NewResource(ctx, cfg)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create resource: %w", err)
	}
	tempLogger.Debug("Resource created", zap.Any("attributes", res.Attributes()))

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	ssp := sdktrace.NewSimpleSpanProcessor(traceExporter)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())),
		sdktrace.WithSpanProcessor(ssp),
	)
	otel.SetTracerProvider(tp)
	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	tempLogger.Debug("TracerProvider initialized and set globally.")

	tempLogger.Debug("Setting up OTLP Metric Exporter", zap.String("endpoint", cfg.OtelExporterOtlpEndpoint))
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithTemporalitySelector(func(kind sdkmetric.InstrumentKind) metricdata.Temporality {
			if kind == sdkmetric.InstrumentKindCounter || kind == sdkmetric.InstrumentKindHistogram {
				return metricdata.DeltaTemporality
			}
			return metricdata.CumulativeTemporality
		}),
		otlpmetricgrpc.WithTimeout(5*time.Second),
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		return shutdown, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	reader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)
	otel.SetMeterProvider(mp)
	shutdownFuncs = append(shutdownFuncs, mp.Shutdown)
	tempLogger.Debug("MeterProvider initialized and set globally.")

	tempLogger.Info("OpenTelemetry SDK initialized successfully (Traces and Metrics).")

	return shutdown, nil
}

func GetTracer(instrumentationName string) oteltrace.Tracer {

	return otel.Tracer(instrumentationName)
}

func GetMeter(instrumentationName string) metric.Meter {

	return otel.Meter(instrumentationName)
}
