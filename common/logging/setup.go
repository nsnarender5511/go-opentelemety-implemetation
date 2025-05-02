package logging

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
)

// Logger defines the standard logging interface used across services.

var globalLogger *logrus.Logger // Keep internal global as concrete type
var loggerOnce sync.Once

// GetLogger returns the globally configured logger instance as a Logger interface.
func GetLogger() *logrus.Logger { // Return interface type
	if globalLogger == nil {
		// This should ideally not happen if SetupLogrus is called early in main.
		// Return a default logger or panic based on desired strictness.
		// For now, return nil to make the dependency explicit.
		// Consider adding a logrus.Warn here using a temporary logger.
		logrus.StandardLogger().Error("FATAL: Attempted to GetLogger before SetupLogrus was called or it failed.")
		return nil
	}
	return globalLogger
}

func SetupLogrus(cfg *config.Config) *logrus.Logger { // Return interface type
	loggerOnce.Do(func() { // Ensure initialization happens only once
		logger := logrus.New()
		level, err := logrus.ParseLevel(cfg.LogLevel)
		if err != nil {
			logger.Warnf("Invalid log level '%s', defaulting to 'info': %v", cfg.LogLevel, err)
			level = logrus.InfoLevel
		}
		logger.SetLevel(level)
		switch strings.ToLower(cfg.LogFormat) {
		case "json":
			logger.SetFormatter(&logrus.JSONFormatter{
				TimestampFormat: time.RFC3339Nano,
				// Ensure Context is logged if present (for the hook)
				// This might not be strictly necessary if the hook always checks entry.Context
			})
		case "text":
			logger.SetFormatter(&logrus.TextFormatter{
				FullTimestamp:   true,
				TimestampFormat: time.RFC3339Nano,
				// DisableColors: true, // Often needed for structured parsing by agents
			})
		default:
			logger.Warnf("Invalid log format '%s', defaulting to 'text'", cfg.LogFormat)
			logger.SetFormatter(&logrus.TextFormatter{
				FullTimestamp:   true,
				TimestampFormat: time.RFC3339Nano,
			})
		}
		logger.SetOutput(os.Stderr) // Log to stderr

		// *** Add the OTel Hook ***
		// Ensure the global OTel LoggerProvider is configured *before* this hook is added
		// if the hook relies on it (which our implementation does).
		// The telemetry setup should happen before or alongside logger setup.
		otelHook := NewOtelHook()
		logger.AddHook(otelHook)
		// *************************

		// Set the global logger instance
		globalLogger = logger

		// Configure the standard logrus logger to match (optional but good practice)
		// Note: Standard logger won't have the hook unless explicitly added.
		// It's generally better to use the globalLogger instance everywhere.
		// logrus.SetLevel(globalLogger.GetLevel())
		// logrus.SetFormatter(globalLogger.Formatter)
		// logrus.SetOutput(globalLogger.Out)
		// logrus.AddHook(otelHook) // Add hook to standard logger too? Consider implications.

		globalLogger.Infof("Logrus initialized globally with level '%s', format '%s', and OTel hook.", globalLogger.GetLevel(), cfg.LogFormat)
	})
	// Ensure setup completed successfully before returning
	if globalLogger == nil {
		// Logrus setup failed within the Once.Do block
		logrus.StandardLogger().Error("FATAL: Logrus setup failed.")
		// Depending on requirements, could panic or return a non-nil default logger
		return nil
	}
	return globalLogger
}
