package metric

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	OpsCountMetricName    = "service.operations.count"
	DurationMetricName    = "service.duration.seconds"
	ErrorsCountMetricName = "service.errors.count"
)

var (
	opsCounter   metric.Int64Counter
	durationHist metric.Float64Histogram
	errorCounter metric.Int64Counter
)

func InitializeCommonMetrics(meter metric.Meter) error {
	var err, multiErr error

	opsCounter, err = meter.Int64Counter(
		OpsCountMetricName,
		metric.WithDescription("Counts service operations by layer and operation name."),
		metric.WithUnit("{operation}"),
	)
	multiErr = errors.Join(multiErr, err)

	durationHist, err = meter.Float64Histogram(
		DurationMetricName,
		metric.WithDescription("Measures the duration of service operations by layer and operation name."),
		metric.WithUnit("s"),
	)
	multiErr = errors.Join(multiErr, err)

	errorCounter, err = meter.Int64Counter(
		ErrorsCountMetricName,
		metric.WithDescription("Counts errors encountered by layer, operation, and type."),
		metric.WithUnit("{error}"),
	)
	multiErr = errors.Join(multiErr, err)

	return multiErr
}

func RecordOperationMetrics(
	ctx context.Context,
	layer string,
	operation string,
	startTime time.Time,
	opErr error,
	attrs ...attribute.KeyValue,
) {

	if durationHist == nil && opsCounter == nil && errorCounter == nil {
		return
	}

	commonAttrs := []attribute.KeyValue{
		attribute.String("layer", layer),
		attribute.String("operation", operation),
	}
	mergedAttrs := append(commonAttrs, attrs...)

	if durationHist != nil {
		duration := time.Since(startTime).Seconds()
		durationAttrs := append(mergedAttrs, attribute.Bool("error", opErr != nil))
		durationHist.Record(ctx, duration, metric.WithAttributes(durationAttrs...))
	}

	if opErr == nil && opsCounter != nil {
		opsCounter.Add(ctx, 1, metric.WithAttributes(mergedAttrs...))
	}

	if opErr != nil && errorCounter != nil {
		errorCounter.Add(ctx, 1, metric.WithAttributes(mergedAttrs...))
	}
}
