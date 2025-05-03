package log

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/narender/common/config"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

var L *slog.Logger

func Init() error {
	if L != nil {
		slog.Warn("Logger already initialized")
		return nil
	}

	var level slog.Level
	switch strings.ToLower(config.Get().LogLevel) {
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
	isProduction := strings.ToLower(config.Get().Environment) == "production"

	if isProduction {
		slog.Info("Production environment: Configuring OTel slog handler.", slog.String("service.name", config.Get().ServiceName))
		handler = otelslog.NewHandler(config.Get().ServiceName)
	} else {
		slog.Info("Non-production environment: Configuring Console slog handler (Tint).")
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			AddSource:  handlerOpts.AddSource,
			Level:      handlerOpts.Level,
			TimeFormat: time.Kitchen,
		})
	}

	L = slog.New(handler)

	// Enrich the logger with common application attributes
	cfg := config.Get()
	L = L.With(
		slog.String("service.name", cfg.ServiceName),
		slog.String("service.version", cfg.ServiceVersion),
		slog.String("deployment.environment", cfg.Environment),
	)

	slog.SetDefault(L)

	L.Info("Logger initialized, enriched with common attributes, and set as default", slog.String("level", level.String()))
	return nil
}
