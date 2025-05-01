# Refactoring Plan based on Roaster Analysis

This document outlines the necessary refactoring tasks identified during the code roast to improve the OpenTelemetry implementation, code consistency, and overall robustness of the `product-service`.

## Critical Failures

### 1. Inconsistent `WithSpan` Usage

*   **Issue:** A mix of the custom `telemetry.WithSpan` helper and direct `tracer.WithSpan` calls are used across the codebase (e.g., in `handler.go`), violating the principle of consistency. The custom helper itself was not part of the original plan.
*   **File(s) Affected:** `product-service/src/handler.go`, `product-service/src/service.go`, `product-service/src/repository.go`, `common/telemetry/trace.go` (if the custom helper exists there).
*   **Fix:** Choose **one** consistent approach:
    *   **Option A (Recommended):** Refactor all instances to use the standard `tracer.WithSpan(ctx, "SpanName", func(ctx context.Context) error { ... })` pattern provided by the `go.opentelemetry.io/otel/trace` package. Remove the custom `telemetry.WithSpan` helper if it exists.
*   **Goal:** Standardize span creation for improved readability and maintainability.

### 2. Module Path Fragility

*   **Issue:** The `product-service/go.mod` file does not explicitly `require` the `common` modules. It relies on the Go tool finding them based on the local filesystem layout (`replace` directive might have been removed, but the direct requirement is still missing).
*   **File(s) Affected:** `product-service/go.mod`
*   **Fix:** Implement a standard Go module dependency mechanism:
    *   **Option A (Sub-modules):** Treat `common` as a proper sub-module. Add `require product-service/common v0.0.0` (or similar) to `product-service/go.mod` and potentially add a `replace product-service/common => ../common` directive *if* necessary during local development (though workspaces are generally preferred now).
*   **Goal:** Create a robust and standard Go module structure that isn't dependent on implicit local paths.

## Major Problems

### 1. Redundant Tracer/Meter Initialization in `main.go`

*   **Issue:** `main.go` uses `telemetry.GetTracer(...)` and `telemetry.GetMeter(...)` after `telemetry.InitTelemetry()` has already set up the global providers. This is unnecessary.
*   **File(s) Affected:** `product-service/src/main.go`
*   **Fix:** Remove the calls to `telemetry.GetTracer` and `telemetry.GetMeter` in `main.go`. Fetch the initial instances for dependency injection using the standard OTel API: `tracer := otel.Tracer("product-service/main")` and `meter := otel.Meter("product-service/main")`.
*   **Goal:** Simplify initialization and adhere to standard OTel practices.

### 2. Metric Recording Location

*   **Issue:** Custom metric counters (`productLookupsCounter`, `productStockCheckCounter`) are incremented *inside* the `WithSpan` lambda functions in `handler.go`.
*   **File(s) Affected:** `product-service/src/handler.go`
*   **Fix:** Move the `*.Add(...)` calls to *after* the `WithSpan` block completes. Use the `err` variable returned by `WithSpan` to determine the success/failure status and set the appropriate attributes (`app.lookup.success`, `app.stock_check.success`) for the metric.
*   **Goal:** Decouple metric recording logic from span execution logic for better clarity and separation of concerns.



## Annoying Issues

### 1. Inconsistent Attribute Key Usage (Magic Strings)

*   **Issue:** A mix of constants defined in `common/telemetry/attributes.go` (e.g., `telemetry.AppProductIDKey`) and magic strings (e.g., `"product_id"`, `"productId"`) are used for attributes, log fields, and map keys.
*   **File(s) Affected:** `product-service/src/handler.go`, `product-service/src/repository.go`, `product-service/src/service.go`
*   **Fix:** Consistently use the constants defined in `common/telemetry/attributes.go` for all OpenTelemetry attribute keys, structured log fields (`logrus.WithField`), and map keys where appropriate. Remove magic strings for these identifiers. Define new constants if necessary (e.g., for log field names).
*   **Goal:** Improve code consistency and reduce the risk of typos.

### 2. Missing Fiber Error Handler Implementation

*   **Issue:** The suggestion to add a Fiber `ErrorHandler` in `main.go` for better OTel integration was commented but not implemented.
*   **File(s) Affected:** `product-service/src/main.go`
*   **Fix:** Implement a Fiber `ErrorHandler` in `fiber.New(fiber.Config{...})`. This handler should:
    *   Extract the request context (`c.UserContext()`).
    *   Log the error using `logrus.WithContext(ctx).Error(...)`.
    *   Get the current span from the context (`trace.SpanFromContext(ctx)`).
    *   Record the error on the span (`span.RecordError(err)`).
    *   Set the span status to error (`span.SetStatus(codes.Error, err.Error())`).
    *   Return the appropriate HTTP response to the client.
*   **Goal:** Ensure unhandled errors and panics are properly captured and reported in logs and traces.

### 3. Granularity of Spans in `repository.readData`

*   **Issue:** The `repository.readData` function wraps the file reading (`os.ReadFile`) in a span, but the subsequent `json.Unmarshal` is handled outside this span without its own trace context.
*   **File(s) Affected:** `product-service/src/repository.go`
*   **Fix:** Wrap the `json.Unmarshal` call within its own dedicated span using `WithSpan` or manual span creation (`tracer.Start`). This isolates the unmarshaling operation for better observability in traces. Ensure the error from unmarshaling is recorded on this new span.
*   **Goal:** Improve trace granularity to distinguish between file I/O errors and data parsing errors.

