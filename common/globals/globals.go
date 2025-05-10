package globals

import (
	"fmt"
	"log"
	"log/slog"
	"reflect"
	"sync"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
	"github.com/narender/common/config"
	commonLog "github.com/narender/common/log"
	commonOtel "github.com/narender/common/telemetry"
)

var (
	cfg    *config.Config
	logger *slog.Logger
	once   sync.Once
)

// Init loads configuration and initializes logger/telemetry once.
// Returns an error if initialization fails.
func Init() error {
	var initErr error
	once.Do(func() {
		if err := godotenv.Load(); err != nil {
			log.Println("Info: .env file not found or error loading, proceeding with environment variables.")
		}

		currentCfg := &config.Config{}
		if err := env.Parse(currentCfg); err != nil {
			log.Printf("CRITICAL: Failed to parse configuration from environment: %+v\n", err)
			initErr = fmt.Errorf("failed to parse configuration: %w", err)
			return
		}
		cfg = currentCfg

		fmt.Println("--- Loaded Configuration ---")
		val := reflect.ValueOf(cfg).Elem()
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			fieldName := typ.Field(i).Name
			fieldValue := val.Field(i).Interface()
			fmt.Printf("Key: %s, Value: %v\n", fieldName, fieldValue)
		}
		fmt.Println("--------------------------")

		if err := commonLog.Init(cfg.LOG_LEVEL, cfg.ENVIRONMENT); err != nil {
			log.Printf("CRITICAL: Logger initialization failed: %v\n", err)
			initErr = fmt.Errorf("failed to initialize logger: %w", err)
			return
		}
		logger = commonLog.L
		if logger == nil {
			log.Println("CRITICAL: Logger initialized successfully but global logger is nil")
			initErr = fmt.Errorf("logger nil after successful initialization")
			return
		}
		logger.Info("Logger initialized", slog.String("level", cfg.LOG_LEVEL))

		if err := commonOtel.InitTelemetry(cfg); err != nil {
			logger.Error("Failed to initialize OpenTelemetry", slog.Any("error", err))
			initErr = fmt.Errorf("failed to initialize telemetry: %w", err)
			return
		}
		logger.Info("OpenTelemetry initialized", slog.String("endpoint", cfg.OTEL_ENDPOINT))

		logger.Info("Application Globals Initialized Successfully.")
	})

	return initErr
}

// Cfg returns the loaded configuration.
// Panics if Init() was not called or failed.
func Cfg() *config.Config {
	if cfg == nil {
		panic("FATAL: Configuration accessed before successful initialization. Call globals.Init() at application start and check for errors.")
	}
	return cfg
}

// Logger returns the initialized global logger.
// Panics if Init() was not called or failed.
func Logger() *slog.Logger {
	if logger == nil {
		panic("FATAL: Logger accessed before successful initialization. Call globals.Init() at application start and check for errors.")
	}
	return logger
}
