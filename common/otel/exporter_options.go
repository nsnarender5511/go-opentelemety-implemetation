package otel

import (
	"strconv"
	"strings"
	"time"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const defaultOtlpTimeout = 10 * time.Second
const defaultOtlpBatchTimeout = 5 * time.Second

func parseKeyValueMap(input string) map[string]string {
	if input == "" {
		return nil
	}
	result := make(map[string]string)
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			if key != "" {
				result[key] = value
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func getEnvDuration(envVar string, defaultVal time.Duration, logger *logrus.Logger) time.Duration {
	valStr := config.OTEL_EXPORTER_OTLP_TIMEOUT_MS // Assuming this is the var name, adjust if needed
	if envVar != "" {
		valStr = envVar
	}
	if valStr != "" {
		if parsed, err := strconv.ParseInt(valStr, 10, 64); err == nil && parsed >= 0 {
			return time.Duration(parsed) * time.Millisecond
		}
		logger.Warnf("Invalid duration value '%s' for OTLP timeout env var, using default %v", valStr, defaultVal)
	}
	return defaultVal
}

func newOtlpGrpcDialOptions() []grpc.DialOption {
	// For now, we always include WithBlock as it was in the original code.
	// Consider making this configurable if startup delays are a concern.
	return []grpc.DialOption{grpc.WithBlock()}
}

func newOtlpTraceGrpcExporterOptions(logger *logrus.Logger) []otlptracegrpc.Option {
	// Get config values
	endpoint := config.OTEL_EXPORTER_OTLP_ENDPOINT
	insecureStr := config.OTEL_EXPORTER_INSECURE
	headersStr := config.OTEL_EXPORTER_OTLP_HEADERS
	timeoutStr := config.OTEL_EXPORTER_OTLP_TIMEOUT_MS

	insecure := strings.ToLower(insecureStr) == "true"
	headers := parseKeyValueMap(headersStr)
	timeout := getEnvDuration(timeoutStr, defaultOtlpTimeout, logger)

	dialOpts := newOtlpGrpcDialOptions()

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithTimeout(timeout),        // Use parsed timeout
		otlptracegrpc.WithDialOption(dialOpts...), // Spread the common dial options
	}

	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		// Use system cert pool by default.
		// #nosec G402 -- Defaulting to system cert pool is reasonable baseline.
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
	}

	if len(headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(headers)) // Add parsed headers
	}

	return opts
}

func newOtlpMetricGrpcExporterOptions(logger *logrus.Logger) []otlpmetricgrpc.Option {
	// Get config values
	endpoint := config.OTEL_EXPORTER_OTLP_ENDPOINT
	insecureStr := config.OTEL_EXPORTER_INSECURE
	headersStr := config.OTEL_EXPORTER_OTLP_HEADERS
	timeoutStr := config.OTEL_EXPORTER_OTLP_TIMEOUT_MS

	insecure := strings.ToLower(insecureStr) == "true"
	headers := parseKeyValueMap(headersStr)
	timeout := getEnvDuration(timeoutStr, defaultOtlpTimeout, logger)

	dialOpts := newOtlpGrpcDialOptions()

	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithTimeout(timeout), // Use parsed timeout
		otlpmetricgrpc.WithDialOption(dialOpts...),
	}

	if insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	} else {
		// Use system cert pool by default.
		// #nosec G402 -- Defaulting to system cert pool is reasonable baseline.
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
	}

	if len(headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(headers)) // Add parsed headers
	}

	return opts
}

func newOtlpLogGrpcExporterOptions(logger *logrus.Logger) []otlploggrpc.Option {
	// Get config values
	endpoint := config.OTEL_EXPORTER_OTLP_ENDPOINT
	insecureStr := config.OTEL_EXPORTER_INSECURE
	headersStr := config.OTEL_EXPORTER_OTLP_HEADERS
	timeoutStr := config.OTEL_EXPORTER_OTLP_TIMEOUT_MS

	insecure := strings.ToLower(insecureStr) == "true"
	headers := parseKeyValueMap(headersStr)
	timeout := getEnvDuration(timeoutStr, defaultOtlpTimeout, logger)

	dialOpts := newOtlpGrpcDialOptions()

	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithTimeout(timeout), // Use parsed timeout
		otlploggrpc.WithDialOption(dialOpts...),
	}

	if insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	} else {
		// Use system cert pool by default.
		// #nosec G402 -- Defaulting to system cert pool is reasonable baseline.
		opts = append(opts, otlploggrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
	}

	if len(headers) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(headers)) // Add parsed headers
	}

	return opts
}
