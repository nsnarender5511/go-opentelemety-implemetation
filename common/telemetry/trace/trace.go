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

// DefaultStatusMapper maps common errors to OpenTelemetry span status codes.
func DefaultStatusMapper(err error) codes.Code {
	if err == nil {
		return codes.Ok
	}
	// Treat NotFound as Ok for spans, as it's often an expected outcome.
	if errors.Is(err, commonerrors.ErrNotFound) {
		return codes.Ok
	}
	// All other errors are treated as actual errors.
	return codes.Error
}

// StatusMapperFunc defines the function signature for mapping an error to a span status code.
type StatusMapperFunc func(error) codes.Code

// SpanController provides methods to manage the lifecycle of an OpenTelemetry span.
type SpanController interface {
	AddEvent(name string, options ...trace.EventOption)
	RecordError(err error, options ...trace.EventOption)
	SetAttributes(kv ...attribute.KeyValue)
	SetStatus(code codes.Code, description string)
	End(err *error, statusMapper StatusMapperFunc, options ...trace.SpanEndOption)
	Span() trace.Span // Expose the underlying span if needed
}

type spanControllerImpl struct {
	span   trace.Span
	logger *slog.Logger // Optional logger for internal use
}

// StartSpan begins a new trace span and returns a controller to manage it.
// tracerName should typically be the instrumented package name (e.g., "product-service/repository").
// operationName describes the specific action (e.g., "GetAllProducts", "UpdateStock").
// layer identifies the architectural layer (e.g., "repository", "service", "handler").
func StartSpan(ctx context.Context, tracerName, operationName, layer string, initialAttrs ...attribute.KeyValue) (context.Context, SpanController) {
	tracer := otel.Tracer(tracerName)

	opts := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindInternal), // Default kind, adjust if needed
		trace.WithAttributes(semconv.CodeFunctionKey.String(operationName)),
		trace.WithAttributes(semconv.CodeNamespaceKey.String(tracerName)), // Use tracerName for namespace
		trace.WithAttributes(attribute.String("app.layer", layer)),
	}
	if len(initialAttrs) > 0 {
		opts = append(opts, trace.WithAttributes(initialAttrs...))
	}

	newCtx, span := tracer.Start(ctx, operationName, opts...)

	// TODO: Decide if a logger should be passed in or created here.
	// logger := slog.Default() // Or get from context/config

	return newCtx, &spanControllerImpl{
		span: span,
		// logger: logger,
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

// End finishes the span, recording the error, setting the status based on the mapper,
// and ensuring the span is always ended.
// It takes a pointer to an error (`*error`) typically captured from a named return variable.
func (sc *spanControllerImpl) End(errPtr *error, statusMapper StatusMapperFunc, options ...trace.SpanEndOption) {
	// Ensure span.End() is always called
	defer sc.span.End(options...)

	if errPtr == nil || *errPtr == nil {
		// No error, set status to OK
		sc.span.SetStatus(codes.Ok, "")
		return
	}

	// Error occurred
	err := *errPtr
	sc.span.RecordError(err) // Record the error details

	// Determine status using the provided mapper (or default if nil)
	mapper := statusMapper
	if mapper == nil {
		mapper = DefaultStatusMapper // Fallback to default
	}
	statusCode := mapper(err)

	statusMsg := ""
	if statusCode == codes.Error {
		statusMsg = err.Error() // Use error message for Error status description
	}

	sc.span.SetStatus(statusCode, statusMsg)
}

func (sc *spanControllerImpl) Span() trace.Span {
	return sc.span
}
