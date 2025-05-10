package config

// Config defines the application configuration structure using environment variables.
type Config struct {
	// Core App Settings
	ENVIRONMENT               string `env:"ENVIRONMENT,required" envDefault:"development"`
	PRODUCT_SERVICE_PORT      string `env:"PRODUCT_SERVICE_PORT,required" envDefault:"8082"`
	MASTER_STORE_SERVICE_PORT string `env:"MASTER_STORE_SERVICE_PORT,required" envDefault:"8083"`
	LOG_LEVEL                 string `env:"LOG_LEVEL" envDefault:"info"`
	// Default path set for container environment; override for local dev using .env or env var.
	PRODUCT_DATA_FILE_PATH string `env:"PRODUCT_DATA_FILE_PATH,required" envDefault:"/product-service/data.json"`
	// URL for the product service API
	PRODUCT_SERVICE_URL string `env:"PRODUCT_SERVICE_URL" envDefault:"http://product-service:8082"`

	// Telemetry Settings
	// Default endpoint suitable for local development; override in Docker.
	OTEL_ENDPOINT   string `env:"OTEL_ENDPOINT,required" envDefault:"localhost:4317"`
	SERVICE_NAME    string `env:"SERVICE_NAME" envDefault:"product-service"`
	SERVICE_VERSION string `env:"SERVICE_VERSION" envDefault:"unknown"`

	// Debug/Simulation Settings
	SimulateDelayEnabled           bool    `env:"SIMULATE_DELAY_ENABLED" envDefault:"false"`
	SimulateDelayMinMs             int     `env:"SIMULATE_DELAY_MIN_MS" envDefault:"10"`
	SimulateDelayMaxMs             int     `env:"SIMULATE_DELAY_MAX_MS" envDefault:"100"`
	SimulateRandomErrorEnabled     bool    `env:"SIMULATE_RANDOM_ERROR_ENABLED" envDefault:"false"`
	SimulateOverallErrorChance     float64 `env:"SIMULATE_OVERALL_ERROR_CHANCE" envDefault:"0.1"`
	SimulateApplicationErrorWeight int     `env:"SIMULATE_APPLICATION_ERROR_WEIGHT" envDefault:"1"`
	SimulateBusinessErrorWeight    int     `env:"SIMULATE_BUSINESS_ERROR_WEIGHT" envDefault:"1"`
}

// NOTE: Removed GetProductionConfig, GetDevelopmentConfig, commonConfig functions
// Configuration is now loaded directly from environment variables / .env file.
