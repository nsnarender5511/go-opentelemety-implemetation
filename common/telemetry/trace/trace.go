package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func DefaultStatusMapper(err error) codes.Code {
	if err == nil {
		return codes.Ok
	}

	return codes.Error
}

type StatusMapperFunc func(error) codes.Code

// StartSpan begins a new OTel span, inferring the operation name from the caller.
// It uses a static tracer name and adds standard code attributes.
// Enhanced to include component and operation as standard attributes.
func StartSpan(ctx context.Context, component, operation string, initialAttrs ...attribute.KeyValue) (context.Context, trace.Span) {
	// Add component and operation as standard attributes
	standardAttrs := []attribute.KeyValue{
		attribute.String("component", component),
		attribute.String("operation", operation),
	}

	// Combine standard and custom attributes
	allAttrs := append(standardAttrs, initialAttrs...)

	operationName := component + " :: " + operation
	tracerName := "static-tracer-for-now"
	tracer := otel.Tracer(tracerName)

	// parentSpanContext := trace.SpanContextFromContext(ctx)
	// fmt.Printf("[DEBUG] StartSpan called | operation: %s | hasParent: %t | parentTraceID: %s | parentSpanID: %s\n",
	// 	operationName,
	// 	parentSpanContext.IsValid(),
	// 	parentSpanContext.TraceID().String(),
	// 	parentSpanContext.SpanID().String(),
	// )

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(semconv.CodeFunctionKey.String(operationName)),
		trace.WithAttributes(semconv.CodeNamespaceKey.String(tracerName)),
	}
	if len(allAttrs) > 0 {
		opts = append(opts, trace.WithAttributes(allAttrs...))
	}

	newCtx, span := tracer.Start(ctx, operationName, opts...)

	return newCtx, span
}

// EndSpan concludes the given span, automatically recording errors and setting status.
// It expects a pointer to an error variable to check for failures.
func EndSpan(span trace.Span, errPtr *error, statusMapper StatusMapperFunc, options ...trace.SpanEndOption) {
	defer span.End(options...)

	if errPtr == nil || *errPtr == nil {
		span.SetStatus(codes.Ok, "")
		return
	}

	err := *errPtr
	span.RecordError(err, trace.WithStackTrace(true))

	mapper := statusMapper
	if mapper == nil {
		mapper = DefaultStatusMapper
	}
	statusCode := mapper(err)

	statusMsg := ""
	if statusCode == codes.Error {
		statusMsg = err.Error()
	}

	span.SetStatus(statusCode, statusMsg)
}
