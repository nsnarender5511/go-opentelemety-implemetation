package trace

import (
	"fmt"

	"github.com/narender/common/telemetry/attributes"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	// semconv "go.opentelemetry.io/otel/semconv/v1.21.0" // Removed direct import
	oteltrace "go.opentelemetry.io/otel/trace"
)

// RecordSpanError records an error on a span, setting status to Error and recording exception details.
func RecordSpanError(span oteltrace.Span, err error, attrs ...attribute.KeyValue) {
	if span == nil || !span.IsRecording() || err == nil {
		return
	}

	// Add standard exception attributes
	extraAttrs := []attribute.KeyValue{
		attributes.ExceptionMessageKey.String(err.Error()),         // Use attributes.ExceptionMessageKey
		attributes.ExceptionTypeKey.String(fmt.Sprintf("%T", err)), // Use attributes.ExceptionTypeKey
	}

	// Combine standard attributes with any provided custom attributes
	allAttrs := append(extraAttrs, attrs...)

	span.RecordError(err, oteltrace.WithAttributes(allAttrs...))
	span.SetStatus(codes.Error, err.Error())
}
