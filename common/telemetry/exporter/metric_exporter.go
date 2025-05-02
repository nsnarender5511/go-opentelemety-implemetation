package exporter

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func NewMetricExporter(ctx context.Context, cfg *config.Config, logger *logrus.Logger) (sdkmetric.Exporter, error) {
	// Construct options directly for the standard exporter
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlpmetricgrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
		// Note: Add Temporality preference if needed via WithTemporalitySelector
		// Note: Add Aggregation preference if needed via WithAggregationSelector
	}

	if cfg.OtelExporterInsecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
		logger.Warn("Using insecure gRPC connection for OTLP metric exporter")
	} else {
		logger.Info("Using secure gRPC connection for OTLP metric exporter")
	}

	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	// Add other options like retry, compression if configured in cfg
	// Example: opts = append(opts, otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{...}))
	// Example: opts = append(opts, otlpmetricgrpc.WithCompression(otlpmetricgrpc.GzipCompression))

	// Create the exporter directly using the configured options
	metricExporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter client: %w", err)
	}
	logger.Info("OTLP metric exporter created successfully")
	return metricExporter, nil
}
