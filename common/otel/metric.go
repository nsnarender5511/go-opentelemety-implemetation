package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type MeterProvider = sdkmetric.MeterProvider

func newMeterProvider(ctx context.Context, res *Resource, logger *logrus.Logger) (*MeterProvider, ShutdownFunc, error) {
	logger.Debug("Creating meter provider...")

	if res == nil {
		return nil, nil, fmt.Errorf("resource cannot be nil")
	}

	// Create OTLP metric exporter options using the helper (pass logger)
	exporterOpts := newOtlpMetricGrpcExporterOptions(logger)

	// Create the exporter
	exp, err := otlpmetricgrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create reader with periodic export
	reader := sdkmetric.NewPeriodicReader(exp,
		sdkmetric.WithInterval(5*time.Second), // Export metrics every 5 seconds
	)

	// Create meter provider with the reader
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res.Unwrap()),
		sdkmetric.WithReader(reader),
	)

	// Create a shutdown function that properly cleans up the meter provider
	shutdown := func(shutdownCtx context.Context) error {
		logger.Debug("Shutting down meter provider...")
		return mp.Shutdown(shutdownCtx)
	}

	logger.Info("Meter provider created successfully")
	return mp, shutdown, nil
}

func GetMeter(name string) metric.Meter {
	return otel.Meter(name)
}

func Counter(name, description string) metric.Int64Counter {
	meter := GetMeter("counter")
	counter, _ := meter.Int64Counter(name, metric.WithDescription(description))
	return counter
}

func Gauge(name, description string) metric.Int64ObservableGauge {
	meter := GetMeter("gauge")
	gauge, _ := meter.Int64ObservableGauge(name, metric.WithDescription(description))
	return gauge
}

func Histogram(name, description string) metric.Int64Histogram {
	meter := GetMeter("histogram")
	histogram, _ := meter.Int64Histogram(name, metric.WithDescription(description))
	return histogram
}
