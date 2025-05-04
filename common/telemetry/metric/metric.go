package metric

import (
	"context"
	"errors"
	"log/slog"
	"time"

	commonerrors "github.com/narender/common/errors"
	"github.com/narender/common/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter           = otel.Meter("common/telemetry/metric")
	operationsTotal metric.Int64Counter
	durationMillis  metric.Float64Histogram
	errorsTotal     metric.Int64Counter
	// initErr variable removed as error handling is encapsulated in helpers

	// Map for error type lookup
	errorTypeMap = map[error]string{
		commonerrors.ErrNotFound:     "not_found",
		commonerrors.ErrValidation:   "validation",
		commonerrors.ErrInternal:     "internal",
		commonerrors.ErrUnauthorized: "unauthorized",
		commonerrors.ErrForbidden:    "forbidden",
	}
)

// Helper function to initialize an Int64Counter
func initInt64Counter(name, description, unit string) (metric.Int64Counter, error) {
	counter, err := meter.Int64Counter(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		slog.Error("Failed to initialize counter", slog.String("name", name), slog.Any("error", err))
	}
	return counter, err
}

// Helper function to initialize a Float64Histogram
func initFloat64Histogram(name, description, unit string) (metric.Float64Histogram, error) {
	histogram, err := meter.Float64Histogram(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		slog.Error("Failed to initialize histogram", slog.String("name", name), slog.Any("error", err))
	}
	return histogram, err
}

func init() {
	var err error // Use local err variable within init
	operationsTotal, err = initInt64Counter(
		"app.operations.total",
		"Total number of operations executed",
		"{operation}",
	)
	if err != nil {
		slog.Error("Failed to initialize counter", slog.String("name", "app.operations.total"), slog.Any("error", err))
	}
	// Error is logged within the helper, continue initialization if possible

	durationMillis, err = initFloat64Histogram(
		"app.operations.duration_milliseconds",
		"Duration of operations in milliseconds",
		"ms",
	)
	if err != nil {
		slog.Error("Failed to initialize histogram", slog.String("name", "app.operations.duration_milliseconds"), slog.Any("error", err))
	}

	errorsTotal, err = initInt64Counter(
		"app.operations.errors.total",
		"Total number of operations that resulted in an error",
		"{error}",
	)
	// Error is logged within the helper
	if err != nil {
		slog.Error("Failed to initialize counter", slog.String("name", "app.operations.errors.total"), slog.Any("error", err))
	}
}

type MetricsController interface {
	End(ctx context.Context, err *error, additionalAttrs ...attribute.KeyValue)
}

type metricsControllerImpl struct {
	startTime time.Time
	operation string
}

func StartMetricsTimer() MetricsController {

	return &metricsControllerImpl{
		startTime: time.Now(),
		operation: utils.GetCallerFunctionName(4),
	}
}

func (mc *metricsControllerImpl) End(ctx context.Context, errPtr *error, additionalAttrs ...attribute.KeyValue) {
	duration := time.Since(mc.startTime)
	durationMs := float64(duration.Microseconds()) / 1000.0

	isError := errPtr != nil && *errPtr != nil

	baseAttrs := []attribute.KeyValue{
		attribute.String("custom.metrics", "true"),
		attribute.String("app.operation", mc.operation),
		attribute.Bool("app.error", isError),
	}
	attrs := append(baseAttrs, additionalAttrs...)

	if isError {
		err := *errPtr
		errorType := "unknown" // Default error type
		// Use map lookup for error type classification
		for errKey, typeStr := range errorTypeMap {
			if errors.Is(err, errKey) {
				errorType = typeStr
				break // Found the most specific type
			}
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
