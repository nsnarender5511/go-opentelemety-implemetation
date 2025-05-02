package exporter

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/manager"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func NewMetricExporter(ctx context.Context, cfg *config.Config) (sdkmetric.Exporter, error) {
	conn, err := newOTLPGrpcConnection(ctx, cfg, "metric")
	if err != nil {
		return nil, err
	}

	metricClientOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithGRPCConn(conn),
		otlpmetricgrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}
	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		metricClientOpts = append(metricClientOpts, otlpmetricgrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	metricExporter, err := otlpmetricgrpc.New(ctx, metricClientOpts...)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create OTLP metric exporter client: %w", err)
	}
	manager.GetLogger().Info("OTLP metric exporter created")
	return metricExporter, nil
}
