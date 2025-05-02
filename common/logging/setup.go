package logging

import (
	"os"
	"strings"
	"time"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
)

func SetupLogrus(cfg *config.Config) *logrus.Logger {
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
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	default:
		logger.Warnf("Invalid log format '%s', defaulting to 'text'", cfg.LogFormat)
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	}
	logger.SetOutput(os.Stderr)

	logrus.SetLevel(logger.GetLevel())
	logrus.SetFormatter(logger.Formatter)
	logrus.SetOutput(logger.Out)

	logger.Infof("Logrus initialized with level '%s' and format '%s'.", logger.GetLevel(), cfg.LogFormat)
	return logger
}
