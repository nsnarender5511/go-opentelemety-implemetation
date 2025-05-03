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
	once   sync.Once
	err    error
)

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

func Cfg() *config.Config {
	if cfg == nil {
		panic("configuration not initialized: call globals.Init() first and check error")
	}
	return cfg
}

func Logger() *slog.Logger {
	if logger == nil {
		panic("logger not initialized: call globals.Init() first and check error")
	}
	return logger
}

func GetCfg() *config.Config {
	return cfg
}

func GetLogger() *slog.Logger {
	return logger
}
