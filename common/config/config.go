package config

import (
	// Added for URI validation
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Constants for keys and validation lists
const (
	envServiceName        = "OTEL_SERVICE_NAME"
	envOtelBatchTimeoutMS = "OTEL_BATCH_TIMEOUT_MS"
)

// Config holds all configuration settings
type Config struct {
	// Service information
	ServiceName    string
	ServiceVersion string

	// OpenTelemetry configuration
	OtelEndpoint     string
	OtelInsecure     bool
	OtelSampleRatio  float64
	OtelBatchTimeout time.Duration

	// Logging configuration
	LogLevel  string
	LogFormat string

	// Application-specific settings
	ProductServicePort string
	DataFilePath       string

	// Shutdown timeouts
	ShutdownTotalTimeout   time.Duration
	ShutdownServerTimeout  time.Duration
	ShutdownOtelMinTimeout time.Duration
}

// NewConfig creates a new Config with the provided options
func NewConfig(opts ...Option) *Config {
	c := &Config{
		// Set sensible defaults
		ServiceName:            "service",
		ServiceVersion:         "dev",
		OtelEndpoint:           "http://localhost:4317",
		OtelInsecure:           false,
		OtelSampleRatio:        1.0,
		OtelBatchTimeout:       5 * time.Second,
		LogLevel:               "info",
		LogFormat:              "text",
		ProductServicePort:     "8080",
		ShutdownTotalTimeout:   30 * time.Second,
		ShutdownServerTimeout:  10 * time.Second,
		ShutdownOtelMinTimeout: 5 * time.Second,
	}

	// Apply all options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Helper functions for environment variable loading
func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return strings.ToLower(val) == "true"
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration, unit time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			return time.Duration(parsed) * unit
		}
	}
	return defaultVal
}

// WithEnv loads configuration from environment variables
func (c *Config) WithEnv() *Config {
	c.ServiceName = getEnvString(envServiceName, c.ServiceName)
	c.ServiceVersion = getEnvString("SERVICE_VERSION", c.ServiceVersion)
	c.OtelEndpoint = getEnvString("OTEL_EXPORTER_OTLP_ENDPOINT", c.OtelEndpoint)
	c.LogLevel = getEnvString("LOG_LEVEL", c.LogLevel)
	c.LogFormat = getEnvString("LOG_FORMAT", c.LogFormat)
	c.ProductServicePort = getEnvString("PRODUCT_SERVICE_PORT", c.ProductServicePort)
	c.DataFilePath = getEnvString("DATA_FILE_PATH", c.DataFilePath)

	c.OtelInsecure = getEnvBool("OTEL_EXPORTER_INSECURE", c.OtelInsecure)
	c.OtelSampleRatio = getEnvFloat("OTEL_SAMPLE_RATIO", c.OtelSampleRatio)

	c.ShutdownTotalTimeout = getEnvDuration("SHUTDOWN_TOTAL_TIMEOUT_SEC", c.ShutdownTotalTimeout, time.Second)
	c.ShutdownServerTimeout = getEnvDuration("SHUTDOWN_SERVER_TIMEOUT_SEC", c.ShutdownServerTimeout, time.Second)
	c.ShutdownOtelMinTimeout = getEnvDuration("SHUTDOWN_OTEL_MIN_TIMEOUT_SEC", c.ShutdownOtelMinTimeout, time.Second)

	c.OtelBatchTimeout = getEnvDuration(envOtelBatchTimeoutMS, c.OtelBatchTimeout, time.Millisecond)

	return c
}

// Validate validates the configuration
func (c *Config) Validate() []error {
	validator := NewValidator()

	// Validate required fields
	validator.RequireNonEmpty("ServiceName", c.ServiceName)
	validator.RequireNonEmpty("ServiceVersion", c.ServiceVersion)
	validator.RequireNonEmpty("OtelEndpoint", c.OtelEndpoint)
	validator.RequireNonEmpty("LogLevel", c.LogLevel)
	validator.RequireNonEmpty("LogFormat", c.LogFormat)
	validator.RequireNonEmpty("ProductServicePort", c.ProductServicePort)

	// Validate values in allowed sets
	validator.RequireOneOf("LogLevel", c.LogLevel, []string{"debug", "info", "warn", "error", "fatal", "panic"})
	validator.RequireOneOf("LogFormat", c.LogFormat, []string{"text", "json"})

	// Validate numeric ranges
	if port, err := strconv.Atoi(c.ProductServicePort); err == nil {
		RequireInRange(validator, "ProductServicePort", port, 1, 65535)
	} else {
		validator.AddError("ProductServicePort", "must be a valid integer")
	}

	RequireInRange(validator, "OtelSampleRatio", c.OtelSampleRatio, 0.0, 1.0)

	// Validate file existence if specified
	if c.DataFilePath != "" {
		if _, err := os.Stat(c.DataFilePath); os.IsNotExist(err) {
			validator.AddError("DataFilePath", "file does not exist: "+c.DataFilePath)
		}
	}

	return validator.Errors()
}

// Log logs the current configuration
func (c *Config) Log() {
	logrus.WithFields(logrus.Fields{
		"service_name":      c.ServiceName,
		"service_version":   c.ServiceVersion,
		"otel_endpoint":     c.OtelEndpoint,
		"otel_insecure":     c.OtelInsecure,
		"otel_sample_ratio": c.OtelSampleRatio,
		"log_level":         c.LogLevel,
		"log_format":        c.LogFormat,
		"port":              c.ProductServicePort,
		"data_file_path":    c.DataFilePath,
		"shutdown_total":    c.ShutdownTotalTimeout,
		"shutdown_server":   c.ShutdownServerTimeout,
		"shutdown_otel":     c.ShutdownOtelMinTimeout,
	}).Info("Configuration loaded")
}
