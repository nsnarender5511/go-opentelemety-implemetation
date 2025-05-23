package trace

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"

	"github.com/narender/common/config"
)

func SetupOtlpTraceExporter(ctx context.Context, cfg *config.Config, connOpts []grpc.DialOption, res *resource.Resource) error {
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTEL_ENDPOINT),
		otlptracegrpc.WithDialOption(connOpts...),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithSpanProcessor(trace.NewBatchSpanProcessor(traceExporter)),
	)
	// Set the global TracerProvider and Propagator for the application.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	log.Println("OTel TracerProvider initialized and set globally.")
	return nil
}
