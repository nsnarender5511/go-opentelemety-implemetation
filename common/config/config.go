package config

import (
	"log"
	"time"
)

type Config struct {
	ServiceName              string `env:"SERVICE_NAME,required"`
	ServiceVersion           string `env:"SERVICE_VERSION,required"`
	Environment              string `env:"ENVIRONMENT,default=development"`
	ProductServicePort       string `env:"PRODUCT_SERVICE_PORT,default=8082"`
	LogLevel                 string `env:"LOG_LEVEL,default=info"`
	DataFilePath             string `env:"DATA_FILE_PATH,default=./data.json"`
	OtelExporterOtlpEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,default=otel-collector:4317"`

	OtelExporterInsecure    bool
	OtelSampleRatio         float64
	OtelSamplerType         string
	OtelBatchTimeout        time.Duration
	OtelExporterOtlpTimeout time.Duration
	OtelExporterOtlpHeaders map[string]string
	OtelEnableExemplars     bool

	ShutdownTimeout       time.Duration
	ServerShutdownTimeout time.Duration

	SimulateDelayEnabled bool `mapstructure:"SIMULATE_DELAY_ENABLED"`
	SimulateDelayMinMs   int  `mapstructure:"SIMULATE_DELAY_MIN_MS"`
	SimulateDelayMaxMs   int  `mapstructure:"SIMULATE_DELAY_MAX_MS"`
}

func LoadConfig() (*Config, error) {
	log.Println("WARN: LoadConfig returning default config; ensure environment variable parsing is implemented.")
	return GetDefaultConfig(), nil
}

func GetDefaultConfig() *Config {
	return &Config{
		ProductServicePort:       "8082",
		ServiceName:              "product-service",
		ServiceVersion:           "1.0.0",
		DataFilePath:             "/app/data.json",
		LogLevel:                 "info",
		Environment:              "production",
		OtelExporterOtlpEndpoint: "otel-collector:4317",
	}
}

