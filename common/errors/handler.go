package errors

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)


func HandleLayerError(ctx context.Context, logger *slog.Logger, spanner interface {
	AddEvent(string, ...trace.EventOption)
}, opErr error, layer, operation string, attrs ...attribute.KeyValue) {
	if opErr == nil {
		return 
	}

	logLevel := slog.LevelError 
	eventName := "error"

	if errors.Is(opErr, ErrNotFound) {
		logLevel = slog.LevelWarn        
		eventName = "resource_not_found" 
	}

	
	logAttrs := []any{
		slog.String("layer", layer),
		slog.String("operation", operation),
		slog.String("error", opErr.Error()),
	}
	
	for _, attr := range attrs {
		logAttrs = append(logAttrs, slog.Any(string(attr.Key), attr.Value.AsInterface()))
	}

	
	logger.Log(ctx, logLevel, "Operation failed", logAttrs...)

	
	if spanner != nil {
		spanAttrs := []attribute.KeyValue{
			attribute.String("layer", layer),
			attribute.String("operation", operation),
			attribute.String("error.message", opErr.Error()),
		}
		
		spanAttrs = append(spanAttrs, attrs...)

		if errors.Is(opErr, ErrNotFound) {
			spanAttrs = append(spanAttrs, attribute.Bool("error.expected", true))
		}

		spanner.AddEvent(eventName, trace.WithAttributes(spanAttrs...))
	}
}
