package config

import (
	"time"
)

type Config struct {
	ProductServicePort string
	ServiceName        string
	ServiceVersion     string
	DataFilePath       string
	LogLevel           string
	LogFormat          string

	OtelExporterOtlpEndpoint string
	OtelExporterInsecure     bool
	OtelSampleRatio          float64
	OtelSamplerType          string
	OtelBatchTimeout         time.Duration
	OtelExporterOtlpTimeout  time.Duration
	OtelExporterOtlpHeaders  map[string]string
	OtelEnableExemplars      bool

	ShutdownTimeout       time.Duration
	ServerShutdownTimeout time.Duration
}

func GetHardcodedConfig() *Config {
	return &Config{
		ProductServicePort: "8082",
		ServiceName:        "product-service",
		ServiceVersion:     "1.0.0",
		DataFilePath:       "product-service/data.json",
		LogLevel:           "info",
		LogFormat:          "json",

		OtelExporterOtlpEndpoint: "localhost:4317",
		OtelExporterInsecure:     true,
		OtelSampleRatio:          1.0,
		OtelSamplerType:          "parentbased_traceidratio",
		OtelBatchTimeout:         5 * time.Second,
		OtelExporterOtlpTimeout:  10 * time.Second,
		OtelExporterOtlpHeaders:  make(map[string]string),
		OtelEnableExemplars:      false,

		ShutdownTimeout:       15 * time.Second,
		ServerShutdownTimeout: 10 * time.Second,
	}
}
