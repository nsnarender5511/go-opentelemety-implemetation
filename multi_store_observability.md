# Comprehensive Multi-Store Observability Strategy

## 1. Introduction and Goals

### 1.1. Purpose
This document outlines a comprehensive strategy for designing and implementing a robust observability solution for the multi-store e-commerce platform, specifically focusing on the `product-service` and its interactions. The goal is to enable deep insights into system behavior, operational health, and business performance across all individual store instances and in aggregate.

### 1.2. Key Observability Goals
*   **Operational Excellence:** Proactively identify and resolve issues, minimize downtime, and ensure service reliability and performance.
*   **Business Insights:** Understand revenue trends, sales patterns, product popularity, and inventory impact on a per-store and platform-wide basis.
*   **Enhanced Developer Productivity:** Accelerate debugging, performance tuning, and understanding of code behavior in production.
*   **Scalability and Maintainability:** Build an observability framework that scales with the growth of services and store instances.

### 1.3. Benefits of Comprehensive Telemetry
A well-implemented observability strategy provides:
*   **Faster Mean Time to Detection (MTTD) and Resolution (MTTR):** Quickly pinpoint root causes of problems.
*   **Data-Driven Decision Making:** For business strategy, inventory management, and technical improvements.
*   **Improved User Experience:** By ensuring service availability, performance, and quick resolution of issues.
*   **Clear Understanding of System Dependencies:** Visualizing how services interact and impact each other.

## 2. Core Observability Pillars
Our strategy is built upon the three core pillars of observability:

*   **Metrics:** Aggregated, numerical data representing the health and performance of the system over time. Ideal for dashboards, alerting, and trend analysis.
*   **Distributed Traces:** Detailed records of a single request's journey as it flows through multiple services or components. Essential for understanding latency, dependencies, and request-specific errors.
*   **Logs:** Timestamped, contextual records of discrete events occurring within the system. Provide ground-truth details for debugging and auditing.

The true power of observability comes from the **correlation** of these pillars – being able to seamlessly navigate from a problematic metric spike to relevant traces, and then to the specific logs for those traces.

## 3. The Telemetry Collection Pipeline: From Instrumentation to Backend (New Section)

Understanding how telemetry data is generated, collected, processed, and exported is key to implementing and maintaining an effective observability solution. This section details the components and flow involved.

### 3.1. Overview of the Telemetry Pipeline

The typical flow of telemetry data in an OpenTelemetry-based system is as follows:

1.  **Instrumentation:** Application code is instrumented using OpenTelemetry APIs and SDKs to generate telemetry signals (traces, metrics, logs).
2.  **OpenTelemetry SDK (in-app):** The SDK within the application collects these signals, processes them (e.g., batching spans), and exports them using configured exporters.
3.  **OpenTelemetry Collector (Optional but Highly Recommended):** A separate agent or gateway service that receives telemetry from multiple sources (including applications and infrastructure like Docker). It can process (e.g., filter, enrich, sample) and export data to one or more telemetry backends.
4.  **Telemetry Backend (e.g., SigNoz):** A specialized system for ingesting, storing, querying, visualizing, and alerting on the collected telemetry data.

### 3.2. Application Instrumentation with OpenTelemetry Go SDK

Instrumentation is the process of adding code to your application to emit telemetry signals. The OpenTelemetry Go SDK provides the tools for this.

#### 3.2.1. Core Concept: The OpenTelemetry Resource

A **Resource** is a fundamental concept in OpenTelemetry. It represents the entity producing telemetry (e.g., your `product-service` instance for `store-alpha`). Attributes attached to a Resource are automatically associated with *all* telemetry emitted by that SDK instance, ensuring consistent tagging.

*   **Definition:** Resources are typically defined once during SDK initialization.
*   **Key Attributes for Us:** `service.name`, `service.version`, `service.instance.id`, and our custom `store.id`.
*   **Setting Resource Attributes:** The most convenient way is via the `OTEL_RESOURCE_ATTRIBUTES` environment variable, as planned:
    `OTEL_RESOURCE_ATTRIBUTES=service.name=product-service,store.id=alpha_store,service.instance.id=product-service-alpha-$(hostname),service.version=1.0.0`
    The Go SDK automatically detects this.

*Go Code Example (Illustrative - SDK detects env var by default):*
```go
// main.go or telemetry_setup.go
import (
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0" // Use appropriate version
)

func newResource(ctx context.Context, serviceName, serviceVersion, storeID, instanceID string) (*resource.Resource, error) {
    // If OTEL_RESOURCE_ATTRIBUTES is set, it often merges with or overrides programmatic defaults.
    // For explicit programmatic setup (can be combined with env var detection):
    res, err := resource.Merge(
        resource.Default(), // Detects OTEL_RESOURCE_ATTRIBUTES and other defaults
        resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
            semconv.ServiceVersionKey.String(serviceVersion),
            semconv.ServiceInstanceIDKey.String(instanceID),
            attribute.String("store.id", storeID), // Our custom attribute
        ),
    )
    return res, err
}
```

#### 3.2.2. SDK Initialization: Providers, Exporters, and Processors

At application startup, you need to initialize the SDK components:

*   **Providers (`TracerProvider`, `MeterProvider`):** These are factories for creating `Tracer` and `Meter` instances, respectively. They are configured with resources, processors (for traces), and exporters.
*   **Exporters:** Responsible for sending telemetry data out of the application. For our setup, the **OTLP (OpenTelemetry Protocol) Exporter** is key, as it sends data in a standard format to the OTel Collector or directly to an OTLP-compatible backend.
*   **Processors (Primarily for Traces):** Define how spans are processed before being exported. The `BatchSpanProcessor` is commonly used to batch spans and send them periodically, which is more efficient than sending each span individually.

*Go Code Example (Simplified Trace Setup):*
```go
// main.go or telemetry_setup.go
import (
	"context"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitTracerProvider initializes an OTLP exporter and a TracerProvider
func InitTracerProvider(ctx context.Context) (func(context.Context) error, error) {
	// Attempt to detect resource attributes from OTEL_RESOURCE_ATTRIBUTES env var
	// If not set, or for additional programmatic attributes, you could use newResource() from above.
	res, err := resource.New(ctx, resource.WithDetectors(resource.DefaultDetectors()...))
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

    // --- Exporter Setup --- 
	// Example: OTLP gRPC Exporter
	// Assumes OTEL_EXPORTER_OTLP_ENDPOINT is set, e.g., "http://otel-collector:4317"
	// Assumes OTEL_EXPORTER_OTLP_INSECURE is set for local dev, e.g., "true"
	exp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

    // --- Processor Setup --- 
	// BatchSpanProcessor is generally recommended for production.
	bsp := tracesdk.NewBatchSpanProcessor(exp)

    // --- Provider Setup --- 
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp), // Alternative to WithBatchSpanProcessor if using older API
        // tracesdk.WithSpanProcessor(bsp), // Use this with newer API if BSP is configured manually
		tracesdk.WithResource(res),
		// Configure sampling if needed (e.g., tracesdk.WithSampler(tracesdk.TraceIDRatioBased(0.1)) for 10%)
		tracesdk.WithSampler(tracesdk.AlwaysSample()), // Sample all for dev/debug
	)
	otel.SetTracerProvider(tp)

	// Set global propagators to W3C Trace Context and Baggage.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Return a shutdown function to ensure telemetry is flushed before app exits.
	return func(ctx context.Context) error {
		 cxt, cancel := context.WithTimeout(ctx, time.Second*5)
		 defer cancel()
		 if err := tp.Shutdown(cxt); err != nil {
			 return fmt.Errorf("failed to shutdown TracerProvider: %w", err)
		 }
		 return nil
	}, nil
}
```
*Note: Metric provider setup (`MeterProvider`) follows a similar pattern with a metric exporter.* 

#### 3.2.3. Instrumenting Traces
Once the `TracerProvider` is set, you obtain a `Tracer` instance to create spans.

*   **Get a Tracer:** `tracer := otel.Tracer("your-instrumentation-scope-name")` (e.g., "product-service/service").
*   **Start a Span:** `ctx, span := tracer.Start(parentContext, "spanName", trace.WithAttributes(...))`.
The `parentContext` is crucial for linking spans into a single trace.
*   **Set Attributes:** `span.SetAttributes(attribute.String("key", "value"))`.
*   **Record Errors:** `span.RecordError(err)` and `span.SetStatus(codes.Error, "operation failed")`.
*   **End a Span:** `defer span.End()` is a common pattern.

*Go Code Example (Basic Span):*
```go
// In a service method
import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

func (s *productService) GetByName(ctx context.Context, name string) (models.Product, error) {
	// Get a tracer. The name should be the instrumentation scope, not the span name.
	// Example: "github.com/yourorg/yourrepo/product-service/services"
	tracer := otel.Tracer("product-service.services") 

	// Start a new span. The context `ctx` is updated with the new span.
	var span trace.Span
	ctx, span = tracer.Start(ctx, "productService.GetByName", trace.WithAttributes(
		attribute.String("product.name.searched", name),
	))
	defer span.End() // Ensure the span is ended when the function returns.

	// ... your business logic ...
	product, err := s.repo.GetByName(ctx, name) // Pass the updated context
	if err != nil {
		span.RecordError(err) // Records the error with stacktrace by default
		span.SetStatus(codes.Error, "failed to get product from repository")
        span.SetAttributes(attribute.Bool("app.product.found", false))
		return models.Product{}, err
	}
    span.SetAttributes(attribute.Bool("app.product.found", true))
    span.SetStatus(codes.Ok, "") // Explicitly set OK status
	return product, nil
}
```

#### 3.2.4. Instrumenting Metrics
Similarly, get a `Meter` from the `MeterProvider` to create and record metrics.

*   **Get a Meter:** `meter := otel.Meter("your-instrumentation-scope-name")`.
*   **Create Instruments:**
    *   `counter, _ := meter.Int64Counter("app.items.sold.count", metric.WithDescription("Items sold"), metric.WithUnit("{item}"))`
    *   `histogram, _ := meter.Float64Histogram("http.server.request.duration", ...)`
*   **Record Values:** `counter.Add(ctx, 1, metric.WithAttributes(attribute.String("product.category", "electronics")))`.

*Go Code Example (Basic Counter):*
```go
// In a service method where an item is sold, e.g., within product-service/src/services/buy_product_service.go
import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var itemsSoldCounter metric.Int64Counter

func init() { // Or in your MeterProvider setup function
    meter := otel.Meter("product-service.services") // Consistent scope name
    var err error
    itemsSoldCounter, err = meter.Int64Counter(
        "app.items.sold.count",
        metric.WithDescription("Total number of items sold"),
        metric.WithUnit("{item}"),
    )
    if err != nil {
        log.Fatalf("Failed to create itemsSoldCounter: %v", err)
    }
}

func (s *productService) RecordSale(ctx context.Context, category string, quantity int) {
    // ... other business logic ... 
    if itemsSoldCounter != nil {
        itemsSoldCounter.Add(ctx, int64(quantity), metric.WithAttributes(
            attribute.String("product.category", category),
            // store.id will be added automatically if it's a resource attribute
        ))
    }
}
```

#### 3.2.5. Instrumenting Logs (Integrating with `slog`)
For logs, the goal is to ensure they are structured and automatically correlated with traces (`trace_id`, `span_id`) and resources (`store.id`, `service.name`).

*   **Use `slog`:** As already planned for structured logging.
*   **Trace Context Injection:** The crucial part is configuring your `slog.Handler` to extract `trace_id` and `span_id` from the `context.Context` and include them in log records. Resource attributes can also be added by the handler.
    *   This might involve a custom `slog.Handler` wrapper or using an OpenTelemetry-provided logging library/exporter for `slog` if one matures for Go (the ecosystem is evolving).
    *   A simpler approach if not using a full OTel logging SDK is to manually extract and add them, but this is less ideal.

*Go Code Example (Conceptual `slog` with manual trace context - a proper handler is better):*
```go
// In a service method, e.g., within a file in product-service/src/services/
import (
	"context"
	"log/slog"
	"go.opentelemetry.io/otel/trace"
)

func (s *productService) ProcessOrder(ctx context.Context, orderID string) {
    span := trace.SpanFromContext(ctx)
    // This manual addition is illustrative; a handler should do this automatically.
    s.logger.InfoContext(ctx, "Processing order",
        slog.String("order_id", orderID),
        slog.String("trace_id", span.SpanContext().TraceID().String()),
        slog.String("span_id", span.SpanContext().SpanID().String()),
        // store_id and other resource attributes would ideally be added by the handler
    )
    // ... logic ...
}
```

### 3.3. OpenTelemetry Collector (OTel Collector)

The OTel Collector is a vendor-agnostic proxy that can receive, process, and export telemetry data. It plays a vital role in a scalable and flexible observability pipeline.

#### 3.3.1. Role and Benefits
*   **Decoupling:** Your application exports data in a standard OTLP format, and the Collector handles routing it to one or more backends. This makes switching backends easier.
*   **Centralized Processing:** The Collector can perform operations like:
    *   **Batching:** Efficiently groups telemetry before sending it to backends.
    *   **Retries:** Handles temporary backend unavailability.
    *   **Attribute Manipulation:** Add, modify, or remove attributes (e.g., scrub sensitive data, add common tags).
    *   **Sampling:** Perform tail-based sampling for traces (make sampling decisions after all spans for a trace are collected).
    *   **Filtering:** Drop unnecessary telemetry.
*   **Infrastructure Metrics:** Can collect metrics from the host or other services (e.g., Docker Stats Receiver, Prometheus Receiver).
*   **Reduced Application Load:** Offloads some processing and export responsibility from the application SDK.

#### 3.3.2. Key Components of the Collector
*   **Receivers:** Define how data gets *into* the Collector.
    *   *Examples:* `otlp` (for OTLP gRPC/HTTP from SDKs), `prometheus` (to scrape Prometheus endpoints), `filelog` (to collect logs from files), `docker_stats` (to collect Docker container metrics).
*   **Processors:** Define how data is *transformed* within the Collector.
    *   *Examples:* `batch` (groups telemetry), `memory_limiter` (prevents out-of-memory issues), `attributes` (modify attributes), `span` (modify spans, e.g., rename), `filter` (drop data based on criteria), `tailsampling` (for traces).
*   **Exporters:** Define how data gets *out* of the Collector to backends.
    *   *Examples:* `otlp` (to SigNoz or other OTLP backends), `prometheusremotewrite`, `loki` (for logs), `logging` (prints telemetry to console, useful for debugging Collector config).
*   **Extensions:** Provide auxiliary capabilities.
    *   *Examples:* `health_check` (HTTP endpoint for Collector health), `pprof` (for Go profiling of Collector), `zpages` (debugging pages).
*   **Connectors:** A newer component type that acts as both an exporter from one pipeline and a receiver for another, enabling more complex data flows within the collector itself.

#### 3.3.3. Service Pipelines
The Collector configuration (`otel-collector-config.yaml`) defines **pipelines** that link receivers, processors, and exporters for each signal type (traces, metrics, logs).

*Example (Conceptual pipeline in `otel-collector-config.yaml`):*
```yaml
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [otlp] # To SigNoz
    metrics:
      receivers: [otlp, docker_stats]
      processors: [batch]
      exporters: [otlp] # To SigNoz
    logs:
      receivers: [otlp] # If apps send OTLP logs, or filelog receiver
      processors: [batch]
      exporters: [otlp] # To SigNoz, or loki
```

### 3.4. Telemetry Backend (e.g., SigNoz)

This is where your telemetry data is ultimately sent for long-term storage, querying, visualization, and alerting. SigNoz, being built on OpenTelemetry, natively supports OTLP and provides integrated views for metrics, traces, and logs.

### 3.5. Data Flow Summary

1.  **`product-service` (Go App):**
    *   Uses OpenTelemetry Go SDK.
    *   **Resource** defined (e.g., `service.name`, `store.id`).
    *   Traces/Metrics instrumented, Logs structured.
    *   Data processed by `BatchSpanProcessor` (for traces).
    *   Data exported via **OTLP Exporter** to the OTel Collector.
2.  **OpenTelemetry Collector:**
    *   **OTLP Receiver** ingests data from the application.
    *   (Optional) **Docker Stats Receiver** collects container metrics.
    *   Data flows through **Processors** (e.g., `batch`, `memory_limiter`, potentially `attributes` or `filter`).
    *   Processed data sent via **OTLP Exporter** to SigNoz.
3.  **SigNoz (Backend):**
    *   Receives, stores, and indexes the telemetry.
    *   Provides UI for dashboards, querying (e.g., ClickHouse SQL for traces/logs, PromQL for metrics), trace exploration, and alerting.

This detailed pipeline ensures that telemetry is consistently tagged, efficiently processed, and made available for comprehensive analysis and visualization, forming the backbone of your multi-store observability strategy.

## 4. Store Identification Strategy: The `store_id` Bedrock

To differentiate telemetry from multiple service instances (each representing a "store"), a unique `store_id` must be consistently associated with all emitted data.

### 4.1. Recommended Approach
*   **Environment Variable (`STORE_ID`):** Each service instance defined in `docker-compose.yml` will be configured with a unique `STORE_ID` environment variable.
    *   *Example:* `STORE_ID=alpha_store`
*   **OpenTelemetry Resource Attribute (`store.id`):** This `STORE_ID` will be set as an OpenTelemetry Resource attribute, preferably named `store.id`. This is best achieved by setting the `OTEL_RESOURCE_ATTRIBUTES` environment variable.
    *   *Example:* `OTEL_RESOURCE_ATTRIBUTES=service.name=product-service,store.id=alpha_store,service.instance.id=product-service-alpha_store-$(hostname)`
    *   The `service.instance.id` (using `hostname` or another unique identifier per container) helps distinguish between multiple replicas of the *same* store if horizontal scaling is implemented for a single store.

### 4.2. Criticality for Multi-Store Views
This explicit `store.id` tagging is fundamental for:
*   Filtering and segmenting data in dashboards and queries on a per-store basis.
*   Aggregating data for a platform-wide overview.
*   Comparing performance and business metrics across different stores.

### 4.3. How Resource Attributes Ensure Telemetry Consistency

The OpenTelemetry SDKs are designed to automatically associate defined **Resource** attributes with all telemetry signals (metrics, traces, and logs) emitted by that SDK instance. This is key to our strategy:

1.  **OpenTelemetry Resource:** A Resource is an immutable representation of the entity producing telemetry (e.g., a specific `product-service` instance for `store-alpha`). It's defined once during SDK initialization.
2.  **`OTEL_RESOURCE_ATTRIBUTES` Environment Variable:** By setting this environment variable (e.g., `OTEL_RESOURCE_ATTRIBUTES=service.name=product-service,store.id=alpha_store,...`), the Go OpenTelemetry SDK automatically configures the Resource for that specific service instance.
3.  **Automatic Association:**
    *   **Metrics:** All metrics will inherently include `store.id`, `service.name`, etc., as dimensions.
    *   **Traces:** All spans within traces generated by this instance will be linked to this Resource.
    *   **Logs:** If the logging setup is integrated with the OpenTelemetry SDK (e.g., a `slog.Handler` that enriches with resource attributes, or an OTel logging exporter), logs will also carry these resource attributes.

This mechanism ensures that every piece of telemetry is automatically and consistently tagged without needing to manually add `store.id` to every instrumentation call.

## 5. Detailed Metrics Strategy

Metrics provide quantifiable insights into system health and business performance.

### 5.1. Guiding Principles for Metric Collection
*   **Actionability:** Focus on metrics that drive action (e.g., RED metrics - Rate, Errors, Duration; USE metrics - Utilization, Saturation, Errors; Business KPIs).
*   **Standardization:** Use consistent naming conventions (e.g., `app.<component>.<measurement>.<unit>`) and leverage OpenTelemetry semantic conventions where applicable.
*   **Dimensionality:** Utilize attributes (labels/tags) extensively to allow for filtering, grouping, and segmentation (e.g., by `store.id`, `product.category`, `http.route`).

### 5.2. Business-Critical Metrics (KPIs)
These metrics directly reflect the business impact of the service.

*   **`app.revenue.total` (Counter)**
    *   **Description:** Total revenue generated from product sales.
    *   **Unit:** Currency (e.g., `USD`).
    *   **Attributes:** `product.name`, `product.category`, `currency_code`, `store.id`.
    *   **Calculation:** Sum of `product.Price * quantity` for each successful sale.
    *   **Aggregations:** Per-minute, per-hour, per-day totals.
*   **`app.items.sold.count` (Counter)**
    *   **Description:** Total number of items sold.
    *   **Unit:** `{item}`.
    *   **Attributes:** `product.name`, `product.category`, `store.id`.
    *   **Calculation:** Sum of `quantity` for each successful sale.
*   **`app.cart.abandonment.rate` (Gauge) - *Aspirational, if cart feature exists***
    *   **Description:** Percentage of carts created but not converted to a sale.
    *   **Unit:** `%`.
    *   **Attributes:** `store.id`.
*   **`app.customer.conversion.rate` (Gauge) - *Aspirational***
    *   **Description:** Percentage of product views or interactions leading to a purchase.
    *   **Unit:** `%`.
    *   **Attributes:** `product.category`, `store.id`.

### 5.3. Operational & Performance Metrics

#### 5.3.1. Inventory Metrics
*   **`app.product.stock.current` (Observable Gauge, derived from `ProductInventoryCountMetric`)**
    *   **Description:** Current stock level for each product.
    *   **Unit:** `{item}`.
    *   **Attributes:** `product.name`, `product.category`, `store.id`.
    *   **Source:** Updated via `commonmetric.UpdateProductStockLevels()`.
*   **`app.product.stock.updates.count` (Counter)**
    *   **Description:** Number of times stock levels for any product have been updated.
    *   **Unit:** `{update}`.
    *   **Attributes:** `product.name` (optional, if updating one product), `update.type` (e.g., "sale", "manual_restock", "initial_load"), `store.id`.
*   **`app.stock.insufficient.events.count` (Counter)**
    *   **Description:** Number of times a purchase attempt failed due to insufficient stock.
    *   **Unit:** `{event}`.
    *   **Attributes:** `product.name`, `product.category`, `requested.quantity`, `available.quantity`, `store.id`.

#### 5.3.2. Application Performance Metrics (SLIs) - Leveraging OpenTelemetry Semantic Conventions
*   **`http.server.request.duration` (Histogram, from `otelhttp` instrumentation)**
    *   **Description:** Distribution of latencies for HTTP requests received by the service.
    *   **Unit:** `s` (seconds) or `ms` (milliseconds).
    *   **Attributes:** `http.method`, `http.route`, `http.status_code`, `store.id` (via resource).
*   **`http.server.active_requests` (UpDownCounter, from `otelhttp` instrumentation)**
    *   **Description:** Number of concurrent, in-flight HTTP requests.
    *   **Unit:** `{request}`.
    *   **Attributes:** `http.method`, `http.route`, `store.id` (via resource).
*   **`http.server.request.count` (Counter, often derived from duration histogram)**
    *   **Description:** Total number of HTTP requests processed.
    *   **Unit:** `{request}`.
    *   **Attributes:** `http.method`, `http.route`, `http.status_code`, `store.id` (via resource).
*   **`app.product.notfound.count` (Counter)**
    *   **Description:** Number of times a product search (e.g., `GetByName`) resulted in "not found".
    *   **Unit:** `{event}`.
    *   **Attributes:** `product.name.searched`, `store.id`.
*   **`app.startup.duration` (Gauge/Histogram)**
    *   **Description:** Time taken for the service to initialize and become ready.
    *   **Unit:** `ms`.
    *   **Attributes:** `store.id`.
*   **`app.config.load.success` (Gauge: 0 or 1)**
    *   **Description:** Indicates if the product data file was loaded successfully at startup.
    *   **Attributes:** `store.id`.

#### 5.3.3. Dependency Metrics
*   **FileDB Operations (Custom Metrics, if `db.FileDatabase` is not auto-instrumented):**
    *   `app.filedb.read.duration` (Histogram): Latency of reading the data file.
    *   `app.filedb.write.duration` (Histogram): Latency of writing to the data file.
    *   `app.filedb.read.errors.count` (Counter): Errors during file read.
    *   `app.filedb.write.errors.count` (Counter): Errors during file write.
    *   **Attributes:** `file.name`, `operation.type` (e.g., "full_read", "stock_update_write"), `store.id`.

### 5.4. Instrumentation Points for Metrics (Detailed)
This section maps specific metrics to their intended instrumentation points in the codebase.

*   **`app.revenue.total`, `app.items.sold.count`:**
    *   **Location:** `product-service/src/services/buy_product_service.go`, within the `BuyProduct` function, after successful stock validation and inventory update.
    *   **Action:** Call corresponding increment functions in `common/telemetry/metric` with appropriate attributes.
*   **`app.product.stock.current` (via `commonmetric.UpdateProductStockLevels`):**
    *   **Location 1:** `product-service/src/repositories/get_all_repository.go`, end of `GetAll` function.
        *   **Action:** Prepare `map[string]int64` of all product stocks and call `commonmetric.UpdateProductStockLevels()`.
    *   **Location 2:** `product-service/src/repositories/update_stock_repository.go`, end of `UpdateStock` function (after successful write).
        *   **Action:** After updating `productsMap`, prepare the full `map[string]int64` and call `commonmetric.UpdateProductStockLevels()`.
    *   **Location 3 (Optional):** A periodic ticker in `product-service/src/repositories/repository.go`'s `recordStockLevels` function (if such a mechanism is implemented there).
*   **`app.product.stock.updates.count`:**
    *   **Location:** `product-service/src/repositories/update_stock_repository.go`, within `UpdateStock` after successful write.
    *   **Action:** Increment counter with attributes indicating product name (if single) and `store.id`.
*   **`app.stock.insufficient.events.count`:**
    *   **Location:** `product-service/src/services/buy_product_service.go`, within `BuyProduct` when `product.Stock < quantity` condition is met.
    *   **Action:** Increment counter with product, quantity, and store attributes.
*   **`app.product.notfound.count`:**
    *   **Location:** `product-service/src/services/get_by_name_service.go`, within `GetByName` if the repository returns a "not found" error.
    *   **Action:** Increment counter with searched name and `store.id`.
*   **`app.startup.duration`, `app.config.load.success`:**
    *   **Location:** `product-service/src/main.go` or equivalent service initialization logic.
    *   **Action:** Record time at start and end of initialization; emit metrics.
*   **FileDB Metrics (`app.filedb.*`):**
    *   **Location:** `common/db/file_database.go` (or similar for `FileDatabase` implementation).
    *   **Action:** Wrap `Read` and `Write` operations with timing and error counting logic.

## 6. Distributed Tracing Strategy

Distributed tracing provides an end-to-end view of requests as they traverse system components.

### 6.1. Importance of Distributed Traces
*   **Latency Analysis:** Pinpoint bottlenecks and slow operations within a request's lifecycle.
*   **Dependency Visualization:** Understand service interactions and critical paths.
*   **Error Triaging:** Correlate errors to specific requests and their context.

### 6.2. Trace Propagation and Context
*   **Standard:** Adhere to W3C Trace Context standard for interoperability.
*   **Mechanism:** OpenTelemetry SDKs typically handle propagation automatically via HTTP headers (e.g., `traceparent`, `tracestate`) or other protocol-specific mechanisms.
*   **Scope:** Ensure context propagates through all service hops, asynchronous boundaries (e.g., goroutines that need the parent context), and any message queue interactions (if added later).

### 6.3. Span Granularity and Naming Conventions
*   **Span Naming:** Use clear, consistent names. A common convention is `<package>.<StructName>.<MethodName>` or `<protocol> <operation>` (e.g., `http GET /products/{name}`, `productService.GetByName`).
*   **Child Spans vs. Span Events:**
    *   **Child Spans:** For distinct logical units of work that have a measurable duration and can potentially fail independently (e.g., a call to the repository from the service, a file read operation).
    *   **Span Events:** For significant point-in-time occurrences within a span (e.g., "stock check started", "cache hit").
*   **Detailed Breakdown Examples:**
    *   **`BuyProduct` (in `product-service/src/services/buy_product_service.go`):**
        *   Parent Span: `productService.BuyProduct`
        *   Span Event: `stock.check.start`
        *   Span Event: `stock.check.end` (with attributes)
        *   Child Span (optional, if repo call is complex): `productRepository.GetByName` (for initial stock check, maps to method in `product-service/src/repositories/get_by_name_repository.go`)
        *   Child Span: `productRepository.UpdateStock` (maps to method in `product-service/src/repositories/update_stock_repository.go`)
        *   Span Event: `inventory.update.end` (with attributes)
    *   **`GetByName` (in `product-service/src/services/get_by_name_service.go`):**
        *   Parent Span: `productService.GetByName`
        *   Child Span: `productRepository.GetByName` (maps to method in `product-service/src/repositories/get_by_name_repository.go`)
    *   **`FileDatabase.Read` (db/file_database.go):**
        *   Parent Span: `FileDatabase.Read`
        *   Span Event: `file.open.attempt`
        *   Span Event: `file.read.success` or `file.read.error`
        *   Span Event: `json.unmarshal.attempt`

### 6.4. Key Span Attributes (Semantic Conventions + Custom)
Enrich spans with attributes for context and filterability. `store.id` will be added via resource attributes.

*   **OpenTelemetry Semantic Conventions:**
    *   **HTTP:** `http.method`, `http.route`, `http.status_code`, `http.request.header.<key>`, `http.response.header.<key>`, `net.peer.ip`.
    *   **RPC:** `rpc.system`, `rpc.service`, `rpc.method`.
    *   **Database:** `db.system`, `db.name`, `db.statement`, `db.operation`.
    *   **Messaging (if used later):** `messaging.system`, `messaging.destination.name`, `messaging.message.id`.
    *   **Error Attributes:** `otel.status_code` (ERROR), `otel.status_description`, `exception.type`, `exception.message`, `exception.stacktrace` (use with caution, can be verbose).
*   **Custom Business/Application Attributes:**
    *   **`GetAll` / `GetByCategory` spans:**
        *   `app.response.product.count`: Number of products returned.
    *   **`GetByName` spans:**
        *   `app.product.name.searched`
        *   `app.product.found`: (boolean) true/false.
        *   If found: `app.product.price`, `app.product.stock_level`.
    *   **`UpdateStock` spans:**
        *   `app.product.name`
        *   `app.request.product.new_stock`
        *   `app.product.old_stock` (actual stock before update).
    *   **`BuyProduct` spans:**
        *   `app.product.name`
        *   `app.product.purchase_quantity`
        *   `app.product.price.at_purchase`
        *   `app.transaction.value` (calculated: price * quantity)
        *   `app.response.product.remaining_stock`
    *   **Error Context:**
        *   `app.error.code`: (e.g., `apierrors.ErrCodeNotFound.String()`)
        *   `app.error.is_retryable`: (boolean, if applicable).

### 6.5. Sampling Strategy
*   **Purpose:** Manage the volume of trace data sent to the backend, especially in high-traffic production environments.
*   **Types:**
    *   **Head-based Sampling:** Decision made at the beginning of a trace. Simpler but can miss important infrequent errors.
    *   **Tail-based Sampling:** Decision made after all spans for a trace are completed. More complex but allows keeping traces with errors or specific characteristics.
*   **Initial Recommendation:**
    *   **Development/Staging:** 100% sampling (AlwaysSample).
    *   **Production:** Start with parent-based sampling (if a parent span is sampled, the child is too) combined with a probabilistic sampler (e.g., sample 10% of root spans). Adjust based on volume and observability needs. Consider tail-based sampling via the OTEL Collector if specific error traces are critical.

## 7. Comprehensive Logging Strategy

Logs provide detailed, event-specific information, acting as the ground truth for system activities.

### 7.1. Philosophy: Logs as Structured Events
Treat logs not just as free-form text but as structured data representing specific events, making them machine-parseable and queryable.

### 7.2. Structured Logging (with `slog`)
*   **Format:** Strictly use key-value pairs for all contextual information. This is native to `slog`.
*   **Output:** Configure `slog` to output JSON in production environments for easier parsing by log management systems. Use a human-readable console handler for local development.
    *   *Example `main.go` setup:*
        ```go
        // if production {
        //  logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
        // } else {
        //  logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}))
        // }
        // slog.SetDefault(logger)
        ```

### 7.3. Trace-Log Correlation (The `trace_id` and `span_id` Link)
This is **critical** for seamless navigation between traces and logs.
*   **Mechanism:** Ensure `trace_id` and `span_id` from the active OpenTelemetry trace context are automatically injected into every structured log entry.
*   **Implementation:**
    *   Use an `slog.Handler` wrapper that extracts trace context from `context.Context` and adds it to log records.
    *   Alternatively, OpenTelemetry logging exporters or bridges for `slog` can handle this. Verify the setup of `globals.Logger()`.
    *   *Example Attribute Names:* `trace_id`, `span_id`.

### 7.4. Essential Log Fields (Standardization)
Ensure these fields are consistently present in logs:
*   `time` (or `ts`): Timestamp in ISO 8601 UTC format.
*   `level` (or `severity`): Log level (e.g., "DEBUG", "INFO", "WARN", "ERROR"). `slog` handles this.
*   `msg` (or `message`): The human-readable log message. `slog` handles this.
*   `trace_id`: The ID of the trace the log belongs to (if available).
*   `span_id`: The ID of the span the log belongs to (if available).
*   `store_id`: The specific store instance identifier (from resource attributes or context).
*   `service.name`, `service.version`, `service.instance.id`: From OpenTelemetry resource attributes.
*   **Contextual Application Fields:** `product.name`, `order.id`, `user.id` (if available), `app.error.code`, etc., specific to the log event.

### 7.5. Log Levels and Their Usage (Guidance for `product-service`)
*   **`slog.LevelDebug`:**
    *   Entry/exit of important functions with key parameters.
    *   Intermediate computations or variable states critical for debugging.
    *   Detailed information about I/O operations (e.g., "Reading file: data.json", "Attempting to parse N products").
    *   *Example (from a file in `product-service/src/services/` or `product-service/src/repositories/`):* `slog.DebugContext(ctx, "Checking stock for product", slog.String("product_name", name), slog.Int("requested_quantity", quantity))`
*   **`slog.LevelInfo`:**
    *   Service startup and shutdown events, including key configurations loaded (e.g., `STORE_ID`, `PRODUCT_DATA_FILE_PATH`).
    *   Significant successful business transactions (e.g., "Product purchased successfully", "Stock updated successfully").
    *   Major lifecycle events or state changes.
    *   *Example (from `product-service/src/services/buy_product_service.go`):* `slog.InfoContext(ctx, "Product purchase processed", slog.String("product_name", name), slog.Int("quantity", quantity), slog.Float64("revenue", revenue))`
*   **`slog.LevelWarn`:**
    *   Expected "negative paths" or recoverable issues that don't halt operations but might indicate potential problems or inefficiencies.
    *   "Product not found" during a search.
    *   "Insufficient stock for purchase attempt".
    *   Retries being attempted for an operation.
    *   Deprecated feature usage.
    *   *Example (from `product-service/src/services/buy_product_service.go`):* `slog.WarnContext(ctx, "Insufficient stock for purchase", slog.String("product_name", name), slog.Int("requested", quantity), slog.Int("available", currentStock))`
*   **`slog.LevelError`:**
    *   Unexpected errors that prevent an operation from completing successfully.
    *   I/O errors (file read/write failures).
    *   Database errors (if applicable).
    *   Failures in critical business logic.
    *   Panics recovered by a middleware.
    *   Always include the `error` object itself for stack trace and details.
    *   *Example (from `product-service/src/repositories/update_stock_repository.go`):* `slog.ErrorContext(ctx, "Failed to update product stock in data file", slog.String("product_name", name), slog.Any("error", err))`

### 7.6. What NOT to Log (Security and Privacy Focus)
*   **Personally Identifiable Information (PII):** Avoid logging raw customer names, addresses, email addresses, phone numbers unless absolutely necessary, pseudonymized, and compliant with privacy regulations (e.g., GDPR, CCPA).
*   **Sensitive Credentials:** Never log passwords, API keys, security tokens, or raw credit card information.
*   **Verbose Stack Traces in Production INFO/DEBUG:** Full stack traces can be noisy. Log them at ERROR level or provide a summary/link.
*   **Large Payloads:** Avoid logging entire large request/response bodies unless specifically for debugging an issue, and consider truncating them.

### 7.7. Log Shipping and Centralization
*   **Mechanism:** The OpenTelemetry Collector will be configured to receive logs from container stdout/stderr (or directly if using an OTEL logging library).
*   **Processing:** The Collector can parse, filter, and enrich logs before exporting them to the chosen backend (e.g., Signoz).

## 8. Docker Compose Configuration for Multi-Store Deployment

This section details the `docker-compose.yml` setup for running multiple `product-service` instances, each representing a distinct store.

### 8.1. Service Definition per Store
Each store requires its own service definition to manage unique configurations, data, and port mappings.

```yaml
networks:
  otel_internal-network:
    driver: bridge

services:
  product-service-store-alpha: # Unique service name for this store instance
    build:
      context: .
      dockerfile: ./product-service/Dockerfile # Assuming a common Dockerfile
    ports:
      - "8091:8082" # Unique host port for store_alpha, mapping to container's 8082
    volumes:
      # Store-specific data volume
      - ./product-service/data-store-alpha.json:/product-service/data.json
    env_file:
      - .env.store-alpha # Optional: For store-specific secrets or complex configs
    environment:
      - OTEL_SERVICE_NAME=product-service # Common service name for grouping in backend
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317 # OTLP gRPC endpoint
      - OTEL_EXPORTER_OTLP_INSECURE=true # For local dev; use TLS in prod
      - PRODUCT_DATA_FILE_PATH=/product-service/data.json
      - STORE_ID=alpha_store # CRITICAL: Unique identifier for this store
      # CRITICAL: Sets resource attributes for all telemetry from this instance
      - OTEL_RESOURCE_ATTRIBUTES=service.name=product-service,store.id=alpha_store,service.instance.id=product-service-alpha-$(hostname),service.version=1.0.0
    networks:
      - otel_internal-network
    depends_on:
      - otel-collector
    deploy: # Optional: for swarm mode or resource limits
      replicas: 1
      resources:
        limits:
          cpus: '0.5'
          memory: 128M

  product-service-store-beta:
    build:
      context: .
      dockerfile: ./product-service/Dockerfile
    ports:
      - "8092:8082" # Unique host port for store_beta
    volumes:
      - ./product-service/data-store-beta.json:/product-service/data.json
    env_file:
      - .env.store-beta
    environment:
      - OTEL_SERVICE_NAME=product-service
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
      - OTEL_EXPORTER_OTLP_INSECURE=true
      - PRODUCT_DATA_FILE_PATH=/product-service/data.json
      - STORE_ID=beta_store
      - OTEL_RESOURCE_ATTRIBUTES=service.name=product-service,store.id=beta_store,service.instance.id=product-service-beta-$(hostname),service.version=1.0.0
    networks:
      - otel_internal-network
    depends_on:
      - otel-collector
    # ... potentially more store instances up to 10 ...

  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.99.0 # Or latest stable
    container_name: otel-collector
    command: ["--config=/etc/otelcol-contrib/config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otelcol-contrib/config.yaml:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro # For Docker stats receiver
    ports:
      - "4317:4317"  # OTLP gRPC
      - "4318:4318"  # OTLP HTTP
      - "13133:13133" # Health check
      - "55679:55679" # zPages (optional)
    networks:
      - otel_internal-network
    # Environment variables for the collector itself, if needed for its exporters (e.g., Signoz endpoint/key)
    # environment:
    #   - SIGNOZ_ENDPOINT=...
    #   - SIGNOZ_INGESTION_KEY=...

  # ... other services like product-simulator ...
```

### 8.2. `otel-collector-config.yaml` Considerations
The collector configuration will define:
*   **Receivers:** OTLP (for application telemetry), Docker Stats Receiver (for container metrics).
*   **Processors:** Batch processor (recommended), attribute manipulation (e.g., ensuring `store.id` consistency if needed, though resource attributes from SDK are preferred).
*   **Exporters:** OTLP (to Signoz or other backend), logging exporter (for debugging collector).
*   **Service Pipelines:** Defining how data flows from receivers through processors to exporters.

## 9. Dashboarding and Visualization Strategy

Dashboards translate raw telemetry into actionable insights.

### 9.1. Principles of Effective Dashboards
*   **Audience-Specific:** Tailor dashboards to their users (e.g., SRE/Ops, Developers, Business Analysts, Store Managers).
*   **Action-Oriented:** Focus on information that leads to decisions or actions, not just data dumps.
*   **Clear Visualizations:** Use appropriate chart types for the data being presented.
*   **Performance:** Dashboards should load quickly. Optimize queries.
*   **Interactivity:** Leverage variables (especially for `$store_id`) to allow users to filter and drill down.
*   **Consistency:** Use consistent naming and layouts across related dashboards.
*   **Leverage Cross-Signal Correlation:** Design dashboards to facilitate navigation between metrics, traces, and logs for a holistic view.
*   **Primary Filters on Resource Attributes:** Use `store_id` and `service.instance.id` as key dashboard variables for targeted views.

### 9.2. Key Dashboard Categories and Panels (Detailed Examples)

#### 9.2.1. Global Multi-Store Overview Dashboard (Audience: Platform Ops, Management)
*   **Variables:** Time range.
*   **Panels:**
    *   **Platform KPIs:**
        *   Total Platform Revenue (Sum of `app.revenue.total` across all stores).
        *   Total Platform Items Sold (Sum of `app.items.sold.count` across all stores).
        *   Platform-Wide Average Request Latency (Avg of `http.server.request.duration`).
        *   Platform-Wide Error Rate (Percentage of error status codes from `http.server.request.count`).
    *   **Per-Store Comparison (Top 5-10 by a metric, or all if manageable):**
        *   Revenue by Store (Bar chart: `sum by (store.id) (app.revenue.total)`).
        *   Items Sold by Store (Bar chart: `sum by (store.id) (app.items.sold.count)`).
        *   Error Rate by Store (Table or Bar chart).
        *   Average Latency by Store (Table or Bar chart).
    *   **Store Instance Health Map:**
        *   Status indicators (Green/Yellow/Red) for each `service.instance.id` based on CPU/Memory/Error Rate thresholds.
    *   **Key Alert Summary:** Display active critical alerts.
    *   **(New) Platform-Wide Error Correlation:**
        *   Top Erroring Stores (Metric: `sum by (store.id) (rate(app_errors_total{app_error_code!="", store_id!=""}[5m]))`).
        *   Panel to show if similar error *types* (based on `app.error.code`) spiked in multiple stores simultaneously, or if common infrastructure metrics (like `otel-collector` health) showed issues concurrently. Helps distinguish store-specific vs. platform-wide problems.

#### 9.2.2. Individual Store Health & Deep-Dive Dashboard (Enhanced) (Audience: Store Managers, Ops for specific store)
*   **Variables:** Time range, `$store_id` (mandatory single select).
*   **Panels (all filtered by the selected `$store_id`):
    *   **Store KPIs (Combined Business & Operational):**
        *   Total Revenue for Store (`app.revenue.total`).
        *   Total Items Sold for Store (`app.items.sold.count`).
        *   P95 Latency for `/products/buy` (Trace-derived: `http.server.request.duration`).
        *   Error Rate for `/products/buy` (Trace-derived: based on `http.server.request.count` with error status codes).
        *   Trend of Revenue/Items Sold (Hourly, Daily).
        *   Top Selling Products (Revenue & Units) for this store.
        *   Sales by Category for this store.
    *   **Inventory Status:**
        *   Current Stock Levels for Top N Products (Table/Gauges - `app.product.stock.current`).
        *   Low Stock Products List.
        *   "Insufficient Stock" Event Count & Trend (`app.stock.insufficient.events.count`).
    *   **Operational Health (for this store's instances):**
        *   API Request Rate, Error Rate, Latency (p90, p95, p99) for other key endpoints.
        *   Container CPU & Memory Utilization.
        *   Container Restart Count.
    *   **Recent Problematic Activity:**
        *   Table of Recent Slow Traces & Traces with Errors for this store (listing Trace ID, Root Span, Duration, Error, Timestamp - linkable).
        *   Feed of Recent Critical/Error Logs for this store (including `trace_id` - linkable).

#### 9.2.3. Application Performance Monitoring (APM) Dashboard (Audience: Developers, SREs)
*   **Variables:** Time range, `$store_id` (optional, can be "All" or specific), `$service_instance_id` (optional), `$http_route`.
*   **Panels:**
    *   **Service Overview (RED Metrics):**
        *   Request Rate (per route, filterable by store).
        *   Error Rate (per route, filterable by store).
        *   Latency Distribution (Histogram/Heatmap - p50, p90, p95, p99 per route, filterable by store).
    *   **Trace Analysis:**
        *   Table of Slowest Traces (with key attributes like `http.route`, `store.id`).
        *   Table of Traces with Errors.
        *   Service Map (visualizing dependencies if other services are called).
    *   **Resource Utilization:**
        *   JVM/Go Runtime Metrics (Heap usage, GC pauses, Goroutine count) - filterable by instance.
    *   **Dependency Performance:**
        *   FileDB Read/Write Latency & Error Rates (if instrumented).
    *   **Custom Metrics Deep Dive:**
        *   `app.product.notfound.count` trend.
        *   `app.stock.insufficient.events.count` trend.

#### 9.2.4. Business Insights Dashboard (Audience: Product Managers, Business Analysts)
*   **Variables:** Time range, `$store_id` (can be "All" or specific), `$product_category`, `$product_name`.
*   **Panels:**
    *   **Revenue & Sales Trends:**
        *   Detailed daily/weekly/monthly revenue and items sold, with comparisons to previous periods.
        *   Growth rates.
    *   **Product Performance:**
        *   Top/Bottom performing products by revenue, units sold, profit margin (if cost data available).
        *   Product sales trends over time.
    *   **Category Analysis:**
        *   Sales performance by product category.
        *   Category contribution to total revenue.
    *   **Customer Behavior (derived from metrics):**
        *   "Product Not Found" trends (top searches, frequency per store).
        *   Impact of "Insufficient Stock" events on potential revenue (estimated).
    *   **Inventory Optimization:**
        *   Products frequently out of stock.
        *   Slow-moving inventory.

#### 9.2.5. Troubleshooting Workbench Dashboard (New) (Audience: Developers, SREs)
*   **Variables:** `$trace_id` (user input, optional), `$store_id` (user input, optional), time range.
*   **Panels (dynamically update based on variable input):
    *   **Trace Waterfall:**
        *   Displays the trace for the input `$trace_id`.
    *   **Logs for Trace/Store:**
        *   If `$trace_id` is provided: Shows all logs where `log.trace_id == $trace_id`.
        *   If only `$store_id` and time range: Shows logs for that store in the time window, filterable by log level or keywords.
    *   **Metrics around Trace/Event Time (for the `$store_id` of the trace/event):**
        *   Shows key service metrics (request rate, error rate, resource utilization for the relevant `service.instance.id`) in a small time window around the trace's execution time or selected event time.
        *   *Purpose:* Provides immediate context of service health during a specific problematic request or event.

### 9.3. Alerting Strategy
While dashboards provide visibility, alerting provides proactive notification of issues.
*   **Alert on SLI/SLO Violations:** High error rates, unacceptable latency spikes.
*   **Resource Saturation:** High CPU/Memory utilization on containers.
*   **Critical Business Metrics:** Drastic drop in revenue or sales rate (anomaly detection).
*   **Inventory Alerts:** Critical low stock levels for key products.
*   **System Errors:** Spike in `ERROR` level logs, specific error codes.
*   **Collector Health:** Issues with the `otel-collector` itself (e.g., high queue, data export failures).
Alerts should be actionable and routed to the appropriate teams, including `store_id` context where relevant.

## 10. Correlating Telemetry Signals for Unified Insights (New Section)

The true power of observability emerges when metrics, traces, and logs are not viewed in isolation but are correlated to provide a unified understanding of system behavior. Consistent resource attributes (especially `store.id`) and direct signal linking are foundational to this.

### 10.1. Trace-Log Correlation

*   **Mechanism:** As detailed in Section 7.3, the injection of `trace_id` and `span_id` from the active OpenTelemetry context into structured logs is paramount.
*   **Benefit:** This enables seamless navigation within observability platforms:
    *   From a specific span in a trace waterfall, directly jump to all logs emitted during that span's execution.
    *   From a log entry (e.g., an error log), link back to the full distributed trace that provided the context for that log, and to the specific span active when the log was written.
    *   This dramatically reduces the Mean Time to Investigation (MTTI) and Mean Time to Resolution (MTTR) by providing immediate, relevant detail.

### 10.2. Resource-Based Correlation

*   **Mechanism:** Since all telemetry signals (metrics, traces, logs) from a given service instance will share the same OpenTelemetry Resource attributes (e.g., `store.id=alpha_store`, `service.name=product-service`, `service.instance.id=...`), these attributes act as common pivot points.
*   **Benefit:** This allows for broader contextual analysis even for signals not part of the same direct execution path:
    *   **Example Scenario 1 (Metric to Traces/Logs):** An alert fires for a spike in the `app.revenue.total` metric for `store_id=alpha_store`. Analysts can then filter traces and logs for `store_id=alpha_store` occurring around the timestamp of the metric anomaly to investigate contributing factors (e.g., a surge in specific product purchases, a new promotion, or even erroneous transactions).
    *   **Example Scenario 2 (Infrastructure to Application):** A high CPU utilization alert fires for a container linked to `service.instance.id=product-service-alpha-XYZ` (which corresponds to `store_id=alpha_store`). Operators can then inspect application-level metrics (like request rate, active requests from `http.server.*` metrics) and examine traces for that specific instance to understand the load characteristics and identify potential performance bottlenecks within the application code running on that instance.
    *   **Example Scenario 3 (Log Anomaly to Metrics/Traces):** An unusual pattern of error logs is detected for `store.id=beta_store`. This can be correlated with performance metrics (latency, error rate metrics) and trace data for `beta_store` during the same period to see if the log anomalies manifested as user-facing issues or performance degradation.

By designing dashboards and analytical workflows that leverage both direct trace-log linking and broader resource-based correlation, teams can gain a much deeper and faster understanding of system state and behavior across the multi-store environment.

## 11. Implementation Roadmap (High-Level Phases)

A phased approach allows for iterative development and value delivery.

*   **Phase 1: Foundational Telemetry & Core Business Metrics**
    *   Implement `STORE_ID` propagation via `OTEL_RESOURCE_ATTRIBUTES`.
    *   Instrument core business metrics: `app.revenue.total`, `app.items.sold.count`.
    *   Basic `http.server.*` metrics via auto-instrumentation.
    *   Implement `app.product.stock.current` (via `UpdateProductStockLevels`).
    *   Basic trace-log correlation (inject `trace_id`, `span_id` into logs).
    *   Setup `otel-collector` for metrics, traces, logs export to Signoz.
    *   Develop initial Global Overview and Single Store Deep-Dive dashboards.
*   **Phase 2: Enhanced Operational Metrics & Tracing**
    *   Instrument detailed operational metrics: `app.stock.insufficient.events.count`, `app.product.notfound.count`, FileDB metrics.
    *   Enrich traces with detailed custom attributes and granular spans/events for key flows (`BuyProduct`).
    *   Refine `slog` usage for consistent structured logging with all essential fields.
    *   Develop APM dashboard.
    *   Implement basic alerting on critical SLIs.
*   **Phase 3: Advanced Insights, Logging, and Optimization**
    *   Implement aspirational business metrics (e.g., conversion rates if feasible).
    *   Further enrich logs with contextual information.
    *   Develop Business Insights dashboard.
    *   Refine alerting rules and notification channels.
    *   Review telemetry data volume and optimize sampling/collection strategies if needed.
    *   Conduct performance testing and use telemetry to identify optimization areas.

## 12. Conclusion

This comprehensive observability strategy, centered around consistent store identification and rich telemetry across metrics, traces, and logs, will provide invaluable insights into the multi-store e-commerce platform. By instrumenting the `product-service` and leveraging the OpenTelemetry ecosystem, we can achieve operational excellence, gain deep business understanding, and build a resilient, scalable system. The phased implementation will ensure continuous value delivery and adaptation to evolving needs. 