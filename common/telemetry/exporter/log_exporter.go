package exporter

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
)


func NewLogExporter(ctx context.Context, cfg *config.Config, logger *zap.Logger) (sdklog.Exporter, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlploggrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}

	if cfg.OtelExporterInsecure {
		opts = append(opts, otlploggrpc.WithInsecure())
		logger.Warn("Using insecure gRPC connection for OTLP log exporter")
	} else {
		logger.Info("Using secure gRPC connection for OTLP log exporter")
		
	}

	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
		logger.Debug("Adding custom headers to OTLP log exporter", zap.Int("count", len(cfg.OtelExporterOtlpHeaders)))
	}

	logExporter, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		logger.Error("Failed to create OTLP log exporter client", zap.Error(err))
		return nil, fmt.Errorf("failed to create OTLP log exporter client: %w", err)
	}

	logger.Info("OTLP log exporter created successfully", zap.String("endpoint", cfg.OtelExporterOtlpEndpoint))
	return logExporter, nil
}
