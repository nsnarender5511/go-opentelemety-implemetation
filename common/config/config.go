package config

import (
	"log"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

// Default configuration values
var defaultConfigs = map[string]interface{}{
	"PRODUCT_SERVICE_PORT":        "8082",
	"LOG_LEVEL":                   "info",
	"LOG_FORMAT":                  "text", // "text" or "json"
	"OTEL_SERVICE_NAME":           "product-service",
	"OTEL_EXPORTER_OTLP_ENDPOINT": "localhost:4317", // Default for local setup
	"OTEL_EXPORTER_INSECURE":      "true",           // Default insecure for local OTLP
	"OTEL_SAMPLE_RATIO":           1.0,              // Default to AlwaysSample
}

// Exported configuration variables, loaded from environment or defaults.
var (
	PRODUCT_SERVICE_PORT        string
	LOG_LEVEL                   string
	LOG_FORMAT                  string
	OTEL_SERVICE_NAME           string
	SERVICE_NAME                string // Alias for OTEL_SERVICE_NAME for convenience
	OTEL_EXPORTER_OTLP_ENDPOINT string
	OTEL_EXPORTER_INSECURE      bool
	OTEL_SAMPLE_RATIO           float64
)

// init loads configuration using Viper when the package is imported.
func init() {
	// Tell viper to look for environment variables
	viper.AutomaticEnv()

	// Set default values
	for key, value := range defaultConfigs {
		viper.SetDefault(key, value)
	}

	// Load configuration values into package variables
	PRODUCT_SERVICE_PORT = viper.GetString("PRODUCT_SERVICE_PORT")
	LOG_LEVEL = viper.GetString("LOG_LEVEL")
	LOG_FORMAT = viper.GetString("LOG_FORMAT")

	OTEL_SERVICE_NAME = viper.GetString("OTEL_SERVICE_NAME")
	SERVICE_NAME = OTEL_SERVICE_NAME // Set the alias

	OTEL_EXPORTER_OTLP_ENDPOINT = viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT")
	OTEL_EXPORTER_INSECURE = viper.GetBool("OTEL_EXPORTER_INSECURE")

	// Load and validate OTEL_SAMPLE_RATIO
	OTEL_SAMPLE_RATIO = viper.GetFloat64("OTEL_SAMPLE_RATIO")
	if OTEL_SAMPLE_RATIO < 0.0 || OTEL_SAMPLE_RATIO > 1.0 {
		log.Printf("Warning: Invalid OTEL_SAMPLE_RATIO '%.2f' from config/env, must be between 0.0 and 1.0. Defaulting to 1.0 (AlwaysSample).", OTEL_SAMPLE_RATIO)
		OTEL_SAMPLE_RATIO = 1.0
	}

	log.Println("Configuration loaded via Viper (env vars + defaults).")
	// Log key values for verification during startup
	log.Printf("  PRODUCT_SERVICE_PORT: %s", PRODUCT_SERVICE_PORT)
	log.Printf("  LOG_LEVEL: %s", LOG_LEVEL)
	log.Printf("  LOG_FORMAT: %s", LOG_FORMAT)
	log.Printf("  OTEL_SERVICE_NAME: %s", OTEL_SERVICE_NAME)
	log.Printf("  OTEL_EXPORTER_OTLP_ENDPOINT: %s", OTEL_EXPORTER_OTLP_ENDPOINT)
	log.Printf("  OTEL_EXPORTER_INSECURE: %t", OTEL_EXPORTER_INSECURE)
	log.Printf("  OTEL_SAMPLE_RATIO: %.2f", OTEL_SAMPLE_RATIO)
}

// GetEnvOrDefault retrieves an environment variable or returns a default value.
// Note: This is kept for potential utility but isn't used by the main viper loading.
func GetEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetEnvOrDefaultBool retrieves a boolean environment variable or returns a default.
// Note: This is kept for potential utility but isn't used by the main viper loading.
func GetEnvOrDefaultBool(key string, defaultValue bool) bool {
	valueStr := GetEnvOrDefault(key, strconv.FormatBool(defaultValue))
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid boolean value '%s' for env var '%s', using default %t. Error: %v", valueStr, key, defaultValue, err)
		return defaultValue
	}
	return value
}

// GetEnvOrDefaultFloat retrieves a float64 environment variable or returns a default.
// Note: This is kept for potential utility but isn't used by the main viper loading.
func GetEnvOrDefaultFloat(key string, defaultValue float64) float64 {
	valueStr := GetEnvOrDefault(key, strconv.FormatFloat(defaultValue, 'f', -1, 64))
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		log.Printf("Warning: Invalid float value '%s' for env var '%s', using default %f. Error: %v", valueStr, key, defaultValue, err)
		return defaultValue
	}
	return value
}
