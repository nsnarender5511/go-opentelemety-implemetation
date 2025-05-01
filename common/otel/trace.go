package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TracerProvider is the OpenTelemetry tracer provider interface
type TracerProvider = sdktrace.TracerProvider

// newTracerProvider creates a new tracer provider with the provided configuration
func newTracerProvider(ctx context.Context, cfg *config.Config, res *Resource, logger *logrus.Logger) (*TracerProvider, ShutdownFunc, error) {
	logger.Debug("Creating tracer provider...")

	if res == nil {
		return nil, nil, fmt.Errorf("resource cannot be nil")
	}

	// Create OTLP exporter
	var opts []otlptracegrpc.Option
	opts = append(opts, otlptracegrpc.WithEndpoint(cfg.OtelEndpoint))

	// Add insecure option if needed
	if cfg.OtelInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		// For secure connections, configure TLS if needed
		opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
	}

	// Add timeout option
	opts = append(opts, otlptracegrpc.WithTimeout(10*time.Second))

	// Add gRPC connection options
	opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithBlock()))

	// Create the exporter
	exp, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create trace provider with the exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(cfg.OtelBatchTimeout),
			sdktrace.WithMaxExportBatchSize(512), // Default batch size
		),
		sdktrace.WithResource(res.Unwrap()),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.OtelSampleRatio)),
	)

	// Create a shutdown function that properly cleans up the tracer provider
	shutdown := func(shutdownCtx context.Context) error {
		logger.Debug("Shutting down tracer provider...")
		return tp.Shutdown(shutdownCtx)
	}

	logger.Info("Tracer provider created successfully")
	return tp, shutdown, nil
}

// GetTracer returns a new tracer from the global provider
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
