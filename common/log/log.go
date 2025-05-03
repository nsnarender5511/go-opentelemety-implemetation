package log

import (
	"log/slog"
	"os"
	"strings"

	"github.com/narender/common/config" // Assuming this path is correct
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// Global slog logger instance
var L *slog.Logger


func Init(cfg *config.Config) error {
	if L != nil {
		slog.Warn("Logger already initialized") // Use default slog before L is set
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
	default: // "info" or anything else
		level = slog.LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{
		AddSource: true, // Include source file/line number
		Level:     level,
	}

	var handler slog.Handler
	isProduction := strings.ToLower(cfg.Environment) == "production"

	if isProduction {
		slog.Info("Production environment: Configuring OTel slog handler.", slog.String("service.name", cfg.ServiceName)) // Log this before L is set
		handler = otelslog.NewHandler(cfg.ServiceName)
	} else {
		slog.Info("Non-production environment: Configuring Console slog handler (JSON).") // Log this before L is set
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	// Create the logger instance with the selected handler
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
