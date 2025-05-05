package config

type Config struct {
	ENVIRONMENT          string `env:"ENVIRONMENT,default=development"`
	PRODUCT_SERVICE_PORT string `env:"PRODUCT_SERVICE_PORT,default=8082"`
	LOG_LEVEL            string `env:"LOG_LEVEL,default=info"`
	OTEL_ENDPOINT        string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,default=otel-collector:4317"`

	PRODUCT_DATA_FILE_PATH string `mapstructure:"PRODUCT_DATA_FILE_PATH"`

	SimulateDelayEnabled bool `mapstructure:"SIMULATE_DELAY_ENABLED"`
	SimulateDelayMinMs   int  `mapstructure:"SIMULATE_DELAY_MIN_MS"`
	SimulateDelayMaxMs   int  `mapstructure:"SIMULATE_DELAY_MAX_MS"`
}

func LoadConfig(env string) (*Config, error) {
	cfg := commonConfig()

	if env == "production" {
		prodCfg := GetProductionConfig()
		cfg.OTEL_ENDPOINT = prodCfg.OTEL_ENDPOINT
		cfg.ENVIRONMENT = prodCfg.ENVIRONMENT
	} else {
		devCfg := GetDevelopmentConfig()
		cfg.OTEL_ENDPOINT = devCfg.OTEL_ENDPOINT
		cfg.ENVIRONMENT = devCfg.ENVIRONMENT
		cfg.PRODUCT_DATA_FILE_PATH = "./product-service/data.json"
	}

	return cfg, nil
}

func GetProductionConfig() *Config {
	// Keep these minimal, only environment differences
	return &Config{
		OTEL_ENDPOINT: "otel-collector:4317",
		ENVIRONMENT:   "production",
	}
}

func GetDevelopmentConfig() *Config {
	// Keep these minimal, only environment differences
	return &Config{
		OTEL_ENDPOINT: "localhost:4317",
		ENVIRONMENT:   "development",
	}
}

func commonConfig() *Config {
	// Base configuration for all environments
	return &Config{
		PRODUCT_SERVICE_PORT:   "8082",
		LOG_LEVEL:              "info",
		PRODUCT_DATA_FILE_PATH: "/app/data.json",
		SimulateDelayEnabled:   true,
		SimulateDelayMinMs:     10,
		SimulateDelayMaxMs:     10,
		// Set a default environment maybe? Or let LoadConfig handle it.
		// ENVIRONMENT: "development",
	}
}
