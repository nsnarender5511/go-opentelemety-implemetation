package otel

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"github.com/sirupsen/logrus"
)

// InitOTel initializes OpenTelemetry with the given config
// It returns a shutdown function to clean up resources
func InitOTel(ctx context.Context, cfg *config.Config) (func(context.Context) error, error) {
	logger := logrus.New()

	// Create a setup with configuration
	setup := NewSetup(cfg, WithLogger(logger))

	// Create and configure the OpenTelemetry components
	var err error

	// Add resource
	setup, err = setup.WithResource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Add propagator
	setup = setup.WithPropagator()

	// Add tracer provider
	setup, err = setup.WithTracing(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup tracing: %w", err)
	}

	// Add meter provider
	setup, err = setup.WithMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup metrics: %w", err)
	}

	// Add logger provider (optional, won't fail if it doesn't work)
	setup, _ = setup.WithLogging(ctx)

	// Return the shutdown function
	return setup.Shutdown, nil
}

// InitTelemetry initializes OpenTelemetry with a simple config for backward compatibility
// This provides a simpler function signature for existing code
func InitTelemetry(ctx context.Context, telemetryConfig struct {
	ServiceName string
	Endpoint    string
	Insecure    bool
	SampleRatio float64
	LogLevel    string
}) (func(context.Context) error, error) {
	// Create config from telemetry config
	cfg := config.NewConfig(
		config.WithServiceName(telemetryConfig.ServiceName),
		config.WithOtelEndpoint(telemetryConfig.Endpoint),
		config.WithOtelInsecure(telemetryConfig.Insecure),
		config.WithOtelSampleRatio(telemetryConfig.SampleRatio),
		config.WithLogLevel(telemetryConfig.LogLevel),
		config.WithLogFormat("text"), // Default to text
	)

	return InitOTel(ctx, cfg)
}
