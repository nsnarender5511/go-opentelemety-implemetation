package config

import (
	"errors"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Initialize a minimal logger for config loading phase
var configLogger = logrus.New()

func init() {
	// Configure minimal logger - output to stderr, text format
	configLogger.SetOutput(os.Stderr)
	configLogger.SetFormatter(&logrus.TextFormatter{DisableColors: true, FullTimestamp: true})
	configLogger.SetLevel(logrus.InfoLevel) // Default level for config warnings
}

const (
	// Default values
	defaultLogLevel           = "info"
	defaultLogFormat          = "text"
	defaultOtelSampleRatio    = 1.0
	defaultOtelBatchTimeoutMS = 5000
	defaultOtelMaxExportSize  = 512
	defaultShutdownTotalSec   = 30
	defaultShutdownServerSec  = 15
	defaultShutdownOtelMinSec = 5

	// --- Log Batch Processor Defaults ---
	defaultOtelLogMaxQueueSize     = 2048
	defaultOtelLogExportTimeoutMS  = 30000 // 30 seconds
	defaultOtelLogExportIntervalMS = 1000  // 1 second
)

var (
	productServicePort   string
	logLevel             string
	logFormat            string
	otelServiceName      string
	serviceName          string
	serviceVersion       string
	otelExporterEndpoint string
	otelExporterInsecure bool
	otelSampleRatio      float64
	dataFilepath         string

	// New config vars
	otelBatchTimeout       time.Duration
	otelMaxExportBatchSize int
	shutdownTotalTimeout   time.Duration
	shutdownServerTimeout  time.Duration
	shutdownOtelMinTimeout time.Duration

	loadOnce sync.Once
	loadErr  error
)

// LoadConfig loads configuration from environment variables and .env files.
// It should be called once at application startup.
// It returns an error if loading fails or essential variables are missing.
func LoadConfig() error {
	viper.AutomaticEnv()
	loadOnce.Do(func() {
		var criticalErrors []error // Slice to accumulate critical errors

		// Use Info level for successful loads, Warn for issues
		configLogger.Debug("Attempting to load .env.default and .env files...")
		errDefault := godotenv.Load("../.env.default") // Load defaults first
		errEnv := godotenv.Load("../.env")             // Override with .env
		if errDefault != nil {
			if os.IsNotExist(errDefault) {
				configLogger.Debug("'.env.default' file not found, skipping.")
			} else {
				configLogger.WithError(errDefault).Warn("Error loading .env.default file (path: ../.env.default)")
			}
		}
		if errEnv != nil {
			if os.IsNotExist(errEnv) {
				configLogger.Info("'.env' file not found, relying on environment variables and defaults.")
			} else {
				configLogger.WithError(errEnv).Warn("Error loading .env file (path: ../.env)")
			}
		}

		productServicePort = getEnvStr("PRODUCT_SERVICE_PORT", "")
		logLevel = getEnvStr("LOG_LEVEL", defaultLogLevel)
		logFormat = getEnvStr("LOG_FORMAT", defaultLogFormat)
		otelServiceName = getEnvStr("OTEL_SERVICE_NAME", "")
		serviceVersion = getEnvStr("SERVICE_VERSION", "")
		otelExporterEndpoint = getEnvStr("OTEL_EXPORTER_OTLP_ENDPOINT", "")
		dataFilepath = getEnvStr("DATA_FILE_PATH", "")
		otelExporterInsecure = getEnvBool("OTEL_EXPORTER_INSECURE", false)
		otelSampleRatio = getEnvFloat("OTEL_SAMPLE_RATIO", defaultOtelSampleRatio)
		serviceName = otelServiceName // Use OTEL_SERVICE_NAME as the canonical service name

		otelBatchTimeout = getEnvDurationMS("OTEL_BATCH_TIMEOUT_MS", defaultOtelBatchTimeoutMS)
		otelMaxExportBatchSize = getEnvInt("OTEL_MAX_EXPORT_BATCH_SIZE", defaultOtelMaxExportSize)
		shutdownTotalTimeout = getEnvDurationSec("SHUTDOWN_TOTAL_TIMEOUT_SEC", defaultShutdownTotalSec)
		shutdownServerTimeout = getEnvDurationSec("SHUTDOWN_SERVER_TIMEOUT_SEC", defaultShutdownServerSec)
		shutdownOtelMinTimeout = getEnvDurationSec("SHUTDOWN_OTEL_MIN_TIMEOUT_SEC", defaultShutdownOtelMinSec)

		// --- Set Log Batch Processor Defaults in Viper ---
		viper.SetDefault("OTEL_LOG_MAX_QUEUE_SIZE", defaultOtelLogMaxQueueSize)
		viper.SetDefault("OTEL_LOG_EXPORT_TIMEOUT_MS", defaultOtelLogExportTimeoutMS)
		viper.SetDefault("OTEL_LOG_EXPORT_INTERVAL_MS", defaultOtelLogExportIntervalMS)

		if productServicePort == "" {
			err := errors.New("CRITICAL: PRODUCT_SERVICE_PORT environment variable is not set")
			configLogger.Error(err.Error())
			criticalErrors = append(criticalErrors, err)
		}
		if otelServiceName == "" {
			err := errors.New("CRITICAL: OTEL_SERVICE_NAME environment variable is not set")
			configLogger.Error(err.Error())
			criticalErrors = append(criticalErrors, err)
		}
		if otelExporterEndpoint == "" {
			err := errors.New("CRITICAL: OTEL_EXPORTER_OTLP_ENDPOINT environment variable is not set - Telemetry export will fail")
			configLogger.Error(err.Error())
			criticalErrors = append(criticalErrors, err)
		}
		if dataFilepath == "" {
			err := errors.New("CRITICAL: DATA_FILE_PATH environment variable is not set")
			configLogger.Error(err.Error())
			criticalErrors = append(criticalErrors, err)
		}

		if len(criticalErrors) > 0 {
			loadErr = errors.Join(criticalErrors...) // Combine errors
			configLogger.Errorf("Configuration loading failed due to %d critical missing variables.", len(criticalErrors))
			return // Return early if critical errors occurred
		}

		configLogger.Info("Configuration initialized successfully.")
		configLogger.WithFields(logrus.Fields{
			"otel_endpoint":        otelExporterEndpoint,
			"otel_insecure":        otelExporterInsecure,
			"otel_service_name":    otelServiceName,
			"otel_sample_ratio":    otelSampleRatio,
			"otel_batch_timeout":   otelBatchTimeout,
			"otel_max_batch_size":  otelMaxExportBatchSize,
			"log_level":            logLevel,
			"log_format":           logFormat,
			"product_service_port": productServicePort,
			"data_file_path":       dataFilepath,
		}).Info("Key configuration values loaded")

		configLogger.WithFields(logrus.Fields{
			"product_service_port": productServicePort,
			"log_level":            logLevel,
			"log_format":           logFormat,
			"otel_service_name":    otelServiceName,
			"service_version":      serviceVersion,
			"otel_endpoint":        otelExporterEndpoint,
			"otel_insecure":        otelExporterInsecure,
			"otel_sample_ratio":    otelSampleRatio,
			"data_file_path":       dataFilepath,
			"otel_batch_timeout":   otelBatchTimeout,
			"otel_max_batch_size":  otelMaxExportBatchSize,
			"shutdown_total":       shutdownTotalTimeout,
			"shutdown_server":      shutdownServerTimeout,
			"shutdown_otel_min":    shutdownOtelMinTimeout,
			"log_queue_size":       OtelLogMaxQueueSize(),
			"log_export_timeout":   OtelLogExportTimeout(),
			"log_export_interval":  OtelLogExportInterval(),
		}).Debug("Full loaded configuration values")
	})
	return loadErr
}

// --- Helper functions for reading/parsing env vars with defaults ---

func getEnvStr(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if valueStr, exists := os.LookupEnv(key); exists {
		if val, err := strconv.ParseBool(valueStr); err == nil {
			return val
		}
		configLogger.Warnf("Could not parse env var %s='%s' as bool, using default %v", key, valueStr, fallback)
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		if val, err := strconv.Atoi(valueStr); err == nil {
			return val
		}
		configLogger.Warnf("Could not parse env var %s='%s' as int, using default %d", key, valueStr, fallback)
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if valueStr, exists := os.LookupEnv(key); exists {
		if val, err := strconv.ParseFloat(valueStr, 64); err == nil {
			return val
		}
		configLogger.Warnf("Could not parse env var %s='%s' as float64, using default %f", key, valueStr, fallback)
	}
	return fallback
}

func getEnvDurationSec(key string, fallbackSec int) time.Duration {
	fbDur := time.Duration(fallbackSec) * time.Second
	if valueStr, exists := os.LookupEnv(key); exists {
		if val, err := strconv.Atoi(valueStr); err == nil {
			return time.Duration(val) * time.Second
		}
		configLogger.Warnf("Could not parse env var %s='%s' as int (seconds), using default %s", key, valueStr, fbDur)
	}
	return fbDur
}

func getEnvDurationMS(key string, fallbackMS int) time.Duration {
	fbDur := time.Duration(fallbackMS) * time.Millisecond
	if valueStr, exists := os.LookupEnv(key); exists {
		if val, err := strconv.Atoi(valueStr); err == nil {
			return time.Duration(val) * time.Millisecond
		}
		configLogger.Warnf("Could not parse env var %s='%s' as int (milliseconds), using default %s", key, valueStr, fbDur)
	}
	return fbDur
}

// Getters for configuration values

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

func ServiceName() string {
	return serviceName
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

// --- Log Batch Processor Accessors ---

func OtelLogMaxQueueSize() int {
	return viper.GetInt("OTEL_LOG_MAX_QUEUE_SIZE")
}

func OtelLogExportTimeout() time.Duration {
	return viper.GetDuration("OTEL_LOG_EXPORT_TIMEOUT_MS") * time.Millisecond
}

func OtelLogExportInterval() time.Duration {
	return viper.GetDuration("OTEL_LOG_EXPORT_INTERVAL_MS") * time.Millisecond
}
