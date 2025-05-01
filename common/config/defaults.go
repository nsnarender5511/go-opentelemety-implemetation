package config

import "time"

// NewDefaultConfig provides a configuration with sensible defaults
func NewDefaultConfig() *Config {
	return &Config{
		// Service information
		ServiceName:    "service",
		ServiceVersion: "dev",

		// OpenTelemetry configuration
		OtelEndpoint:     "http://localhost:4317",
		OtelInsecure:     false,
		OtelSampleRatio:  1.0,
		OtelBatchTimeout: 5 * time.Second,

		// Logging configuration
		LogLevel:  "info",
		LogFormat: "text",

		// Application-specific settings
		ProductServicePort: "8080",

		// Shutdown timeouts
		ShutdownTotalTimeout:   30 * time.Second,
		ShutdownServerTimeout:  10 * time.Second,
		ShutdownOtelMinTimeout: 5 * time.Second,
	}
}
