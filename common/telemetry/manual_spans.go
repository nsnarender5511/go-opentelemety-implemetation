package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StartSpan begins a new span using the global tracer provider.
// tracerName should typically be the instrumentation scope name (e.g., package path).
func StartSpan(ctx context.Context, tracerName string, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, spanName, opts...)
}

// AddAttribute adds a single attribute to a span.
// It handles different value types by converting them to OTel Attributes.
func AddAttribute(span trace.Span, key string, value interface{}) {
	if span == nil || !span.IsRecording() {
		return
	}

	var attr attribute.KeyValue
	switch v := value.(type) {
	case string:
		attr = attribute.String(key, v)
	case int:
		attr = attribute.Int(key, v)
	case int64:
		attr = attribute.Int64(key, v)
	case float64:
		attr = attribute.Float64(key, v)
	case bool:
		attr = attribute.Bool(key, v)
	default:
		// For unsupported types, convert to string
		attr = attribute.String(key, fmt.Sprintf("%v", v))
	}
	span.SetAttributes(attr)
}

// AddAttributes adds multiple attributes to a span.
func AddAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span == nil || !span.IsRecording() {
		return
	}
	span.SetAttributes(attrs...)
}
