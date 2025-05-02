package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Provider interface {
	GetValue(key string) (string, bool)
}

type EnvironmentProvider struct{}

func (p *EnvironmentProvider) GetValue(key string) (string, bool) {
	val, ok := os.LookupEnv(key)
	return val, ok
}

func NewEnvironmentProvider() *EnvironmentProvider {
	return &EnvironmentProvider{}
}

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

func LoadConfig(path string, provider Provider) (*Config, error) {
	if err := godotenv.Load(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("Warning: Failed to load .env file from %s: %v", path, err)
		}
	}

	cfg := &Config{}
	var loadErrors []error

	cfg.ProductServicePort = getString(provider, "PRODUCT_SERVICE_PORT", "8082")
	cfg.ServiceName = getString(provider, "OTEL_SERVICE_NAME", "product-service")
	cfg.ServiceVersion = getString(provider, "SERVICE_VERSION", "1.0.0")
	cfg.DataFilePath = getString(provider, "DATA_FILE_PATH", "data.json")
	cfg.LogLevel = getString(provider, "LOG_LEVEL", "info")
	cfg.LogFormat = getString(provider, "LOG_FORMAT", "text")

	cfg.OtelExporterOtlpEndpoint = getString(provider, "OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317")

	var err error // Declare error variable once for reuse
	cfg.OtelExporterInsecure, err = getBool(provider, "OTEL_EXPORTER_INSECURE", false)
	if err != nil {
		loadErrors = append(loadErrors, err)
	}

	cfg.OtelSampleRatio, err = getFloat64(provider, "OTEL_SAMPLE_RATIO", 1.0)
	if err != nil {
		loadErrors = append(loadErrors, err)
	}

	cfg.OtelSamplerType = getString(provider, "OTEL_SAMPLER_TYPE", "parentbased_traceidratio")
	cfg.OtelSamplerType = strings.ToLower(cfg.OtelSamplerType)
	switch cfg.OtelSamplerType {
	case "always_on", "always_off", "traceidratio", "parentbased_traceidratio":
		// valid
	default:
		// Format error directly here for sampler type
		loadErrors = append(loadErrors, fmt.Errorf("invalid OTEL_SAMPLER_TYPE: invalid value %q, must be one of [always_on, always_off, traceidratio, parentbased_traceidratio]", cfg.OtelSamplerType))
	}

	cfg.OtelBatchTimeout, err = getDuration(provider, "OTEL_BATCH_TIMEOUT_MS", 5000*time.Millisecond, time.Millisecond)
	if err != nil {
		loadErrors = append(loadErrors, err)
	}

	cfg.OtelExporterOtlpTimeout, err = getDuration(provider, "OTEL_EXPORTER_OTLP_TIMEOUT_MS", 10000*time.Millisecond, time.Millisecond)
	if err != nil {
		loadErrors = append(loadErrors, err)
	}

	cfg.OtelExporterOtlpHeaders = getHeaders(provider, "OTEL_EXPORTER_OTLP_HEADERS", "")

	cfg.OtelEnableExemplars, err = getBool(provider, "OTEL_ENABLE_EXEMPLARS", false)
	if err != nil {
		loadErrors = append(loadErrors, err)
	}

	cfg.ShutdownTimeout, err = getDuration(provider, "SHUTDOWN_TIMEOUT_SECONDS", 15*time.Second, time.Second)
	if err != nil {
		loadErrors = append(loadErrors, err)
	}

	cfg.ServerShutdownTimeout, err = getDuration(provider, "SERVER_SHUTDOWN_TIMEOUT_SECONDS", 10*time.Second, time.Second)
	if err != nil {
		loadErrors = append(loadErrors, err)
	}

	if len(loadErrors) > 0 {
		errMsg := "failed to load configuration:"
		for _, e := range loadErrors {
			errMsg += "\n\t- " + e.Error()
		}
		return nil, errors.New(errMsg)
	}

	return cfg, nil
}

func getString(provider Provider, key string, defaultValue string) string {
	value, ok := provider.GetValue(key)
	if !ok || value == "" {
		return defaultValue
	}
	return value
}

func getBool(provider Provider, key string, defaultValue bool) (bool, error) {
	return loadValue(provider, key, defaultValue, func(s string) (bool, error) {
		val, err := strconv.ParseBool(strings.ToLower(s))
		if err != nil {
			return false, fmt.Errorf("invalid boolean value %q", s)
		}
		return val, nil
	})
}

func getFloat64(provider Provider, key string, defaultValue float64) (float64, error) {
	return loadValue(provider, key, defaultValue, func(s string) (float64, error) {
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid float64 value %q", s)
		}
		return val, nil
	})
}

func getDuration(provider Provider, key string, defaultValue time.Duration, unit time.Duration) (time.Duration, error) {
	return loadValue(provider, key, defaultValue, func(s string) (time.Duration, error) {
		i, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("invalid integer value %q", s)
		}
		return time.Duration(i) * unit, nil
	})
}

func getHeaders(provider Provider, key string, defaultValue string) map[string]string {
	value := getString(provider, key, defaultValue)
	headers := make(map[string]string)
	if value == "" {
		return headers
	}
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			if key != "" {
				headers[key] = val
			}
		}
	}
	return headers
}

func loadValue[T any](provider Provider, key string, defaultValue T, parser func(string) (T, error)) (T, error) {
	valueStr, ok := provider.GetValue(key)
	if !ok || valueStr == "" {
		return defaultValue, nil
	}
	val, err := parser(valueStr)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("key %q: %w", key, err)
	}
	return val, nil
}
