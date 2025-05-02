# OpenTelemetry Refactoring Plan

## Summary
This plan details the refactoring of the OpenTelemetry (OTel) integration within the `common/otel` module and its usage across the services (initially `product-service`). The goals are to eliminate global state via Dependency Injection (DI), ensure secure OTLP export by default, standardize telemetry attributes, decouple components from OTel internals, and remove dead/unnecessary code, thereby improving testability, maintainability, security, and adherence to Go best practices.

## Identified Code Quality Issues
1.  **Global State for OTel Providers:** Use of global variables (`globalTracerProvider`, `globalMeterProvider`) accessed via `GetTracer()`/`GetMeter()` hinders testability and clarity.
2.  **Insecure OTLP Exporter Default:** Defaults to insecure gRPC connection, ignoring the `OTEL_EXPORTER_INSECURE=false` setting and posing a security risk.
3.  **Inconsistent Attribute Keys:** Mix of constants and raw string literals for attribute keys leads to potential typos and analysis issues.
4.  **Tight Coupling in Metrics Callback:** `productRepository.ObserveStockLevels` is coupled to OTel SDK types (`Observer`, `ObservableGauge`).
5.  **Boilerplate Span Creation:** Repetitive code for starting/ending spans and recording errors clutters logic.
6.  **Potentially Redundant OTel Helpers:** Files like `trace.go`, `tracer.go`, `meter.go` might be unnecessary after DI.
7.  **Dead Code:** Commented-out code exists in `common/config/config.go`.

---

## Refactoring Phases and Steps

### Phase 1: Implement Dependency Injection & Secure Exporter

**Goal:** Eliminate global state for OTel providers and fix the insecure exporter default.

**Step 1.1: Modify Structs & Constructors for DI**
*   **Files:** `product-service/src/repository.go`, `product-service/src/service.go`, `product-service/src/handler.go` (and any others using OTel directly).
*   **Action:**
    *   Add fields for `oteltrace.TracerProvider` and `metric.MeterProvider` (or specific `oteltrace.Tracer`/`metric.Meter` instances) to the structs.
    *   Update corresponding constructor functions (e.g., `NewProductRepository`) to accept these providers/meters as arguments.
    *   Initialize the struct fields within the constructors. Example (`repository.go`):
        ```diff
        // product-service/src/repository.go
        type productRepository struct {
            products map[string]Product
            mu       sync.RWMutex
            filePath string
        +   tracer   oteltrace.Tracer
        +   meter    metric.Meter // Add Meter if repository creates metrics directly
        }

        -func NewProductRepository(dataFilePath string) (ProductRepository, error) {
        +func NewProductRepository(dataFilePath string, tracerProvider oteltrace.TracerProvider, meterProvider metric.MeterProvider) (ProductRepository, error) {
            // ... existing setup ...
            repo := &productRepository{
                products: make(map[string]Product),
                filePath: dataFilePath,
        +       tracer:   tracerProvider.Tracer("product-service/repository"), // Initialize tracer
        +       meter:    meterProvider.Meter("product-service/repository"),   // Initialize meter
            }
            // ... rest of initialization ...
            return repo, nil
        }
        ```

**Step 1.2: Update OTel Initialization and Service Wiring**
*   **File:** `product-service/src/main.go`
*   **Action:**
    *   Call `commonotel.InitTelemetry` to get the `tracerProvider` and `meterProvider`.
    *   Pass these providers to the constructors of services, repositories, and handlers during application setup.
    *   Example:
        ```diff
        // product-service/src/main.go
        func main() {
            // ... load config ...
        -   shutdown, err := commonotel.InitTelemetry(context.Background(), cfg)
        +   otelShutdown, tracerProvider, meterProvider, err := commonotel.InitTelemetry(context.Background(), cfg) // Assuming InitTelemetry returns providers now
            if err != nil {
                logrus.Fatalf("Failed to initialize OpenTelemetry: %v", err)
            }
        -   defer func() { /* handle shutdown */ }()
        +   defer func() { /* handle otelShutdown */ }() // Adjust shutdown handling

            // ... other setup ...

        -   repo, err := NewProductRepository(cfg.DataFilePath)
        +   repo, err := NewProductRepository(cfg.DataFilePath, tracerProvider, meterProvider)
            if err != nil {
                 logrus.Fatalf("Failed to create repository: %v", err)
            }

        -   svc := NewProductService(repo)
        +   svc := NewProductService(repo, tracerProvider, meterProvider) // Pass providers to service if needed

        -   h := NewHandler(svc)
        +   h := NewHandler(svc, tracerProvider, meterProvider) // Pass providers to handler if needed

            // ... setup router and server ...
        }
        ```

**Step 1.3: Update OTel Usage within Components**
*   **Files:** `product-service/src/repository.go`, `product-service/src/service.go`, `product-service/src/handler.go`.
*   **Action:** Modify methods that previously used `commonotel.GetTracer()` or `commonotel.GetMeter()` to use the injected instance fields (e.g., `r.tracer`, `s.meter`).
*   Example (`repository.go`):
    ```diff
    // product-service/src/repository.go - Inside a method like GetAll
    -   tracer := commonotel.GetTracer("product-service/repository")
    -   ctx, span := tracer.Start(ctx, "ProductRepository.GetAll", ...)
    +   ctx, span := r.tracer.Start(ctx, "ProductRepository.GetAll", ...) // Use injected tracer
        defer span.End()
        // ... rest of method ...
    ```

**Step 1.4: Modify `InitTelemetry` Return Signature**
*   **File:** `common/otel/provider.go`
*   **Action:** Change `InitTelemetry` to return the initialized `tracerProvider` and `meterProvider` along with the shutdown function and error. Update the calling code in `main.go` accordingly (as shown in Step 1.2).
    ```diff
    // common/otel/provider.go
    - func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
    + func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, tp *trace.TracerProvider, mp *metric.MeterProvider, err error) {
        // ... setup ...
        // Keep local variables: tracerProvider, meterProvider

        // ... existing setup for tracerProvider and meterProvider ...

        // Before returning:
    -   otel.SetTracerProvider(tracerProvider)
    -   otel.SetMeterProvider(meterProvider)
    -   setGlobalProviders(tracerProvider, meterProvider) // Remove global setting
    -   otel.SetTextMapPropagator(prop)

        // Instead, just set the global propagator and return the providers
        otel.SetTextMapPropagator(prop)
        logrustr.Info("Global OpenTelemetry propagator configured. Providers must be injected.")

    -   return shutdown, nil
    +   return shutdown, tracerProvider, meterProvider, nil // Return providers
    }
    ```

**Step 1.5: Implement Secure Exporter Logic**
*   **File:** `common/otel/provider.go`
*   **Action:** Implement the correct TLS configuration logic within `InitTelemetry` based on `cfg.OtelExporterInsecure`. Default to secure. Remove the misleading warning log.
    ```diff
    // common/otel/provider.go - Inside InitTelemetry
        var transportCreds credentials.TransportCredentials
    -   if cfg.OtelExporterInsecure {
    -       transportCreds = insecure.NewCredentials()
    -       logrustr.Warn("Using insecure connection for OTLP exporter")
    -   } else {
    -       logrustr.Warn("TLS configuration for OTLP exporter not implemented, using insecure connection as fallback.") // Remove this
    -       transportCreds = insecure.NewCredentials()
    -   }
    +   if cfg.OtelExporterInsecure {
    +       transportCreds = insecure.NewCredentials()
    +       logrustr.Warn("Using insecure gRPC connection for OTLP exporter as per OTEL_EXPORTER_INSECURE=true")
    +   } else {
    +       // Default to secure connection using system certs or configure as needed
    +       logrustr.Info("Using secure gRPC connection for OTLP exporter (verify collector TLS settings)")
    +       // Basic TLS config (adjust as needed for custom CAs etc.)
    +       transportCreds = credentials.NewTLS(&tls.Config{}) // Relies on system cert pool
    +   }
        exporterOpts = append(exporterOpts, grpc.WithTransportCredentials(transportCreds))
        // ... rest of exporter setup ...
    ```

**Step 1.6: Remove Global Accessors & Setters**
*   **Files:** `common/otel/provider.go`, `common/otel/accessors.go`
*   **Action:**
    *   Delete the global variables `globalTracerProvider`, `globalMeterProvider` (if they exist explicitly).
    *   Delete the `setGlobalProviders` function from `provider.go`.
    *   Delete the functions `GetTracer`, `GetMeter` from `accessors.go`. Ensure `accessors.go` is either removed or only contains truly necessary global accessors (ideally none related to providers).

### Phase 2: Standardize Attributes & Decouple Metrics

**Goal:** Ensure consistent attribute usage and decouple repository logic from OTel metric callbacks.

**Step 2.1: Define Attribute Constants**
*   **File:** `common/otel/attributes.go`
*   **Action:** Define constants for all custom attribute keys. Use `semconv` constants where applicable.
    ```go
    package otel

    import (
        "go.opentelemetry.io/otel/attribute"
        semconv "go.opentelemetry.io/otel/semconv/v1.21.0" // Or newer version
    )

    // Use semantic conventions directly or define constants referencing them
    var (
        DBSystemKey       = semconv.DBSystemKey
        DBOperationKey    = semconv.DBOperationKey
        DBStatementKey    = semconv.DBStatementKey // Example: useful for SQL or operations
        NetPeerNameKey    = semconv.NetPeerNameKey
        HTTPMethodKey     = semconv.HTTPMethodKey
        HTTPRouteKey      = semconv.HTTPRouteKey
        HTTPStatusCodeKey = semconv.HTTPStatusCodeKey
        // Add other relevant semconv keys...

        // Custom Attributes (Namespace appropriately if needed)
        DBFilePathKey       = attribute.Key("db.file.path")
        AppProductIDKey     = attribute.Key("app.product.id")
        ProductNewStockKey  = attribute.Key("product.new_stock")
        // Add other custom keys...
    )
    ```

**Step 2.2: Update Attribute Usage**
*   **Files:** `product-service/src/repository.go`, `product-service/src/handler.go` (anywhere attributes are set).
*   **Action:** Replace all string literals used for attribute keys with the constants defined in `attributes.go`.
    ```diff
    // product-service/src/repository.go - Inside GetByID
        defer span.End()
        r.mu.RLock()
        defer r.mu.RUnlock()
        product, ok := r.products[id]
        if !ok {
            // ... error handling ...
    -       span.RecordError(errNotFound, attribute.String("app.product.id", id))
    +       span.RecordError(errNotFound, oteltrace.WithAttributes(commonotel.AppProductIDKey.String(id)))
            return Product{}, errNotFound
        }
    -   span.SetAttributes(attribute.String("db.system", "file")) // Example, assuming this was set
    +   span.SetAttributes(commonotel.DBSystemKey.String("file")) // Use constant

    // product-service/src/repository.go - Inside UpdateStock
        span.SetAttributes(
    -       attribute.String("db.key", productID),
    -       attribute.Int("product.new_stock", newStock),
    +       commonotel.AppProductIDKey.String(productID), // Assuming product ID is the key here
    +       commonotel.ProductNewStockKey.Int(newStock),
        )
    ```

**Step 2.3: Refactor Repository Metric Observation**
*   **File:** `product-service/src/repository.go`
*   **Action:**
    *   Change `ObserveStockLevels` to a method like `GetCurrentStockLevels` that retrieves and returns data (`map[string]int`) without OTel SDK types.
    *   Remove the `metric.Observer` and `metric.Int64ObservableGauge` parameters from the repository method.
    ```diff
    // product-service/src/repository.go
    -func (r *productRepository) ObserveStockLevels(ctx context.Context, observer metric.Observer, stockGauge metric.Int64ObservableGauge) error {
    +func (r *productRepository) GetCurrentStockLevels(ctx context.Context) (map[string]int, error) {
    +   // Optional: Add tracing for this read operation using r.tracer
    +   ctx, span := r.tracer.Start(ctx, "ProductRepository.GetCurrentStockLevels")
    +   defer span.End()

        logrus.Debug("Repository: GetCurrentStockLevels called")
        r.mu.RLock()
        defer r.mu.RUnlock()
    -   for _, product := range r.products {
    -       observer.ObserveInt64(
    -           stockGauge,
    -           int64(product.Stock),
    -           metric.WithAttributes(
    -               // Use constants here before removing
    -               commonotel.AppProductIDKey.String(product.ProductID),
    -           ),
    -       )
    -   }
    -   return nil
    +   stockLevels := make(map[string]int, len(r.products))
    +   for id, product := range r.products {
    +        stockLevels[id] = product.Stock
    +   }
    +   span.SetAttributes(attribute.Int("app.products.count", len(stockLevels))) // Example attribute
    +   return stockLevels, nil
    }

    // Ensure ProductRepository interface is updated if necessary
    type ProductRepository interface {
        // ... other methods ...
    -   ObserveStockLevels(ctx context.Context, observer metric.Observer, stockGauge metric.Int64ObservableGauge) error
    +   GetCurrentStockLevels(ctx context.Context) (map[string]int, error)
    }

    ```

**Step 2.4: Update Metric Callback Registration**
*   **File:** `common/otel/metrics.go` (or wherever `RegisterCallback` happens, likely needs access to the repo instance now). This might need restructuring, perhaps moving the callback registration to `main.go` where both the `meterProvider` and `repo` instance are available.
*   **Action:** Modify the OTel metric callback function to:
    1.  Call the repository's `GetCurrentStockLevels` method.
    2.  Use the `metric.Observer` provided by the callback to record the retrieved values using the correct instrument (`stockGauge`) and attribute constants.
    ```diff
    // common/otel/metrics.go - Or potentially main.go after DI
    -// Assume RegisterStockMetrics gets Meter and potentially the repo instance now
    -func RegisterStockMetrics(meter metric.Meter, repo ProductRepository) error { // Example signature change
    +func RegisterStockMetrics(meter metric.Meter, repo ProductRepository) error { // ProductRepository needs to be accessible
        stockGauge, err := meter.Int64ObservableGauge(
            "product.stock", // Use semantic conventions if applicable, e.g., `item.stock`?
            metric.WithDescription("Current number of products in stock"),
            metric.WithUnit("{items}"), // Standard unit
        )
        // ... handle error ...

        _, err = meter.RegisterCallback(
            func(ctx context.Context, o metric.Observer) error {
    -           // Old way: Pass observer/gauge down to repo
    -           // return repo.ObserveStockLevels(ctx, o, stockGauge)
    +           // New way: Get data from repo, observe here
    +           levels, err := repo.GetCurrentStockLevels(ctx) // Call the new repo method
    +           if err != nil {
    +               logrus.WithError(err).Error("Failed to get stock levels for metric callback")
    +               // Decide if error should halt observation
    +               return err
    +           }
    +           for id, stock := range levels {
    +                o.ObserveInt64(stockGauge, int64(stock), metric.WithAttributes(
    +                    commonotel.AppProductIDKey.String(id), // Use constant
    +                ))
    +           }
    +           return nil
            },
            stockGauge,
        )
        // ... handle registration error ...
        return nil // Or accumulated error
    }
    ```

### Phase 3: Cleanup and Simplification

**Goal:** Remove dead code and potentially redundant helper abstractions.

**Step 3.1: Remove Dead Code**
*   **File:** `common/config/config.go`
*   **Action:** Delete the large commented-out block of code.

**Step 3.2: Review/Simplify OTel Helper Files**
*   **Files:** `common/otel/trace.go`, `tracer.go`, `meter.go`.
*   **Action:**
    *   Analyze the necessity of the interfaces and helper functions within these files now that DI is used.
    *   If components can directly use `oteltrace.Tracer` and `metric.Meter` interfaces from the SDK effectively, remove the custom abstractions.
    *   Consolidate any genuinely useful, non-redundant helpers (e.g., perhaps span creation wrappers if deemed valuable after Step 5 from the original plan).

**Step 3.3: (Optional) Refactor Span Creation Boilerplate**
*   **Files:** `product-service/src/repository.go`, potentially new helper in `common/otel`.
*   **Action:** If significant boilerplate remains after DI, implement helper functions or wrappers as discussed in the initial plan (Refactoring Plan item #5) to abstract the span start/end/error recording logic.

---

## Verification Strategy
*   **Manual Testing:** After each phase (especially Phase 1 and 2), run the service and the simulator.
*   **Observability Check:** Verify traces and metrics (especially `product.stock`) appear correctly in SigNoz (or the configured backend). Check for correct attributes and values.
*   **Security Check:** Test OTLP connection with TLS enabled on the collector side (default behavior). Test with `OTEL_EXPORTER_INSECURE=true` against an insecure collector endpoint.
*   **Functionality Check:** Ensure core API endpoints (`/products`, `/products/{id}`, `/products/{id}/stock`) still function as expected.

## Expected Outcomes
*   Testable components decoupled from global OTel state.
*   Secure OTLP export enabled by default.
*   Consistent and maintainable telemetry attributes.
*   Improved separation of concerns (repository vs. metrics).
*   Cleaner codebase with reduced boilerplate and no dead code.
