package log

import (
	"fmt"

	"github.com/narender/common/config"

	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *otelzap.Logger

func Init(cfg *config.Config) error {
	var zapLogger *zap.Logger
	var err error

	zapCfg := zap.NewProductionConfig()

	logLevel := zapcore.InfoLevel
	if cfg.LogLevel != "" {
		err := logLevel.UnmarshalText([]byte(cfg.LogLevel))
		if err == nil {
			zapCfg.Level = zap.NewAtomicLevelAt(logLevel)
		} else {

			tempLogger, _ := zap.NewDevelopment()
			tempLogger.Warn("Failed to parse log level from config, using default",
				zap.String("configValue", cfg.LogLevel),
				zap.Error(err),
				zap.String("defaultLevel", logLevel.String()),
			)
			zapCfg.Level = zap.NewAtomicLevelAt(logLevel)
			_ = tempLogger.Sync()
		}
	} else {
		zapCfg.Level = zap.NewAtomicLevelAt(logLevel)
	}

	zapLogger, err = zapCfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		return fmt.Errorf("failed to create zap logger: %w", err)
	}

	L = otelzap.New(zapLogger)

	L.Info("Logger initialized", zap.String("level", logLevel.String()))

	return nil
}

func Cleanup() {
	if L != nil {

		L.Debug("Flushing buffered logs...")
		if err := L.Sync(); err != nil {

			tempLogger, _ := zap.NewDevelopment()
			tempLogger.Error("Failed to sync logger", zap.Error(err))
			_ = tempLogger.Sync()
		}
	}
}
