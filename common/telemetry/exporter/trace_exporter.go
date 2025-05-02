package exporter

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
)

func NewTraceExporter(ctx context.Context, cfg *config.Config, logger *zap.Logger) (sdktrace.SpanExporter, error) {
	
	if logger == nil {
		logger = zap.NewNop()
	}

	
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlptracegrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}

	if cfg.OtelExporterInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
		logger.Warn("Using insecure gRPC connection for OTLP trace exporter")
	} else {
		
		logger.Info("Using secure gRPC connection for OTLP trace exporter")
		
	}

	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	
	
	

	
	traceExporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		
		return nil, fmt.Errorf("failed to create OTLP trace exporter client: %w", err)
	}
	
	logger.Info("OTLP trace exporter created successfully")

	return traceExporter, nil
}
