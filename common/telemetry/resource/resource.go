package resource

import (
	"context"
	"fmt"

	"github.com/narender/common/config"
	
	"go.opentelemetry.io/otel/sdk/resource"
	
)


func NewResource(ctx context.Context, cfg *config.Config) (*resource.Resource, error) {
	
	
	res, err := resource.New(ctx,
		resource.WithFromEnv(),      
		resource.WithProcess(),      
		resource.WithOS(),           
		resource.WithHost(),         
		resource.WithContainer(),    
		resource.WithTelemetrySDK(), 
		
		
		
		
		
		
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTel resource: %w", err)
	}
	
	
	return res, nil
}
