package exporter

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/manager"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func NewLogExporter(ctx context.Context, cfg *config.Config) (sdklog.Exporter, error) {
	conn, err := newOTLPGrpcConnection(ctx, cfg, "log")
	if err != nil {
		return nil, err
	}

	logClientOpts := []otlploggrpc.Option{
		otlploggrpc.WithGRPCConn(conn),
		otlploggrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}
	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		logClientOpts = append(logClientOpts, otlploggrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	logExporter, err := otlploggrpc.New(ctx, logClientOpts...)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create OTLP log exporter client: %w", err)
	}
	manager.GetLogger().Info("OTLP log exporter created")
	return logExporter, nil
}
