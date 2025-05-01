# OpenTelemetry Integration Checklist for Go Microservice

This checklist outlines the steps required to integrate OpenTelemetry into the Go microservice, based on the detailed plan in [gemini_analysis.md](gemini_analysis.md).

## Phase 1: Setup & Dependencies

### Step 1: Create the `telemetry` Package Structure
*   [x] Create `common/telemetry` directory. ([Details](gemini_analysis.md#step-1-create-the-telemetry-package-structure))
*   [x] Create `common/telemetry/init.go`. ([Details](gemini_analysis.md#step-1-create-the-telemetry-package-structure))
*   [x] Create `common/telemetry/config.go`. ([Details](gemini_analysis.md#step-1-create-the-telemetry-package-structure))
*   [x] Create `common/telemetry/resource.go`. ([Details](gemini_analysis.md#step-1-create-the-telemetry-package-structure))
*   [x] Create `common/telemetry/trace.go`. ([Details](gemini_analysis.md#step-1-create-the-telemetry-package-structure))
*   [x] Create `common/telemetry/metric.go`. ([Details](gemini_analysis.md#step-1-create-the-telemetry-package-structure))
*   [x] Create `common/telemetry/log.go`. ([Details](gemini_analysis.md#step-1-create-the-telemetry-package-structure))

### Step 2: Add OTel Dependencies
*   [x] Add OTel Core & SDK dependencies (`go get go.opentelemetry.io/otel...`). ([Details](gemini_analysis.md#step-2-add-otel-dependencies))
*   [x] Add OTLP Exporter dependencies (`go get go.opentelemetry.io/otel/exporters/otlp...`). ([Details](gemini_analysis.md#step-2-add-otel-dependencies))
*   [x] Add Instrumentation library dependencies (`go get go.opentelemetry.io/contrib/instrumentation...`). ([Details](gemini_analysis.md#step-2-add-otel-dependencies))
*   [x] Add other necessary libraries (e.g., `grpc`). ([Details](gemini_analysis.md#step-2-add-otel-dependencies))
*   [x] Run `go mod tidy`. ([Details](gemini_analysis.md#step-2-add-otel-dependencies))

## Phase 2: Configuration & Core Components

### Step 3: Configure Telemetry Settings
*   [x] Update `common/config/config.go` with OTel environment variables and defaults (`OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_INSECURE`). ([Details](gemini_analysis.md#step-3-configure-telemetry-settings))
*   [x] Create `common/telemetry/config.go` to load telemetry-specific config. Ensure correct module path is used. ([Details](gemini_analysis.md#step-3-configure-telemetry-settings))

### Step 4: Create the OTel Resource
*   [x] Implement `newResource` function in `common/telemetry/resource.go` to define service attributes (name, version, environment) using `semconv`. ([Details](gemini_analysis.md#step-4-create-the-otel-resource))
*   [x] Include automatic detection attributes (`WithFromEnv`, `WithHost`, `WithProcess`, `WithProcessRuntimeDescription`). ([Details](gemini_analysis.md#step-4-create-the-otel-resource))
*   [x] Handle potential errors during resource creation and merging. ([Details](gemini_analysis.md#step-4-create-the-otel-resource))

### Step 5: Set Up Tracing
*   [x] Implement `initTracerProvider` in `common/telemetry/trace.go`. ([Details](gemini_analysis.md#step-5-set-up-tracing))
*   [x] Configure OTLP Trace Exporter (`otlptracegrpc`) with endpoint and security options (insecure/TLS). ([Details](gemini_analysis.md#step-5-set-up-tracing))
*   [x] Create and configure a Batch Span Processor (`sdktrace.NewBatchSpanProcessor`). ([Details](gemini_analysis.md#step-5-set-up-tracing))
*   [x] Create the Tracer Provider (`sdktrace.NewTracerProvider`) with resource and sampler (`AlwaysSample` or ratio-based). ([Details](gemini_analysis.md#step-5-set-up-tracing))
*   [x] Set the global Tracer Provider (`otel.SetTracerProvider`). ([Details](gemini_analysis.md#step-5-set-up-tracing))
*   [x] Set the global Text Map Propagator (`propagation.TraceContext`, `propagation.Baggage`). ([Details](gemini_analysis.md#step-5-set-up-tracing))
*   [x] Implement `GetTracer` helper function. ([Details](gemini_analysis.md#step-5-set-up-tracing))
*   [x] Return a shutdown function for the tracer provider. ([Details](gemini_analysis.md#step-5-set-up-tracing))

### Step 6: Set Up Metrics
*   [x] Implement `initMeterProvider` in `common/telemetry/metric.go`. ([Details](gemini_analysis.md#step-6-set-up-metrics))
*   [x] Configure OTLP Metric Exporter (`otlpmetricgrpc`) with endpoint and security options. ([Details](gemini_analysis.md#step-6-set-up-metrics))
*   [x] Create a Periodic Reader (`sdkmetric.NewPeriodicReader`) with a suitable interval. ([Details](gemini_analysis.md#step-6-set-up-metrics))
*   [x] Create the Meter Provider (`sdkmetric.NewMeterProvider`) with resource and reader. ([Details](gemini_analysis.md#step-6-set-up-metrics))
*   [x] Set the global Meter Provider (`otel.SetMeterProvider`). ([Details](gemini_analysis.md#step-6-set-up-metrics))
*   [x] Start host metrics collection (`host.Start`). ([Details](gemini_analysis.md#step-6-set-up-metrics))
*   [x] Start Go runtime metrics collection (`runtime.Start`). ([Details](gemini_analysis.md#step-6-set-up-metrics))
*   [x] Implement `GetMeter` helper function. ([Details](gemini_analysis.md#step-6-set-up-metrics))
*   [x] Return a shutdown function for the meter provider. ([Details](gemini_analysis.md#step-6-set-up-metrics))

### Step 7: Set Up Logging (Logrus Hook)
*   [x] Implement `initLoggerProvider` in `common/telemetry/log.go`. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Configure OTLP Log Exporter (`otlplogsgrpc`) with endpoint and security options. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Create a Batch Log Record Processor (`sdklog.NewBatchProcessor`). ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Create the Logger Provider (`sdklog.NewLoggerProvider`) with resource and processor. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Set the global Logger Provider (`global.SetLoggerProvider`). ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Obtain an OTel Logger instance (`otelLogger = loggerProvider.Logger(...)`). Ensure correct module path is used. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Return a shutdown function for the logger provider. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Implement the `OtelHook` struct for Logrus (`logrus.Hook`). ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Implement the `Levels()` method for the hook. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Implement the `Fire()` method for the hook:
    *   [x] Get context and trace information (`trace.SpanFromContext`). ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
    *   [x] Map Logrus level to OTel severity (`mapLogLevel`). ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
    *   [x] Create OTel Log Record (`otelLogs.Record`). ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
    *   [x] Set timestamp, severity, body. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
    *   [x] Add trace ID, span ID, trace flags if available. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
    *   [x] Add attributes from Logrus fields and caller info. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
    *   [x] Emit the record using the `otelLogger`. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Implement `mapLogLevel` helper function. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))
*   [x] Implement `ConfigureLogrus` function to add the `OtelHook` to the global Logrus instance. ([Details](gemini_analysis.md#step-7-set-up-logging-logrus-hook))

### Step 8: Initialize Telemetry Components
*   [x] Implement `InitTelemetry` in `common/telemetry/init.go`. ([Details](gemini_analysis.md#step-8-initialize-telemetry-components))
*   [x] Load configuration (`LoadConfig`). ([Details](gemini_analysis.md#step-8-initialize-telemetry-components))
*   [x] Create the resource (`newResource`). ([Details](gemini_analysis.md#step-8-initialize-telemetry-components))
*   [x] Call `initTracerProvider`, `initMeterProvider`, `initLoggerProvider` in order, collecting shutdown functions and handling errors. ([Details](gemini_analysis.md#step-8-initialize-telemetry-components))
*   [x] Call `ConfigureLogrus` *after* `initLoggerProvider` succeeds. ([Details](gemini_analysis.md#step-8-initialize-telemetry-components))
*   [x] Implement `createMasterShutdown` to execute all collected shutdown functions concurrently. ([Details](gemini_analysis.md#step-8-initialize-telemetry-components))
*   [x] Return the master shutdown function and any initialization error from `InitTelemetry`. ([Details](gemini_analysis.md#step-8-initialize-telemetry-components))

## Phase 3: Service Integration

### Step 9: Integrate Telemetry into `product-service`
*   [x] Update `product-service/src/main.go`:
    *   [x] Call `telemetry.InitTelemetry()` at the beginning of `main`. ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))
    *   [x] Defer the returned OTel shutdown function with appropriate timeout and context. ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))
    *   [x] Configure Logrus level and formatter *after* `InitTelemetry` call. ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))
    *   [x] Replace old logger usage with direct Logrus calls. ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))
    *   [x] Add Fiber OTel middleware (`otelgofiber.Middleware`) early in the middleware chain. ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))
    *   [x] Ensure graceful shutdown of Fiber server and OTel components. ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))
*   [x] Update `product-service/src/handler.go`, `product-service/src/service.go`, `product-service/src/repository.go`:
    *   [x] Modify function signatures to accept `context.Context` as the first argument. ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))
    *   [x] Pass the context down through layers (handler -> service -> repository). ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))
    *   [x] Replace old logger calls with `logrus.WithContext(ctx)...`. ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))
    *   [x] Add custom spans (`tracer.WithSpan`) around key operations (e.g., service calls, database interactions). Ensure correct module path is used for `GetTracer`. ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))
    *   [x] Add relevant attributes and record errors on custom spans. ([Details](gemini_analysis.md#step-9-integrate-telemetry-into-product-service))

### Step 10: Add Custom Metrics
*   [x] Define custom metric instruments (e.g., Counters like `app.product.lookups`, `app.product.stock_checks`) using `telemetry.GetMeter()`. Preferably define them once (e.g., in an `init` block or handler struct). Ensure correct module path is used for `GetMeter`. ([Details](gemini_analysis.md#step-10-add-custom-metrics))
*   [x] Handle errors during instrument creation. ([Details](gemini_analysis.md#step-10-add-custom-metrics))
*   [x] Record metrics within relevant code sections (e.g., handler methods) using the instrument's `Add` method. ([Details](gemini_analysis.md#step-10-add-custom-metrics))
*   [x] Pass the `requestCtx` to the metric recording method. ([Details](gemini_analysis.md#step-10-add-custom-metrics))
*   [x] Add relevant attributes (`metric.WithAttributeSet`) to provide dimensions (e.g., product ID, success status, error type). ([Details](gemini_analysis.md#step-10-add-custom-metrics))

## Phase 4: Verification

### Step 11: Running the Service
*   [ ] Set required environment variables (`OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_INSECURE`, `LOG_LEVEL`, etc.). ([Details](gemini_analysis.md#step-11-running-the-service))
*   [ ] Run the `product-service`. ([Details](gemini_analysis.md#step-11-running-the-service))
*   [ ] Run the test script (`simulate_product_service.py`) or interact with the API. ([Details](gemini_analysis.md#step-11-running-the-service))
*   [ ] Verify traces, metrics (host, runtime, custom), and logs (with trace correlation) appear in the OTel Collector backend (e.g., SigNoz, Jaeger). 