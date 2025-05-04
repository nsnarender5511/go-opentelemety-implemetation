package config

type Config struct {
	ENVIRONMENT          string `env:"ENVIRONMENT,default=development"`
	PRODUCT_SERVICE_PORT string `env:"PRODUCT_SERVICE_PORT,default=8082"`
	LOG_LEVEL            string `env:"LOG_LEVEL,default=info"`
	OTEL_ENDPOINT        string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,default=otel-collector:4317"`

	PRODUCT_DATA_FILE_PATH string `env:"PRODUCT_DATA_FILE_PATH,default=/app/data.json"`

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
		cfg.PRODUCT_DATA_FILE_PATH = prodCfg.PRODUCT_DATA_FILE_PATH
	} else {
		devCfg := GetDevelopmentConfig()
		cfg.OTEL_ENDPOINT = devCfg.OTEL_ENDPOINT
		cfg.ENVIRONMENT = devCfg.ENVIRONMENT
		cfg.PRODUCT_DATA_FILE_PATH = devCfg.PRODUCT_DATA_FILE_PATH
	}

	return cfg, nil
}

func GetProductionConfig() *Config {
	return &Config{
		OTEL_ENDPOINT:          "otel-collector:4317",
		ENVIRONMENT:            "production",
		PRODUCT_DATA_FILE_PATH: "/app/products.json",
	}
}

func GetDevelopmentConfig() *Config {
	return &Config{
		OTEL_ENDPOINT:          "localhost:4317",
		ENVIRONMENT:            "development",
		PRODUCT_DATA_FILE_PATH: "products.json",
	}
}

func commonConfig() *Config {
	return &Config{
		PRODUCT_SERVICE_PORT: "8082",
		LOG_LEVEL:            "debug",
		SimulateDelayEnabled: true,
		SimulateDelayMinMs:   10,
		SimulateDelayMaxMs:   100,
	}
}
