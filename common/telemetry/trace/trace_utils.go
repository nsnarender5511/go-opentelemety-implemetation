package trace

import (
	"fmt"

	"github.com/narender/common/telemetry/attributes"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	
	oteltrace "go.opentelemetry.io/otel/trace"
)


func RecordSpanError(span oteltrace.Span, err error, attrs ...attribute.KeyValue) {
	if span == nil || !span.IsRecording() || err == nil {
		return
	}

	
	extraAttrs := []attribute.KeyValue{
		attributes.ExceptionMessageKey.String(err.Error()),         
		attributes.ExceptionTypeKey.String(fmt.Sprintf("%T", err)), 
	}

	
	allAttrs := append(extraAttrs, attrs...)

	span.RecordError(err, oteltrace.WithAttributes(allAttrs...))
	span.SetStatus(codes.Error, err.Error())
}
