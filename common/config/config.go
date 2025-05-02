package config

import (
	"errors"
	"fmt"
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
	
	_ = godotenv.Load(path)

	cfg := &Config{}
	var err error
	var loadErrors []error

	
	cfg.ProductServicePort = getString(provider, "PRODUCT_SERVICE_PORT", "8082") 
	cfg.ServiceName = getString(provider, "OTEL_SERVICE_NAME", "product-service")
	cfg.ServiceVersion = getString(provider, "SERVICE_VERSION", "1.0.0")
	cfg.DataFilePath = getString(provider, "DATA_FILE_PATH", "data.json") 
	cfg.LogLevel = getString(provider, "LOG_LEVEL", "info")
	cfg.LogFormat = getString(provider, "LOG_FORMAT", "text") 

	
	cfg.OtelExporterOtlpEndpoint = getString(provider, "OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317")
	cfg.OtelExporterInsecure, err = getBool(provider, "OTEL_EXPORTER_INSECURE", false)
	if err != nil {
		loadErrors = append(loadErrors, fmt.Errorf("invalid OTEL_EXPORTER_INSECURE: %w", err))
	}
	cfg.OtelSampleRatio, err = getFloat64(provider, "OTEL_SAMPLE_RATIO", 1.0)
	if err != nil {
		loadErrors = append(loadErrors, fmt.Errorf("invalid OTEL_SAMPLE_RATIO: %w", err))
	}
	cfg.OtelSamplerType = getString(provider, "OTEL_SAMPLER_TYPE", "parentbased_traceidratio")
	cfg.OtelSamplerType = strings.ToLower(cfg.OtelSamplerType) 
	cfg.OtelBatchTimeout, err = getDurationMs(provider, "OTEL_BATCH_TIMEOUT_MS", 5000*time.Millisecond)
	if err != nil {
		loadErrors = append(loadErrors, fmt.Errorf("invalid OTEL_BATCH_TIMEOUT_MS: %w", err))
	}
	cfg.OtelExporterOtlpTimeout, err = getDurationMs(provider, "OTEL_EXPORTER_OTLP_TIMEOUT_MS", 10000*time.Millisecond)
	if err != nil {
		loadErrors = append(loadErrors, fmt.Errorf("invalid OTEL_EXPORTER_OTLP_TIMEOUT_MS: %w", err))
	}
	cfg.OtelExporterOtlpHeaders = getHeaders(provider, "OTEL_EXPORTER_OTLP_HEADERS", "")

	cfg.OtelEnableExemplars, err = getBool(provider, "OTEL_ENABLE_EXEMPLARS", false)
	if err != nil {
		loadErrors = append(loadErrors, fmt.Errorf("invalid OTEL_ENABLE_EXEMPLARS: %w", err))
	}

	
	cfg.ShutdownTimeout, err = getDurationSec(provider, "SHUTDOWN_TIMEOUT_SECONDS", 15*time.Second) 
	if err != nil {
		loadErrors = append(loadErrors, fmt.Errorf("invalid SHUTDOWN_TIMEOUT_SECONDS: %w", err))
	}
	cfg.ServerShutdownTimeout, err = getDurationSec(provider, "SERVER_SHUTDOWN_TIMEOUT_SECONDS", 10*time.Second)
	if err != nil {
		loadErrors = append(loadErrors, fmt.Errorf("invalid SERVER_SHUTDOWN_TIMEOUT_SECONDS: %w", err))
	}

	if len(loadErrors) > 0 {
		
		errMsg := "failed to load configuration:"
		for _, e := range loadErrors {
			errMsg += "\n\t- " + e.Error()
		}
		return nil, errors.New(errMsg)
	}

	
	
	if cfg.OtelExporterOtlpEndpoint == "" {
		fmt.Println("Warning: OTEL_EXPORTER_OTLP_ENDPOINT is not set. Telemetry export might fail.")
	}
	if cfg.DataFilePath == "" {
		fmt.Println("Warning: DATA_FILE_PATH is not set. Repository might fail or use defaults.")
	}

	
	switch cfg.OtelSamplerType {
	case "always_on", "always_off", "traceidratio", "parentbased_traceidratio":
		
	default:
		loadErrors = append(loadErrors, fmt.Errorf("invalid OTEL_SAMPLER_TYPE: %q, must be one of [always_on, always_off, traceidratio, parentbased_traceidratio]", cfg.OtelSamplerType))
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
	valueStr, ok := provider.GetValue(key)
	if !ok || valueStr == "" {
		return defaultValue, nil
	}
	val, err := strconv.ParseBool(strings.ToLower(valueStr))
	if err != nil {
		return false, fmt.Errorf("invalid boolean value for %s: %q: %w", key, valueStr, err)
	}
	return val, nil
}

func getFloat64(provider Provider, key string, defaultValue float64) (float64, error) {
	valueStr, ok := provider.GetValue(key)
	if !ok || valueStr == "" {
		return defaultValue, nil
	}
	val, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float64 value for %s: %q: %w", key, valueStr, err)
	}
	return val, nil
}

func getDurationMs(provider Provider, key string, defaultValue time.Duration) (time.Duration, error) {
	valueStr, ok := provider.GetValue(key)
	if !ok || valueStr == "" {
		return defaultValue, nil
	}
	ms, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value for %s (milliseconds): %q: %w", key, valueStr, err)
	}
	return time.Duration(ms) * time.Millisecond, nil
}

func getDurationSec(provider Provider, key string, defaultValue time.Duration) (time.Duration, error) {
	valueStr, ok := provider.GetValue(key)
	if !ok || valueStr == "" {
		return defaultValue, nil
	}
	sec, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value for %s (seconds): %q: %w", key, valueStr, err)
	}
	return time.Duration(sec) * time.Second, nil
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
