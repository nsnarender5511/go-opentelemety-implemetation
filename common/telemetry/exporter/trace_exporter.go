package exporter

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func NewTraceExporter(ctx context.Context, cfg *config.Config, logger *logrus.Logger) (sdktrace.SpanExporter, error) {
	// Construct options directly for the standard exporter
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlptracegrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}

	if cfg.OtelExporterInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
		logger.Warn("Using insecure gRPC connection for OTLP trace exporter")
	} else {
		// Secure is the default, but log for clarity if needed
		logger.Info("Using secure gRPC connection for OTLP trace exporter")
		// opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(&tls.Config{}))) // Example if specific TLS needed
	}

	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	// Add other options like retry, compression if configured in cfg
	// Example: opts = append(opts, otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{...}))
	// Example: opts = append(opts, otlptracegrpc.WithCompression(otlptracegrpc.GzipCompression))

	// Create the exporter directly using the configured options
	traceExporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		// No connection to close here as the exporter handles it
		return nil, fmt.Errorf("failed to create OTLP trace exporter client: %w", err)
	}
	logger.Info("OTLP trace exporter created successfully")

	return traceExporter, nil
}
