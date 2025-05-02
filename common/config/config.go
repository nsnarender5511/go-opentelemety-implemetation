package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration parameters for the application.
type Config struct {
	// Service specific
	ProductServicePort string
	ServiceName        string
	ServiceVersion     string
	DataFilePath       string

	// Logging
	LogLevel  string
	LogFormat string // e.g., "json", "text"

	// OTel related
	OtelExporterOtlpEndpoint string
	OtelExporterInsecure     bool              // Use bool for clarity
	OtelSampleRatio          float64           // Use float for ratio
	OtelBatchTimeout         time.Duration     // Use time.Duration
	OtelExporterOtlpTimeout  time.Duration     // Use time.Duration
	OtelExporterOtlpHeaders  map[string]string // Use map for headers

	// Shutdown
	ShutdownTimeout       time.Duration // Use time.Duration
	ServerShutdownTimeout time.Duration // Use time.Duration
}

// LoadConfig loads configuration from environment variables or a .env file.
// It prioritizes environment variables over the .env file.
func LoadConfig(path string) (*Config, error) {
	// Attempt to load .env file, ignore error if it doesn't exist
	_ = godotenv.Load(path)

	cfg := &Config{}
	var err error

	cfg.ProductServicePort = getenv("PRODUCT_SERVICE_PORT", "8080")
	cfg.ServiceName = getenv("OTEL_SERVICE_NAME", "product-service") // Use OTEL_SERVICE_NAME as ServiceName
	cfg.ServiceVersion = getenv("SERVICE_VERSION", "1.0.0")
	cfg.DataFilePath = getenv("DATA_FILE_PATH", "") // Expect path to be set explicitly

	cfg.LogLevel = getenv("LOG_LEVEL", "info")
	cfg.LogFormat = getenv("LOG_FORMAT", "json")

	cfg.OtelExporterOtlpEndpoint = getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317")

	insecureStr := getenv("OTEL_EXPORTER_INSECURE", "false")
	cfg.OtelExporterInsecure, err = strconv.ParseBool(insecureStr)
	if err != nil {
		return nil, fmt.Errorf("invalid OTEL_EXPORTER_INSECURE value %q: %w", insecureStr, err)
	}

	sampleRatioStr := getenv("OTEL_SAMPLE_RATIO", "1.0")
	cfg.OtelSampleRatio, err = strconv.ParseFloat(sampleRatioStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid OTEL_SAMPLE_RATIO value %q: %w", sampleRatioStr, err)
	}

	cfg.OtelBatchTimeout, err = parseDurationMs(getenv("OTEL_BATCH_TIMEOUT_MS", "5000"), "OTEL_BATCH_TIMEOUT_MS")
	if err != nil {
		return nil, err
	}

	cfg.OtelExporterOtlpTimeout, err = parseDurationMs(getenv("OTEL_EXPORTER_OTLP_TIMEOUT_MS", "10000"), "OTEL_EXPORTER_OTLP_TIMEOUT_MS")
	if err != nil {
		return nil, err
	}

	cfg.OtelExporterOtlpHeaders = parseHeaders(getenv("OTEL_EXPORTER_OTLP_HEADERS", ""))

	cfg.ShutdownTimeout, err = parseDurationSec(getenv("SHUTDOWN_TIMEOUT_SECONDS", "10"), "SHUTDOWN_TIMEOUT_SECONDS")
	if err != nil {
		return nil, err
	}

	cfg.ServerShutdownTimeout, err = parseDurationSec(getenv("SERVER_SHUTDOWN_TIMEOUT_SECONDS", "10"), "SERVER_SHUTDOWN_TIMEOUT_SECONDS")
	if err != nil {
		return nil, err
	}

	if cfg.DataFilePath == "" {
		// Optionally return an error if the data file path is mandatory
		// return nil, fmt.Errorf("DATA_FILE_PATH environment variable is not set")
		fmt.Println("Warning: DATA_FILE_PATH environment variable is not set.") // Or just log a warning
	}

	return cfg, nil
}

// getenv retrieves the value of the environment variable named by the key.
// It returns the defaultValue if the variable is not set or empty.
func getenv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// parseDurationMs parses a string representing milliseconds into a time.Duration.
func parseDurationMs(value string, envVarName string) (time.Duration, error) {
	ms, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q: must be an integer (milliseconds): %w", envVarName, value, err)
	}
	return time.Duration(ms) * time.Millisecond, nil
}

// parseDurationSec parses a string representing seconds into a time.Duration.
func parseDurationSec(value string, envVarName string) (time.Duration, error) {
	sec, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q: must be an integer (seconds): %w", envVarName, value, err)
	}
	return time.Duration(sec) * time.Second, nil
}

// parseHeaders parses a comma-separated key=value string into a map.
func parseHeaders(value string) map[string]string {
	headers := make(map[string]string)
	if value == "" {
		return headers
	}
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			if key != "" {
				headers[key] = val
			}
		}
	}
	return headers
}
