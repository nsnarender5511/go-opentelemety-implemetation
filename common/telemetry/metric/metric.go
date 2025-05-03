package metric

import (
	"context"
	"errors"
	"log/slog"
	"time"

	commonerrors "github.com/narender/common/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)





var (
	meter           = otel.Meter("common/telemetry/metric") 
	operationsTotal metric.Int64Counter
	durationMillis  metric.Float64Histogram
	errorsTotal     metric.Int64Counter
	initErr         error
)

func init() {
	
	operationsTotal, initErr = meter.Int64Counter(
		"app.operations.total",
		metric.WithDescription("Total number of operations executed"),
		metric.WithUnit("{operation}"),
	)
	if initErr != nil {
		slog.Error("Failed to initialize operationsTotal counter", slog.Any("error", initErr))
	}

	durationMillis, initErr = meter.Float64Histogram(
		"app.operations.duration_milliseconds",
		metric.WithDescription("Duration of operations in milliseconds"),
		metric.WithUnit("ms"),
	)
	if initErr != nil {
		slog.Error("Failed to initialize durationMillis histogram", slog.Any("error", initErr))
	}

	errorsTotal, initErr = meter.Int64Counter(
		"app.operations.errors.total",
		metric.WithDescription("Total number of operations that resulted in an error"),
		metric.WithUnit("{error}"),
	)
	if initErr != nil {
		slog.Error("Failed to initialize errorsTotal counter", slog.Any("error", initErr))
	}
}




type MetricsController interface {
	End(ctx context.Context, err *error, additionalAttrs ...attribute.KeyValue)
}

type metricsControllerImpl struct {
	startTime time.Time
	layer     string
	operation string
	
}


func StartMetricsTimer(layer, operation string) MetricsController {
	return &metricsControllerImpl{
		startTime: time.Now(),
		layer:     layer,
		operation: operation,
	}
}



func (mc *metricsControllerImpl) End(ctx context.Context, errPtr *error, additionalAttrs ...attribute.KeyValue) {
	duration := time.Since(mc.startTime)
	durationMs := float64(duration.Microseconds()) / 1000.0 

	isError := errPtr != nil && *errPtr != nil

	
	baseAttrs := []attribute.KeyValue{
		attribute.String("app.layer", mc.layer),
		attribute.String("app.operation", mc.operation),
		attribute.Bool("app.error", isError),
	}
	attrs := append(baseAttrs, additionalAttrs...)

	
	if isError {
		err := *errPtr
		errorType := "unknown"
		if errors.Is(err, commonerrors.ErrNotFound) {
			errorType = "not_found"
		} else if errors.Is(err, commonerrors.ErrValidation) {
			errorType = "validation"
		} else if errors.Is(err, commonerrors.ErrInternal) {
			errorType = "internal"
		} else if errors.Is(err, commonerrors.ErrUnauthorized) {
			errorType = "unauthorized"
		} else if errors.Is(err, commonerrors.ErrForbidden) {
			errorType = "forbidden"
		}
		attrs = append(attrs, attribute.String("app.error.type", errorType))
	}

	opt := metric.WithAttributes(attrs...)

	
	if operationsTotal != nil {
		operationsTotal.Add(ctx, 1, opt)
	}
	if durationMillis != nil {
		durationMillis.Record(ctx, durationMs, opt)
	}
	if isError && errorsTotal != nil {
		errorsTotal.Add(ctx, 1, opt)
	}
}
