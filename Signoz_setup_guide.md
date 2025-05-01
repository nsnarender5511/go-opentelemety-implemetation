# Signoz OpenTelemetry Setup Guide

This guide outlines the steps to correctly implement OpenTelemetry (OTel) in the Go project, aligning the codebase with the `gemini_analysis.md` plan and addressing identified discrepancies and issues.

**Prerequisites:**

*   An OTel Collector (e.g., SigNoz) running and accessible (default endpoint assumed: `localhost:4317`).
*   Go environment set up.
*   Access to the project's `go.mod` file.

---

## Phase 1: Correcting Setup & Implementing Core Telemetry

### Step 1: Determine Go Module Path

1.  Open the `product-service/go.mod` file located in the `product-service` directory.
2.  The module path is declared on the first line: `module product-service`.
3.  **Note this path:** `product-service`. You will use it in the next step. (The `replace` directive in `go.mod` should eventually be removed after fixing imports).

### Step 2: Fix Import Paths

The codebase currently uses `common/...` which works locally due to a `replace` directive in `product-service/go.mod`. For clarity and standard practice, update these to use the actual local module path (`product-service/common/...`).

Modify the import blocks in the following files:

*   **`product-service/src/main.go`**:
    *   Change `config "common/config"` to `config "product-service/common/config"`
    *   Change `"common/telemetry"` to `"product-service/common/telemetry"`
*   **`product-service/src/handler.go`**:
    *   Change `"common/errors"` to `"product-service/common/errors"` (This path will be created in Step 4)
    *   Change `"common/telemetry"` to `"product-service/common/telemetry"`
*   **`product-service/src/service.go`**:
    *   Change `"common/telemetry"` to `"product-service/common/telemetry"`
*   **`product-service/src/repository.go`**:
    *   Change `"common/errors"` to `"product-service/common/errors"` (This path will be created in Step 4)
    *   Change `"common/telemetry"` to `"product-service/common/telemetry"`
*   **`common/utils.go`** (This file will be removed in Step 4):
    *   Change `commonerrors "common/errors"` to `commonerrors "product-service/common/errors"`
    *   Change `"common/logger"` to `"product-service/common/logger"` (This import will be removed later).

### Step 3: Complete `common/telemetry` Implementation

The `common/telemetry` directory is currently incomplete. Create the missing files using the code provided in the corresponding steps of `gemini_analysis.md`.

1.  **Create `common/telemetry/config.go`**: Use the code from `gemini_analysis.md`, Step 3, part 2. Remember to adjust the import path within this file:
    *   Change `import "your_module_path/common/config"` to `import "product-service/common/config"`.
2.  **Create `common/telemetry/resource.go`**: Use the code from `gemini_analysis.md`, Step 4.
3.  **Create `common/telemetry/trace.go`**: Use the code from `gemini_analysis.md`, Step 5.
4.  **Create `common/telemetry/metric.go`**: Use the code from `gemini_analysis.md`, Step 6.
5.  **Create `common/telemetry/init.go`**: Use the code from `gemini_analysis.md`, Step 8.
6.  **Update `common/telemetry/log.go`**:
    *   Locate the line: `otelLogger = loggerProvider.Logger("common/telemetry")` (around line 74).
    *   Change it to use your module path: `otelLogger = loggerProvider.Logger("product-service/common/telemetry")`.
    *   Verify that the import `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc` is correct (it seems fixed in the attached version, but double-check against `gemini_analysis.md` Step 7 if issues arise).
    *   Ensure the attribute setting logic in `Fire` uses `otelLogs.KeyValue` or `attribute.KeyValue` consistently as per the OTel API version you are using (the attached `log.go` seems to use `otelLogs.KeyValue`, while `gemini_analysis.md` used `attribute.KeyValue`. Ensure consistency with the rest of the telemetry setup).

### Step 4: Refactor Error Handling to `common/errors`

Centralize error definitions and handling logic.

1.  **Create Directory**: `mkdir -p common/errors`
2.  **Create `common/errors/errors.go`**:
    *   Add `package errors` at the top.
    *   Define common error variables as shown in `gemini_analysis.md` (e.g., `ErrProductNotFound = errors.New("product not found")`, `ErrDatabaseOperation = errors.New("database operation failed")`). Add other domain-specific errors as needed (e.g., `ErrUserNotFound`). Include a generic `ErrNotFound = errors.New("resource not found")`.
    *   Import the necessary packages (`errors`, `fmt`, `net/http`, `github.com/gofiber/fiber/v2`, `github.com/sirupsen/logrus`). **Do not** import `common/logger`.
    *   Move the `HandleServiceError` function from `common/utils.go` into this file.
    *   Update `HandleServiceError`:
        *   Remove the `l := logger.Get()` line and the import for `common/logger`.
        *   Add logging *within* the function using `logrus.WithContext(c.UserContext()).WithError(err).Errorf("Failed to %s", action)`. (Ensure `logrus` is imported).
        *   Update the `switch` statement to use the locally defined error variables (e.g., `case errors.Is(err, ErrProductNotFound):`).
3.  **Update Callers**:
    *   Modify `product-service/src/handler.go`:
        *   Change the import from `"your_module_path/common/utils"` (or wherever `HandleServiceError` was) to `"product-service/common/errors"`.
        *   Ensure calls to `errors.HandleServiceError` are correct.
    *   Modify `product-service/src/repository.go`:
        *   Change the import from `"common/errors"` to `"product-service/common/errors"`.
        *   Return the appropriate error variables defined in `common/errors/errors.go` (e.g., `return Product{}, errors.ErrProductNotFound`).
4.  **Delete `common/utils.go`**: If `HandleServiceError` was its only content, delete the file: `rm common/utils.go`.

---

## Phase 2: Refining Implementation & Best Practices

### Step 5: Standardize Span Creation

Use the `tracer.WithSpan` helper function for consistency and safety, instead of manual `tracer.Start` and `defer span.End()`.

1.  **Update `product-service/src/handler.go`**:
    *   In methods like `GetAllProducts`, `GetProductByID`, `GetProductStock`, replace `ctxSpan, span := h.tracer.Start(...)` and `defer span.End()` blocks.
    *   Wrap the core logic (e.g., the call to the service layer) within `err := h.tracer.WithSpan(ctx, "SpanName", func(ctx context.Context) error { ... })`.
    *   The `WithSpan` helper automatically handles `span.End()`, records errors, and sets the span status based on the returned error. Remove manual calls to `span.RecordError` and `span.SetStatus` *if* they are solely based on the error returned by the wrapped function. You can still add attributes inside the `WithSpan` function body using `span := trace.SpanFromContext(ctx); span.SetAttributes(...)`.
    *   Ensure metric recording (`.Add`) happens *outside* the `WithSpan` call, but uses the context returned by it if needed, and considers the `err` returned by `WithSpan`.
2.  **Update `product-service/src/service.go`**:
    *   Apply the same pattern as above in `GetAll`, `GetByID`, `GetStock`. Wrap the calls to `s.repo.*` methods within `s.tracer.WithSpan(...)`.
3.  **Update `product-service/src/repository.go`**:
    *   Apply the same pattern in `FindAll`, `FindByProductID`, `FindStockByProductID`, and `readData`. Wrap the core file I/O or data manipulation logic within `r.tracer.WithSpan(...)`. Remember to return errors appropriately from the inner function for `WithSpan` to record them.

### Step 6: Fix Repository Panic

Modify `NewProductRepository` to return an error instead of panicking on file issues.

1.  **Update `product-service/src/repository.go`**:
    *   Change the signature: `func NewProductRepository() (ProductRepository, error)`
    *   Replace `panic(...)` calls with `return nil, fmt.Errorf(...)`.
    *   Return `r, nil` at the end on success.
2.  **Update `product-service/src/main.go`**:
    *   Change the call: `productRepo, err := NewProductRepository()`
    *   Add error handling: `if err != nil { logrus.WithError(err).Fatal("Failed to initialize ProductRepository") }`

### Step 7: Remove Redundant Fiber Logger

The OTel middleware and Logrus hook provide sufficient request logging.

1.  **Update `product-service/src/main.go`**:
    *   Remove the line `app.Use(fiberlogger.New())`.

### Step 8: Standardize OTel Attribute Keys

Use constants and semantic conventions for attribute keys to avoid typos and improve consistency.

1.  **(Optional) Create `common/telemetry/attributes.go`**:
    *   Add `package telemetry`.
    *   Import `go.opentelemetry.io/otel/attribute` and `semconv "go.opentelemetry.io/otel/semconv/v1.25.0"`.
    *   Define constants for frequently used custom attributes (e.g., `ProductCountKey = attribute.Key("product.count")`, `LookupSuccessKey = attribute.Key("app.lookup.success")`).
    *   Reference semantic convention keys directly where applicable (e.g., `semconv.DBSystemKey`, `semconv.DBOperationKey`, `semconv.HTTPMethodKey`).
2.  **Update Code**:
    *   Modify `handler.go`, `service.go`, `repository.go`.
    *   Replace magic strings in `attribute.String(...)`, `attribute.Int(...)` etc., with the defined constants or `semconv` keys (e.g., `attribute.String("product.id", productID)` becomes `productIDKey.String(productID)` or potentially `semconv.CodeFunctionKey.String(...)` if appropriate).

### Step 9: (Optional) Use Dependency Injection for Tracer/Meter

Improve testability and clarify dependencies by injecting Tracer and Meter instances.

1.  **Update Constructors**:
    *   Modify `NewProductHandler(service ProductService)` to `NewProductHandler(service ProductService, tracer trace.Tracer, meter metric.Meter)`. Store `tracer` and `meter` on the struct. Remove `telemetry.GetTracer` and `telemetry.GetMeter` calls from within the constructor. Initialize counters using the passed-in `meter`.
    *   Modify `NewProductService(repo ProductRepository)` to `NewProductService(repo ProductRepository, tracer trace.Tracer)`. Store `tracer`. Remove `telemetry.GetTracer`.
    *   Modify `NewProductRepository()` to `NewProductRepository(tracer trace.Tracer) (ProductRepository, error)`. Store `tracer`. Remove `telemetry.GetTracer`.
2.  **Update `product-service/src/main.go`**:
    *   After `telemetry.InitTelemetry()`, get the global tracer and meter once:
        ```go
        tracer := otel.Tracer("product-service/product-service") // Or a more specific name like product-service/main
        meter := otel.Meter("product-service/product-service")   // Or a more specific name like product-service/main
        ```
    *   Pass these instances when calling the constructors: `productRepo, err := NewProductRepository(tracer)`, `productService := NewProductService(productRepo, tracer)`, `productHandler := NewProductHandler(productService, tracer, meter)`.

---

## Phase 3: Verification

### Step 10: Run the Service

1.  Ensure your OTel collector is running.
2.  Set required environment variables (adjust values as needed):
    ```bash
    export OTEL_SERVICE_NAME="product-service-final"
    export OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317" # Your collector's gRPC endpoint
    export OTEL_EXPORTER_INSECURE="true" # Use "false" if your collector uses TLS
    export LOG_LEVEL="info" # Or "debug" for more verbose logs
    export LOG_FORMAT="json" # Recommended for structured logs
    export PRODUCT_SERVICE_PORT="8082"
    ```
3.  Navigate to the service directory: `cd product-service/src`
4.  Tidy dependencies: `go mod tidy` (Run this from the `product-service` directory, *not* `product-service/src`)
5.  Run the service: `go run .` (Run this from the `product-service/src` directory)

### Step 11: Test and Verify

1.  Send requests to the service endpoints (e.g., using `curl` or a tool like Postman).
    *   `curl http://localhost:8082/products`
    *   `curl http://localhost:8082/products/PROD-123`
    *   `curl http://localhost:8082/products/PROD-123/stock`
    *   `curl http://localhost:8082/products/INVALID-ID` (to test errors)
2.  Check your OTel backend (e.g., SigNoz UI):
    *   **Traces:** Look for traces named after the service (`product-service-final`) and spans corresponding to HTTP requests (`GET /products/:productId`), service calls (`GetByIDService`), repository calls (`FindByProductIDRepo`), etc. Verify trace IDs propagate correctly and errors are marked on spans.
    *   **Metrics:** Look for runtime/host metrics and your custom application metrics (`app.product.lookups`, `app.product.stock_checks`). Check if counts increase and attributes (`product.id`, `app.lookup.success`) are present.
    *   **Logs:** Find logs from the service. Verify they have `trace_id` and `span_id` attributes when generated within a request context, allowing correlation with traces. Check severity levels and log messages.

---

By following these steps, you should have a robust and correctly implemented OpenTelemetry setup aligned with the original plan and best practices. 