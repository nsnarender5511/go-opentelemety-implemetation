package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// LoggerProvider is the OpenTelemetry logger provider interface
type LoggerProvider = sdklog.LoggerProvider

// newLoggerProvider creates a new logger provider with the provided configuration
func newLoggerProvider(ctx context.Context, cfg *config.Config, res *Resource, logger *logrus.Logger) (*LoggerProvider, ShutdownFunc, error) {
	logger.Debug("Creating logger provider...")

	if res == nil {
		return nil, nil, fmt.Errorf("resource cannot be nil")
	}

	// Create OTLP exporter
	var opts []otlploggrpc.Option
	opts = append(opts, otlploggrpc.WithEndpoint(cfg.OtelEndpoint))

	// Add insecure option if needed
	if cfg.OtelInsecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	} else {
		// For secure connections, configure TLS if needed
		opts = append(opts, otlploggrpc.WithTLSCredentials(insecure.NewCredentials()))
	}

	// Add timeout option
	opts = append(opts, otlploggrpc.WithTimeout(10*time.Second))

	// Add gRPC connection options
	opts = append(opts, otlploggrpc.WithDialOption(grpc.WithBlock()))

	// Create the exporter
	exp, err := otlploggrpc.New(ctx, opts...)
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

// configureLogrus configures logrus with OpenTelemetry
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
