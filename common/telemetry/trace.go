package telemetry

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
)

// newTraceProvider creates and configures the OTLP trace provider.
func newTraceProvider(ctx context.Context, telemetryCfg TelemetryConfig, res *resource.Resource, setupLogger *logrus.Logger) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	setupLogger.Debug("Creating OTLP trace exporter...")

	var clientOpts []otlptracegrpc.Option
	clientOpts = append(clientOpts, otlptracegrpc.WithEndpoint(telemetryCfg.Endpoint))

	if telemetryCfg.Insecure {
		clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
		setupLogger.Debug("Trace exporter configured with insecure connection.")
	} else {
		clientOpts = append(clientOpts, otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
		setupLogger.Debug("Trace exporter configured with secure connection.")
	}

	traceExporter, err := otlptracegrpc.New(ctx, clientOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}
	setupLogger.Debug("OTLP trace exporter created successfully.")

	// Use config accessors from common/config
	batchTimeout := config.OtelBatchTimeout()
	maxExportBatchSize := config.OtelMaxExportBatchSize()

	setupLogger.WithFields(logrus.Fields{
		"batchTimeout": batchTimeout,
		"maxBatchSize": maxExportBatchSize,
	}).Debug("Configuring Batch Span Processor with values from config package")

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter,
		sdktrace.WithBatchTimeout(batchTimeout),
		sdktrace.WithMaxExportBatchSize(maxExportBatchSize),
	)

	// Determine sampler based on config struct value
	var sampler sdktrace.Sampler
	if telemetryCfg.SampleRatio > 0 && telemetryCfg.SampleRatio <= 1 {
		sampler = sdktrace.TraceIDRatioBased(telemetryCfg.SampleRatio)
		setupLogger.Infof("Trace sampling enabled with ratio: %.2f", telemetryCfg.SampleRatio)
	} else {
		sampler = sdktrace.AlwaysSample() // Default to always sample if ratio is invalid or 0/absent
		setupLogger.Info("Trace sampling configured to always sample.")
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	setupLogger.Debug("Trace provider configured.")
	// OTel SDK v1.x TracerProvider *does* have Shutdown which handles processors.

	// Define and return the shutdown function as a closure
	shutdownFunc := func(shutdownCtx context.Context) error {
		// Capture setupLogger for use within the closure
		localSetupLogger := setupLogger
		localSetupLogger.Debug("Shutting down OTel Trace Provider...")
		// Rely on the timeout applied to shutdownCtx by the caller (masterShutdown)
		err := tracerProvider.Shutdown(shutdownCtx)
		if err != nil {
			localSetupLogger.WithError(err).Error("Error shutting down OTel Trace Provider")
		} else {
			localSetupLogger.Debug("OTel Trace Provider shutdown complete.")
		}
		return err
	}

	return tracerProvider, shutdownFunc, nil
}
