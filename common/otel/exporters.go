package otel

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/narender/common/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func newOTLPGrpcConnection(ctx context.Context, cfg *config.Config, signalType string) (*grpc.ClientConn, error) {
	logger := GetLogger()
	var transportCreds credentials.TransportCredentials
	if cfg.OtelExporterInsecure {
		transportCreds = insecure.NewCredentials()
		logger.Warnf("Using insecure gRPC connection for OTLP %s exporter", signalType)
	} else {
		logger.Infof("Using secure gRPC connection for OTLP %s exporter", signalType)
		transportCreds = credentials.NewTLS(&tls.Config{})
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
	}

	dialCtx, cancel := context.WithTimeout(ctx, cfg.OtelExporterOtlpTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, cfg.OtelExporterOtlpEndpoint, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial OTLP %s exporter endpoint %s: %w", signalType, cfg.OtelExporterOtlpEndpoint, err)
	}
	logger.Infof("Successfully connected to OTLP gRPC endpoint for %s: %s", signalType, cfg.OtelExporterOtlpEndpoint)
	return conn, nil
}

func newTraceExporter(ctx context.Context, cfg *config.Config) (sdktrace.SpanExporter, error) {
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
	GetLogger().Info("OTLP trace exporter created")

	return traceExporter, nil
}

func newMetricExporter(ctx context.Context, cfg *config.Config) (sdkmetric.Exporter, error) {
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
	GetLogger().Info("OTLP metric exporter created")
	return metricExporter, nil
}

func newLogExporter(ctx context.Context, cfg *config.Config) (sdklog.Exporter, error) {
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
	GetLogger().Info("OTLP log exporter created")
	return logExporter, nil
}
