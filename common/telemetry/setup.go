package telemetry

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/narender/common/config"
	commonlog "github.com/narender/common/log"
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

func InitTelemetry(cfg *config.Config) (*slog.Logger, error) {

	if err := commonlog.Init(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize application logger: %w", err)
	}
	logger := commonlog.L

	res, err := resource.NewResource(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	log.Println("OTel Resource created.")

	if cfg.Environment == "production" {
		log.Println("Production environment detected. Initializing OTLP Trace, Metric, and Log providers.")

		connOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		}

		otlpEndpoint := cfg.OtelExporterOtlpEndpoint

		traceExporter, err := otlptracegrpc.New(context.Background(),
			otlptracegrpc.WithEndpoint(otlpEndpoint),
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
		log.Println("OTel TracerProvider initialized and set globally.")

		metricExporter, err := otlpmetricgrpc.New(context.Background(),
			otlpmetricgrpc.WithEndpoint(otlpEndpoint),
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
		log.Println("OTel MeterProvider initialized and set globally.")

		logExporter, err := otlploggrpc.New(context.Background(),
			otlploggrpc.WithEndpoint(otlpEndpoint),
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
		log.Println("OTel LoggerProvider initialized and set globally.")

	} else {
		log.Printf("Non-production environment (%s) detected. Skipping OTLP exporter setup. Using No-Op providers.", cfg.Environment)
	}

	log.Println("OpenTelemetry SDK initialization sequence complete.")
	return logger, nil
}

func GetTracer(instrumentationName string) oteltrace.Tracer {
	return otel.Tracer(instrumentationName)
}

func GetMeter(instrumentationName string) metric.Meter {
	return otel.Meter(instrumentationName)
}
