package config

import (
	"errors"
	"fmt"
	"net/url" // Added for URI validation
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
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

// Package-level variables to hold configuration
var (
	productServicePort   string
	logLevel             string
	logFormat            string
	otelServiceName      string // Note: Renamed from serviceName for clarity with OTEL_SERVICE_NAME env var
	serviceVersion       string
	otelExporterEndpoint string
	otelExporterInsecure bool
	otelSampleRatio      float64
	dataFilepath         string

	shutdownTotalTimeout   time.Duration
	shutdownServerTimeout  time.Duration
	shutdownOtelMinTimeout time.Duration

	// Optional advanced OTel values (can be added if needed)
	otelBatchTimeout       time.Duration
	otelMaxExportBatchSize int
	otelLogMaxQueueSize    int
	otelLogExportTimeout   time.Duration
	otelLogExportInterval  time.Duration

	loadOnce sync.Once
	loadErr  error
)

// LoadConfig loads configuration from environment variables and .env files.
// It relies solely on godotenv and standard Go libraries.
// It returns an aggregated error if loading or validation fails.
func LoadConfig() error {
	loadOnce.Do(func() {
		configLogger.Info("Attempting to load configuration...")

		// Load .env files first (default then override)
		_ = godotenv.Load(".env.default") // Ignore error if default doesn't exist
		_ = godotenv.Load(".env")         // Ignore error if override doesn't exist

		var validationErrors []string

		// --- Read and Validate Required Variables ---

		otelServiceName, loadErr = getEnvStrRequired(envServiceName)
		if loadErr != nil {
			validationErrors = append(validationErrors, loadErr.Error())
		}
		serviceVersion, loadErr = getEnvStrRequired(envServiceVersion)
		if loadErr != nil {
			validationErrors = append(validationErrors, loadErr.Error())
		}

		otelExporterEndpoint, loadErr = getEnvStrRequired(envOtelExporterEndpoint)
		if loadErr != nil {
			validationErrors = append(validationErrors, loadErr.Error())
		} else if _, err := url.ParseRequestURI(otelExporterEndpoint); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("invalid URI format for %s: %v", envOtelExporterEndpoint, err))
		}

		logLevel, loadErr = getEnvStrRequired(envLogLevel)
		if loadErr != nil {
			validationErrors = append(validationErrors, loadErr.Error())
		} else if _, ok := allowedLogLevels[strings.ToLower(logLevel)]; !ok {
			validationErrors = append(validationErrors, fmt.Sprintf("invalid value for %s: '%s', allowed: %v", envLogLevel, logLevel, getMapKeys(allowedLogLevels)))
		}

		logFormat, loadErr = getEnvStrRequired(envLogFormat)
		if loadErr != nil {
			validationErrors = append(validationErrors, loadErr.Error())
		} else if _, ok := allowedLogFormats[strings.ToLower(logFormat)]; !ok {
			validationErrors = append(validationErrors, fmt.Sprintf("invalid value for %s: '%s', allowed: %v", envLogFormat, logFormat, getMapKeys(allowedLogFormats)))
		}

		productServicePort, loadErr = getEnvStrRequired(envProductServicePort)
		if loadErr != nil {
			validationErrors = append(validationErrors, loadErr.Error())
		} else {
			if portNum, err := strconv.Atoi(productServicePort); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("invalid integer format for %s: '%s'", envProductServicePort, productServicePort))
			} else if portNum <= 0 || portNum > 65535 {
				validationErrors = append(validationErrors, fmt.Sprintf("port %s must be between 1 and 65535, got %d", envProductServicePort, portNum))
			}
		}

		dataFilepath, loadErr = getEnvStrRequired(envDataFilePath)
		if loadErr != nil {
			validationErrors = append(validationErrors, loadErr.Error())
		} else if _, err := os.Stat(dataFilepath); os.IsNotExist(err) {
			validationErrors = append(validationErrors, fmt.Sprintf("file specified by %s does not exist: %s", envDataFilePath, dataFilepath))
		}

		shutdownTotalSec, loadErr := getEnvIntRequired(envShutdownTotalTimeout)
		if loadErr != nil {
			validationErrors = append(validationErrors, loadErr.Error())
		} else if shutdownTotalSec < 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("%s must be >= 0, got %d", envShutdownTotalTimeout, shutdownTotalSec))
		} else {
			shutdownTotalTimeout = time.Duration(shutdownTotalSec) * time.Second
		}

		shutdownServerSec, loadErr := getEnvIntRequired(envShutdownServerTimeout)
		if loadErr != nil {
			validationErrors = append(validationErrors, loadErr.Error())
		} else if shutdownServerSec < 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("%s must be >= 0, got %d", envShutdownServerTimeout, shutdownServerSec))
		} else {
			shutdownServerTimeout = time.Duration(shutdownServerSec) * time.Second
		}

		shutdownOtelMinSec, loadErr := getEnvIntRequired(envShutdownOtelMinTimeout)
		if loadErr != nil {
			validationErrors = append(validationErrors, loadErr.Error())
		} else if shutdownOtelMinSec < 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("%s must be >= 0, got %d", envShutdownOtelMinTimeout, shutdownOtelMinSec))
		} else {
			shutdownOtelMinTimeout = time.Duration(shutdownOtelMinSec) * time.Second
		}

		// --- Read Optional Variables (Provide defaults here or handle missing values) ---
		// Example: Read OTEL_EXPORTER_INSECURE with a default of false
		var insecureErr error
		otelExporterInsecure, insecureErr = getEnvBool(envOtelExporterInsecure, false)
		if insecureErr != nil { // Log parse errors for optional bools but don't fail validation
			configLogger.Warnf("Could not parse %s: %v, using default 'false'", envOtelExporterInsecure, insecureErr)
		}

		// Example: Read OTEL_SAMPLE_RATIO with a default of 1.0 and validation
		var sampleRatioErr error
		otelSampleRatio, sampleRatioErr = getEnvFloat(envOtelSampleRatio, 1.0)
		if sampleRatioErr != nil {
			configLogger.Warnf("Could not parse %s: %v, using default '1.0'", envOtelSampleRatio, sampleRatioErr)
			// If parsing fails, default is used, which is valid. Only validate if parsing succeeds.
		} else if otelSampleRatio < 0 || otelSampleRatio > 1.0 {
			validationErrors = append(validationErrors, fmt.Sprintf("%s must be between 0.0 and 1.0, got %f", envOtelSampleRatio, otelSampleRatio))
		}

		// --- Read Optional Advanced OTel Vars (Example) ---
		var err error
		otelBatchTimeout, err = getEnvDurationMS(envOtelBatchTimeoutMS, 5000)
		if err != nil {
			configLogger.Warnf("Error parsing %s: %v, using default", envOtelBatchTimeoutMS, err)
		}
		otelMaxExportBatchSize, err = getEnvInt(envOtelMaxExportBatchSize, 512)
		if err != nil {
			configLogger.Warnf("Error parsing %s: %v, using default", envOtelMaxExportBatchSize, err)
		}
		otelLogMaxQueueSize, err = getEnvInt(envOtelLogMaxQueueSize, 2048)
		if err != nil {
			configLogger.Warnf("Error parsing %s: %v, using default", envOtelLogMaxQueueSize, err)
		}
		otelLogExportTimeout, err = getEnvDurationMS(envOtelLogExportTimeoutMS, 30000)
		if err != nil {
			configLogger.Warnf("Error parsing %s: %v, using default", envOtelLogExportTimeoutMS, err)
		}
		otelLogExportInterval, err = getEnvDurationMS(envOtelLogExportIntervalMS, 1000)
		if err != nil {
			configLogger.Warnf("Error parsing %s: %v, using default", envOtelLogExportIntervalMS, err)
		}

		// --- Final Validation Check ---
		if len(validationErrors) > 0 {
			finalErrorMsg := fmt.Sprintf("configuration validation failed: %s", strings.Join(validationErrors, "; "))
			loadErr = errors.New(finalErrorMsg) // Set the aggregated error to loadErr
			configLogger.Error(loadErr)
			return // Stop further processing
		}

		// --- Log Loaded Configuration ---
		configLogger.Info("Configuration loaded and validated successfully.")
		configLogger.WithFields(logrus.Fields{
			"otel_service_name":    otelServiceName,
			"service_version":      serviceVersion,
			"otel_endpoint":        otelExporterEndpoint,
			"otel_insecure":        otelExporterInsecure,
			"otel_sample_ratio":    otelSampleRatio,
			"log_level":            logLevel,
			"log_format":           logFormat,
			"product_service_port": productServicePort,
			"data_file_path":       dataFilepath,
			"shutdown_total":       shutdownTotalTimeout,
			"shutdown_server":      shutdownServerTimeout,
			"shutdown_otel_min":    shutdownOtelMinTimeout,
			// Optional advanced values
			"otel_batch_timeout":  otelBatchTimeout,
			"otel_max_batch_size": otelMaxExportBatchSize,
			"log_queue_size":      otelLogMaxQueueSize,
			"log_export_timeout":  otelLogExportTimeout,
			"log_export_interval": otelLogExportInterval,
		}).Debug("Full loaded configuration values")

	})

	return loadErr // Return the error captured within loadOnce.Do (nil on success)
}

// --- Helper function to get keys from a map[string]struct{} for error messages ---
func getMapKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// --- Helper functions for reading/parsing env vars ---

// Reads a required string environment variable.
func getEnvStrRequired(key string) (string, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	if value == "" {
		return "", fmt.Errorf("required environment variable %s must not be empty", key)
	}
	return value, nil
}

// Reads a required integer environment variable.
func getEnvIntRequired(key string) (int, error) {
	valueStr, err := getEnvStrRequired(key)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid integer format for %s: %w", key, err)
	}
	return value, nil
}

// Reads an optional boolean environment variable.
// Returns fallback if not set. Returns error ONLY on parse failure.
func getEnvBool(key string, fallback bool) (bool, error) {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return fallback, nil
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		// Don't wrap here, let caller decide how to handle parse error vs missing
		return fallback, fmt.Errorf("invalid boolean format for %s ('%s'): %w", key, valueStr, err)
	}
	return value, nil
}

// Reads an optional integer environment variable.
// Returns fallback if not set. Returns error ONLY on parse failure.
func getEnvInt(key string, fallback int) (int, error) {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return fallback, nil
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallback, fmt.Errorf("invalid integer format for %s ('%s'): %w", key, valueStr, err)
	}
	return value, nil
}

// Reads an optional float environment variable.
// Returns fallback if not set. Returns error ONLY on parse failure.
func getEnvFloat(key string, fallback float64) (float64, error) {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return fallback, nil
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return fallback, fmt.Errorf("invalid float format for %s ('%s'): %w", key, valueStr, err)
	}
	return value, nil
}

// Reads an optional duration environment variable (expecting milliseconds).
// Returns fallback if not set. Returns error ONLY on parse failure.
func getEnvDurationMS(key string, fallbackMS int) (time.Duration, error) {
	fallbackDuration := time.Duration(fallbackMS) * time.Millisecond
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return fallbackDuration, nil
	}
	valueMS, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallbackDuration, fmt.Errorf("invalid integer format for duration (ms) %s ('%s'): %w", key, valueStr, err)
	}
	return time.Duration(valueMS) * time.Millisecond, nil
}

// Reads an optional duration environment variable (expecting seconds).
// Returns fallback if not set. Returns error ONLY on parse failure.
func getEnvDurationSec(key string, fallbackSec int) (time.Duration, error) {
	fallbackDuration := time.Duration(fallbackSec) * time.Second
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return fallbackDuration, nil
	}
	valueSec, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallbackDuration, fmt.Errorf("invalid integer format for duration (sec) %s ('%s'): %w", key, valueStr, err)
	}
	return time.Duration(valueSec) * time.Second, nil
}

// --- Getters for configuration values (Unchanged) ---

func ProductServicePort() string {
	return productServicePort
}

func LogLevel() string {
	return logLevel
}

func LogFormat() string {
	return logFormat
}

func OtelServiceName() string {
	return otelServiceName
}

func ServiceName() string { // Keep original getter name if preferred
	return otelServiceName
}

func ServiceVersion() string {
	return serviceVersion
}

func OtelExporterEndpoint() string {
	return otelExporterEndpoint
}

func IsOtelExporterInsecure() bool {
	return otelExporterInsecure
}

func OtelSampleRatio() float64 {
	return otelSampleRatio
}

func DataFilepath() string {
	return dataFilepath
}

func OtelBatchTimeout() time.Duration {
	return otelBatchTimeout
}

func OtelMaxExportBatchSize() int {
	return otelMaxExportBatchSize
}

func ShutdownTotalTimeout() time.Duration {
	return shutdownTotalTimeout
}

func ShutdownServerTimeout() time.Duration {
	return shutdownServerTimeout
}

func ShutdownOtelMinTimeout() time.Duration {
	return shutdownOtelMinTimeout
}

// --- Log Batch Processor Accessors (read from globals) ---

func OtelLogMaxQueueSize() int {
	return otelLogMaxQueueSize
}

func OtelLogExportTimeout() time.Duration {
	return otelLogExportTimeout
}

func OtelLogExportInterval() time.Duration {
	return otelLogExportInterval
}
