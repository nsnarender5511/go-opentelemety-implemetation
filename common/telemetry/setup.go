package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/resource"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otelgloballog "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	metricdata "go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func deltaTemporalitySelector(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	switch kind {
	case sdkmetric.InstrumentKindCounter,
		sdkmetric.InstrumentKindHistogram,
		sdkmetric.InstrumentKindObservableCounter:
		return metricdata.DeltaTemporality
	case sdkmetric.InstrumentKindUpDownCounter,
		sdkmetric.InstrumentKindObservableUpDownCounter:
		return metricdata.CumulativeTemporality
	default:

		return metricdata.CumulativeTemporality
	}
}

func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		if len(shutdownFuncs) == 0 {
			log.Println("OpenTelemetry shutdown: No providers initialized (likely non-production environment).")
			return nil
		}
		var shutdownErr error
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			shutdownErr = errors.Join(shutdownErr, shutdownFuncs[i](ctx))
		}
		shutdownFuncs = nil
		log.Println("OpenTelemetry resources shutdown sequence completed (production).")
		return shutdownErr
	}

	defer func() {
		if err != nil {
			log.Printf("ERROR: OpenTelemetry SDK initialization failed: %v", err)
			if shutdownErr := shutdown(context.Background()); shutdownErr != nil {
				log.Printf("ERROR: OTel cleanup after setup failure: %v", shutdownErr)
			}
		}
	}()

	isProduction := strings.ToLower(cfg.Environment) == "production"

	res, err := resource.NewResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	log.Println("OTel Resource created.")

	if isProduction {
		log.Println("Production environment detected. Initializing OTLP Trace, Metric, and Log providers.")

		connOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		}

		traceExporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
			otlptracegrpc.WithDialOption(connOpts...),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
		}
		ssp := sdktrace.NewSimpleSpanProcessor(traceExporter)
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(ssp),
		)
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
		shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
		log.Println("OTel TracerProvider initialized and set globally.")

		metricExporter, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
			otlpmetricgrpc.WithDialOption(connOpts...),
			otlpmetricgrpc.WithTemporalitySelector(deltaTemporalitySelector),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
		}
		reader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))
		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(reader),
		)
		otel.SetMeterProvider(mp)
		shutdownFuncs = append(shutdownFuncs, mp.Shutdown)
		log.Println("OTel MeterProvider initialized and set globally.")

		logExporter, err := otlploggrpc.New(ctx,
			otlploggrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
			otlploggrpc.WithDialOption(connOpts...),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
		}
		logProcessor := sdklog.NewBatchProcessor(logExporter)
		loggerProvider := sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(logProcessor),
		)
		otelgloballog.SetLoggerProvider(loggerProvider)
		shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
		log.Println("OTel LoggerProvider initialized and set globally.")

	} else {
		log.Printf("Non-production environment (%s) detected. Skipping OTLP exporter setup. Using No-Op providers.", cfg.Environment)
	}

	log.Println("OpenTelemetry SDK initialization sequence complete.")
	return shutdown, nil
}

func GetTracer(instrumentationName string) oteltrace.Tracer {
	return otel.Tracer(instrumentationName)
}

func GetMeter(instrumentationName string) metric.Meter {
	return otel.Meter(instrumentationName)
}
