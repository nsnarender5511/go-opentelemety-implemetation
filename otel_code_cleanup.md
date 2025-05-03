# Refactoring Plan: Common Telemetry Module Cleanup

## 1. Feature Overview

This plan details the refactoring of the `common` module's telemetry initialization and management code. The goals are:
*   Simplify the application's telemetry setup by delegating tasks like complex resource detection, batching, and final endpoint exporting to the OpenTelemetry Collector.
*   Remove redundant configurations and custom abstractions within the `common` module.
*   Align the setup with standard OpenTelemetry practices and SDK usage.
*   Ensure telemetry (traces, metrics, logs) correctly flows from the application to the OTel Collector.

**Target Architecture:**

```
[Application (e.g., product-service)]
    |
    | OTLP (gRPC/HTTP to Collector)
    V
[OTel Collector (otel-collector service)]
    | - Receives OTLP data
    | - Detects/Enriches Resources (docker, env, system, static)
    | - Batches data
    | - (Optional) Samples data
    | - Exports OTLP data (to Backend)
    V
[Backend (e.g., ingest.in.signoz.cloud:443)]
```

**Logging Strategy:** Application logs will be written to stdout using the configured Zap logger. The OTel Collector can be configured separately to collect these logs (e.g., via Docker logging driver or `filelog` receiver). OTel Logging SDK setup will be removed from the `common` module.

---

## 2. Implementation Plan

### Phase 1: Configuration & Resource Simplification

**Step 1: Simplify Application Configuration**

*   **What:** Remove backend-specific OTLP settings and add a setting for the Collector endpoint.
*   **Where:** `common/config/config.go`
*   **Why:** The application should only know about its immediate telemetry destination (the Collector), not the final backend. This decouples the application from the backend infrastructure and centralizes backend configuration in the Collector.
*   **How:**
    *   Remove fields like `OtlpExporterEndpoint`, `OtlpExporterHeaders`, `OtlpInsecure` (if they exist based on exporter setup).
    *   Add a field for the collector endpoint.

    ```diff
    // common/config/config.go
    type Config struct {
        ServiceName             string `env:"SERVICE_NAME,required"`
        ServiceVersion          string `env:"SERVICE_VERSION,required"`
        Environment             string `env:"ENVIRONMENT,default=development"`
        ProductServicePort      string `env:"PRODUCT_SERVICE_PORT,default=8082"`
        LogLevel                string `env:"LOG_LEVEL,default=info"`
        DataFilePath            string `env:"DATA_FILE_PATH,default=./data.json"`
    +   OtelExporterOtlpEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,default=otel-collector:4317"`
    -   // Remove any fields related to specific OTLP backend endpoint, headers, TLS, etc.
    -   // Example: OtlpEndpoint string `env:"OTLP_ENDPOINT"`
    -   // Example: OtlpHeaders  map[string]string `env:"OTLP_HEADERS"`
    }

    func LoadConfig(logger *zap.Logger) (*Config, error) {
        // ... (ensure new field is loaded)
    }

    // Ensure GetDefaultConfig reflects the changes if applicable
    func GetDefaultConfig() *Config {
        return &Config{
            ProductServicePort: "8082",
            ServiceName:        "product-service", // Default, should be overridden by env
            ServiceVersion:     "1.0.0",         // Default, should be overridden by env
            DataFilePath:       "/app/data.json",
            LogLevel:           "info",
            Environment:        "development",
    +       OtelExporterOtlpEndpoint: "otel-collector:4317", // Default collector endpoint
            // Remove defaults for removed fields
        }
    }

    ```

**Step 2: Simplify Resource Creation**

*   **What:** Rely more on standard OTel environment variables and automatic detection for resource attributes.
*   **Where:** `common/telemetry/resource/resource.go` (or wherever `NewResource` is defined).
*   **Why:** Reduces boilerplate code in the application. Leverages the standard `OTEL_SERVICE_NAME` and `OTEL_RESOURCE_ATTRIBUTES` environment variables and lets the Collector handle more complex detection (like container ID, host info).
*   **How:** Modify the `NewResource` function to use default detectors and only add essential attributes programmatically if necessary.

    ```diff
    // common/telemetry/resource/resource.go (Example structure)
    package resource

    import (
        "context"
        "fmt"

        "github.com/narender/common/config"
        "go.opentelemetry.io/otel"
    +   "go.opentelemetry.io/otel/sdk/resource"
    +   semconv "go.opentelemetry.io/otel/semconv/v1.21.0" // Use appropriate version
    )

    // NewResource creates a resource with default detectors and essential attributes.
    func NewResource(ctx context.Context, cfg *config.Config) (*resource.Resource, error) {
        // Start with default detectors. This automatically includes OTEL_SERVICE_NAME
        // and OTEL_RESOURCE_ATTRIBUTES environment variables.
    +   res, err := resource.New(ctx,
    +       resource.WithDetectors(otel.DefaultResourceDetectors()...), // Standard detectors
    +       resource.WithTelemetrySDK(),                            // Adds SDK info
    +       // Add other essential *static* attributes if NOT covered by OTEL_RESOURCE_ATTRIBUTES
    +       // Example: resource.WithAttributes(semconv.DeploymentEnvironmentKey.String(cfg.Environment)),
    +       // BUT prefer setting via OTEL_RESOURCE_ATTRIBUTES if possible.
    +       // Explicitly setting service name/version might be redundant if OTEL_SERVICE_NAME is set.
    +       // resource.WithAttributes(semconv.ServiceNameKey.String(cfg.ServiceName)),
    +       // resource.WithAttributes(semconv.ServiceVersionKey.String(cfg.ServiceVersion)),
    +   )
    +   if err != nil {
    +       return nil, fmt.Errorf("failed to create resource: %w", err)
    +   }
    +   return res, nil

    -   // Remove manual creation of attributes like host.name, os.type, etc.
    -   // Example: hostName, _ := os.Hostname()
    -   // attributes := []attribute.KeyValue{
    -   //     semconv.ServiceNameKey.String(cfg.ServiceName),
    -   //     semconv.ServiceVersionKey.String(cfg.ServiceVersion),
    -   //     semconv.DeploymentEnvironmentKey.String(cfg.Environment),
    -   //     attribute.String("host.name", hostName), // Let collector detect this
    -   //     // ... other manually set attributes
    -   // }
    -   // return resource.New(ctx, resource.WithAttributes(attributes...))
    }

    ```

**Step 3: Update Docker Compose Environment**

*   **What:** Set standard OTel environment variables for the application service.
*   **Where:** `docker-compose.yml`
*   **Why:** To provide necessary configuration (service name, collector endpoint, basic resource attributes) to the application and the OTel SDK according to standard practices.
*   **How:** Add/update the `environment` section for `product-service`.

    ```diff
    # docker-compose.yml
    services:
      product-service:
        build:
          context: .
          dockerfile: ./product-service/Dockerfile
        # labels: # Removed unused label
        #   - "collect_stats=true"
        expose:
          - "8082"
        volumes:
          - ./product-service/data.json:/app/data.json:ro
    +   environment:
    +     - OTEL_SERVICE_NAME=product-service
    +     - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317 # Point to collector service
    +     - OTEL_RESOURCE_ATTRIBUTES=deployment.environment=development,service.version=1.0.0 # Example
    +     - LOG_LEVEL=${LOG_LEVEL:-info} # Keep existing env vars needed by the app
        healthcheck:
          test: ["CMD", "curl", "-f", "http://localhost:8082/health"]
          interval: 10s
        networks:
          - otel_internal-network
        depends_on:
          otel-collector:
            condition: service_healthy
          # ... other dependencies

    ```

---

### Phase 2: Refactor Telemetry Setup Code

**Step 4: Delete Unused Exporter Code**

*   **What:** Remove the custom OTLP exporter setup code.
*   **Where:** `common/telemetry/exporter/` directory.
*   **Why:** The application will use the standard OTLP exporter provided by the OTel Go SDK, configured simply with the Collector endpoint. Custom logic for headers, specific backend details, etc., is no longer needed here.
*   **How:** Delete the entire `common/telemetry/exporter/` directory. Run `go mod tidy` afterwards.

**Step 5: Refactor Telemetry Accessors**

*   **What:** Remove the `TelemetryManager` singleton struct, its state, and initialization logic. Keep simplified accessor functions that delegate directly to the standard OTel global providers.
*   **Where:** `common/telemetry/manager/global_manager.go` (or move simplified functions to a new file like `common/telemetry/accessors.go` and delete the manager file).
*   **Why:** Removes unnecessary custom abstraction and state management while still providing a minimal layer for service code to call, insulating it from direct `otel.*` calls if desired.
*   **How:** Remove the struct, singleton, mutex, and `InitializeGlobalManager`. Simplify `GetTracer` and `GetMeter`.

    ```diff
    // common/telemetry/manager/global_manager.go OR common/telemetry/accessors.go
    package manager // or package telemetry

    import (
    -   "sync"
    -
        "go.opentelemetry.io/otel"
        "go.opentelemetry.io/otel/metric"
    -   metricnoop "go.opentelemetry.io/otel/metric/noop"
    -   sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    -   sdktrace "go.opentelemetry.io/otel/sdk/trace"
        oteltrace "go.opentelemetry.io/otel/trace"
    -   tracenoop "go.opentelemetry.io/otel/trace/noop"
    -   "go.uber.org/zap"
    )

    -// Remove TelemetryManager struct definition
    -/*
    -type TelemetryManager struct {
    -   tracerProvider *sdktrace.TracerProvider
    -   meterProvider  *sdkmetric.MeterProvider
    -   tracer         oteltrace.Tracer
    -   meter          metric.Meter
    -   logger         *zap.Logger
    -
    -   serviceName    string
    -   serviceVersion string
    -}
    -*/

    -// Remove singleton variables
    -/*
    -var (
    -   globalManager *TelemetryManager
    -   once          sync.Once
    -   managerMutex  sync.RWMutex
    -)
    -*/

    -// Remove InitializeGlobalManager function
    -/*
    -func InitializeGlobalManager(tp *sdktrace.TracerProvider, mp *sdkmetric.MeterProvider, log *zap.Logger, serviceName, serviceVersion string) {
    -    // ... (implementation removed)
    -}
    -*/

    // GetTracer returns a tracer from the global provider.
    // The instrumentationName should generally be the package path of the calling code.
    func GetTracer(instrumentationName string) oteltrace.Tracer {
    -   managerMutex.RLock()
    -   defer managerMutex.RUnlock()
    -
    -   // ... (old logic checking globalManager removed)
    -
    -   // Return specific tracer based on name and version (old logic removed)
    -
    +   // Directly delegate to the standard OTel global function
    +   // OTel handles returning a no-op tracer if the provider is not initialized.
    +   return otel.Tracer(instrumentationName)
    }

    // GetMeter returns a meter from the global provider.
    // The instrumentationName should generally be the package path of the calling code.
    func GetMeter(instrumentationName string) metric.Meter {
    -   managerMutex.RLock()
    -   defer managerMutex.RUnlock()
    -
    -   // ... (old logic checking globalManager removed)
    -
    -   // Return specific meter based on name and version (old logic removed)
    -
    +   // Directly delegate to the standard OTel global function
    +   // OTel handles returning a no-op meter if the provider is not initialized.
    +   return otel.Meter(instrumentationName)
    }

    -// Remove GetLogger, GetTracerProvider, GetMeterProvider (these were tied to the manager struct)
    -/*
    -func GetLogger() *zap.Logger { ... }
    -func GetTracerProvider() *sdktrace.TracerProvider { ... }
    -func GetMeterProvider() metric.MeterProvider { ... }
    -*/
    
    ```

**Step 6: Delete Custom Propagator Setup (If Redundant)**

*   **What:** Remove the custom propagator setup function.
*   **Where:** `common/telemetry/propagator/propagator.go`.
*   **Why:** The default global propagator in the OTel SDK is already W3C TraceContext and Baggage, which is the standard. Explicitly setting it again is often redundant unless specific non-default propagators were being configured.
*   **How:** Delete the file `common/telemetry/propagator/propagator.go`. Run `go mod tidy` afterwards. (If non-standard propagators *were* needed, this step needs reconsideration).

**Step 7: Delete Custom Sampler (If Standard is Sufficient)**

*   **What:** Remove custom trace sampler code if standard sampling is adequate for the SDK.
*   **Where:** `common/telemetry/trace/sampler.go` (or similar).
*   **Why:** Simplifies the application code. Standard OTel samplers (`ParentBased(AlwaysSample)`, `ParentBased(TraceIDRatioBased)`) cover common use cases. Complex sampling is often better handled centrally in the Collector.
*   **How:** Delete the sampler file. Run `go mod tidy`. The `InitTelemetry` function will be modified in the next step to use a standard sampler.

**Step 8: Refactor `InitTelemetry` Function**

*   **What:** Rewrite the main telemetry setup function to use standard OTLP exporters, simple processors, default samplers, remove OTel logging setup, and remove dependencies on the deleted packages.
*   **Where:** `common/telemetry/setup.go`
*   **Why:** To align with the new design: export to Collector, simplify processing within the SDK, remove logging redundancy, and use standard components.
*   **How:**

    ```diff
    // common/telemetry/setup.go
    package telemetry

    import (
        "context"
        "errors"
        "fmt"
        "time" // Add if needed for exporter timeouts

        "github.com/narender/common/config"
    -   "github.com/narender/common/telemetry/exporter"  // Remove
    -   "github.com/narender/common/telemetry/manager"    // Remove
    -   "github.com/narender/common/telemetry/metric"     // Keep (or adjust if metric helpers used)
    -   "github.com/narender/common/telemetry/propagator" // Remove
        "github.com/narender/common/telemetry/resource"
    -   "github.com/narender/common/telemetry/trace"      // Keep (or adjust if trace helpers used)
        "go.uber.org/zap"

        "go.opentelemetry.io/otel"
    +   "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc" // Add
    +   "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc" // Add
    -   logglobal "go.opentelemetry.io/otel/log/global" // Remove
    -   sdklog "go.opentelemetry.io/otel/sdk/log"       // Remove
        sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    +   "go.opentelemetry.io/otel/sdk/metric/metricdata" // Add for TemporalitySelector if needed
        sdktrace "go.opentelemetry.io/otel/sdk/trace"
    +   "google.golang.org/grpc" // Add
    )

    func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
        tempLogger := zap.NewExample() // Use a temporary logger for setup
        defer tempLogger.Sync()

        var shutdownFuncs []func(context.Context) error

        // shutdown function to clean up resources
        shutdown = func(ctx context.Context) error {
            var shutdownErr error
            // Execute shutdown functions in reverse order.
            for i := len(shutdownFuncs) - 1; i >= 0; i-- {
                shutdownErr = errors.Join(shutdownErr, shutdownFuncs[i](ctx))
            }
            shutdownFuncs = nil // Clear funcs to prevent double execution
            tempLogger.Info("OpenTelemetry resources shutdown sequence completed.")
            return shutdownErr
        }

        // Defer shutdown in case of initialization errors.
        defer func() {
            if err != nil {
                tempLogger.Error("OpenTelemetry SDK initialization failed", zap.Error(err))
                if shutdownErr := shutdown(context.Background()); shutdownErr != nil {
                    tempLogger.Error("Error during OTel cleanup after setup failure", zap.Error(shutdownErr))
                }
            }
        }()

        // --- Resource ---
        res, err := resource.NewResource(ctx, cfg) // Use simplified resource creation
        if err != nil {
            return shutdown, fmt.Errorf("failed to create resource: %w", err)
        }
        tempLogger.Debug("Resource created", zap.Any("attributes", res.Attributes()))

        // --- Propagator ---
        // No explicit setup needed, SDK defaults to W3C TraceContext and Baggage.
    -   // propagator.SetupPropagators() // Remove

        // --- Trace Provider ---
        tempLogger.Debug("Setting up OTLP Trace Exporter", zap.String("endpoint", cfg.OtelExporterOtlpEndpoint))
    +   traceExporter, err := otlptracegrpc.New(ctx,
    +       otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
    +       otlptracegrpc.WithInsecure(), // Use insecure connection for local collector
    +       // Add other options like WithTimeout if needed
    +       // otlptracegrpc.WithTimeout(5*time.Second),
    +       otlptracegrpc.WithDialOption(grpc.WithBlock()), // Recommended for startup validation
    +   )
        if err != nil {
            return shutdown, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
        }
    +   // Use SimpleSpanProcessor for direct export to local collector
    +   ssp := sdktrace.NewSimpleSpanProcessor(traceExporter)
    -   // sampler := trace.NewSampler(cfg) // Remove custom sampler if using default
    -   // bsp := sdktrace.NewBatchSpanProcessor(traceExporter) // Remove batch processor
    -   // shutdownFuncs = append(shutdownFuncs, bsp.Shutdown) // Remove BSP shutdown

        tp := sdktrace.NewTracerProvider(
            sdktrace.WithResource(res),
    -       // sdktrace.WithSampler(sampler), // Remove if using default
    +       sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.AlwaysSample())), // Or TraceIDRatioBased
    -       // sdktrace.WithSpanProcessor(bsp),
    +       sdktrace.WithSpanProcessor(ssp),
        )
        otel.SetTracerProvider(tp) // Set the global TracerProvider
        shutdownFuncs = append(shutdownFuncs, tp.Shutdown) // Add TracerProvider shutdown
        tempLogger.Debug("TracerProvider initialized and set globally.")

        // --- Meter Provider ---
        tempLogger.Debug("Setting up OTLP Metric Exporter", zap.String("endpoint", cfg.OtelExporterOtlpEndpoint))
    +   metricExporter, err := otlpmetricgrpc.New(ctx,
    +       otlpmetricgrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
    +       otlpmetricgrpc.WithInsecure(),
    +       // Use Delta temporality for counters/histograms, Cumulative for up/down counters
    +       otlpmetricgrpc.WithTemporalitySelector(func(kind sdkmetric.InstrumentKind) metricdata.Temporality {
    +           if kind == sdkmetric.InstrumentKindCounter || kind == sdkmetric.InstrumentKindHistogram {
    +               return metricdata.DeltaTemporality
    +           }
    +           return metricdata.CumulativeTemporality // Default for UpDownCounter, Observable Counters/Gauges
    +       }),
    +       otlpmetricgrpc.WithTimeout(5*time.Second),
    +       otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
    +   )
        if err != nil {
            return shutdown, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
        }

        // Use a periodic reader
    +   reader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second)) // Adjust interval as needed
        mp := sdkmetric.NewMeterProvider(
            sdkmetric.WithResource(res),
            sdkmetric.WithReader(reader),
        )
        otel.SetMeterProvider(mp) // Set the global MeterProvider
        shutdownFuncs = append(shutdownFuncs, mp.Shutdown) // Add MeterProvider shutdown
        tempLogger.Debug("MeterProvider initialized and set globally.")

        // --- Logger Provider ---
    -   // Remove all OTel Logging SDK setup
    -   // logExporter, err := exporter.NewLogExporter(ctx, cfg, tempLogger) ...
    -   // logProcessor := sdklog.NewBatchProcessor(logExporter) ...
    -   // lp := sdklog.NewLoggerProvider(...) ...
    -   // logglobal.SetLoggerProvider(lp) ...
    -   // shutdownFuncs = append(shutdownFuncs, logProcessor.Shutdown) ...

    -   // manager.InitializeGlobalManager(tp, mp, nil, cfg.ServiceName, cfg.ServiceVersion) // Remove
        tempLogger.Info("OpenTelemetry SDK initialized successfully (Traces and Metrics).")

        return shutdown, nil // Return the combined shutdown function and nil error
    }

    ```

**Step 9: Keep Zap Logging Setup**

*   **What:** Ensure the Zap logger setup remains functional for application logging.
*   **Where:** `common/log/setup.go`
*   **Why:** Provides standard application logging, writing to stdout, which can be collected separately if needed.
*   **How:** No changes needed unless it depended on the removed `TelemetryManager`. Ensure it returns a configured `*zap.Logger`.

---

### Phase 3: Update Application Code Usage

**Step 10: Replace Custom Manager Usage**

*   **What:** Update all code that used the custom `TelemetryManager` to use the **new simplified accessor functions** (`telemetry.GetTracer`, `telemetry.GetMeter`) from the common module.
*   **Where:** Search across the entire codebase, especially in `product-service` and potentially other `common` sub-packages.
*   **Why:** To use the refactored, minimal abstraction layer provided by the `common` module.
*   **How:**

    *   **Tracers:**
        ```diff
        // Example in some_handler.go
        import (
        -   "github.com/narender/common/telemetry/manager" // Old import
        +   "github.com/narender/common/telemetry" // New import (or specific package if moved)
        +   "go.opentelemetry.io/otel/attribute"
        +   semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
        )

        func HandleRequest(ctx context.Context) {
        -   // tracer := manager.GetTracer("my-handler-instrumentation") // Old way
        +   tracer := telemetry.GetTracer("github.com/your-org/your-repo/product-service/handler") // New way

            ctx, span := tracer.Start(ctx, "HandleRequest")
            defer span.End()

            span.SetAttributes(attribute.String("key", "value"))
            // ...
        }
        ```

    *   **Meters:**
        ```diff
        // Example in some_service.go
        import (
            "context"
        -   "github.com/narender/common/telemetry/manager" // Old import
        +   "github.com/narender/common/telemetry" // New import (or specific package if moved)
            "go.opentelemetry.io/otel/metric"
        +   "go.opentelemetry.io/otel" // Needed for otel.Handle
        )

        var (
        - // meter = manager.GetMeter("my-service-instrumentation") // Old way init
        + meter = telemetry.GetMeter("github.com/your-org/your-repo/product-service/service") // New way init
          requestCounter metric.Int64Counter
        )

        func init() { // Or in a setup function
            var err error
            requestCounter, err = meter.Int64Counter("requests_processed",
                metric.WithDescription("Number of requests processed"),
                metric.WithUnit("{count}"),
            )
            if err != nil {
                // Handle error - perhaps log using Zap or panic if critical
                otel.Handle(err) // Use default OTel error handler
            }
        }


        func ProcessData(ctx context.Context) {
             // ... processing ...

            if requestCounter != nil {
                requestCounter.Add(ctx, 1)
            }
        }
        ```

    *   **Loggers:**
        ```diff
        // Example in product-service/src/main.go or handlers/services

        // Assume 'logger' is a *zap.Logger obtained from log.NewLogger(cfg)
        // and passed down or stored appropriately.

        - // oldLogger := manager.GetLogger() // Old way (function removed)

        + logger.Info("Some message", zap.String("key", "value")) // Use Zap
        + logger.Error("An error occurred", zap.Error(err))
        ```

**Step 11: Verify `main.go`**

*   **What:** Ensure the application entry point correctly initializes logging and telemetry and handles shutdown.
*   **Where:** `product-service/src/main.go`
*   **Why:** Proper initialization and shutdown are critical for telemetry function and data flushing.
*   **How:** Review the `main` function.

    ```go
    // product-service/src/main.go (Conceptual)
    package main

    import (
        "context"
        "os"
        "os/signal"
        "syscall"
        "time"

        "github.com/narender/common/config"
        "github.com/narender/common/log" // Use the log package
        "github.com/narender/common/telemetry" // Use the telemetry package
        // ... other imports (fiber, handler, repository, service)
        "go.uber.org/zap"
    )

    func main() {
        // Temporary logger for config loading
        tempLogger := zap.NewExample()
        cfg, err := config.LoadConfig(tempLogger)
        if err != nil {
            tempLogger.Fatal("Failed to load configuration", zap.Error(err))
        }

        // Initialize proper application logger
        appLogger := log.NewLogger(cfg) // Get the Zap logger
        defer appLogger.Sync()          // Ensure logs are flushed on exit

        appLogger.Info("Starting product-service",
            zap.String("version", cfg.ServiceVersion),
            zap.String("environment", cfg.Environment),
        )

        // Initialize Telemetry (Traces and Metrics)
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Timeout for setup
        otelShutdown, err := telemetry.InitTelemetry(ctx, cfg)
        cancel() // Release context resources
        if err != nil {
            appLogger.Fatal("Failed to initialize OpenTelemetry", zap.Error(err))
        }
        // Defer the shutdown function to ensure telemetry is flushed gracefully.
        defer func() {
            shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second) // Timeout for shutdown
            defer cancelShutdown()
            if err := otelShutdown(shutdownCtx); err != nil {
                appLogger.Error("Error shutting down OpenTelemetry", zap.Error(err))
            } else {
                 appLogger.Info("OpenTelemetry shutdown complete.")
            }
        }()


        // ... Initialize repository, service, handler, fiber app ...
        // Ensure the 'appLogger' instance is passed to components that need logging.
        // repo := repository.New(...)
        // svc := service.New(repo, appLogger) // Example passing logger
        // hdlr := handler.New(svc, appLogger) // Example passing logger
        // app := fiber.New(...)
        // api.SetupRoutes(app, hdlr, appLogger) // Pass logger if needed

        // ... Start the fiber server ...
        // go func() {
        //    if err := app.Listen(":" + cfg.ProductServicePort); err != nil {
        //        appLogger.Fatal("Fiber server failed to start", zap.Error(err))
        //    }
        // }()


        // Wait for termination signal
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        <-quit

        appLogger.Info("Shutting down server...")

        // ... Gracefully shutdown the fiber server ...
        // shutdownCtx, cancelServerShutdown := context.WithTimeout(context.Background(), 5*time.Second)
        // defer cancelServerShutdown()
        // if err := app.ShutdownWithContext(shutdownCtx); err != nil {
        //    appLogger.Error("Fiber server shutdown failed", zap.Error(err))
        // }

        appLogger.Info("Server exiting")
    }
    ```

---

### Phase 4: Collector Configuration Verification

**Step 12: Review OTel Collector Config**

*   **What:** Ensure the collector configuration aligns with the application changes and best practices.
*   **Where:** `otel-collector-config.yaml`
*   **Why:** The collector is now central to processing and exporting. Its config must correctly receive, process (resource detection, batching), and export telemetry.
*   **How:** Review the existing config (`otel-collector-config.yaml` attached in previous messages shows a good starting point after removing filters). Key points:
    *   **Receivers:** `otlp` receiver must be enabled for `grpc` (on `:4317`) and potentially `http` (on `:4318`). `docker_stats` is separate and likely okay.
    *   **Processors:**
        *   `batch`: Keep enabled. Fine-tune batch sizes/timeouts if needed later.
        *   `resourcedetection`: Ensure `docker`, `env`, `system` detectors are present to automatically add relevant attributes. `override: false` is usually correct.
        *   `resource`: Use this to add *static* attributes not discoverable otherwise (e.g., `deployment.environment` if not reliably set via `OTEL_RESOURCE_ATTRIBUTES`). *Remove* the `host.name` upsert, as the `docker` or `system` detector should handle this.
    *   **Exporters:** `otlp` exporter configured for the final backend (`ingest.in.signoz.cloud:443`) with correct `headers` (like `signoz-ingestion-key`) and `tls` settings (`insecure: false`). `debug` exporter can be helpful during testing.
    *   **Pipelines:**
        *   `traces`: `receivers: [otlp]`, `processors: [resourcedetection, resource, batch]`, `exporters: [otlp]`
        *   `metrics`: `receivers: [otlp, docker_stats]`, `processors: [resourcedetection, resource, batch]`, `exporters: [otlp]`
        *   `logs`: Remove this pipeline entirely for now, as the application is no longer sending OTLP logs. (If Docker log collection is added later via a different receiver like `filelog` or `fluentforward`, a logs pipeline would be needed).
    *   **Service:** Ensure extensions (`health_check`, etc.) and telemetry settings (`logs.level: debug`) are appropriate.

    ```diff
    # otel-collector-config.yaml (Illustrative changes/verification points)
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317 # Listen for app OTLP gRPC
          http:
            endpoint: 0.0.0.0:4318 # Listen for app OTLP HTTP
      docker_stats: # Keep this for host metrics
        endpoint: "unix:///var/run/docker.sock"
        collection_interval: 10s
        timeout: 20s # Increased timeout
        api_version: "1.47"

    processors:
      batch: {}
      resourcedetection:
        detectors: [env, system, docker] # Ensure these are present
        override: false
      resource:
        attributes:
    -     # Remove static host.name if detected by docker/system
    -     # - key: host.name
    -     #   value: "my-static-host"
    -     #   action: upsert
          - key: deployment.environment # Keep if needed and not set via OTEL_RESOURCE_ATTRIBUTES
            value: "development"
            action: insert # Use 'insert' if it should only be added if missing

    exporters:
      otlp:
        endpoint: "ingest.in.signoz.cloud:443" # Final backend
        tls:
          insecure: false # Should be false for secure backend
        headers:
          signoz-ingestion-key: dtsa409InZNwTUjVBS7WPrNLBsANF1ZnZJXd # Keep auth header
      debug: # Optional: useful for debugging
        verbosity: detailed

    # ... extensions ...

    service:
      telemetry:
        logs:
          level: "debug" # Keep debug for collector's own logs if needed
          encoding: "json"
      extensions: [health_check, pprof, zpages]
      pipelines:
        traces:
          receivers: [otlp]
          processors: [resourcedetection, resource, batch] # Order matters
          exporters: [otlp] # Add debug exporter here for testing if needed: [otlp, debug]
        metrics:
          receivers: [otlp, docker_stats]
          processors: [resourcedetection, resource, batch] # Order matters
          exporters: [otlp] # Add debug exporter here for testing if needed: [otlp, debug]
    -   logs: # Remove the OTLP logs pipeline
    -     receivers: [otlp]
    -     processors: [resourcedetection, resource, batch]
    -     exporters: [otlp]

    ```

---

This detailed plan should guide the refactoring process effectively. Remember to test thoroughly after each phase. 