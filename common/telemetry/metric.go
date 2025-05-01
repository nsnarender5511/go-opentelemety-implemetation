package telemetry

import (
	"context"
	"fmt"
	"time"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"
)

// newMeterProvider creates and configures the OTLP meter provider.
func newMeterProvider(ctx context.Context, telemetryCfg TelemetryConfig, res *resource.Resource, setupLogger *logrus.Logger) (*sdkmetric.MeterProvider, func(context.Context) error, error) {
	setupLogger.Debug("Creating OTLP metric exporter...")

	var clientOpts []otlpmetricgrpc.Option
	clientOpts = append(clientOpts, otlpmetricgrpc.WithEndpoint(telemetryCfg.Endpoint))

	if telemetryCfg.Insecure {
		clientOpts = append(clientOpts, otlpmetricgrpc.WithInsecure())
		setupLogger.Debug("Metric exporter configured with insecure connection.")
	} else {
		clientOpts = append(clientOpts, otlpmetricgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
		setupLogger.Debug("Metric exporter configured with secure connection.")
	}

	metricExporter, err := otlpmetricgrpc.New(ctx, clientOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}
	setupLogger.Debug("OTLP metric exporter created successfully.")

	// Configure MeterProvider
	// Using default interval for PeriodicReader, can be configured if needed.
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))), // Example interval
	)

	setupLogger.Debug("Meter provider configured.")

	// Define and return the shutdown function as a closure
	shutdownFunc := func(shutdownCtx context.Context) error {
		// Capture setupLogger for use within the closure
		localSetupLogger := setupLogger
		localSetupLogger.Debug("Shutting down OTel Meter Provider...")
		// Rely on the timeout applied to shutdownCtx by the caller (masterShutdown)
		err := mp.Shutdown(shutdownCtx)
		if err != nil {
			localSetupLogger.WithError(err).Error("Error shutting down OTel Meter Provider")
		} else {
			localSetupLogger.Debug("OTel Meter Provider shutdown complete.")
		}
		return err
	}

	return mp, shutdownFunc, nil
}

// GetMeter returns a named meter instance from the global provider
func GetMeter(name string) metric.Meter {
	return otel.Meter(name)
}
