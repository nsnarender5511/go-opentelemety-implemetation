package resource

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/sdk/resource"
)

func NewResource(ctx context.Context) (*resource.Resource, error) {

	res, err := resource.New(ctx,
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTel resource: %w", err)
	}

	return res, nil
}
