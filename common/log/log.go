package log

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/narender/common/config"
	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

var L *slog.Logger

func Init() error {
	if L != nil {
		slog.Warn("Logger already initialized")
		return nil
	}

	// Determine log level from config, default to Info
	var level slog.Level = slog.LevelInfo // Default level
	logLevelStr := strings.ToLower(config.Get().LogLevel)
	if err := level.UnmarshalText([]byte(logLevelStr)); err != nil {
		slog.Warn("Invalid log level configured, defaulting to INFO", slog.String("configuredLevel", logLevelStr), slog.Any("error", err))
		level = slog.LevelInfo // Ensure default on error
	}

	handlerOpts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}

	var handler slog.Handler
	cfg := config.Get()
	isProduction := strings.ToLower(cfg.Environment) == "production"

	if isProduction {
		slog.Info("Production environment: Configuring OTLP and Console (Tint) slog handlers.")

		otlpHandler := otelslog.NewHandler("default_logger")

		consoleHandler := tint.NewHandler(os.Stdout, &tint.Options{
			AddSource:  handlerOpts.AddSource,
			Level:      handlerOpts.Level,
			TimeFormat: time.RFC3339,
		})

		handler = slogmulti.Fanout(otlpHandler, consoleHandler)

	} else {
		slog.Info("Non-production environment: Configuring Console slog handler (Tint).")
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			AddSource:  handlerOpts.AddSource,
			Level:      handlerOpts.Level,
			TimeFormat: time.Kitchen,
		})
	}

	L = slog.New(handler)

	slog.SetDefault(L)

	L.Info("Logger initialized, enriched with common attributes, and set as default", slog.String("level", level.String()))
	return nil
}
