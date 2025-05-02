# Common Module (`github.com/narender/common`)

This module provides shared utilities and foundational components used across different services within the application ecosystem. It aims to promote code reuse, consistency, and maintainability.

## Sub-packages

The `common` module is organized into the following sub-packages:

1.  [config](#config): Environment variable loading and configuration management.
2.  [errors](#errors): Standardized application error types and handling.
3.  [logging](#logging): Application logging setup (using Zap) and context propagation.
4.  [middleware](#middleware): Reusable HTTP middleware (e.g., error handling, request logging, context logger).
5.  [telemetry](#telemetry): OpenTelemetry setup and instrumentation (traces, metrics, logs), including common wrappers.

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

**Purpose:** Initializes the application's primary Zap logger and provides helpers for injecting/retrieving a logger from `context.Context`.

**Usage:**

1.  Initialize the logger early in application startup (typically done within `telemetry.InitTelemetry` which calls `logging.InitZapLogger`):

    ```go
    // Inside telemetry.InitTelemetry:
    baseLogger, err := logging.InitZapLogger(cfg)
    if err != nil {
        // Handle error (InitZapLogger logs internally)
        return nil, fmt.Errorf("failed to initialize base logger: %w", err)
    }
    // The baseLogger is stored globally via manager.InitializeGlobalManager
    ```

2.  Inject a request-scoped logger into context using `middleware.ContextLoggerMiddleware` (see [middleware](#middleware) section).

3.  Access the context-aware logger within handlers or service functions:

    ```go
    package service

    import (
        "context"
        "github.com/narender/common/logging"
        "go.uber.org/zap"
    )

    func doSomething(ctx context.Context) {
        logger := logging.LoggerFromContext(ctx) // Retrieves logger with trace context
        logger.Info("User performed action", zap.Int("user_id", 123))
    }
    ```

4.  Access the base global logger (if context is unavailable, use sparingly):

    ```go
    import "github.com/narender/common/telemetry/manager"

    func backgroundTask() {
        logger := manager.GetLogger() // Gets the base logger
        logger.Info("Running background task")
    }
    ```

**Explanation:**
`InitZapLogger` configures a Zap instance based on `Config` (level, encoding). `ContextLoggerMiddleware` adds trace/span IDs to a cloned logger and injects it into the request's context. `LoggerFromContext` retrieves this enhanced logger. `manager.GetLogger()` provides access to the base logger instance.

---

### `middleware`

**Purpose:** Provides reusable HTTP middleware for common cross-cutting concerns like error handling, request logging, and injecting the context logger.

**Usage (Example with Fiber):**

1.  Register the middleware in the correct order:

    ```go
    package main

    import (
        "github.com/gofiber/fiber/v2"
        otelfiber "github.com/gofiber/contrib/otelfiber/v2"
        "github.com/narender/common/middleware"
        "github.com/narender/common/telemetry/manager"
        // ... other imports
    )

    func main() {
        // ... config loading, telemetry init ...
        baseLogger := manager.GetLogger()

        app := fiber.New(fiber.Config{
            // ErrorHandler uses LoggerFromContext if request context is available,
            // otherwise falls back to the provided baseLogger.
            ErrorHandler: middleware.NewErrorHandler(baseLogger, nil), // Pass base logger for fallback
        })

        // 1. OTel Middleware (adds trace info to context)
        app.Use(otelfiber.Middleware(otelfiber.WithServerName(cfg.ServiceName)))

        // 2. ContextLogger Middleware (adds trace-aware logger to context)
        //    MUST run AFTER OTel middleware
        app.Use(middleware.ContextLoggerMiddleware(baseLogger))

        // 3. RequestLogger Middleware (uses logger from context)
        app.Use(middleware.NewRequestLogger())

        // ... setup routes ...

        app.Listen(":8080")
    }
    ```

**Explanation:**
-   `otelfiber.Middleware`: Instruments requests, adding trace information to the `context.Context`.
-   `ContextLoggerMiddleware`: Clones the `baseLogger`, extracts trace/span IDs from the context (populated by the OTel middleware), adds them to the cloned logger's fields, and injects this *new* logger instance back into the context using `logging.NewContextWithLogger`.
-   `RequestLogger`: Retrieves the logger *from the context* using `logging.LoggerFromContext` and logs request/response details. Because it runs *after* `ContextLoggerMiddleware`, the logger it gets includes trace IDs.
-   `ErrorHandler`: Tries to get the logger from context first. If an error occurs very early (before context logger is set) or outside a request context, it uses the `baseLogger` provided during initialization.

---

### `telemetry`

**Purpose:** Configures and manages OpenTelemetry instrumentation (tracing, metrics, logging). Provides accessors (`manager`) and common wrappers (`trace`, `metric`) for simplified instrumentation.

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

2.  Start spans using the common wrapper:

    ```go
    package service

    import (
        "context"
        "time"
        "github.com/narender/common/logging" // For logger from context
        "github.com/narender/common/telemetry/trace" // For StartSpan
        "github.com/narender/common/telemetry/metric" // For RecordOperationMetrics
        "go.opentelemetry.io/otel/attribute"
    )

    const serviceLayerName = "service"
    const serviceScopeName = "github.com/user/myapp/service"

    func ProcessRequest(ctx context.Context, data string) (err error) {
        logger := logging.LoggerFromContext(ctx)
        startTime := time.Now()
        operation := "ProcessRequest"

        // Use the wrapper to start the span
        ctx, span := trace.StartSpan(ctx, serviceScopeName, operation, attribute.String("data.size", fmt.Sprintf("%d", len(data))))
        defer span.End()

        // Use the wrapper to record metrics (defer AFTER span end)
        defer func() {
             metric.RecordOperationMetrics(ctx, serviceLayerName, operation, startTime, err, attribute.String("data.size", fmt.Sprintf("%d", len(data))))
        }()

        logger.Info("Processing request...")
        // ... perform processing ...

        // if err != nil {
        //     span.RecordError(err) // Use span directly for errors
        //     span.SetStatus(codes.Error, err.Error())
        //     logger.Error("Processing failed", zap.Error(err))
        //     return err // opErr in defer will capture this
        // }

        logger.Info("Processing successful")
        return nil // opErr in defer will be nil
    }
    ```

3.  Record metrics using the common wrapper (shown in the span example above with `RecordOperationMetrics`).

4.  Logging is handled via `logging.LoggerFromContext` (shown above), automatically correlating with traces thanks to `middleware.ContextLoggerMiddleware`.

5.  Automatic HTTP instrumentation is typically handled by framework-specific middleware like `otelfiber` (shown in the [middleware](#middleware) section) rather than the generic `instrumentation.NewHTTPHandler` unless using `net/http` directly.

**Explanation:**
`InitTelemetry` sets up the SDK. `manager` provides global accessors. `trace.StartSpan` simplifies creating spans with the correct tracer. `metric.RecordOperationMetrics` standardizes duration, count, and error metrics. Logging uses Zap via `logging.LoggerFromContext` for automatic trace correlation.

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