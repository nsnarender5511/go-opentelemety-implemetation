package otel

import (
	"time"

	"github.com/narender/common/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Default OTLP exporter timeout
const defaultOtlpTimeout = 10 * time.Second

// newOtlpGrpcDialOptions creates common gRPC dial options for OTLP exporters.
func newOtlpGrpcDialOptions(cfg *config.Config) []grpc.DialOption {
	// For now, we always include WithBlock as it was in the original code.
	// Consider making this configurable if startup delays are a concern.
	return []grpc.DialOption{grpc.WithBlock()}
}

// newOtlpTraceGrpcExporterOptions creates common options for the OTLP trace gRPC exporter.
func newOtlpTraceGrpcExporterOptions(cfg *config.Config, dialOpts []grpc.DialOption) []otlptracegrpc.Option {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OtelEndpoint),
		otlptracegrpc.WithTimeout(defaultOtlpTimeout), // Use a consistent timeout
		otlptracegrpc.WithDialOption(dialOpts...),     // Spread the common dial options
	}

	if cfg.OtelInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		// Assume secure connection needed, use default TLS (often sufficient for localhost/internal)
		// For production, more sophisticated TLS config might be required.
		opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials())) // TODO: Revisit for production TLS
	}

	return opts
}

// newOtlpMetricGrpcExporterOptions creates common options for the OTLP metric gRPC exporter.
func newOtlpMetricGrpcExporterOptions(cfg *config.Config, dialOpts []grpc.DialOption) []otlpmetricgrpc.Option {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OtelEndpoint),
		otlpmetricgrpc.WithTimeout(defaultOtlpTimeout),
		otlpmetricgrpc.WithDialOption(dialOpts...),
	}

	if cfg.OtelInsecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	} else {
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials())) // TODO: Revisit for production TLS
	}

	return opts
}

// newOtlpLogGrpcExporterOptions creates common options for the OTLP log gRPC exporter.
func newOtlpLogGrpcExporterOptions(cfg *config.Config, dialOpts []grpc.DialOption) []otlploggrpc.Option {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.OtelEndpoint),
		otlploggrpc.WithTimeout(defaultOtlpTimeout),
		otlploggrpc.WithDialOption(dialOpts...),
	}

	if cfg.OtelInsecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	} else {
		opts = append(opts, otlploggrpc.WithTLSCredentials(insecure.NewCredentials())) // TODO: Revisit for production TLS
	}

	return opts
}
