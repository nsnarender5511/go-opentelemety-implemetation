package metric

import (
	"context"
	"fmt"
	"time"

	"github.com/narender/common/telemetry/manager"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

const (
	HTTPMetricsInstrumentationName = "github.com/narender/common/telemetry/metric"
)

const (
	httpServerRequestCount    = "http.server.request.count"
	httpServerRequestDuration = "http.server.request.duration"
	httpServerActiveRequests  = "http.server.active_requests"
)



type HTTPMetrics struct {
	httpReqCounter          otelmetric.Int64Counter
	httpReqDurationHist     otelmetric.Float64Histogram
	httpActiveRequestsGauge otelmetric.Int64UpDownCounter
}



func NewHTTPMetrics() (*HTTPMetrics, error) {

	meter := manager.GetMeter(HTTPMetricsInstrumentationName)
	var err error
	
	appMetrics := &HTTPMetrics{}

	appMetrics.httpReqCounter, err = meter.Int64Counter(
		httpServerRequestCount,
		otelmetric.WithDescription("Number of HTTP requests received"),
		otelmetric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s counter: %w", httpServerRequestCount, err)
	}

	appMetrics.httpReqDurationHist, err = meter.Float64Histogram(
		httpServerRequestDuration,
		otelmetric.WithDescription("Duration of HTTP requests"),
		otelmetric.WithUnit("ms"),
		otelmetric.WithExplicitBucketBoundaries(
			5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s histogram: %w", httpServerRequestDuration, err)
	}

	appMetrics.httpActiveRequestsGauge, err = meter.Int64UpDownCounter(
		httpServerActiveRequests,
		otelmetric.WithDescription("Number of active HTTP requests"),
		otelmetric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s updowncounter: %w", httpServerActiveRequests, err)
	}

	manager.GetLogger().Info("Common HTTP metrics initialized.")
	return appMetrics, nil
}



func (m *HTTPMetrics) RecordHTTPRequestDuration(ctx context.Context, duration time.Duration, attributes ...attribute.KeyValue) {
	if m == nil {
		return
	}

	metricOpts := []otelmetric.AddOption{
		otelmetric.WithAttributes(attributes...),
	}
	metricRecordOpts := []otelmetric.RecordOption{
		otelmetric.WithAttributes(attributes...),
	}

	if m.httpReqCounter != nil {
		m.httpReqCounter.Add(ctx, 1, metricOpts...)
	}
	if m.httpReqDurationHist != nil {
		m.httpReqDurationHist.Record(ctx, float64(duration.Milliseconds()), metricRecordOpts...)
	}
}



func (m *HTTPMetrics) AddActiveRequest(ctx context.Context, delta int64, attributes ...attribute.KeyValue) {
	if m == nil || m.httpActiveRequestsGauge == nil {
		return
	}
	m.httpActiveRequestsGauge.Add(ctx, delta, otelmetric.WithAttributes(attributes...))
}
