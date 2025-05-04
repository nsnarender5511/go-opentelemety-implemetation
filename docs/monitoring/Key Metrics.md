# Key Metrics

**Purpose:** This page details the key application-level metrics defined and collected via OpenTelemetry.
**Audience:** Developers, DevOps, SREs, Students
**Prerequisites:** Understanding of OpenTelemetry Metrics concepts (Counter, Histogram). See [Glossary](../Glossary.md).
**Related Pages:** [Telemetry Setup](./Telemetry Setup.md), [Monitoring Overview](./README.md), [common/telemetry/metric/metric.go]()

---

## 1. Overview & Key Concepts

The application emits custom metrics using the OpenTelemetry Go SDK to provide insights into operation counts, durations, and errors.

*   **Key Concept: Metric Instruments:** Specific types of instruments (Counter, Histogram) are used to record different kinds of measurements.
*   **Key Concept: Attributes/Dimensions:** Metrics are recorded with key-value attributes (dimensions) that allow filtering and grouping during analysis (e.g., show duration by operation name).
*   **Core Responsibility:** Quantify application performance and behavior.
*   **Why it Matters:** Metrics are essential for building dashboards, setting alerts, and understanding performance trends over time.

---

## 2. Configuration & Setup

Metrics are defined in `common/telemetry/metric/metric.go` using a named `Meter` (`common/telemetry/metric`). The OTLP export pipeline is configured in [Telemetry Setup](./Telemetry Setup.md).

**Relevant Files:**
*   `common/telemetry/metric/metric.go`: Defines metric instruments (`app.operations.*`) and the `MetricsController` helper.
*   `common/telemetry/metric/exporter.go`: Configures the OTLP exporter and global `MeterProvider`.
*   `product-service/src/repository.go`: Example usage of the `MetricsController` pattern.

**Initialization:**
*   Metric instruments (`operationsTotal`, `durationMillis`, `errorsTotal`) are created in the `common/telemetry/metric/metric.go` package `init()` function.
*   The global `MeterProvider` configured in [Telemetry Setup](./Telemetry Setup.md) makes these metrics available for recording.

---

## 3. Defined Custom Metrics (`app.operations.*`)

The following custom metrics are defined in `common/telemetry/metric/metric.go` and used by the `productRepository`:

1.  **`app.operations.total`**
    *   **Type:** Counter (Int64)
    *   **Unit:** `{operation}`
    *   **Description:** Total number of operations executed (e.g., within the repository layer).
    *   **Key Attributes:** `app.layer`, `app.operation`, `app.error`, plus any additional attributes passed via `MetricsController.End`.

2.  **`app.operations.duration_milliseconds`**
    *   **Type:** Histogram (Float64)
    *   **Unit:** `ms`
    *   **Description:** Duration of operations in milliseconds.
    *   **Key Attributes:** `app.layer`, `app.operation`, `app.error`, plus any additional attributes passed via `MetricsController.End`.

3.  **`app.operations.errors.total`**
    *   **Type:** Counter (Int64)
    *   **Unit:** `{error}`
    *   **Description:** Total number of operations that resulted in an error.
    *   **Key Attributes:** `app.layer`, `app.operation`, `app.error` (always true for this metric), plus any additional attributes passed via `MetricsController.End`.

**(Note:** The file `common/telemetry/metric/wrappers.go` contains definitions for a similar but different set of metrics (`service.*`) and a different recording function. However, analysis of `repository.go` shows that the `app.operations.*` metrics and the `MetricsController` from `metric.go` are the ones currently being used.)

---

## 4. Implementation Details & Usage

A `MetricsController` helper pattern (`common/telemetry/metric/metric.go`) is used in `productRepository` to simplify recording the three core metrics together.

**`MetricsController` Pattern:**
*   Call `metric.StartMetricsTimer(layer, operation)` at the beginning of an operation, providing the layer (e.g., "repository") and operation name (e.g., "GetByID").
*   Use `defer timer.End(ctx, &err, ...)` to ensure metrics are recorded when the operation finishes.

**Example Usage (from `repository.go` - corrected to show layer/operation passing):**
```go
func (r *productRepository) GetByID(ctx context.Context, id string) (product Product, opErr error) {
    productIdAttr := attribute.String(attributes.AppProductID, id)
    // Start timer, passing layer and operation name
    mc := commonmetric.StartMetricsTimer("repository", "GetByID")
    // Ensure End is called, passing error pointer and additional attributes
    defer mc.End(ctx, &opErr, productIdAttr)

    ctx, spanner := commontrace.StartSpan(ctx, productIdAttr)
    defer commontrace.EndSpan(spanner, &opErr, nil)

    // ... repository logic that might set opErr ...

    return product, opErr
}
```
*   The `End` method automatically calculates duration and records values for the three `app.operations.*` metrics.
*   It attaches standard attributes (`app.layer`, `app.operation` - provided to `StartMetricsTimer`) and `app.error` (based on the error pointer), plus any additional attributes passed to `End`.
*   **Clarification on Attributes:** The `StartMetricsTimer` function accepts `layer` and `operation` strings. The calling code (e.g., `repository.go`) must pass these values for the `app.layer` and `app.operation` attributes to be correctly populated on the emitted metrics.

---

## 5. Monitoring & Observability Integration

*   These `app.operations.*` metrics are exported via the OTLP pipeline configured in [Telemetry Setup](./Telemetry Setup.md).
*   They will appear in SigNoz and can be used to build dashboards and alerts focusing on repository performance.
*   The common attributes allow filtering/grouping (e.g., view average duration per repository operation using `app.operation`, or filter by `app.layer`).

---

## 6. Visuals & Diagrams

<!-- 
[USER ACTION REQUIRED]
Insert actual screenshot(s) from SigNoz showing time-series charts for repository metrics.
Example: A dashboard panel showing app.operations.total, app.operations.duration_milliseconds (P95), and app.operations.errors.total for the repository layer, potentially grouped by app.operation.

Example Markdown:
![Repository Metrics Dashboard Panel](../assets/images/repo_metrics_dashboard.png)
*Fig 1: Example Repository Metrics Charts from SigNoz.*
-->

*Placeholder for Repository Metrics Charts.*

---

## 7. Teaching Points & Demo Walkthrough

*   **Key Takeaway:** Custom metrics provide application-specific insights. Using helpers like `MetricsController` can standardize the collection of related metrics (count, duration, error count) for specific layers like the repository.
*   **Demo Steps:**
    1.  Show `common/telemetry/metric/metric.go`, highlighting the three `app.operations.*` metric definitions and the `MetricsController`.
    2.  Show the usage of the `StartMetricsTimer`/`End` pattern in `product-service/src/repository.go`, ensuring `layer` and `operation` are passed.
    3.  Run the application and generate some operations that interact with the repository (e.g., `GET /products`).
    4.  In SigNoz, show dashboards/charts built using `app.operations.total`, `app.operations.duration_milliseconds`, and `app.operations.errors.total`.
    5.  Demonstrate filtering/grouping using attributes like `app.operation` and `app.layer`.
*   **Common Pitfalls / Questions:**
    *   Why might `app.layer` and `app.operation` attributes be missing? (The caller code needs to be updated to pass these strings to `StartMetricsTimer` when creating the timer).
    *   What do the units `{operation}` and `{error}` mean? (They are placeholders indicating the unit is essentially a count of operations or errors, respectively).
*   **Simplification Analogy:** The `MetricsController` used by the repository is like a dedicated timekeeper for the file storage room. When someone enters, they tell the timekeeper which room they are entering (`app.layer="repository"`) and what they are doing (`app.operation="GetByID"`). The timekeeper then counts how many times someone goes in (`app.operations.total`), times how long they stay (`app.operations.duration_milliseconds`), and counts how many times someone comes out saying they couldn't find what they needed (`app.operations.errors.total`), noting the room and task on each record.

---

**Last Updated:** 2024-07-30
