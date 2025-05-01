package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

type Resource struct {
	resource *resource.Resource
}

func newResource(ctx context.Context, serviceName, serviceVersion string) (*Resource, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
		resource.WithFromEnv(),      // Pull attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables
		resource.WithProcessPID(),   // Add process ID as a resource attribute
		resource.WithProcessOwner(), // Add process owner as a resource attribute
		resource.WithHost(),         // Add host information as resource attributes
		resource.WithOS(),           // Add OS information as resource attributes
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	return &Resource{resource: res}, nil
}

func (r *Resource) Unwrap() *resource.Resource {
	return r.resource
}
