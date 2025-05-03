package config

import (
	"log"
	"sync"
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

var (
	globalConfig *Config
	configOnce   sync.Once
	configMutex  sync.RWMutex
)

func LoadConfig() (*Config, error) {
	var loadErr error
	configOnce.Do(func() {
		// TODO: Implement actual config loading logic (e.g., from env vars, file)
		log.Println("WARN: LoadConfig returning default config; ensure environment variable parsing is implemented.")
		cfg := GetDefaultConfig()

		configMutex.Lock()
		globalConfig = cfg
		configMutex.Unlock()
		// If loading fails, set loadErr
		// loadErr = someError
	})

	configMutex.RLock()
	defer configMutex.RUnlock()
	if loadErr != nil {
		return nil, loadErr // Return error if loading failed
	}
	return globalConfig, nil // Return potentially nil config if Do block didn't run/set it? Check logic.
}

// Get returns a read-only copy of the globally loaded configuration.
// It assumes LoadConfig has been called successfully at least once.
func Get() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	// Return a copy to prevent modification? Or assume read-only usage?
	// For simplicity, returning direct pointer for now.
	if globalConfig == nil {
		log.Println("WARN: config.Get() called before LoadConfig() or LoadConfig() failed, returning default config.")
		// Potential panic? Or return default? Returning default for resilience.
		return GetDefaultConfig()
	}
	return globalConfig
}

func GetDefaultConfig() *Config {
	return &Config{
		ProductServicePort:       "8082",
		ServiceName:              "product-service",
		ServiceVersion:           "1.0.0",
		DataFilePath:             "/app/data.json", // Adjusted default path
		LogLevel:                 "info",
		Environment:              "development", // Or "development"
		OtelExporterOtlpEndpoint: "otel-collector:4317",
		// Default delay settings
		SimulateDelayEnabled: false,
		SimulateDelayMinMs:   10,
		SimulateDelayMaxMs:   100,
		// Add other defaults as needed
		ShutdownTimeout:       5 * time.Second,
		ServerShutdownTimeout: 10 * time.Second,
	}
}
