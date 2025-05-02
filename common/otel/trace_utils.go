package otel

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	// semconv "go.opentelemetry.io/otel/semconv/v1.21.0" // Removed direct import
	oteltrace "go.opentelemetry.io/otel/trace"
)

// RecordSpanError records an error on a span, setting status to Error and recording exception details.
func RecordSpanError(span oteltrace.Span, err error, attributes ...attribute.KeyValue) {
	if span == nil || !span.IsRecording() || err == nil {
		return
	}

	// Add standard exception attributes
	extraAttrs := []attribute.KeyValue{
		ExceptionMessageKey.String(err.Error()),         // Use package-level key
		ExceptionTypeKey.String(fmt.Sprintf("%T", err)), // Use package-level key
	}

	// Combine standard attributes with any provided custom attributes
	allAttrs := append(extraAttrs, attributes...)

	span.RecordError(err, oteltrace.WithAttributes(allAttrs...))
	span.SetStatus(codes.Error, err.Error())
}
