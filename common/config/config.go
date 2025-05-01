package config

import (
	// Added for URI validation
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Initialize a minimal logger for config loading phase
var configLogger = logrus.New()

func init() {
	// Configure minimal logger - output to stderr, text format
	configLogger.SetOutput(os.Stderr)
	configLogger.SetFormatter(&logrus.TextFormatter{DisableColors: true, FullTimestamp: true})
	configLogger.SetLevel(logrus.InfoLevel)
}

// Constants for keys and validation lists
const (
	envServiceName            = "OTEL_SERVICE_NAME"
	envServiceVersion         = "SERVICE_VERSION"
	envOtelExporterEndpoint   = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envOtelExporterInsecure   = "OTEL_EXPORTER_INSECURE"
	envOtelSampleRatio        = "OTEL_SAMPLE_RATIO"
	envLogLevel               = "LOG_LEVEL"
	envLogFormat              = "LOG_FORMAT"
	envProductServicePort     = "PRODUCT_SERVICE_PORT"
	envDataFilePath           = "DATA_FILE_PATH"
	envShutdownTotalTimeout   = "SHUTDOWN_TOTAL_TIMEOUT_SEC"
	envShutdownServerTimeout  = "SHUTDOWN_SERVER_TIMEOUT_SEC"
	envShutdownOtelMinTimeout = "SHUTDOWN_OTEL_MIN_TIMEOUT_SEC"
	// Optional advanced OTel keys (if needed)
	envOtelBatchTimeoutMS      = "OTEL_BATCH_TIMEOUT_MS"
	envOtelMaxExportBatchSize  = "OTEL_MAX_EXPORT_BATCH_SIZE"
	envOtelLogMaxQueueSize     = "OTEL_LOG_MAX_QUEUE_SIZE"
	envOtelLogExportTimeoutMS  = "OTEL_LOG_EXPORT_TIMEOUT_MS"
	envOtelLogExportIntervalMS = "OTEL_LOG_EXPORT_INTERVAL_MS"
)

var (
	// Allowed values for validation
	allowedLogLevels  = map[string]struct{}{"debug": {}, "info": {}, "warn": {}, "error": {}, "fatal": {}, "panic": {}}
	allowedLogFormats = map[string]struct{}{"text": {}, "json": {}}
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

// WithEnv loads configuration from environment variables
func (c *Config) WithEnv() *Config {
	// Map of environment variables to config fields
	envMappings := map[string]*string{
		envServiceName:                &c.ServiceName,
		"SERVICE_VERSION":             &c.ServiceVersion,
		"OTEL_EXPORTER_OTLP_ENDPOINT": &c.OtelEndpoint,
		"LOG_LEVEL":                   &c.LogLevel,
		"LOG_FORMAT":                  &c.LogFormat,
		"PRODUCT_SERVICE_PORT":        &c.ProductServicePort,
		"DATA_FILE_PATH":              &c.DataFilePath,
	}

	// Apply environment variables if they exist
	for env, field := range envMappings {
		if val := os.Getenv(env); val != "" {
			*field = val
		}
	}

	// Handle boolean values
	if val := os.Getenv("OTEL_EXPORTER_INSECURE"); val != "" {
		c.OtelInsecure = strings.ToLower(val) == "true"
	}

	// Handle float values
	if val := os.Getenv("OTEL_SAMPLE_RATIO"); val != "" {
		if ratio, err := strconv.ParseFloat(val, 64); err == nil {
			c.OtelSampleRatio = ratio
		}
	}

	// Handle time durations (in seconds)
	if val := os.Getenv("SHUTDOWN_TOTAL_TIMEOUT_SEC"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds >= 0 {
			c.ShutdownTotalTimeout = time.Duration(seconds) * time.Second
		}
	}

	if val := os.Getenv("SHUTDOWN_SERVER_TIMEOUT_SEC"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds >= 0 {
			c.ShutdownServerTimeout = time.Duration(seconds) * time.Second
		}
	}

	if val := os.Getenv("SHUTDOWN_OTEL_MIN_TIMEOUT_SEC"); val != "" {
		if seconds, err := strconv.Atoi(val); err == nil && seconds >= 0 {
			c.ShutdownOtelMinTimeout = time.Duration(seconds) * time.Second
		}
	}

	// Handle time durations (in milliseconds)
	if val := os.Getenv(envOtelBatchTimeoutMS); val != "" {
		if ms, err := strconv.Atoi(val); err == nil && ms >= 0 {
			c.OtelBatchTimeout = time.Duration(ms) * time.Millisecond
		}
	}

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
		validator.RequireInRange("ProductServicePort", port, 1, 65535)
	} else {
		validator.AddError("ProductServicePort", "must be a valid integer")
	}

	validator.RequireInRange("OtelSampleRatio", c.OtelSampleRatio, 0.0, 1.0)

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
