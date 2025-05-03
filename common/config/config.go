package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
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

func LoadConfig(logger *zap.Logger) (*Config, error) {

	logger.Warn("LoadConfig returning default config; ensure environment variable parsing is implemented.")
	return GetDefaultConfig(), nil
}

func GetDefaultConfig() *Config {
	return &Config{
		ProductServicePort:       "8082",
		ServiceName:              "product-service",
		ServiceVersion:           "1.0.0",
		DataFilePath:             "/app/data.json",
		LogLevel:                 "info",
		Environment:              "development",
		OtelExporterOtlpEndpoint: "otel-collector:4317",
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return fallback
}

func getEnvAsMap(key string) map[string]string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return nil
	}
	pairs := strings.Split(valueStr, ",")
	headers := make(map[string]string)
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}
