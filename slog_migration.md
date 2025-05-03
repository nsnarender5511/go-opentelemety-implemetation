# Slog Migration Plan: From Zap to log/slog with OTLP Export

## 1. Feature Overview

**What:** Migrate the logging system in the Go application (`common` and `product-service` modules) from `go.uber.org/zap` to the standard Go `log/slog` package. Configure `slog` based on the environment:
*   **Production:** Output logs *only* to the OpenTelemetry Collector via OTLP, leveraging the existing telemetry infrastructure in `common/telemetry` and using the global OTel logger provider pattern. Traces and metrics are also exported via OTLP.
*   **Development (or other non-production):** Output logs *only* to the console (stdout). Traces and metrics export via OTLP are disabled (using No-Op providers).

**Why:** To standardize on Go's built-in structured logging, integrate seamlessly with OpenTelemetry in production for correlated logs, traces, and metrics, while maintaining simple console output and disabling OTel export overhead during development.

## 2. Requirements Analysis

*   Replace `zap` usage entirely with `slog`.
*   **Conditional Logging:**
    *   In `production` environment: Export logs *only* via OTLP/gRPC to the configured OTel Collector.
    *   In non-`production` environments: Output structured logs *only* to the console (stdout).
*   **Conditional Telemetry (Traces/Metrics):**
    *   In `production` environment: Export traces and metrics via OTLP.
    *   In non-`production` environments: Disable OTLP export for traces and metrics (effectively use No-Op providers).
*   Reuse `common/telemetry/setup.go` for OTel resource creation and conditional OTLP exporter configuration.
*   Adapt the existing `common/log` package (`Init`, `L`, `Cleanup`) to use `slog` conditionally.
*   Ensure logs exported via OTLP (in production) automatically include `trace_id` and `span_id` when available.
*   Utilize the global OTel logger provider pattern (`global.SetLoggerProvider`) *only* in production.
*   Minimize structural code changes within `product-service/src`; focus on updating log call syntax and ensuring correct initialization order.

## 3. Solution Design (Revised)

1.  **Dependencies:** Add OTel packages (`otlploggrpc`, `sdklog`, `otelglobal`, `otelslog`). Remove `zap` later.
2.  **Telemetry Setup (`common/telemetry/setup.go`):**
    *   Check `cfg.Environment`.
    *   If `"production"`: Initialize OTLP trace, metric, and log exporters and their corresponding `sdk.*Provider` instances (using shared resource). Set them globally via `otel.*Provider` and `otelglobal.SetLoggerProvider`. Return a combined shutdown function for these providers.
    *   If *not* `"production"`: Do *not* initialize OTLP exporters or SDK providers. OpenTelemetry will default to No-Op providers. Return a no-op shutdown function.
3.  **Logging Setup (`common/log/log.go`):**
    *   Adapt `Init` to check `cfg.Environment`.
    *   If `"production"`: Create *only* an `otelslog.NewHandler()` (which reads the global provider set by `InitTelemetry`).
    *   If *not* `"production"`: Create *only* a console handler (e.g., `slog.NewJSONHandler`).
    *   Initialize global `L *slog.Logger` with the selected handler.
    *   Set `L` as `slog.Default`.
    *   **Crucially, `commonlog.Init` must run *after* `telemetry.InitTelemetry`.**
4.  **Application (`product-service/src/main.go`):** Call `InitTelemetry` *first*, then `commonlog.Init`. Update all `zap` log calls to `slog` syntax.
5.  **Middleware (`common/middleware/*`):** Update log call syntax from `zap.Field` to `slog.Attr`.

## 4. Technical Specifications (Revised)

*   **Log Exporter (Prod):** `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc`
*   **Trace/Metric Exporters (Prod):** `otlptracegrpc`, `otlpmetricgrpc`
*   **SDKs (Prod):** `go.opentelemetry.io/otel/sdk/log`, `sdktrace`, `sdkmetric`
*   **Global Providers (Prod):** `go.opentelemetry.io/otel/log/global`, `otel.SetTracerProvider`, `otel.SetMeterProvider`
*   **Slog Bridge (Prod):** `go.opentelemetry.io/contrib/bridges/otelslog`
*   **Slog Handlers:** `log/slog.NewJSONHandler` (Dev) or `log/slog.NewTextHandler` (Dev), `otelslog.NewHandler` (Prod).

## 5. Implementation Plan

---

### Step 5.1: Add Dependencies

*   **What:** Add the required OpenTelemetry packages for log export and the `slog` bridge.
*   **Why:** To make the necessary functions and types available for building the OTel logging pipeline and integrating it with `slog`.
*   **When:** First step, before modifying any code.
*   **How:** Execute the following commands in your terminal:

    ```bash
    # Navigate to the common module directory
    cd common 
    
    # Get the necessary packages
    go get go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc
    go get go.opentelemetry.io/otel/sdk/log
    go get go.opentelemetry.io/contrib/bridges/otelslog
    go get go.opentelemetry.io/otel/log/global
    
    # Tidy up dependencies
    go mod tidy
    
    # Navigate to the product-service module directory
    cd ../product-service
    
    # Tidy up dependencies (to ensure consistency)
    go mod tidy
    cd .. 
    ```

---

### Step 5.2: Modify Telemetry Setup (`common/telemetry/setup.go`) (Revised)

*   **What:** Conditionally initialize OTLP Exporters (Trace, Metric, Log) and OTel SDK Providers *only* if `cfg.Environment == "production"`. Set global providers only in production. Return a conditional shutdown function. Remove `zap` usage.
*   **Why:** To configure the full OTel pipeline (traces, metrics, logs) only for production, allowing development environments to default to No-Op providers, thus avoiding unnecessary export attempts and overhead.
*   **When:** After adding dependencies. Before modifying `common/log`.
*   **How:** Edit `common/telemetry/setup.go`:

    **Before (Relevant Parts):**
    ```go
    import (
    	// ... other imports ...
    	"github.com/narender/common/config"
    	"github.com/narender/common/telemetry/resource"
    	"go.uber.org/zap" // Will be removed
    	// ... other OTel trace/metric imports ...
    )

    func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
    	tempLogger := zap.NewNop() // Will be removed/replaced
    	var shutdownFuncs []func(context.Context) error
        // ... shutdown assignment ...
    	defer func() { // Update this defer
    		if err != nil {
    			// tempLogger.Error(...) // Replace with std log
    			// ... existing shutdown call ...
    		}
    	}()

        // ... Resource (res) creation ...
        // ... Trace provider setup ...
        // ... Metric provider setup ...

        // tempLogger.Info(...) // Replace with std log
    	return shutdown, nil
    }
    ```

    **After (Conditional Initialization):**
    ```go
    package telemetry

    import (
    	"context"
    	"errors"
    	"fmt"
    	"log" // Use standard log for setup messages
    	"strings" // For environment check
    	"time"

    	"github.com/narender/common/config"
    	"github.com/narender/common/telemetry/resource"

    	"go.opentelemetry.io/otel"
    	// Log specific imports (needed even if conditional)
    	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
    	otelgloballog "go.opentelemetry.io/otel/log/global"
    	sdklog "go.opentelemetry.io/otel/sdk/log"

    	// Existing trace/metric imports
    	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
    	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    	"go.opentelemetry.io/otel/metric"
    	"go.opentelemetry.io/otel/propagation" // Needed for propagator setup
    	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
    	sdkresource "go.opentelemetry.io/otel/sdk/resource" // Alias for clarity if needed
    	sdktrace "go.opentelemetry.io/otel/sdk/trace"
    	oteltrace "go.opentelemetry.io/otel/trace"
    	"google.golang.org/grpc"
        "google.golang.org/grpc/credentials/insecure" // Explicit insecure credentials if needed
    )

    // InitTelemetry initializes OpenTelemetry components conditionally based on environment.
    func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
    	var shutdownFuncs []func(context.Context) error

    	// Define shutdown function early
    	shutdown = func(ctx context.Context) error {
            if len(shutdownFuncs) == 0 {
                log.Println("OpenTelemetry shutdown: No providers initialized (likely non-production environment).")
                return nil
            }
    		var shutdownErr error
    		// Execute shutdowns in reverse order
    		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
    			shutdownErr = errors.Join(shutdownErr, shutdownFuncs[i](ctx))
    		}
    		shutdownFuncs = nil // Clear to prevent double execution
    		log.Println("OpenTelemetry resources shutdown sequence completed (production).")
    		return shutdownErr
    	}

    	// Defer simplified error handling
    	defer func() {
    		if err != nil {
    			log.Printf("ERROR: OpenTelemetry SDK initialization failed: %v", err)
    			// Attempt cleanup even on partial failure
    			if shutdownErr := shutdown(context.Background()); shutdownErr != nil {
    				log.Printf("ERROR: OTel cleanup after setup failure: %v", shutdownErr)
    			}
    		}
    	}()

        isProduction := strings.ToLower(cfg.Environment) == "production"

        // --- Shared Resource (Always create) ---
    	res, err := resource.NewResource(ctx, cfg) // Ensure this function exists and works
    	if err != nil {
    		// Error is handled by defer
    		return nil, fmt.Errorf("failed to create resource: %w", err)
    	}
    	log.Println("OTel Resource created.")


        // --- Conditional OTLP Setup (Production Only) ---
        if isProduction {
            log.Println("Production environment detected. Initializing OTLP Trace, Metric, and Log providers.")

            // Common OTLP connection options
            connOpts := []grpc.DialOption{
                grpc.WithTransportCredentials(insecure.NewCredentials()), // Assuming insecure from context
                grpc.WithBlock(),
            }

            // --- Trace Provider ---
            traceExporter, err := otlptracegrpc.New(ctx,
                otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
                otlptracegrpc.WithGRPCDialOption(connOpts...),
            )
            if err != nil { return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err) }
            // Consider BatchSpanProcessor for production instead of SimpleSpanProcessor
            // bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
            ssp := sdktrace.NewSimpleSpanProcessor(traceExporter) // Kept SimpleSpanProcessor as per original plan example
            tp := sdktrace.NewTracerProvider(
                sdktrace.WithResource(res),
                sdktrace.WithSpanProcessor(ssp), // Use bsp for production ideally
            )
            otel.SetTracerProvider(tp)
            // Set the TextMapPropagator globally. Needed for context propagation.
            otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
            shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
            log.Println("OTel TracerProvider initialized and set globally.")


            // --- Metric Provider ---
            metricExporter, err := otlpmetricgrpc.New(ctx,
                otlpmetricgrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
                otlpmetricgrpc.WithGRPCDialOption(connOpts...),
                otlpmetricgrpc.WithTemporalitySelector(sdkmetric.DeltaTemporalitySelector), // Example: Explicit temporality
            )
            if err != nil { return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err) }
            reader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))
            mp := sdkmetric.NewMeterProvider(
                sdkmetric.WithResource(res),
                sdkmetric.WithReader(reader),
            )
            otel.SetMeterProvider(mp)
            shutdownFuncs = append(shutdownFuncs, mp.Shutdown)
            log.Println("OTel MeterProvider initialized and set globally.")


            // --- Log Provider ---
            logExporter, err := otlploggrpc.New(ctx,
                otlploggrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
                otlploggrpc.WithGRPCDialOption(connOpts...),
            )
            if err != nil {
                return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
            }
            logProcessor := sdklog.NewBatchProcessor(logExporter)
            loggerProvider := sdklog.NewLoggerProvider(
                sdklog.WithResource(res),
                sdklog.WithProcessor(logProcessor),
            )
            // *** Set the global LoggerProvider (Only in Production) ***
            otelgloballog.SetLoggerProvider(loggerProvider)
            shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
            log.Println("OTel LoggerProvider initialized and set globally.")

        } else {
             log.Printf("Non-production environment (%s) detected. Skipping OTLP exporter setup. Using No-Op providers.", cfg.Environment)
             // OTel defaults to No-Op providers if none are set globally.
             // No shutdown needed for No-Op providers.
        }

    	log.Println("OpenTelemetry SDK initialization sequence complete.")
    	return shutdown, nil // Return conditional shutdown and nil error
    }

    // Keep GetTracer and GetMeter functions as they are. They will return No-Op
    // Tracer/Meter instances if the global providers are not set (i.e., non-prod).
    func GetTracer(instrumentationName string) oteltrace.Tracer {
        return otel.Tracer(instrumentationName)
    }
    func GetMeter(instrumentationName string) metric.Meter {
        return otel.Meter(instrumentationName)
    }
    ```

---

### Step 5.3: Refactor Logging Setup (`common/log/log.go`) (Revised)

*   **What:** Modify the `common/log` package to use `slog` conditionally. Define a global `*slog.Logger`. Update `Init` to check the environment: create *only* the OTel bridge handler (Prod) or *only* a console handler (Dev/Other), then set the global `L` and `slog.Default`. Remove the `MultiplexHandler`.
*   **Why:** To provide the application with a correctly configured `slog` logger based on the environment, reusing the existing `common/log` structure but without multiplexing.
*   **When:** After modifying `common/telemetry`. Before modifying `product-service/src/main.go`.
*   **How:** Edit/Create `common/log/log.go`:

    **Before (Conceptual - assuming it wrapped zap):**
    ```go
    package log
    
    import "go.uber.org/zap"
    // ... other imports ...

    var L *zap.Logger

    func Init(cfg *config.Config) error {
        // ... zap initialization logic ...
        L = zapLoggerInstance
        return nil
    }

    func Cleanup() {
        _ = L.Sync()
    }
    ```

    **After (Conditional Slog Handler):**
    ```go
    package log

    import (
    	"context" // Keep context for potential future handler needs
        "errors" // For error joining if needed later
    	"log/slog"
    	"os"
    	"strings"

    	"github.com/narender/common/config" // Assuming this path is correct
    	// Only import the bridge if potentially used (Go handles this)
    	"go.opentelemetry.io/contrib/bridges/otelslog"
    )

    // Global slog logger instance
    var L *slog.Logger

    // --- REMOVED MultiplexHandler Implementation ---

    // --- Initialization (Conditional) ---

    // Init initializes the global logger L based on the environment.
    // IMPORTANT: Must be called AFTER telemetry.InitTelemetry, especially for production
    // as it relies on the global logger provider being set.
    func Init(cfg *config.Config) error {
    	if L != nil {
    		slog.Warn("Logger already initialized") // Use default slog before L is set
    		return nil
    	}

    	var level slog.Level
    	switch strings.ToLower(cfg.LogLevel) {
    	case "debug":
    		level = slog.LevelDebug
    	case "warn":
    		level = slog.LevelWarn
    	case "error":
    		level = slog.LevelError
    	default: // "info" or anything else
    		level = slog.LevelInfo
    	}

    	handlerOpts := &slog.HandlerOptions{
    		AddSource: true, // Include source file/line number
    		Level:     level,
    		// ReplaceAttr: // Optional: Customize attribute output if needed
    	}

        var handler slog.Handler
        isProduction := strings.ToLower(cfg.Environment) == "production"

        if isProduction {
            // Production: Use OTel Handler (which uses the global provider set by InitTelemetry)
            slog.Info("Production environment: Configuring OTel slog handler.") // Log this before L is set
            handler = otelslog.NewHandler()
        } else {
            // Development/Other: Use Console Handler (JSON recommended for structure)
            slog.Info("Non-production environment: Configuring Console slog handler (JSON).") // Log this before L is set
            handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
            // Alternative: handler = slog.NewTextHandler(os.Stdout, handlerOpts)
        }

    	// Create the logger instance with the selected handler
    	L = slog.New(handler)

    	// Set it as the default for the standard library
    	slog.SetDefault(L)

    	L.Info("Logger initialized and set as default", slog.String("environment", cfg.Environment), slog.String("level", level.String()))
    	return nil
    }

    // Cleanup remains a no-op for slog, but kept for consistency.
    func Cleanup() {
    	if L != nil {
    		L.Debug("Logger cleanup called (noop).")
    	} else {
    		slog.Debug("Logger cleanup called (noop, logger not initialized).")
    	}
    }
    ```

---

### Step 5.4: Modify Application Entrypoint (`product-service/src/main.go`)

*   **What:** Update `main.go` to call `telemetry.InitTelemetry` *before* `commonlog.Init`, remove `zap` imports/usage, and change all logging calls to use `slog` syntax.
*   **Why:** To correctly initialize the telemetry and logging systems using the new conditional `slog` setup and ensure the application uses the standard logging functions. **Enforcing the initialization order (`InitTelemetry` then `commonlog.Init`) is absolutely critical for the production environment OTel integration to work.**
*   **When:** After `common/log` and `common/telemetry` are updated.
*   **How:** Edit `product-service/src/main.go`:

    **Before (Relevant Parts):**
    ```go
    import (
    	// ... other imports ...
        commonlog "github.com/narender/common/log" // This stays
        "github.com/narender/common/telemetry"
        "go.uber.org/zap" // Remove this
    )

    func main() {
        tempLogger := zap.NewExample() // Remove this
        // ... load config using tempLogger ...

        if err := commonlog.Init(cfg); err != nil { // Called early
             tempLogger.Fatal(...) // Use std log.Fatal before logger init
        }
        defer commonlog.Cleanup()
        appLogger := commonlog.L // This was *zap.Logger

        // ... InitTelemetry called later ...
        otelShutdown, err := telemetry.InitTelemetry(startupCtx, cfg)
        // ... handle error ...

        appLogger.Info("Message", zap.String("k", "v"), zap.Int("n", 1)) // Zap syntax
        // ... other zap calls ...
        appLogger.Fatal("Message", zap.Error(err)) // Zap fatal
    }
    ```

    **After (Corrected Order, Slog Syntax):**
    ```go
    package main

    import (
    	"context"
    	"errors"
    	"log" // Use standard log for fatal errors before init
    	"net/http"
    	"os"      // For os.Exit
    	"os/signal"
    	"syscall"
    	"time"
        "log/slog" // Import slog

    	"github.com/gofiber/contrib/otelfiber/v2"
    	"github.com/gofiber/fiber/v2"
    	"github.com/gofiber/fiber/v2/middleware/cors"
    	"github.com/gofiber/fiber/v2/middleware/recover"

    	"github.com/narender/common/config"
    	commonlog "github.com/narender/common/log" // Keep using this alias
    	"github.com/narender/common/middleware"
    	"github.com/narender/common/telemetry"
    )

    var appConfig *config.Config

    func main() {
        // Load config first (using std log if needed for errors here)
    	cfg, err := config.LoadConfig(nil) // Assuming LoadConfig can handle nil logger or uses std log
    	if err != nil {
    		log.Fatalf("Failed to load configuration: %v", err) // Use std log fatal
    	}
    	appConfig = cfg // Store globally if needed by handlers etc.

    	// --- Initialize OTel First ---
    	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 15*time.Second)
    	otelShutdown, err := telemetry.InitTelemetry(startupCtx, cfg) // Sets global log provider
    	cancelStartup()
    	if err != nil {
    		log.Fatalf("Failed to initialize OpenTelemetry: %v", err) // Std log fatal
    	}
    	log.Println("OpenTelemetry initialized successfully.") // Std log info

    	// --- Initialize Slog Second (uses global OTel provider) ---
    	if err := commonlog.Init(cfg); err != nil {
    		log.Fatalf("Failed to initialize application logger: %v", err) // Std log fatal
    	}
    	defer commonlog.Cleanup()

    	// --- Now use commonlog.L (which is *slog.Logger) ---
    	appLogger := commonlog.L // Or use slog.Default() directly

    	appLogger.Info("Starting service",
    		slog.String("service.name", cfg.ServiceName),
    		slog.String("service.version", cfg.ServiceVersion),
    		slog.String("environment", cfg.Environment),
    	)

        // ... Rest of main function ...

        // --- Update logging calls ---
        // Example: Repository Init Error
    	repo, err := NewProductRepository(cfg.DataFilePath)
    	if err != nil {
            // Use slog Error + os.Exit instead of Fatal
    		appLogger.Error("Failed to initialize product repository", slog.Any("error", err))
            os.Exit(1)
    	}
        // Example: Server Start
        appLogger.Info("Server starting to listen", slog.String("address", addr))
        // Example: Server Listen Error
        if err := app.Listen(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
            appLogger.Error("Server listener failed", slog.Any("error", err))
            os.Exit(1) // Exit after logging error
        }
        // Example: Shutdown Message
        appLogger.Info("Shutdown signal received, initiating graceful server shutdown...")
        // Example: Shutdown Error
        if err := app.ShutdownWithContext(serverShutdownCtx); err != nil {
             appLogger.Error("Fiber server graceful shutdown failed", slog.Any("error", err))
        } else {
             appLogger.Info("Fiber server shutdown complete.")
        }
        // Example: OTel Shutdown Error
        if err := otelShutdown(shutdownCtx); err != nil {
            appLogger.Error("Error during OpenTelemetry shutdown", slog.Any("error", err))
        } else {
            appLogger.Info("OpenTelemetry shutdown complete.")
        }

        appLogger.Info("Application exiting.")
    }

    ```

---

### Step 5.5: Modify Middleware (`common/middleware/*.go`)

*   **What:** Update any custom middleware that uses the logger to use `slog` syntax.
*   **Why:** To ensure consistent logging format and correctly pass attributes.
*   **When:** After `main.go` is updated.
*   **How:** Find files like `common/middleware/request_logger.go` or `error_handler.go`.

    **Before (Conceptual):**
    ```go
    // In request logger middleware
    logger.Info("Request received",
        zap.String("method", c.Method()),
        zap.String("path", c.Path()),
        zap.Duration("latency", latency),
    )
    // In error handler
    logger.Error("Unhandled error",
        zap.Error(err),
        zap.String("path", c.Path()),
    )
    ```

    **After (Slog Syntax):**
    ```go
    import "log/slog" // Add import

    // In request logger middleware
    logger.Info("Request received",
        slog.String("method", c.Method()),
        slog.String("path", c.Path()),
        slog.Duration("latency", latency),
    )
    // In error handler
    logger.Error("Unhandled error",
        slog.Any("error", err), // Use slog.Any for errors
        slog.String("path", c.Path()),
    )
    ```

---

### Step 5.6: Cleanup OTel Collector Config (`otel-collector-config.yaml`)

*   **What:** Ensure the collector is configured to receive OTLP logs and remove any fallback configurations (like `filelog`).
*   **Why:** To correctly process logs sent from the newly configured application.
*   **When:** After application code changes are complete.
*   **How:** Edit `otel-collector-config.yaml`:

    **Ensure/Modify:**
    ```yaml
    receivers:
      otlp: # Make sure OTLP receiver is present
        protocols:
          grpc: # Ensure grpc is enabled (or http if app uses http exporter)
            endpoint: 0.0.0.0:4317
          # http:
            # endpoint: 0.0.0.0:4318
      # REMOVE filelog receiver if present
      # filelog:
      #   include: [ /var/lib/docker/containers/*/*-json.log ]
      #   ... etc ...
      docker_stats: # Keep this for metrics
        # ...

    service:
      pipelines:
        logs:
          receivers: [otlp] # Ensure 'otlp' is listed, remove 'filelog' if present
          processors: [resourcedetection, resource, batch]
          exporters: [otlp, debug] # Keep exporters
    ```

---

### Step 5.7: Cleanup Docker Compose (`docker-compose.yml`)

*   **What:** Remove any volume mounts added specifically for the `filelog` receiver.
*   **Why:** The collector no longer needs direct access to host log files.
*   **When:** After application code changes are complete.
*   **How:** Edit `docker-compose.yml`:

    **Remove (if present in otel-collector service volumes):**
    ```diff
    services:
      otel-collector:
        # ... other config ...
        volumes:
          - ./otel-collector-config.yaml:/etc/otelcol-contrib/config.yaml
          - /var/run/docker.sock:/var/run/docker.sock:ro
    -     - /var/lib/docker/containers:/var/lib/docker/containers:ro # REMOVE THIS LINE
    ```

---

### Step 5.8: Remove Zap Dependency

*   **What:** Remove the `go.uber.org/zap` dependency from the modules.
*   **Why:** It's no longer used and should be removed to keep dependencies clean.
*   **When:** Final step, after verifying the application runs correctly with `slog`.
*   **How:** Execute `go mod tidy` in both `common` and `product-service` directories. This should automatically detect that `zap` is no longer imported and remove it from `go.mod` and `go.sum`.

    ```bash
    cd common && go mod tidy && cd ..
    cd product-service && go mod tidy && cd ..
    ```
    Verify `go.uber.org/zap` is gone from the `require` sections of both `go.mod` files.

---

## 6. Testing Strategy (Revised)

1.  **Build & Run (Both Environments):**
    *   **Development:** Execute `docker-compose up -d --build` (assuming default compose file uses a non-production `ENVIRONMENT` variable or none is set). Check `docker logs signoz_assignment-product-service-1`. Verify logs appear *only* in console format (JSON/Text) with `slog` structure. Verify no OTLP export attempts in collector logs (`docker logs otel-collector`). Check OTLP backend (SigNoz) - no logs, traces, or metrics should arrive from this run.
    *   **Production:** Modify `docker-compose.yml` to set `ENVIRONMENT=production` for the `product-service`. Execute `docker-compose down && docker-compose up -d --build`. Check `docker logs signoz_assignment-product-service-1`. Verify *no* logs appear in the console output (or only minimal bootstrap messages before logger init). Check collector logs (`docker logs otel-collector`) for OTLP log reception. Check OTLP backend (SigNoz) - logs, traces, and metrics should arrive.
2.  **Trace Correlation (Production Only):** Generate requests. Find the corresponding trace in SigNoz. Verify that log records associated with that request contain the correct `trace_id` and `span_id` attributes.

## 7. Technical Risks and Mitigations (Revised)

*   **Initialization Order:** Calling `commonlog.Init` before `telemetry.InitTelemetry` breaks the OTel integration *in production*. **Mitigation:** Strict code review of `main.go` order, add clear comments emphasizing the dependency, potentially add runtime checks if feasible.
*   **`otelslog` Bridge Behavior:** Ensure `otelslog.NewHandler()` correctly picks up the global provider *when set in production*. **Mitigation:** Test production environment thoroughly; consult `otelslog` docs if issues occur.
*   **Environment Configuration:** Incorrect `ENVIRONMENT` variable setting leads to wrong logging/telemetry behavior. **Mitigation:** Clear documentation on environment variable usage, robust configuration loading in `common/config`.
*   **Migration Errors:** Missing `zap` to `slog` syntax conversions. **Mitigation:** Careful code review, utilize compiler errors, test application functionality in both environments.

</rewritten_file> 