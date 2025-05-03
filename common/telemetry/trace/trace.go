package trace

import (
	"context"
	"errors"
	"log/slog"

	commonerrors "github.com/narender/common/errors"
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
	
	if errors.Is(err, commonerrors.ErrNotFound) {
		return codes.Ok
	}
	
	return codes.Error
}


type StatusMapperFunc func(error) codes.Code


type SpanController interface {
	AddEvent(name string, options ...trace.EventOption)
	RecordError(err error, options ...trace.EventOption)
	SetAttributes(kv ...attribute.KeyValue)
	SetStatus(code codes.Code, description string)
	End(err *error, statusMapper StatusMapperFunc, options ...trace.SpanEndOption)
	Span() trace.Span 
}

type spanControllerImpl struct {
	span   trace.Span
	logger *slog.Logger 
}





func StartSpan(ctx context.Context, tracerName, operationName, layer string, initialAttrs ...attribute.KeyValue) (context.Context, SpanController) {
	tracer := otel.Tracer(tracerName)

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindInternal), 
		trace.WithAttributes(semconv.CodeFunctionKey.String(operationName)),
		trace.WithAttributes(semconv.CodeNamespaceKey.String(tracerName)), 
		trace.WithAttributes(attribute.String("app.layer", layer)),
	}
	if len(initialAttrs) > 0 {
		opts = append(opts, trace.WithAttributes(initialAttrs...))
	}

	newCtx, span := tracer.Start(ctx, operationName, opts...)

	
	

	return newCtx, &spanControllerImpl{
		span: span,
		
	}
}

func (sc *spanControllerImpl) AddEvent(name string, options ...trace.EventOption) {
	sc.span.AddEvent(name, options...)
}

func (sc *spanControllerImpl) RecordError(err error, options ...trace.EventOption) {
	sc.span.RecordError(err, options...)
}

func (sc *spanControllerImpl) SetAttributes(kv ...attribute.KeyValue) {
	sc.span.SetAttributes(kv...)
}

func (sc *spanControllerImpl) SetStatus(code codes.Code, description string) {
	sc.span.SetStatus(code, description)
}




func (sc *spanControllerImpl) End(errPtr *error, statusMapper StatusMapperFunc, options ...trace.SpanEndOption) {
	
	defer sc.span.End(options...)

	if errPtr == nil || *errPtr == nil {
		
		sc.span.SetStatus(codes.Ok, "")
		return
	}

	
	err := *errPtr
	sc.span.RecordError(err) 

	
	mapper := statusMapper
	if mapper == nil {
		mapper = DefaultStatusMapper 
	}
	statusCode := mapper(err)

	statusMsg := ""
	if statusCode == codes.Error {
		statusMsg = err.Error() 
	}

	sc.span.SetStatus(statusCode, statusMsg)
}

func (sc *spanControllerImpl) Span() trace.Span {
	return sc.span
}
