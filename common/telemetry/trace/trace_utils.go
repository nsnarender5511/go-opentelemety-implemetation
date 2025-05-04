package trace

import (
	"fmt"

	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func RecordSpanError(span oteltrace.Span, err error, attrs ...attribute.KeyValue) {
	if span == nil || !span.IsRecording() || err == nil {
		return
	}

	extraAttrs := []attribute.KeyValue{
		semconv.ExceptionMessageKey.String(err.Error()),
		semconv.ExceptionTypeKey.String(fmt.Sprintf("%T", err)),
	}

	allAttrs := append(extraAttrs, attrs...)

	span.RecordError(err, oteltrace.WithAttributes(allAttrs...))
	span.SetStatus(codes.Error, err.Error())
}
