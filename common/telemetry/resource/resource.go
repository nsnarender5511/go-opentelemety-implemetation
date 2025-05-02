package resource

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	"github.com/narender/common/telemetry/manager"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func NewResource(ctx context.Context, cfg *config.Config) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithHost(),
		resource.WithOS(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTel resource: %w", err)
	}
	manager.GetLogger().Debug("OpenTelemetry resource created")
	return res, nil
}
