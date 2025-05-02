package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InstrumentationName is the name of the instrumentation library.
const InstrumentationName = "github.com/narender/common/otel"

// Standard metric names
const (
	httpServerRequestCount    = "http.server.request.count"
	httpServerRequestDuration = "http.server.request.duration"
	httpServerActiveRequests  = "http.server.active_requests"
)

// Standard attribute keys (Consider moving generic ones to a central place if reused across trace/metric/log)
var (
	AttrHTTPRequestMethod  = semconv.HTTPRequestMethodKey
	AttrHTTPResponseStatus = semconv.HTTPResponseStatusCodeKey
	AttrNetHostName        = semconv.NetHostNameKey // e.g., "example.com"
	AttrNetHostPort        = semconv.NetHostPortKey // e.g., 8080
	AttrURLPath            = semconv.URLPathKey     // e.g., "/users/:userID"
	AttrURLScheme          = semconv.URLSchemeKey   // e.g., "http", "https"
)

// Metrics provides helper methods to record common application metrics.
// It encapsulates OTel metric instruments.
type Metrics struct {
	meter                   otelmetric.Meter
	httpReqCounter          otelmetric.Int64Counter
	httpReqDurationHist     otelmetric.Float64Histogram
	httpActiveRequestsGauge otelmetric.Int64UpDownCounter
	// Add other common instruments here (e.g., db call duration, cache hits/misses)
}

// NewMetrics creates a new Metrics helper.
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
		// Add explicit bucket boundaries if needed for duration
		// otelmetric.WithExplicitBucketBoundaries(10, 50, 100, 200, 500, 1000, 5000),
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

// RecordHTTPRequestDuration records the duration of an HTTP request and increments the count.
// Attributes should include http method, route/path, status code.
func (m *Metrics) RecordHTTPRequestDuration(ctx context.Context, duration time.Duration, attributes ...attribute.KeyValue) {
	if m == nil {
		return // Or log a warning
	}

	// Add 1 to the request counter with the same attributes
	m.httpReqCounter.Add(ctx, 1, otelmetric.WithAttributes(attributes...))

	// Record duration in milliseconds
	m.httpReqDurationHist.Record(ctx, float64(duration.Milliseconds()), otelmetric.WithAttributes(attributes...))
}

// AddActiveRequest increments or decrements the gauge for active HTTP requests.
// Use delta=1 when a request starts, delta=-1 when it ends.
// Attributes should ideally include method and route/path, potentially host.
func (m *Metrics) AddActiveRequest(ctx context.Context, delta int64, attributes ...attribute.KeyValue) {
	if m == nil {
		return // Or log a warning
	}
	m.httpActiveRequestsGauge.Add(ctx, delta, otelmetric.WithAttributes(attributes...))
}

// Note on Observable Gauges:
// As mentioned in the plan, registering specific observable gauge callbacks (like product stock)
// often requires access to service-specific logic (e.g., a repository or service client).
// Therefore, such gauges are typically registered within the service itself (e.g., in main.go or handler.go)
// using the Meter obtained via GetMeter() or from the MeterProvider.
//
// func registerProductStockCallback(meter otelmetric.Meter, service ProductFetcher) error {
// 	 stockGauge, err := meter.Int64ObservableGauge("product.stock.level", ...)
// 	 if err != nil { return err }
// 	 _, err = meter.RegisterCallback(
// 		 func(ctx context.Context, obs otelmetric.Observer) error {
// 			 products, _ := service.GetAllProducts(ctx) // Fetch data
// 			 for _, p := range products {
// 				 obs.ObserveInt64(stockGauge, int64(p.Stock), otelmetric.WithAttributes(attribute.String("product.id", p.ID)))
// 			 }
// 			 return nil
// 		 },
// 		 stockGauge,
// 	 )
// 	 return err
// }
