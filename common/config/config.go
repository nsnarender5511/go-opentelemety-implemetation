package config

// Remove package-level cfg and configOnce as globals package handles singleton logic.
/*
var (
	cfg *Config
	configOnce   sync.Once
)
*/

type Config struct {
	ServiceName              string `env:"SERVICE_NAME,required"`
	ServiceVersion           string `env:"SERVICE_VERSION,required"`
	Environment              string `env:"ENVIRONMENT,default=development"`
	ProductServicePort       string `env:"PRODUCT_SERVICE_PORT,default=8082"`
	LogLevel                 string `env:"LOG_LEVEL,default=info"`
	OtelExporterOtlpEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,default=otel-collector:4317"`

	SimulateDelayEnabled bool `mapstructure:"SIMULATE_DELAY_ENABLED"`
	SimulateDelayMinMs   int  `mapstructure:"SIMULATE_DELAY_MIN_MS"`
	SimulateDelayMaxMs   int  `mapstructure:"SIMULATE_DELAY_MAX_MS"`
}

// LoadConfig now directly returns the default configuration.
// The responsibility of ensuring it's loaded once is moved to the globals package.
func LoadConfig() (*Config, error) {
	// configOnce.Do(func() { // Remove sync.Once
	// 	cfg = GetDefaultConfig()
	//
	// })
	// return cfg, nil // Return directly
	return GetDefaultConfig(), nil
}

func GetDefaultConfig() *Config {
	return &Config{
		ProductServicePort:       "8082",
		ServiceName:              "product-service",
		ServiceVersion:           "1.0.0",
		LogLevel:                 "debug",
		Environment:              "development",
		OtelExporterOtlpEndpoint: "otel-collector:4317",

		SimulateDelayEnabled: false,
		SimulateDelayMinMs:   10,
		SimulateDelayMaxMs:   100,
	}
}
