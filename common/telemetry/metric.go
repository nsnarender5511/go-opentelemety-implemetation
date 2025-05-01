package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus" // Host metrics (CPU, memory)
	// Go runtime metrics (GC, goroutines)
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// newMeterProvider creates and configures a new MeterProvider
func newMeterProvider(ctx context.Context, config TelemetryConfig, res *resource.Resource) (*sdk.MeterProvider, error) {
	var err error

	logger := config.Logger
	if logger == nil {
		logger = getLogger()
	}

	logger.WithFields(logrus.Fields{
		"endpoint": config.Endpoint,
		"insecure": config.Insecure,
	}).Debug("Creating metric exporter")

	// Configure security options
	var secureOption otlpmetricgrpc.Option
	if config.Insecure {
		secureOption = otlpmetricgrpc.WithInsecure()
		logger.Debug("Using insecure connection for metric exporter")
	} else {
		// Use TLS credentials
		creds := credentials.NewClientTLSFromCert(nil, "")
		secureOption = otlpmetricgrpc.WithTLSCredentials(creds)
		logger.Debug("Using secure connection for metric exporter")
	}

	// Configure headers
	var headers map[string]string
	if config.Headers != nil {
		headers = config.Headers
	} else {
		headers = make(map[string]string)
	}

	// Create OTLP metric exporter
	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(config.Endpoint),
		secureOption,
		otlpmetricgrpc.WithHeaders(headers),
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Configure reader options
	readerOptions := []sdk.PeriodicReaderOption{
		// Default to 30s collection interval if not specified
		sdk.WithInterval(30 * time.Second),
	}

	// Create and configure meter provider
	mp := sdk.NewMeterProvider(
		sdk.WithResource(res),
		sdk.WithReader(sdk.NewPeriodicReader(exp, readerOptions...)),
	)

	logger.Info("Meter provider initialized successfully")
	return mp, nil
}

// GetMeter returns a named meter instance from the global provider
func GetMeter(name string) metric.Meter {
	return otel.Meter(name)
}
