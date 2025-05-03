package log

import (
	"log/slog"
	"os"
	"strings"

	"github.com/narender/common/config"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

var L *slog.Logger

func Init(cfg *config.Config) error {
	if L != nil {
		slog.Warn("Logger already initialized")
		return nil
	}

	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}

	var handler slog.Handler
	isProduction := strings.ToLower(cfg.Environment) == "production"

	if isProduction {
		slog.Info("Production environment: Configuring OTel slog handler.", slog.String("service.name", cfg.ServiceName))
		handler = otelslog.NewHandler(cfg.ServiceName)
	} else {
		slog.Info("Non-production environment: Configuring Console slog handler (JSON).")
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	L = slog.New(handler)

	slog.SetDefault(L)

	L.Info("Logger initialized and set as default", slog.String("environment", cfg.Environment), slog.String("level", level.String()))
	return nil
}

func Cleanup() {
	if L != nil {
		L.Debug("Logger cleanup called (noop).")
	} else {
		slog.Debug("Logger cleanup called (noop, logger not initialized).")
	}
}
