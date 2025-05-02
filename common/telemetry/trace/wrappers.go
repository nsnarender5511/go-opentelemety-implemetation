package trace

import (
	"context"

	"github.com/narender/common/telemetry/manager" 
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)




func StartSpan(ctx context.Context, scopeName, spanName string, initialAttrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := manager.GetTracer(scopeName) 
	
	if ctx == nil {
		ctx = context.Background()
	}
	return tracer.Start(ctx, spanName, trace.WithAttributes(initialAttrs...))
}



