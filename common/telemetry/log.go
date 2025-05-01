package telemetry

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"
)

// otelShutdownFunc defines the signature for shutdown functions returned by providers.
type otelShutdownFunc func(context.Context) error

// createOtlpLogProvider creates and configures the OTLP log provider.
// Returns the provider, a shutdown function, and an error.
func createOtlpLogProvider(ctx context.Context, endpoint string, insecure bool, res *resource.Resource, setupLogger *logrus.Logger) (*sdklog.LoggerProvider, otelShutdownFunc, error) {
	setupLogger.Debug("Creating OTLP log exporter...")

	var clientOpts []otlploggrpc.Option
	if endpoint == "" {
		// This case should ideally be caught by config validation, but double-check here.
		setupLogger.Error("OTLP endpoint is empty in createOtlpLogProvider. Cannot create exporter.")
		return nil, nil, fmt.Errorf("cannot create OTLP log exporter: endpoint is empty")
	}
	clientOpts = append(clientOpts, otlploggrpc.WithEndpoint(endpoint))

	if insecure {
		clientOpts = append(clientOpts, otlploggrpc.WithInsecure())
		setupLogger.Debug("Log exporter configured with insecure connection.")
	} else {
		clientOpts = append(clientOpts, otlploggrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
		setupLogger.Debug("Log exporter configured with secure connection.")
	}

	logExporter, err := otlploggrpc.New(ctx, clientOpts...)
	if err != nil {
		// Don't return fatal error here, allow telemetry init to continue without OTel logging if needed
		setupLogger.WithError(err).WithFields(logrus.Fields{
			"endpoint": endpoint,
			"insecure": insecure,
		}).Warn("Failed to create OTLP log exporter. OTel logging via Logrus hook will NOT function.")
		return nil, nil, fmt.Errorf("failed to create OTLP log exporter (endpoint: %s): %w", endpoint, err)
	}
	setupLogger.WithFields(logrus.Fields{
		"endpoint": endpoint,
		"insecure": insecure,
	}).Info("OTLP log exporter created successfully.")

	// Configure LogRecordProcessor using values from common/config
	maxQueueSize := config.OtelLogMaxQueueSize()
	exportTimeout := config.OtelLogExportTimeout()
	exportInterval := config.OtelLogExportInterval() // Assuming we add this config

	setupLogger.WithFields(logrus.Fields{
		"maxQueueSize":   maxQueueSize,
		"exportTimeout":  exportTimeout,
		"exportInterval": exportInterval, // Log the interval if configured
	}).Debug("Configuring Batch Log Record Processor with values from config package")

	// Add relevant options available in the SDK version being used.
	// Example options (verify against your SDK version):
	batchProcessorOpts := []sdklog.BatchProcessorOption{
		sdklog.WithMaxQueueSize(maxQueueSize),
		sdklog.WithExportTimeout(exportTimeout),
		sdklog.WithExportInterval(exportInterval), // Correct option for export interval
		// sdklog.WithMaxExportBatchSize(...) // Optional: Configure batch size too if needed
	}
	blrp := sdklog.NewBatchProcessor(logExporter, batchProcessorOpts...)

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(blrp),
	)

	setupLogger.Debug("OTel Logger provider configured.")

	// Return the provider and its shutdown function
	shutdownFunc := func(shutdownCtx context.Context) error {
		setupLogger.Debug("Shutting down OTel Logger Provider...")
		// Rely on the timeout applied to shutdownCtx by the caller (masterShutdown)
		err := loggerProvider.Shutdown(shutdownCtx)
		if err != nil {
			setupLogger.WithError(err).Error("Error shutting down OTel Logger Provider")
		} else {
			setupLogger.Debug("OTel Logger Provider shutdown complete.")
		}
		return err
	}

	return loggerProvider, shutdownFunc, nil
}

// configureLogrus sets up the global Logrus instance.
// Moved here from init.go as part of file reorganization.
// It now requires the otelLogProvider to be non-nil.
func configureLogrus(level logrus.Level, otelLogProvider *sdklog.LoggerProvider, setupLogger *logrus.Logger) {
	// Level is now pre-parsed and passed in
	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	logrus.SetOutput(os.Stdout)

	setupLogger.Infof("Standard Logrus configured with level '%s' and JSON formatter.", level.String())

	// Check if provider is nil before adding hook
	if otelLogProvider == nil {
		setupLogger.Warn("OTel Logger Provider is nil. Cannot add Logrus OTel hook. Logs will NOT be sent to OTel.")
		return // Do not attempt to add the hook
	}

	// Configure the otellogrus hook.
	// It implicitly uses the globally registered OTel Logger Provider.
	hook := otellogrus.NewHook(otellogrus.WithLevels(
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	))

	logrus.AddHook(hook)
	setupLogger.Info("Logrus OTel hook added successfully. Logs at specified levels will be sent to OTel.")
}
