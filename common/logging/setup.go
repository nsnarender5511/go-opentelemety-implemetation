package logging

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/narender/common/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var baseLogger *zap.Logger
var loggerOnce sync.Once



func InitZapLogger(cfg *config.Config) (*zap.Logger, error) {
	var initErr error
	loggerOnce.Do(func() {
		var zapCfg zap.Config
		if strings.ToLower(cfg.Environment) == "development" {
			zapCfg = zap.NewDevelopmentConfig()
			
			zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder 
		} else {
			zapCfg = zap.NewProductionConfig()
			
			zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		}

		
		level, err := zapcore.ParseLevel(cfg.LogLevel)
		if err != nil {
			
			fmt.Fprintf(os.Stderr, "Warning: Invalid log level '%s', defaulting to 'info': %v\n", cfg.LogLevel, err)
			level = zapcore.InfoLevel
		}
		zapCfg.Level = zap.NewAtomicLevelAt(level)

		
		
		zapCfg.OutputPaths = []string{"stderr"}
		zapCfg.ErrorOutputPaths = []string{"stderr"}

		
		logger, err := zapCfg.Build(zap.AddCallerSkip(1)) 
		if err != nil {
			initErr = fmt.Errorf("failed to build zap logger: %w", err)
			return
		}

		
		baseLogger = logger

		baseLogger.Info("Zap logger initialized globally",
			zap.String("level", level.String()),
			zap.String("environment", cfg.Environment),
		)
	})

	if initErr != nil {
		return nil, initErr
	}
	if baseLogger == nil {
		
		return nil, fmt.Errorf("zap logger initialization failed or was skipped")
	}
	return baseLogger, nil
}



func GetBaseLogger() *zap.Logger {
	
	if baseLogger == nil {
		
		fmt.Fprintln(os.Stderr, "FATAL: GetBaseLogger called before InitZapLogger or initialization failed.")
	}
	return baseLogger
}
