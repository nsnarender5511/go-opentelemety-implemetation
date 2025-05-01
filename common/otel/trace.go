package otel

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type TracerProvider = sdktrace.TracerProvider

func newTracerProvider(ctx context.Context, res *Resource, logger *logrus.Logger) (*TracerProvider, ShutdownFunc, error) {
	logger.Debug("Creating tracer provider...")

	if res == nil {
		return nil, nil, fmt.Errorf("resource cannot be nil")
	}

	// Create OTLP trace exporter options using the helper (pass logger)
	exporterOpts := newOtlpTraceGrpcExporterOptions(logger)

	// Create the exporter
	exp, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Parse sample ratio
	sampleRatio := 1.0 // Default
	if config.OTEL_SAMPLE_RATIO != "" {
		parsedRatio, err := strconv.ParseFloat(config.OTEL_SAMPLE_RATIO, 64)
		if err == nil && parsedRatio >= 0.0 && parsedRatio <= 1.0 {
			sampleRatio = parsedRatio
		} else {
			logger.Warnf("Invalid OTEL_SAMPLE_RATIO '%s', using default 1.0", config.OTEL_SAMPLE_RATIO)
		}
	}

	// Parse batch timeout
	batchTimeout := defaultOtlpBatchTimeout // Use default from exporter_options.go
	if config.OTEL_BATCH_TIMEOUT_MS != "" {
		parsedMs, err := strconv.ParseInt(config.OTEL_BATCH_TIMEOUT_MS, 10, 64)
		if err == nil && parsedMs >= 0 {
			batchTimeout = time.Duration(parsedMs) * time.Millisecond
		} else {
			logger.Warnf("Invalid OTEL_BATCH_TIMEOUT_MS '%s', using default %v", config.OTEL_BATCH_TIMEOUT_MS, batchTimeout)
		}
	}

	// Create trace provider with the exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(batchTimeout), // Use parsed batch timeout
			sdktrace.WithMaxExportBatchSize(512),    // Default batch size
		),
		sdktrace.WithResource(res.Unwrap()),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(sampleRatio)), // Use parsed sample ratio
	)

	// Create a shutdown function that properly cleans up the tracer provider
	shutdown := func(shutdownCtx context.Context) error {
		logger.Debug("Shutting down tracer provider...")
		return tp.Shutdown(shutdownCtx)
	}

	logger.Info("Tracer provider created successfully")
	return tp, shutdown, nil
}

func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
