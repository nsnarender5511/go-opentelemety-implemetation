package resource

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// NewResource creates a new OpenTelemetry resource with standard attributes.
// These attributes describe the entity producing telemetry (e.g., process, SDK).
func NewResource(ctx context.Context) (*resource.Resource, error) {

	res, err := resource.New(ctx,
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(os.Getenv("SERVICE_NAME")),
			semconv.ServiceVersionKey.String(os.Getenv("SERVICE_VERSION")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTel resource: %w", err)
	}

	return res, nil
}
