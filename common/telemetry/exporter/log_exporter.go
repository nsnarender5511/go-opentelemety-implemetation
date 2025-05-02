package exporter

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	logSdk "go.opentelemetry.io/otel/sdk/log"
)

func NewLogExporter(ctx context.Context, cfg *config.Config, logger *logrus.Logger) (logSdk.Exporter, error) {

	// Construct options directly for the standard exporter
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlploggrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}

	if cfg.OtelExporterInsecure {
		opts = append(opts, otlploggrpc.WithInsecure())
		logger.Warn("OTLP log exporter: using insecure gRPC connection.")
	} else {
		logger.Info("OTLP log exporter: using secure gRPC connection.")
	}

	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	// Add other options like retry, compression if configured in cfg

	logExporter, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		logger.Errorf("Failed to create OTLP log exporter client: %v", err)
		return nil, fmt.Errorf("failed to create OTLP log exporter client: %w", err)
	}

	logger.Info("OTLP log exporter created successfully.")
	return logExporter, nil
}
