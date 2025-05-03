package errors

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HandleLayerError standardizes error logging and span event recording for different layers.
// It logs the error using the provided logger and adds an event to the span.
// The log level is adjusted based on whether the error is ErrNotFound (Warn) or something else (Error).
// spanner should be the SpanController obtained from trace.StartSpan.
func HandleLayerError(ctx context.Context, logger *slog.Logger, spanner interface {
	AddEvent(string, ...trace.EventOption)
}, opErr error, layer, operation string, attrs ...attribute.KeyValue) {
	if opErr == nil {
		return // No error to handle
	}

	logLevel := slog.LevelError // Default to Error level
	eventName := "error"

	if errors.Is(opErr, ErrNotFound) {
		logLevel = slog.LevelWarn        // Downgrade log level for expected "not found" errors
		eventName = "resource_not_found" // More specific event name
	}

	// Prepare log attributes
	logAttrs := []any{
		slog.String("layer", layer),
		slog.String("operation", operation),
		slog.String("error", opErr.Error()),
	}
	// Convert OTel attributes to slog attributes if needed, or handle separately
	for _, attr := range attrs {
		logAttrs = append(logAttrs, slog.Any(string(attr.Key), attr.Value.AsInterface()))
	}

	// Log the error
	logger.Log(ctx, logLevel, "Operation failed", logAttrs...)

	// Add event to the span
	if spanner != nil {
		spanAttrs := []attribute.KeyValue{
			attribute.String("layer", layer),
			attribute.String("operation", operation),
			attribute.String("error.message", opErr.Error()),
		}
		// Include original attributes in the span event as well
		spanAttrs = append(spanAttrs, attrs...)

		if errors.Is(opErr, ErrNotFound) {
			spanAttrs = append(spanAttrs, attribute.Bool("error.expected", true))
		}

		spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
	}
}
