package globals

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/narender/common/config"
	"github.com/narender/common/log"
	"github.com/narender/common/telemetry"
)

var (
	cfg    *config.Config
	logger *slog.Logger
	// once ensures that initialization logic runs exactly once.
	once sync.Once
	err  error
)

// Init initializes global configuration, logging, and telemetry setup.
// It ensures this initialization happens only once using sync.Once.
// Returns an error if any initialization step fails.
func Init() error {
	once.Do(func() {

		cfg, err = config.LoadConfig("production")
		if err != nil {
			err = fmt.Errorf("failed to load config during init: %w", err)
			return
		}
		if cfg == nil {
			err = fmt.Errorf("config loaded as nil without error during init")
			return
		}

		if initErr := log.Init(cfg.LOG_LEVEL, cfg.ENVIRONMENT); initErr != nil {

			err = fmt.Errorf("failed to initialize logger during init: %w", initErr)

			fmt.Printf("CRITICAL: Logger initialization failed: %v\n", err)

			return
		}
		logger = log.L
		if logger == nil {
			err = fmt.Errorf("log.Init() succeeded but log.L is nil")
			return
		}

		if err = telemetry.InitTelemetry(cfg); err != nil {
			err = fmt.Errorf("failed to initialize telemetry setup during init: %w", err)

			logger.Error("Telemetry initialization failed", slog.Any("error", err))

			return
		}

	})

	return err
}

// Cfg returns the loaded configuration, panicking if Init hasn't been successfully called.
func Cfg() *config.Config {
	if cfg == nil {
		panic("configuration not initialized: call globals.Init() first and check error")
	}
	return cfg
}

// Logger returns the initialized logger, panicking if Init hasn't been successfully called.
func Logger() *slog.Logger {
	if logger == nil {
		panic("logger not initialized: call globals.Init() first and check error")
	}
	return logger
}

// GetCfg returns the loaded configuration, potentially nil if Init failed or wasn't called.
func GetCfg() *config.Config {
	return cfg
}

// GetLogger returns the initialized logger, potentially nil if Init failed or wasn't called.
func GetLogger() *slog.Logger {
	return logger
}
