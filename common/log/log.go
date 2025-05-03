package log

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

var L *slog.Logger

func Init(logLevelStr, environment string) error {
	if L != nil {
		slog.Warn("Logger already initialized")
		return nil
	}

	// Determine log level from parameter, default to Info
	var level slog.Level = slog.LevelInfo // Default level
	logLevelLower := strings.ToLower(logLevelStr)
	if err := level.UnmarshalText([]byte(logLevelLower)); err != nil {
		slog.Warn("Invalid log level provided, defaulting to INFO", slog.String("providedLevel", logLevelStr), slog.Any("error", err))
		level = slog.LevelInfo // Ensure default on error
	}

	handlerOpts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}

	var handler slog.Handler
	isProduction := strings.ToLower(environment) == "production"

	if isProduction {
		slog.Info("Production environment: Configuring OTLP and Console (Tint) slog handlers.")

		otlpHandler := otelslog.NewHandler("otlp_logger_placeholder")

		consoleHandler := tint.NewHandler(os.Stdout, &tint.Options{
			AddSource:  handlerOpts.AddSource,
			Level:      handlerOpts.Level,
			TimeFormat: time.RFC3339,
		})

		handler = slogmulti.Fanout(otlpHandler, consoleHandler)

	} else {
		slog.Info("Non-production environment: Configuring Console slog handler (Tint).", slog.String("environment", environment))
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			AddSource:  handlerOpts.AddSource,
			Level:      handlerOpts.Level,
			TimeFormat: time.Kitchen,
		})
	}

	L = slog.New(handler)

	slog.SetDefault(L)

	L.Info("Logger initialized and set as default", slog.String("level", level.String()), slog.String("environment", environment))
	return nil
}
