package telemetry

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// GetOTelPropagator returns the globally configured OpenTelemetry TextMapPropagator.
// This is typically used by instrumentation libraries (like HTTP middleware)
// to inject and extract context across process boundaries.
func GetOTelPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}

// StartSpan begins a new OpenTelemetry span using the tracer associated with tracerName.
// It returns the new context containing the span and the span itself.
// The caller is responsible for ending the span using `defer span.End()`.
func StartSpan(ctx context.Context, tracerName, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return GetTracer(tracerName).Start(ctx, spanName, opts...)
}

// RecordError marks a span as having encountered an error.
// It sets the span status to Error, records the error details,
// and logs the error using the globally configured logrus logger (if available).
// Additional attributes can be provided to add context to the error record.
func RecordError(span trace.Span, err error, description string, attrs ...attribute.KeyValue) {
	if span == nil || err == nil {
		return
	}

	// Set span status and record error
	span.SetStatus(codes.Error, description)
	span.RecordError(err, trace.WithAttributes(attrs...))

	// Log the error with logrus if logger is configured
	if globalLogger != nil {
		fields := logrus.Fields{
			"span_name":   span.SpanContext().SpanID().String(),
			"trace_id":    span.SpanContext().TraceID().String(),
			"description": description,
		}

		// Add attributes to logrus fields
		for _, attr := range attrs {
			fields[string(attr.Key)] = attr.Value.AsInterface()
		}

		globalLogger.WithFields(fields).WithError(err).Error("Error recorded in span")
	}
}

// AddAttribute adds a single key-value attribute to the provided span.
// It handles common Go types (string, int64, int, float64, bool).
// Other types are converted to a string representation.
func AddAttribute(span trace.Span, key string, value interface{}) {
	if span == nil {
		return
	}

	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int64:
		span.SetAttributes(attribute.Int64(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	default:
		// Convert to string for unknown types
		span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

// AddAttributes adds multiple OpenTelemetry attributes to the provided span.
func AddAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span == nil || len(attrs) == 0 {
		return
	}
	span.SetAttributes(attrs...)
}

// IncrementCounter increments an OpenTelemetry Int64Counter metric.
// It uses the meter associated with meterName to find or create the counter
// identified by counterName, then adds the specified value.
// Attributes can be provided for dimensionality. Logs the increment at Debug level if configured.
func IncrementCounter(ctx context.Context, meterName, counterName string, value int64, attrs ...attribute.KeyValue) {
	meter := GetMeter(meterName)
	counter, _ := meter.Int64Counter(counterName)
	counter.Add(ctx, value, metric.WithAttributes(attrs...))

	if globalLogger != nil && globalLogger.IsLevelEnabled(logrus.DebugLevel) {
		fields := logrus.Fields{
			"counter_name": counterName,
			"value":        value,
			"meter_name":   meterName,
		}
		for _, attr := range attrs {
			fields[string(attr.Key)] = attr.Value.AsInterface()
		}
		globalLogger.WithFields(fields).Debug("Counter incremented")
	}
}

// RecordHistogram records a value to an OpenTelemetry Float64Histogram metric.
// It uses the meter associated with meterName to find or create the histogram
// identified by histogramName, then records the specified value.
// Attributes can be provided for dimensionality. Logs the recording at Debug level if configured.
func RecordHistogram(ctx context.Context, meterName, histogramName string, value float64, attrs ...attribute.KeyValue) {
	meter := GetMeter(meterName)
	histogram, _ := meter.Float64Histogram(histogramName)
	histogram.Record(ctx, value, metric.WithAttributes(attrs...))

	if globalLogger != nil && globalLogger.IsLevelEnabled(logrus.DebugLevel) {
		fields := logrus.Fields{
			"histogram_name": histogramName,
			"value":          value,
			"meter_name":     meterName,
		}
		for _, attr := range attrs {
			fields[string(attr.Key)] = attr.Value.AsInterface()
		}
		globalLogger.WithFields(fields).Debug("Histogram value recorded")
	}
}

// SetGauge attempts to set an OpenTelemetry gauge-like metric using a Float64UpDownCounter.
// Note: True gauges observe a value, while UpDownCounters are cumulative. This function
// simply adds the value to the counter, which approximates a gauge set if the previous value
// isn't tracked externally. Use with caution for true gauge semantics.
// Attributes can be provided for dimensionality. Logs the operation at Debug level if configured.
func SetGauge(ctx context.Context, meterName, gaugeName string, value float64, attrs ...attribute.KeyValue) {
	meter := GetMeter(meterName)
	gauge, _ := meter.Float64UpDownCounter(gaugeName)

	// Since UpDownCounter is cumulative, we'd need to track previous value to make it behave like a gauge
	// For simplicity, this is a basic implementation
	gauge.Add(ctx, value, metric.WithAttributes(attrs...))

	if globalLogger != nil && globalLogger.IsLevelEnabled(logrus.DebugLevel) {
		fields := logrus.Fields{
			"gauge_name": gaugeName,
			"value":      value,
			"meter_name": meterName,
		}
		for _, attr := range attrs {
			fields[string(attr.Key)] = attr.Value.AsInterface()
		}
		globalLogger.WithFields(fields).Debug("Gauge value set")
	}
}
