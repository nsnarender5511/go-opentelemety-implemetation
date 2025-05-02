package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

const InstrumentationName = "github.com/narender/common/otel"

const (
	httpServerRequestCount    = "http.server.request.count"
	httpServerRequestDuration = "http.server.request.duration"
	httpServerActiveRequests  = "http.server.active_requests"
)

var (
	AttrHTTPRequestMethod  = semconv.HTTPRequestMethodKey
	AttrHTTPResponseStatus = semconv.HTTPResponseStatusCodeKey
	AttrNetHostName        = semconv.NetHostNameKey
	AttrNetHostPort        = semconv.NetHostPortKey
	AttrURLPath            = semconv.URLPathKey
	AttrURLScheme          = semconv.URLSchemeKey
)

type Metrics struct {
	meter                   otelmetric.Meter
	httpReqCounter          otelmetric.Int64Counter
	httpReqDurationHist     otelmetric.Float64Histogram
	httpActiveRequestsGauge otelmetric.Int64UpDownCounter
}

func NewMetrics(provider otelmetric.MeterProvider) (*Metrics, error) {
	meter := provider.Meter(InstrumentationName)

	httpReqCounter, err := meter.Int64Counter(
		httpServerRequestCount,
		otelmetric.WithDescription("Number of HTTP requests received"),
		otelmetric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, err
	}

	httpReqDurationHist, err := meter.Float64Histogram(
		httpServerRequestDuration,
		otelmetric.WithDescription("Duration of HTTP requests"),
		otelmetric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	httpActiveRequestsGauge, err := meter.Int64UpDownCounter(
		httpServerActiveRequests,
		otelmetric.WithDescription("Number of active HTTP requests"),
		otelmetric.WithUnit("{requests}"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		meter:                   meter,
		httpReqCounter:          httpReqCounter,
		httpReqDurationHist:     httpReqDurationHist,
		httpActiveRequestsGauge: httpActiveRequestsGauge,
	}, nil
}

func (m *Metrics) RecordHTTPRequestDuration(ctx context.Context, duration time.Duration, attributes ...attribute.KeyValue) {
	if m == nil {
		return
	}

	m.httpReqCounter.Add(ctx, 1, otelmetric.WithAttributes(attributes...))

	m.httpReqDurationHist.Record(ctx, float64(duration.Milliseconds()), otelmetric.WithAttributes(attributes...))
}

func (m *Metrics) AddActiveRequest(ctx context.Context, delta int64, attributes ...attribute.KeyValue) {
	if m == nil {
		return
	}
	m.httpActiveRequestsGauge.Add(ctx, delta, otelmetric.WithAttributes(attributes...))
}

