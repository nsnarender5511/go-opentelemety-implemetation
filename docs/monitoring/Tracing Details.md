# Tracing Details

**Purpose:** This page explains how distributed tracing is implemented using OpenTelemetry, including SDK setup, span creation, and context propagation.
**Audience:** Developers, DevOps, Students
**Prerequisites:** Understanding of OpenTelemetry Tracing concepts (Span, Trace Context, Propagation). See [Glossary](../Glossary.md).
**Related Pages:** [Telemetry Setup](./Telemetry%20Setup.md), [Monitoring Overview](./README.md), [`common/telemetry/trace/trace.go`](../../common/telemetry/trace/trace.go), [`common/telemetry/trace/exporter.go`](../../common/telemetry/trace/exporter.go), [`common/telemetry/trace/trace_utils.go`](../../common/telemetry/trace/trace_utils.go)

---

## 1. Overview & Key Concepts

Distributed tracing allows tracking requests as they propagate across different services or components within a service.

*   **Key Concept: Span:** Represents a single unit of work within a trace (e.g., an HTTP request, a database call, a function execution).
*   **Key Concept: Trace Context:** Information (Trace ID, Span ID) propagated between services to link spans into a single trace.
*   **Core Responsibility:** Provide end-to-end visibility for requests, making it easier to pinpoint bottlenecks and errors in distributed systems.
*   **Why it Matters:** Essential for debugging complex interactions, understanding latency, and visualizing service dependencies.

---

## 2. Configuration & Setup

### 2.1 Trace Exporter & Provider Setup (`common/telemetry/trace/exporter.go`)

The OpenTelemetry TracerProvider and OTLP exporter are configured as part of the [Telemetry Setup](./Telemetry%20Setup.md) process, specifically within the `trace.SetupOtlpTraceExporter` function when `ENVIRONMENT=production`.

*   **Exporter:** An `otlptracegrpc` exporter sends trace data to the configured `OTEL_ENDPOINT`.
*   **Processor:** A `BatchSpanProcessor` is used to export spans efficiently in batches.
*   **Provider:** A `TracerProvider` is created, linking the OTel Resource and the span processor.
*   **Global Registration:** The `TracerProvider` is registered globally (`otel.SetTracerProvider`).
*   **Propagation:** Standard W3C `TraceContext` and `Baggage` propagation formats are enabled globally (`otel.SetTextMapPropagator`).

### 2.2 Tracer Instantiation

Code obtains a `Tracer` instance by calling `otel.Tracer(tracerName)`.
*   The `trace.StartSpan` wrapper currently uses a **hardcoded tracer name:** `"static-tracer-for-now"`. **// Fix Applied**
*   **Note:** Best practice typically involves using a meaningful instrumentation scope name (e.g., package path, library name) instead of a static string. This documentation accurately reflects the current code, but refining the tracer name is a potential future improvement.

---

## 3. Implementation Details & Usage

Helper functions in `common/telemetry/trace/trace.go` simplify span creation and management.

**`StartSpan(ctx context.Context, initialAttrs ...attribute.KeyValue)`:**
*   **Purpose:** Starts a new OTel span.
*   **Span Name:** Automatically derived using the caller's function name (`utils.GetCallerFunctionName(3)`).
*   **Attributes Added:**
    *   `code.function`: Caller function name.
    *   `code.namespace`: Hardcoded as `"static-tracer-for-now"`.
    *   Any `initialAttrs` passed to the function.
*   **Span Kind:** Defaults to `Internal`.
*   **Returns:** A new context containing the active span, and the `trace.Span` itself.
*   **Note:** Includes debug `fmt.Printf` calls.

**`EndSpan(span trace.Span, errPtr *error, statusMapper StatusMapperFunc, options ...trace.SpanEndOption)`:**
*   **Purpose:** Ends the provided span, handling errors and status.
*   **Error Handling:** If `errPtr` points to a non-nil error, it records the error on the span (`span.RecordError`) with a stack trace and sets the span status to `Error` (using the error message as the status description).
*   **Status Mapping:** If no error, sets status to `Ok`. Allows custom mapping via `StatusMapperFunc` (defaults to simple Ok/Error mapping).
*   **Usage:** Typically called via `defer` after `StartSpan`.

**Common Usage Pattern:**
```go
ctx, span := trace.StartSpan(ctx, attribute.String("my.attr", "value"))
var err error
defer trace.EndSpan(span, &err, nil) // Use default error mapping

// ... function logic ...
err = someOperationThatMightFail()
if err != nil {
    // Optionally add attributes before returning
    span.SetAttributes(attribute.String("failure.reason", "specific reason"))
    return // Let defer handle span ending and status
}
// ... more logic ...
```

**Utility Functions (`common/telemetry/trace/trace_utils.go`):**
*   **`RecordSpanError(span oteltrace.Span, err error, attrs ...attribute.KeyValue)`:**
    *   **Purpose:** Standardized way to record an error on an active span.
    *   **Functionality:** Checks if the provided span is valid and the error is non-nil. Records the error on the span using `span.RecordError(err)`. Adds standard `exception.message` (from `err.Error()`) and `exception.type` (from `common/telemetry/attributes`) attributes, plus any additional `attrs` passed in. Sets the span status to `codes.Error` with the error message as the description.
    *   **Note:** Provides similar functionality to the error handling within `trace.EndSpan` but allows recording errors mid-span if needed, and explicitly uses the predefined `exception.*` attribute keys.

---

## 4. Monitoring & Observability Integration

*   Spans created using these wrappers are exported via the OTLP pipeline to the Collector and then to SigNoz.
*   Trace context propagation ensures spans from different services (if any) or within the service (e.g., HTTP handler -> DB call) are linked correctly.
*   Attributes added to spans provide valuable context for filtering and analysis in SigNoz.
*   Errors recorded on spans are clearly visible in trace views.

---

## 5. Visuals & Diagrams

<!-- 
[USER ACTION REQUIRED]
Insert actual screenshot from SigNoz showing a trace waterfall for a product-service request.
Example: Should show nested spans like otelfiber -> handler -> service -> repository -> db spans.

Example Markdown:
![Example Trace Waterfall](../assets/images/trace_waterfall_example.png)
*Fig 1: Example Trace Waterfall Diagram from SigNoz.*
-->

*Placeholder for Example Trace Waterfall Diagram.*

---

## 6. Teaching Points & Demo Walkthrough

*   **Key Takeaway:** Tracing provides a request-centric view of execution flow. Helper functions can standardize span creation and error handling. Context propagation is key for linking distributed work.
*   **Demo Steps:**
    1.  Show `common/telemetry/trace/trace.go`, explaining `StartSpan` and `EndSpan`.
    2.  Highlight the automatic span naming and attribute addition in `StartSpan`. Note the hardcoded tracer name.
    3.  Show the error recording and status setting logic in `EndSpan`.
    4.  Show example usage of the `StartSpan`/`EndSpan` pattern in `product-service` code (e.g., around a database call or HTTP handler).
    5.  Run the application and make a request.
    6.  Find the corresponding trace in SigNoz. Show the waterfall diagram, span details, attributes (including `code.function`), and any recorded errors.
*   **Common Pitfalls / Questions:**
    *   Why is my span name `"<unknown>"`? (The `GetCallerFunctionName` utility failed, potentially due to stack depth or function type).
    *   Why aren't my services linked in traces? (Ensure context propagation is working: propagator is set globally, context is passed correctly between functions/services, HTTP clients/servers use OTel instrumentation).
    *   Should I create spans for every function? (No, typically around meaningful units of work like incoming requests, outgoing calls, significant computations, or database interactions).
*   **Simplification Analogy:** Imagine a package delivery. `StartSpan` is like putting a unique barcode (Trace ID + new Span ID) on the package when a step begins (e.g., leaving the warehouse). Attributes are notes written on the package label (function name, parameters). `EndSpan` is scanning the barcode when the step finishes, noting if there were any problems (errors) on the delivery slip.

---

**Last Updated:** 2024-07-30


**Handler Spans:**
*   Note that the handler methods in `product-service/src/handler.go` also explicitly create spans using `commontrace.StartSpan`. This means traces will likely show nested spans: one outer span created by the `otelfiber` middleware covering the whole HTTP request, and inner spans created by the handler for its specific operation, which in turn contain spans from the service and repository layers.
