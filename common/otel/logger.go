package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type LoggerProvider = sdklog.LoggerProvider

func newLoggerProvider(ctx context.Context, res *Resource, logger *logrus.Logger) (*LoggerProvider, ShutdownFunc, error) {
	logger.Debug("Creating logger provider...")

	if res == nil {
		return nil, nil, fmt.Errorf("resource cannot be nil")
	}

	// Create OTLP log exporter options using the helper (pass logger)
	exporterOpts := newOtlpLogGrpcExporterOptions(logger)

	// Create the exporter
	exp, err := otlploggrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	// Create logger provider with the exporter
	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res.Unwrap()),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp,
			sdklog.WithExportInterval(5*time.Second),
			sdklog.WithExportTimeout(30*time.Second),
			sdklog.WithMaxQueueSize(2048),
			sdklog.WithExportMaxBatchSize(512),
		)),
	)

	// Create a shutdown function that properly cleans up the logger provider
	shutdown := func(shutdownCtx context.Context) error {
		logger.Debug("Shutting down logger provider...")
		return lp.Shutdown(shutdownCtx)
	}

	// Set as global provider
	global.SetLoggerProvider(lp)

	logger.Info("Logger provider created successfully")
	return lp, shutdown, nil
}

func configureLogrus(logger *logrus.Logger, provider *LoggerProvider, logLevel string) {
	// Set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
		logger.Warnf("Invalid log level '%s', defaulting to 'info'", logLevel)
	}
	logger.SetLevel(level)

	// Configure formatter based on user preference (text or json)
	if logLevel == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	}

	// Add OpenTelemetry hook if provider is available
	if provider != nil {
		// Add OTel hook for logrus
		logger.AddHook(otellogrus.NewHook(
			otellogrus.WithLevels(
				logrus.PanicLevel,
				logrus.FatalLevel,
				logrus.ErrorLevel,
				logrus.WarnLevel,
				logrus.InfoLevel,
				logrus.DebugLevel,
			),
		))
		logger.Debug("Configured logrus with OpenTelemetry hook")
	}
}
