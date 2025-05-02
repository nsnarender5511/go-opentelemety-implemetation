package exporter

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/manager"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func NewTraceExporter(ctx context.Context, cfg *config.Config) (sdktrace.SpanExporter, error) {
	conn, err := newOTLPGrpcConnection(ctx, cfg, "trace")
	if err != nil {
		return nil, err
	}

	traceClientOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithGRPCConn(conn),
		otlptracegrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}
	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		traceClientOpts = append(traceClientOpts, otlptracegrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	traceExporter, err := otlptracegrpc.New(ctx, traceClientOpts...)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create OTLP trace exporter client: %w", err)
	}
	manager.GetLogger().Info("OTLP trace exporter created")

	return traceExporter, nil
}
