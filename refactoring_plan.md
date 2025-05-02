# Refactoring Plan: Centralize Telemetry and Reduce Boilerplate

**Goal:** Refactor the `common` and `product-service` modules to replace repetitive telemetry (logging, tracing, metrics) boilerplate with centralized helper functions, improving maintainability and consistency.

**Core Strategy:**

1.  Enhance the `common/telemetry` module with wrappers for common tracing and metrics patterns.
2.  Remove layer-specific telemetry initialization and recording helpers from `product-service`.
3.  Refactor method bodies in `product-service` (handler, service, repository) to use the new common wrappers.
4.  Ensure metrics are initialized centrally.

---

## Phase 1: Enhance `common/telemetry` Module

1.  **Tracing Wrappers (`common/telemetry/trace/wrappers.go`):**
    *   Create `StartSpan(ctx context.Context, scopeName, spanName string, initialAttrs ...attribute.KeyValue) (context.Context, trace.Span)`:
        *   Gets tracer using `manager.GetTracer(scopeName)`.
        *   Starts span: `tracer.Start(ctx, spanName, trace.WithAttributes(initialAttrs...))`.
        *   Returns new context and span.
    *   *(Decision: Keep `defer span.End()` separate in application code instead of an `EndSpan` helper for simplicity, but ensure `RecordSpanError` is called explicitly on error paths)*.
    *   Ensure `RecordSpanError` exists and handles nil errors gracefully.

2.  **Metrics Wrappers (`common/telemetry/metric/wrappers.go`):**
    *   **Standardize Metric Names:** Define exported constants:
        ```go
        const (
            OpsCountMetricName   = "service.operations.count"
            DurationMetricName = "service.duration.seconds"
            ErrorsCountMetricName  = "service.errors.count"
            // Add others like FileIODurationMetricName if needed
        )
        ```
    *   **Central Error Counter:**
        *   Declare a single, package-level (or globally accessible via manager) `errorCounter metric.Int64Counter` variable within `common/telemetry/metric` or `common/telemetry/manager`.
        *   Create `InitializeCommonMetrics(meter metric.Meter)` function (or similar) to initialize `errorCounter`, `opsCounter`, `durationHist` using the standardized names. Handle initialization errors.
        *   This `InitializeCommonMetrics` should be called once during startup.
    *   **Create `RecordOperationMetrics` Function:**
        ```go
        package metric // or manager?

        import (
            "context"
            "errors" // Needed for error checking
            "time"

            "github.com/narender/common/errors" // Assuming this exists for ErrNotFound etc.
            "go.opentelemetry.io/otel/attribute"
            "go.opentelemetry.io/otel/metric"
            // Potentially import manager if accessing global counters/histograms
        )

        // Assume these are initialized globally/package-level by InitializeCommonMetrics
        var opsCounter metric.Int64Counter
        var durationHist metric.Float64Histogram
        var errorCounter metric.Int64Counter

        // InitializeCommonMetrics initializes the standard metric instruments.
        // Call this once during startup.
        func InitializeCommonMetrics(meter metric.Meter) error {
            var err, multiErr error
            opsCounter, err = meter.Int64Counter(
                OpsCountMetricName, 
                metric.WithDescription("Counts service operations."),
                metric.WithUnit("{operation}"),
            )
            multiErr = errors.Join(multiErr, err)
            
            durationHist, err = meter.Float64Histogram(
                DurationMetricName, 
                metric.WithDescription("Measures the duration of service operations."),
                metric.WithUnit("s"),
            )
            multiErr = errors.Join(multiErr, err)
            
            errorCounter, err = meter.Int64Counter(
                ErrorsCountMetricName,
                metric.WithDescription("Counts errors encountered by layer and type."),
                metric.WithUnit("{error}"),
            )
             multiErr = errors.Join(multiErr, err)
             
            // Add other common metrics like file I/O duration if needed
            
            return multiErr
        }

        // RecordOperationMetrics records standard duration, success count, and error count metrics.
        func RecordOperationMetrics(
            ctx context.Context,
            layer string,     // e.g., "repository", "service", "handler"
            operation string, // e.g., "GetAll", "GetByID"
            startTime time.Time,
            opErr error,      // The error returned by the operation
            attrs ...attribute.KeyValue, // Additional operation-specific attributes
        ) {
            // Ensure instruments were initialized
            if durationHist == nil && opsCounter == nil && errorCounter == nil {
                // Maybe log a warning using base logger?
                return
            }

            // Common attributes for all metrics related to this op
            commonAttrs := []attribute.KeyValue{
                attribute.String("layer", layer),
                attribute.String("operation", operation),
            }
            mergedAttrs := append(commonAttrs, attrs...)

            // 1. Record Duration (always, add error tag)
            if durationHist != nil {
                duration := time.Since(startTime).Seconds()
                durationAttrs := append(mergedAttrs, attribute.Bool("error", opErr != nil))
                durationHist.Record(ctx, duration, metric.WithAttributes(durationAttrs...))
            }

            // 2. Record Operation Count (Only Success)
            if opErr == nil && opsCounter != nil {
                opsCounter.Add(ctx, 1, metric.WithAttributes(mergedAttrs...))
            }

            // 3. Record Error Count
            if opErr != nil && errorCounter != nil {
                // Determine error type (refine this mapping)
                errorType := "internal"
                if errors.Is(opErr, commonerrors.ErrNotFound) { // Use specific error type if available
                    errorType = "not_found"
                } else if errors.Is(opErr, os.ErrNotExist) { // Example for file errors
                    errorType = "file_not_found"
                } // Add more mappings based on common/errors definitions

                errorAttrs := append(mergedAttrs, attribute.String("error_type", errorType))
                errorCounter.Add(ctx, 1, metric.WithAttributes(errorAttrs...))
            }
        }
        ```

## Phase 2: Refactor `product-service` Layers

1.  **Remove Layer-Specific Helpers:**
    *   Delete `initMetrics` from `repository.go`.
    *   Delete `recordRepositoryError` from `repository.go`.
    *   Delete `initServiceMetrics` from `service.go`.
    *   Delete `recordServiceError` from `service.go`.
    *   Ensure `productErrorsCounter` variable declaration is removed from all layers (handler, service, repository) as it's now handled centrally.

2.  **Refactor Method Bodies (Handler, Service, Repository - Apply pattern to all relevant methods like `GetAll`, `GetByID`, `UpdateStock`, `loadData`, `saveData`):**
    *   **Ensure Context:** Verify `ctx context.Context` is the first argument.
    *   **Named Error Return:** Use named return values for errors (e.g., `func (...) (resultType, opErr error)`) to easily capture the error for the deferred metric call.
    *   **Start Time:** Add `startTime := time.Now()` at the beginning.
    *   **Get Logger:** Add `logger := logging.LoggerFromContext(ctx)`.
    *   **Start Span:** Replace `tracer.Start` with `ctx, span := trace.StartSpan(ctx, scopeName, "ComponentName.MethodName", attributes...)`. Use appropriate `scopeName` (e.g., `repositoryScopeName`, `serviceScopeName`).
    *   **Defer Span End:** Add `defer span.End()` immediately after `StartSpan`.
    *   **Defer Metrics:** Add `defer metric.RecordOperationMetrics(ctx, layerName, operationName, startTime, opErr, attributes...)` *after* the span defer. Pass relevant `layerName` (e.g., `repoLayerName`), `operationName`, and any specific attributes.
    *   **Logging:** Replace Logrus/old Zap calls with `logger.Info/Warn/Debug/Error` using structured `zap.String`, `zap.Error`, etc.
    *   **Error Handling:**
        *   Assign errors to the named return variable `opErr` before returning.
        *   Keep explicit `span.RecordError(opErr, ...)` calls to add detailed error info to traces.
        *   Keep explicit `span.SetStatus(codes.Error, "message")`.
        *   Remove calls to the old `record...Error` functions.
    *   **Success Handling:** Ensure `span.SetStatus(codes.Ok, "")` is called on successful paths.

## Phase 3: Central Metrics Initialization

1.  **`product-service/main.go`:**
    *   After `telemetry.InitTelemetry(...)` succeeds:
    *   Get the meter: `meter := manager.GetMeter(ServiceName)` (or a dedicated init scope name).
    *   Call the central initialization: `err := metric.InitializeCommonMetrics(meter)`.
    *   Log/handle the initialization error using the `baseLogger`.

## Phase 4: Cleanup

1.  Remove unused imports across all modified files.
2.  Remove any remaining declarations of layer-specific metric variables (like `productOpsCounter`, `productRepoDurationHist`, etc.) if they are now fully handled by the common wrappers accessing the centrally initialized instruments.
3.  Run `go mod tidy` in both `common` and `product-service`.
4.  Run `go fmt ./...` or `goimports`.

---

This refactoring aims to significantly reduce code duplication related to observability setup and teardown within the application logic, making the code cleaner and easier to maintain. 