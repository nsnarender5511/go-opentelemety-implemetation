# Metrics Standardization and Product Metrics Implementation Plan (Minimalist Approach)

## 1. Goal

This document outlines the **minimalist plan** to:
1.  Implement new product-specific counters (`product.creation.total`, `product.updates.total`) within the `product-service` by modifying the existing `MetricsController` pattern in `common/telemetry/metric/metric.go`.

**Note:** This approach prioritizes simplicity over comprehensive refactoring or implementing all desired metrics immediately. Refactoring of common components and implementation of the inventory gauge are deferred.

## 2. Analysis Summary (Context)

*   **`common/telemetry/attributes/attributes.go`:** Contains unused attribute keys.
*   **`common/telemetry/metric/metric.go`:**
    *   A `MetricsController` helper (`StartMetricsTimer`/`End`) exists for common metrics.
    *   The `StartMetricsTimer` implementation does not capture `layer`/`operation`, resulting in missing attributes for common metrics.
    *   Common metrics are eagerly initialized in `init()`.

**(Decision):** For simplicity, we will *not* address the unused attributes or the `StartMetricsTimer` issues in this iteration.

## 3. Implementation Plan (Counters Only)

### 3.1. Modify `common/telemetry/metric/metric.go`

*   **Action:** Define and initialize the two new product counters using the existing common `meter` within the package `init()` block.
    ```diff
    var (
    	meter           = otel.Meter("common/telemetry/metric")
    	operationsTotal metric.Int64Counter
    	durationMillis  metric.Float64Histogram
    	errorsTotal     metric.Int64Counter
    +	productCreationCounter metric.Int64Counter // New
    +	productUpdateCounter   metric.Int64Counter // New
    	initErr         error
    )

    func init() {
    	// ... existing initializations for operationsTotal, durationMillis, errorsTotal ...

    +	// Initialize new product counters
    +	productCreationCounter, initErr = meter.Int64Counter(
    +		"product.creation.total",
    +		metric.WithDescription("Total number of products created."),
    +		metric.WithUnit("{product}"),
    +	)
    +	if initErr != nil {
    +		slog.Error("Failed to initialize productCreationCounter", slog.Any("error", initErr))
    +	}
    +
    +	productUpdateCounter, initErr = meter.Int64Counter(
    +		"product.updates.total",
    +		metric.WithDescription("Total number of products updated."),
    +		metric.WithUnit("{product}"),
    +	)
    +	if initErr != nil {
    +		slog.Error("Failed to initialize productUpdateCounter", slog.Any("error", initErr))
    +	}
    }
    ```
*   **Action:** Add new methods to the `MetricsController` interface and implement them in `metricsControllerImpl`.
    ```diff
    type MetricsController interface {
    	End(ctx context.Context, err *error, additionalAttrs ...attribute.KeyValue)
    +	IncrementProductCreated(ctx context.Context)
    +	IncrementProductUpdated(ctx context.Context)
    }

    type metricsControllerImpl struct {
    	startTime time.Time
    	// layer and operation fields are NOT added in this approach
    }

    // ... StartMetricsTimer() remains unchanged ...

    // ... End() remains unchanged ...

    +// IncrementProductCreated increments the product creation counter.
    +func (mc *metricsControllerImpl) IncrementProductCreated(ctx context.Context) {
    +	if productCreationCounter != nil { // Check if initialized
    +		productCreationCounter.Add(ctx, 1)
    +       // Optional: Add debug log if needed
    +       // slog.DebugContext(ctx, "Incremented product creation counter via controller")
    +	}
    +}
    +
    +// IncrementProductUpdated increments the product update counter.
    +func (mc *metricsControllerImpl) IncrementProductUpdated(ctx context.Context) {
    +	if productUpdateCounter != nil { // Check if initialized
    +		productUpdateCounter.Add(ctx, 1)
    +       // Optional: Add debug log if needed
    +       // slog.DebugContext(ctx, "Incremented product update counter via controller")
    +	}
    +}

    ```
*   **Rationale:** Leverages the existing controller pattern and common meter for the new counters with minimal changes to the common package structure.

### 3.2. Update `product-service/src/repository.go`

*   **Action:** Keep existing `StartMetricsTimer()` and `End()` calls as they are (without passing layer/operation).
*   **Action:** Call the new `MetricsController` methods after successful database writes in `Create` and `UpdateStock`.
    ```diff
    // In Create method, after successful repo.database.Write:
    	repo.logger.InfoContext(ctx, "Repository: Product created successfully", slog.String("productID", product.ProductID))
    +	mc.IncrementProductCreated(ctx) // Call new controller method
    	return nil // Success

    // In UpdateStock method, after successful r.database.Write:
    	r.logger.InfoContext(ctx, "Repository: Product stock updated and saved via FileDatabase", slog.String("productID", productID), slog.Int("new_stock", newStock))
    +	mc.IncrementProductUpdated(ctx) // Call new controller method
    	return nil
    ```
*   **Rationale:** Integrates the new counter increments into the existing operation flow using the extended controller.

## 4. Deferred Items

*   **Refactoring:** Cleanup of `common/telemetry/attributes/attributes.go` and fixing `app.layer`/`app.operation` attributes in `common/telemetry/metric/metric.go` are deferred.
*   **Inventory Gauge:** Implementation of the `product.inventory.current` observable gauge metric is deferred.

## 5. Summary of Benefits (Minimalist Approach)

*   **Simplicity:** Achieves the goal of adding the two counters with minimal code changes and no new files.
*   **Leverages Existing Pattern:** Uses the `MetricsController` already in place.
