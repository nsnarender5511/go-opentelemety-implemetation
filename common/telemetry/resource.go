package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

// newResource creates a resource describing this application.
func newResource(ctx context.Context, serviceName string) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithFromEnv(),      // Pull attributes from OTEL_RESOURCE_ATTRIBUTES
		resource.WithTelemetrySDK(), // Basic SDK info
		resource.WithHost(),         // Hostname, etc.
		resource.WithAttributes(
			// Logical service name
			semconv.ServiceName(serviceName),
			// Add other identifying attributes here if needed
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	// Merge with default resource detector (OS, etc.)
	return resource.Merge(resource.Default(), res)
}
