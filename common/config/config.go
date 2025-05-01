package config

import (
	"log"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

var defaultConfigs = map[string]interface{}{
	"PRODUCT_SERVICE_PORT":        "8082",
	"LOG_LEVEL":                   "info",
	"LOG_FORMAT":                  "text",
	"OTEL_SERVICE_NAME":           "product-service",
	"OTEL_EXPORTER_OTLP_ENDPOINT": "localhost:4317",
	"OTEL_EXPORTER_INSECURE":      "true",
	"OTEL_SAMPLE_RATIO":           1.0, // Default to AlwaysSample
}

var (
	PRODUCT_SERVICE_PORT        string
	LOG_LEVEL                   string
	LOG_FORMAT                  string
	OTEL_SERVICE_NAME           string
	SERVICE_NAME                string
	OTEL_EXPORTER_OTLP_ENDPOINT string
	OTEL_EXPORTER_INSECURE      bool
	OTEL_SAMPLE_RATIO           float64
)

func init() {
	viper.AutomaticEnv()

	for key, value := range defaultConfigs {
		viper.SetDefault(key, value)
	}

	PRODUCT_SERVICE_PORT = viper.GetString("PRODUCT_SERVICE_PORT")
	LOG_LEVEL = viper.GetString("LOG_LEVEL")
	LOG_FORMAT = viper.GetString("LOG_FORMAT")

	OTEL_SERVICE_NAME = viper.GetString("OTEL_SERVICE_NAME")
	SERVICE_NAME = OTEL_SERVICE_NAME

	OTEL_EXPORTER_OTLP_ENDPOINT = viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT")
	OTEL_EXPORTER_INSECURE = viper.GetBool("OTEL_EXPORTER_INSECURE")

	// Load and validate OTEL_SAMPLE_RATIO
	OTEL_SAMPLE_RATIO = viper.GetFloat64("OTEL_SAMPLE_RATIO")
	if OTEL_SAMPLE_RATIO < 0.0 || OTEL_SAMPLE_RATIO > 1.0 {
		log.Printf("Warning: Invalid OTEL_SAMPLE_RATIO '%f' from config/env, must be between 0.0 and 1.0. Defaulting to 1.0 (AlwaysSample).", OTEL_SAMPLE_RATIO)
		OTEL_SAMPLE_RATIO = 1.0
	}
}

// GetEnvOrDefault retrieves an environment variable or returns a default value.
func GetEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetEnvOrDefaultBool retrieves a boolean environment variable or returns a default.
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
func GetEnvOrDefaultFloat(key string, defaultValue float64) float64 {
	valueStr := GetEnvOrDefault(key, strconv.FormatFloat(defaultValue, 'f', -1, 64))
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		log.Printf("Warning: Invalid float value '%s' for env var '%s', using default %f. Error: %v", valueStr, key, defaultValue, err)
		return defaultValue
	}
	return value
}
