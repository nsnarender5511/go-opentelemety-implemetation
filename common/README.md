# Common Module (`github.com/narender/common`)

This module provides shared utilities and foundational components used across different services within the application ecosystem. It aims to promote code reuse, consistency, and maintainability.

## Sub-packages

The `common` module is organized into the following sub-packages:

1.  [config](#config): Environment variable loading and configuration management.
2.  [errors](#errors): Standardized application error types and handling.
3.  [logging](#logging): Application logging setup (currently using Logrus).
4.  [middleware](#middleware): Reusable HTTP middleware (e.g., error handling).
5.  [telemetry](#telemetry): OpenTelemetry setup and instrumentation (traces, metrics, logs).

---

### `config`

**Purpose:** Handles loading application configuration, typically from environment variables or `.env` files.

**Usage:**

1.  Define a configuration struct (e.g., in `config/config.go`):

    ```go
    package config

    type Config struct {
        ServiceName    string `env:"SERVICE_NAME,required"`
        ServiceVersion string `env:"SERVICE_VERSION,required"`
        LogLevel       string `env:"LOG_LEVEL" envDefault:"info"`
        // ... other config fields
        OtelExporterOtlpEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required"`
        // ... other Otel fields
    }
    ```

2.  Load the configuration at application startup:

    ```go
    package main

    import (
        "log"
        "github.com/narender/common/config"
    )

    func main() {
        cfg, err := config.LoadConfig(".") // Load from .env in the current dir
        if err != nil {
            log.Fatalf("Failed to load config: %v", err)
        }

        // Use cfg fields
        log.Printf("Service Name: %s", cfg.ServiceName)
    }
    ```

**Explanation:**
The `LoadConfig` function reads environment variables (optionally loading a `.env` file first) and populates the fields of the `Config` struct based on the `env` tags.

---

### `errors`

**Purpose:** Provides standardized error types (`AppError`, `ValidationError`, etc.) for consistent error handling and propagation across services.

**Usage:**

1.  Create specific application errors:

    ```go
    package service

    import (
        "net/http"
        "github.com/narender/common/errors"
    )

    func GetProduct(id string) (*Product, error) {
        if id == "" {
            // Use a predefined error type
            return nil, errors.NewAppError(http.StatusBadRequest, errors.TypeValidation, "Product ID cannot be empty", nil)
        }

        product, err := db.FindProduct(id)
        if err != nil {
            if errors.Is(err, errors.ErrNotFound) { // Check for specific sentinel error
                 return nil, errors.NewAppError(http.StatusNotFound, errors.TypeNotFound, "Product not found", map[string]interface{}{"product_id": id})
            }
            // Wrap underlying database error
            return nil, errors.WrapDatabaseError(err, "Failed to query product", map[string]interface{}{"product_id": id})
        }
        return product, nil
    }
    ```

**Explanation:**
Use `NewAppError` or specific constructors like `WrapDatabaseError` to create structured errors containing an HTTP status code, an internal type, a user-friendly message, and optional context details. The `middleware.ErrorHandler` can then use this information to generate appropriate HTTP responses and log details consistently.

---

### `logging`

**Purpose:** Initializes the application's primary logger (Logrus).

**Usage:**

1.  Initialize the logger early in application startup (often done within `telemetry.InitTelemetry`):

    ```go
    // Typically called inside InitTelemetry, but can be called directly if needed earlier
    logger := logging.SetupLogrus(cfg)
    logger.Info("Application starting...")
    ```

2.  Access the configured global logger via the `telemetry/manager`:

    ```go
    package main

    import "github.com/narender/common/telemetry/manager"

    func doSomething() {
        logger := manager.GetLogger()
        logger.WithField("user_id", 123).Info("User performed action")
    }
    ```

**Explanation:**
`SetupLogrus` configures a Logrus instance based on the `Config` (level, format). The globally accessible instance can then be retrieved using `manager.GetLogger()` throughout the application.

---

### `middleware`

**Purpose:** Provides reusable HTTP middleware for common cross-cutting concerns.

**Usage (Example with Fiber):**

1.  Register the `ErrorHandler` middleware:

    ```go
    package main

    import (
        "github.com/gofiber/fiber/v2"
        "github.com/narender/common/middleware"
        "github.com/narender/common/telemetry/manager"
        // ... other imports
    )

    func main() {
        // ... config loading, telemetry init ...
        logger := manager.GetLogger()
        // httpMetrics should be initialized if metrics are needed by the handler
        // httpMetrics, _ := metric.NewHTTPMetrics()

        app := fiber.New(fiber.Config{
            ErrorHandler: middleware.NewErrorHandler(logger, nil), // Pass logger, and optionally metrics
        })

        // ... setup routes ...

        app.Listen(":8080")
    }
    ```

**Explanation:**
The `NewErrorHandler` creates a Fiber-compatible error handler. It intercepts errors returned from route handlers. If the error is an `errors.AppError` (or similar), it uses the structured information to create an appropriate JSON error response and logs the details. Otherwise, it treats it as a generic internal server error.

---

### `telemetry`

**Purpose:** Configures and manages OpenTelemetry instrumentation (tracing, metrics, logging). Provides accessors for obtaining Tracers and Meters.

**Usage:**

1.  Initialize Telemetry at application startup:

    ```go
    package main

    import (
        "context"
        "log"
        "github.com/narender/common/config"
        "github.com/narender/common/telemetry"
        // ... other imports
    )

    func main() {
        cfg, err := config.LoadConfig(".")
        if err != nil {
            log.Fatalf("Failed to load config: %v", err)
        }

        // Initialize telemetry
        shutdown, err := telemetry.InitTelemetry(context.Background(), cfg)
        if err != nil {
            log.Fatalf("Failed to initialize telemetry: %v", err)
        }
        defer func() {
            if err := shutdown(context.Background()); err != nil {
                log.Printf("Error shutting down telemetry: %v", err)
            }
        }()

        // ... rest of application setup ...
    }
    ```

2.  Get a Tracer and start a span:

    ```go
    package service

    import (
        "context"
        "github.com/narender/common/telemetry/manager"
    )

    const tracerName = "my-service-tracer" // Or use a more specific name

    func ProcessRequest(ctx context.Context, data string) error {
        tracer := manager.GetTracer(tracerName) // Get the global tracer
        ctx, span := tracer.Start(ctx, "ProcessRequest")
        defer span.End()

        // ... perform processing ...

        span.SetAttributes(attribute.String("app.data.size", fmt.Sprintf("%d", len(data))))

        // ... handle errors and potentially record them on the span ...
        // if err != nil {
        //     trace.RecordSpanError(span, err) // Using trace package utility
        //     return err
        // }

        return nil
    }
    ```

3.  Get a Meter and record a metric (example using HTTP metrics):

    ```go
    package middleware // Example usage in middleware

    import (
        "github.com/narender/common/telemetry/metric"
        "github.com/narender/common/telemetry/attributes"
        "go.opentelemetry.io/otel/attribute"
        // ... other imports
    )

    // Assuming httpMetrics were initialized and passed to where needed
    var httpMetrics *metric.HTTPMetrics

    func observeRequest(c *fiber.Ctx) {
         start := time.Now()
         // ... handle request ...
         duration := time.Since(start)
         statusCode := c.Response().StatusCode()

         if httpMetrics != nil {
             attrs := []attribute.KeyValue{
                 attributes.HTTPMethodKey.String(c.Method()),
                 attributes.HTTPRouteKey.String(c.Route().Path),
                 attributes.HTTPStatusCodeKey.Int(statusCode),
             }
             httpMetrics.RecordHTTPRequestDuration(c.UserContext(), duration, attrs...)
         }
    }
    ```

4.  Wrap an HTTP handler for automatic tracing (using `instrumentation` package):

    ```go
    import (
        "net/http"
        "github.com/narender/common/telemetry/instrumentation"
    )

    func main() {
        mux := http.NewServeMux()
        finalHandler := http.HandlerFunc(myActualHandler)

        // Wrap the handler
        otelHandler := instrumentation.NewHTTPHandler(finalHandler, "myActualHandlerOperation")

        mux.Handle("/myendpoint", otelHandler)
        // ... serve ...
    }

    func myActualHandler(w http.ResponseWriter, r *http.Request) {
         // Handler logic
         w.Write([]byte("Hello"))
    }
    ```

**Explanation:**
`InitTelemetry` sets up the entire OpenTelemetry SDK based on the application `Config`. It registers global providers and initializes a `TelemetryManager`. The `manager` package provides `GetTracer`, `GetMeter`, and `GetLogger` for accessing the configured instances. The `instrumentation` package provides wrappers (like `NewHTTPHandler`) for common libraries to simplify adding telemetry. The `trace` package contains utilities like `RecordSpanError`. 