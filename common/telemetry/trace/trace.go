package trace

import (
	"context"

	"github.com/narender/common/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)


type StatusMapperFunc func(error) codes.Code

func StartSpan(ctx context.Context, initialAttrs ...attribute.KeyValue) (context.Context, trace.Span) {
	operationName := utils.GetCallerFunctionName(3)

	tracerName := "static-tracer-for-now"
	tracer := otel.Tracer(tracerName)

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(semconv.CodeFunctionKey.String(operationName)),
		trace.WithAttributes(semconv.CodeNamespaceKey.String(tracerName)),
	}
	if len(initialAttrs) > 0 {
		opts = append(opts, trace.WithAttributes(initialAttrs...))
	}
	newCtx, span := tracer.Start(ctx, operationName, opts...)

	return newCtx, span
}

func EndSpan(span trace.Span, errPtr *error, options ...trace.SpanEndOption) {
	defer span.End(options...)

	if errPtr == nil || *errPtr == nil {
		span.SetStatus(codes.Ok, "SUCCESS")
		return
	}
	err := *errPtr
	span.RecordError(err, trace.WithStackTrace(true))
	statusMsg := err.Error()
	span.SetStatus(codes.Error, statusMsg)
}
