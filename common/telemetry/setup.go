package telemetry

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/narender/common/config"
	otelemetryResource "github.com/narender/common/telemetry/resource"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	otelgloballog "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)



func InitTelemetry(cfg *config.Config) error {

	
	

	
	res, err := otelemetryResource.NewResource(context.Background())
	if err != nil {
		
		log.Printf("ERROR: Failed to create OTel resource: %v\n", err)
		return fmt.Errorf("failed to create resource: %w", err)
	}
	log.Println("OTel Resource created.")

	
	if cfg.Environment == "production" {
		log.Println("Production environment detected. Initializing OTLP Trace, Metric, and Log providers.")

		ctx := context.Background()
		connOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}

		if err := setupOtlpTraceExporter(ctx, cfg, connOpts, res); err != nil {
			log.Printf("ERROR: OTLP Trace exporter setup failed: %v\n", err)
			return fmt.Errorf("trace exporter setup failed: %w", err)
		}

		if err := setupOtlpMetricExporter(ctx, cfg, connOpts, res); err != nil {
			log.Printf("ERROR: OTLP Metric exporter setup failed: %v\n", err)
			return fmt.Errorf("metric exporter setup failed: %w", err)
		}

		if err := setupOtlpLogExporter(ctx, cfg, connOpts, res); err != nil {
			log.Printf("ERROR: OTLP Log exporter setup failed: %v\n", err)
			return fmt.Errorf("log exporter setup failed: %w", err)
		}

	} else {
		
		log.Printf("Non-production environment (%s) detected. Skipping OTLP exporter setup. Using No-Op providers.", cfg.Environment)
		
	}

	log.Println("OpenTelemetry SDK initialization sequence complete.")
	return nil 
}


func setupOtlpTraceExporter(ctx context.Context, cfg *config.Config, connOpts []grpc.DialOption, res *sdkresource.Resource) error {
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlptracegrpc.WithDialOption(connOpts...),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(traceExporter)), 
		
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	log.Println("OTel TracerProvider initialized and set globally.")
	return nil
}


func setupOtlpMetricExporter(ctx context.Context, cfg *config.Config, connOpts []grpc.DialOption, res *sdkresource.Resource) error {
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlpmetricgrpc.WithDialOption(connOpts...),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	reader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)
	otel.SetMeterProvider(mp)
	log.Println("OTel MeterProvider initialized and set globally.")
	return nil
}


func setupOtlpLogExporter(ctx context.Context, cfg *config.Config, connOpts []grpc.DialOption, res *sdkresource.Resource) error {
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlploggrpc.WithDialOption(connOpts...),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	logProcessor := sdklog.NewBatchProcessor(logExporter)
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(logProcessor),
	)
	otelgloballog.SetLoggerProvider(loggerProvider) 
	log.Println("OTel LoggerProvider initialized and set globally.")
	return nil
}
