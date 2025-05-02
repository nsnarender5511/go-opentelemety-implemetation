package exporter

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

func NewMetricExporter(ctx context.Context, cfg *config.Config, logger *zap.Logger) (sdkmetric.Exporter, error) {
	
	if logger == nil {
		logger = zap.NewNop()
	}

	
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlpmetricgrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
		
		
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

	
	
	

	
	metricExporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter client: %w", err)
	}
	logger.Info("OTLP metric exporter created successfully")
	return metricExporter, nil
}
