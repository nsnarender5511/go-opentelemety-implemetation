# Best Practices & Improvement Plan

This document outlines identified areas for improvement in the demo microservice project, based on a review focusing on best practices, particularly concerning OpenTelemetry, configuration, and general Go service design. Each section details the problem, analyzes the affected files, and provides a phased plan for implementation.

---

## 1. Inconsistent OpenTelemetry Resource Attribute Handling [COMPLETED]

**Problem:** Resource attributes (like `service.name`, `service.version`) are defined inconsistently. The `Makefile` uses `OTEL_RESOURCE_ATTRIBUTES` with a potentially incorrect service name (`dice`?) for local runs, while the `Dockerfile` uses `OTEL_SERVICE_NAME`. The `common/telemetry/resource.go` file only uses the `serviceName` parameter passed to it and doesn't explicitly include `service.version`.

**Context Exploration:**
-   `Makefile`: Sets `OTEL_RESOURCE_ATTRIBUTES="service.name=dice,service.version=0.1.0"` for `go run .`.
-   `product-service/Dockerfile`: Sets `OTEL_SERVICE_NAME=$(SERVICE_NAME)` (which is `product-service`).
-   `common/telemetry/resource.go`: Reads `serviceName` parameter, uses `resource.WithFromEnv()`, `resource.WithHost()`, `resource.WithProcess()`, `resource.WithProcessRuntimeDescription()`. It lacks explicit setting of `service.version` and relies on the caller (`common/telemetry/init.go`) to pass the correct `serviceName`.
-   `common/config/config.go`: Loads `OTEL_SERVICE_NAME` from env/defaults, but doesn't have a dedicated `SERVICE_VERSION` variable.
-   `common/telemetry/init.go`: Calls `newResource(initCtx, config.OTEL_SERVICE_NAME)`, passing the name loaded from config.

**Feature Plan: Standardize OTel Resource Definition**

**Goal:** Ensure consistent and comprehensive OpenTelemetry resource attributes are applied regardless of how the service is run (local Go, Docker). Centralize resource definition logic. [COMPLETED]

**Phase 1: Centralize Configuration**
1.  **Modify `common/config/config.go`:**
    *   Add a new default config entry: `"SERVICE_VERSION": "0.1.0"` (or a suitable default).
    *   Add a new exported variable: `var SERVICE_VERSION string`.
    *   Load `SERVICE_VERSION = viper.GetString("SERVICE_VERSION")` in the `init()` function.
    *   Add logging for `SERVICE_VERSION` in `init()`.
2.  **Modify `Makefile`:**
    *   Remove the `OTEL_RESOURCE_ATTRIBUTES` line entirely from the `run` target.
    *   Ensure the `run` target sets `SERVICE_NAME=product-service` and `SERVICE_VERSION=0.1.0` (or desired version) environment variables if not already set implicitly.
3.  **Modify `product-service/Dockerfile`:**
    *   Keep `ENV OTEL_SERVICE_NAME=$SERVICE_NAME` (using ARG/ENV for build-time variable).
    *   Add `ARG SERVICE_VERSION=0.1.0` (default version).
    *   Add `ENV SERVICE_VERSION=$SERVICE_VERSION`.
    *   Pass build args during build: `docker build --build-arg SERVICE_NAME=product-service --build-arg SERVICE_VERSION=0.2.0 ...`

**Phase 2: Enhance Resource Definition**
1.  **Modify `common/telemetry/resource.go`:**
    *   Update `newResource` function signature: Remove the `serviceName string` parameter. It will now get config directly.
    *   Inside `newResource`, access `config.SERVICE_NAME` and `config.SERVICE_VERSION` directly.
    *   Modify `resource.New` call:
        *   Replace `semconv.ServiceName(serviceName)` with `semconv.ServiceName(config.SERVICE_NAME)`.
        *   Add `semconv.ServiceVersion(config.SERVICE_VERSION)`.
        *   Consider adding `semconv.DeploymentEnvironment` based on another config variable (e.g., `ENVIRONMENT`, defaulting to "development").
    *   Update the logging inside `newResource` to reflect it's using `config.SERVICE_NAME`.
2.  **Modify `common/telemetry/init.go`:**
    *   Update the call to `newResource`: Change `newResource(initCtx, config.OTEL_SERVICE_NAME)` to just `newResource(initCtx)`.

**Testing:** Verify telemetry backend (e.g., SigNoz) shows consistent `service.name` and `service.version` attributes for services run via `make run` and `docker run`.

---

## 2. Configuration Loading Obscurity

**Problem:** While `common/config/config.go` uses Viper and loads from environment variables and defaults, the precedence rules aren't explicitly documented. It also doesn't support configuration files, which can be useful.

**Context Exploration:**
-   `common/config/config.go`: Uses `viper.AutomaticEnv()` and `viper.SetDefault()`. No file loading mentioned. Exports package-level variables.
-   `Makefile`, `product-service/Dockerfile`: Set environment variables, implicitly taking highest precedence over defaults.

**Feature Plan: Clarify and Enhance Configuration Loading**

**Goal:** Make configuration loading more transparent and flexible by adding optional file support and documenting precedence.

**Phase 1: Documentation and File Support**
1.  **Modify `common/config/config.go`:**
    *   Add package comments explaining the configuration loading mechanism:
        ```go
        // Package config handles loading application configuration.
        // It uses Viper to load settings with the following precedence:
        // 1. Environment Variables (e.g., PRODUCT_SERVICE_PORT)
        // 2. Configuration file (`config.yaml` or `config.json` in CWD or /etc/app/) - Added in Phase 1
        // 3. Default values set in the code.
        package config 
        ```
    *   In the `init()` function, before loading variables:
        ```go
        // Optional: Add configuration file paths
        viper.SetConfigName("config") // Name of config file (without extension)
        viper.SetConfigType("yaml")  // REQUIRED if the config file does not have the extension in the name
        viper.AddConfigPath("/etc/app/") // Path to look for the config file in
        viper.AddConfigPath(".")        // Optionally look for config in the working directory
        
        err := viper.ReadInConfig() // Find and read the config file
        if err != nil { // Handle errors reading the config file
            if _, ok := err.(viper.ConfigFileNotFoundError); ok {
                // Config file not found; ignore error if desired
                log.Println("No config file found, using defaults and environment variables.")
            } else {
                // Config file was found but another error was produced
                log.Printf("Error reading config file: %s", err)
            }
        } else {
             log.Printf("Using config file: %s", viper.ConfigFileUsed())
        }

        viper.AutomaticEnv() // Env vars override file & defaults
        ```
2.  **(Optional) Create `config.yaml.example`:**
    *   Add an example config file in the root or `product-service` directory showing the structure.

**Testing:** Verify that settings from environment variables override defaults. If a `config.yaml` is created, verify its settings override defaults but are overridden by environment variables. Check logs for confirmation of which config source is used.

---

## 4. Dockerfile `host.docker.internal` Hack

**Problem:** The OTLP endpoint in `Makefile` (`host.docker.internal:4317`) and the default in `common/config/config.go` (`localhost:4317`) rely on Docker Desktop's DNS resolution or assume the collector runs on the host. This isn't portable, especially the `host.docker.internal` part. The switch from `distroless` to `alpine` in the Dockerfile might be related to DNS issues.

**Context Exploration:**
-   `product-service/Dockerfile`: Mentions potential issues with `host.docker.internal` in `distroless`. Uses `alpine`. Sets `OTEL_EXPORTER_OTLP_ENDPOINT=host.docker.internal:4317` via env var in `Makefile`'s `run` target.
-   `common/config/config.go`: Defaults `OTEL_EXPORTER_OTLP_ENDPOINT` to `localhost:4317`.
-   `common/telemetry/init.go`: Passes `config.OTEL_EXPORTER_OTLP_ENDPOINT` to trace, metric, and log provider initializers.
-   `common/telemetry/trace.go`, `metric.go`, `log.go`: (Code not shown) Presumably use this endpoint string when creating OTLP gRPC exporters.

**Feature Plan: Improve OTLP Endpoint Configuration Robustness**

**Goal:** Make the OTLP endpoint configuration more standard and less reliant on Docker Desktop specifics. Allow easier configuration for different environments (local Docker, Kubernetes, etc.).

**Phase 1: Use Standard Docker Networking**
1.  **Modify `Makefile` (`run` target):**
    *   Instead of relying on `host.docker.internal`, use Docker networking.
    *   Add a Docker network: `docker network create signoz-net || true` (run before starting containers).
    *   Run the `product-service` container on this network: Add `--network signoz-net`.
    *   Change the OTLP endpoint env var: `-e OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317` (assuming the SigNoz collector service will be named `otel-collector` in Docker).
    *   Ensure the SigNoz container (when run, e.g., via compose from Problem 5) is also on `signoz-net` and named `otel-collector`.
2.  **Modify `common/config/config.go`:**
    *   Change the default `OTEL_EXPORTER_OTLP_ENDPOINT` to `otel-collector:4317`. Add a comment explaining this assumes Docker networking and a service named `otel-collector`.
3.  **Modify `product-service/Dockerfile`:**
    *   **(Optional but recommended)** Try switching back to `gcr.io/distroless/static-debian11` or similar `distroless/base`. With standard DNS resolution within a Docker network, the previous issues might be resolved. Test thoroughly.

**Testing:** Run the service via `make run` (after implementing the simplified SigNoz setup from Problem 5). Verify telemetry data reaches the SigNoz collector running as `otel-collector` on the same Docker network.

---

## 6. `data.json` Loading Hardcoded Path

**Problem:** The path to the data file (`data.json`) is hardcoded as a constant (`../data.json`) in `product-service/src/repository.go`. This relies on the binary being run from a specific location relative to the data file (which works currently due to `WORKDIR /app/product-service` and `COPY ... ./data.json` in Dockerfile, but it's brittle).

**Context Exploration:**
-   `product-service/src/repository.go`: Defines `const dataFilePath = "../data.json"` and uses it in `NewProductRepository` and `readData`.
-   `product-service/Dockerfile`: Copies `data.json` to `./data.json` relative to `WORKDIR /app`. The binary is run from `/app`. The build happens in `/app/product-service`, making the relative path `../data.json` work *during build* but likely incorrect at *runtime* from `/app`. **Correction**: The binary is built to `/app/product-service-binary`. `WORKDIR /app` is set in the final stage. `COPY --from=builder /app/product-service/data.json ./data.json` copies it to `/app/data.json`. `ENTRYPOINT ["/app/product-service-binary"]`. The hardcoded path `../data.json` relative to the binary's location `/app/product-service-binary` is incorrect. It should likely be `./data.json` relative to the `/app` workdir. The current hardcoding is fragile and potentially wrong.
-   `common/config/config.go`: Does not currently have a config variable for the data path.

**Feature Plan: Make Data File Path Configurable**

**Goal:** Remove the hardcoded data file path and load it from configuration for better flexibility and robustness.

**Phase 1: Configuration**
1.  **Modify `common/config/config.go`:**
    *   Add a new default config entry: `"DATA_FILE_PATH": "./data.json"` (relative to workdir `/app`).
    *   Add a new exported variable: `var DATA_FILE_PATH string`.
    *   Load `DATA_FILE_PATH = viper.GetString("DATA_FILE_PATH")` in `init()`.
    *   Add logging for `DATA_FILE_PATH`.
2.  **Modify `product-service/Dockerfile`:**
    *   Ensure the `COPY --from=builder /app/product-service/data.json ./data.json` command remains, placing the file at `/app/data.json`.
    *   Add an environment variable: `ENV DATA_FILE_PATH=./data.json` (matches the default, can be overridden at runtime if needed).

**Phase 2: Update Repository**
1.  **Modify `product-service/src/repository.go`:**
    *   Remove the constant: `const dataFilePath = "../data.json"`.
    *   Update the `productRepository` struct: Change `filePath string` to hold the configured path.
    *   Update `NewProductRepository`:
        *   Accept the path from config: `r := &productRepository{filePath: config.DATA_FILE_PATH}`.
        *   Update logging/error messages to use `r.filePath`.
    *   Update `readData`: Use `r.filePath` wherever `dataFilePath` was used. Update span attributes (`telemetry.DBFilePathKey.String(r.filePath)`).

**Testing:** Run the service via `make run`. Verify it starts correctly and can read product data. Check logs for confirmation of the data path being used. Optionally, override `DATA_FILE_PATH` via env var to test flexibility.

---


## 8. Missing Health Check Endpoint

**Problem:** The service lacks standard `/health` or `/readyz` endpoints, making it difficult for orchestration systems (like Kubernetes) or load balancers to determine its status.

**Context Exploration:**
-   `product-service/src/main.go`: Sets up Fiber routes but doesn't include health checks.
-   `product-service/src/handler.go`: Contains product-related handlers only.

**Feature Plan: Add Standard Health Check Endpoint**

**Goal:** Implement basic liveness (`/healthz`) and readiness (`/readyz`) endpoints.

**Phase 1: Implement Endpoints**
1.  **Modify `product-service/src/handler.go`:**
    *   Add a new handler function for liveness:
      ```go
      // HealthLiveness handles GET /healthz
      func (h *ProductHandler) HealthLiveness(c *fiber.Ctx) error {
          // Basic liveness check - service is running
          return c.Status(http.StatusOK).JSON(fiber.Map{"status": "UP"})
      }
      ```
    *   Add a new handler function for readiness:
      ```go
      // HealthReadiness handles GET /readyz
      func (h *ProductHandler) HealthReadiness(c *fiber.Ctx) error {
          ctx := c.UserContext()
          log := logrus.WithContext(ctx)
          // TODO: Implement actual readiness checks if needed
          // - Check database connection (if applicable)
          // - Check connection to downstream services (if applicable)
          // For this demo with only a file, basic check is okay.
          _, err := h.service.GetAll(ctx) // Example: Try a basic read operation
          if err != nil {
               log.WithError(err).Error("Readiness check failed: Error performing basic service operation")
               // Don't record span errors for health checks unless desired
               return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
                    "status": "DOWN",
                    "reason": "Failed basic service operation check",
               })
          }
          return c.Status(http.StatusOK).JSON(fiber.Map{"status": "READY"})
      }
      ```
2.  **Modify `product-service/src/main.go`:**
    *   Inside `registerHooksAndStartServer` (or wherever routes are defined if not using `fx`):
        *   Register the health check routes *before* the main API group, and ideally *without* the `otelfiber` middleware if you don't want health checks traced/metered by default.
        ```go
         // --- Health Checks (Registered on root, potentially without OTel middleware) ---
         healthApp := fiber.New() // Separate Fiber instance for health checks? Or register on root app.
         healthApp.Get("/healthz", productHandler.HealthLiveness) // Use handler instance
         healthApp.Get("/readyz", productHandler.HealthReadiness) // Use handler instance
         app.Mount("/internal", healthApp) // Mount health checks under /internal path? Or root?
         // OR register directly on app if no separate middleware needed:
         // app.Get("/healthz", productHandler.HealthLiveness) 
         // app.Get("/readyz", productHandler.HealthReadiness)

         // --- OTel Middleware for main API ---
         app.Use(otelfiber.Middleware()) // Apply middleware *after* health checks if they should skip it

         // --- Main API Routes ---
         api := app.Group(APIPathBase)
         // ... rest of API routes
        ```

**Testing:** Start the service. Use `curl` or a browser to access `/healthz` and `/readyz` (or `/internal/healthz`, `/internal/readyz` depending on registration). Verify they return HTTP 200 OK with the expected JSON payload when the service is running correctly. Test the readiness check failure case if possible (e.g., by temporarily making `data.json` unreadable).

---


## 10. Error Handling Granularity

**Problem:** The Fiber error handler in `product-service/src/main.go` handles specific custom errors (`ValidationError`, `DatabaseError`, `ErrProductNotFound`) well, but might fall back to a generic 500 Internal Server Error for other potential issues without specific logging or span status codes. The `common/errors/errors.go` file defines several sentinel errors but they aren't all handled in the Fiber handler.

**Context Exploration:**
-   `product-service/src/main.go`: Contains the Fiber `ErrorHandler`. It checks for `ValidationError`, `DatabaseError`, `ErrProductNotFound`. Other errors result in a default 500, logging the error, and setting span status to `codes.Error`.
-   `common/errors/errors.go`: Defines `ErrNotFound`, `ErrProductNotFound`, `ErrUserNotFound`, `ErrCartNotFound`, `ErrOrderNotFound`, `ErrDatabaseOperation`, `ErrServiceCallFailed`. Also defines `ValidationError` and `DatabaseError` types. Contains a commented-out `HandleServiceError` function which suggests more extensive mapping was previously considered.
-   `product-service/src/repository.go`: Returns `errors.ErrProductNotFound` and wraps file I/O or JSON errors in `&errors.DatabaseError{}`.

**Feature Plan: Enhance Error Handling Specificity**

**Goal:** Improve error handling to provide more specific feedback (HTTP status codes, log messages, OTel span status) for a wider range of potential errors, leveraging the definitions in `common/errors`.

**Phase 1: Enhance Fiber Error Handler**
1.  **Modify `product-service/src/main.go` (`fiberErrorHandler` function):**
    *   Expand the `switch` or `if/else if` block to handle more errors defined in `common/errors`.
    *   Use `errors.Is` for sentinel errors and `errors.As` for typed errors.
    *   Map errors like `ErrServiceCallFailed` (if this service were to call others) to appropriate HTTP codes (e.g., 502 Bad Gateway or 503 Service Unavailable).
    *   Consider a fallback check for the generic `commonErrors.ErrNotFound`.
    *   Set more specific OTel span status codes where appropriate (e.g., `codes.NotFound`, `codes.InvalidArgument`, `codes.Unavailable`) instead of just `codes.Error` for all non-2xx cases.

    ```go
    // Inside fiberErrorHandler function:
    // ... (existing code) ...

    var validationErr *commonErrors.ValidationError
    var dbErr *commonErrors.DatabaseError
    // Add other expected custom error types here

    if errors.As(err, &validationErr) {
        code = http.StatusBadRequest
        httpErrMessage = validationErr.Error()
        span.SetStatus(codes.InvalidArgument, httpErrMessage) // More specific code
    } else if errors.Is(err, commonErrors.ErrProductNotFound) {
        code = http.StatusNotFound
        httpErrMessage = commonErrors.ErrProductNotFound.Error()
        span.SetStatus(codes.NotFound, httpErrMessage) // More specific code
    } else if errors.As(err, &dbErr) {
        code = http.StatusInternalServerError // Keep 500 for DB errors
        httpErrMessage = "An internal database error occurred"
        logEntry.Errorf("Database error during operation: %s - %+v", dbErr.Operation, dbErr.Err) // Log underlying
        span.SetStatus(codes.Internal, httpErrMessage) // More specific code? Or keep Error?
    } else if errors.Is(err, commonErrors.ErrServiceCallFailed) { // Example
         code = http.StatusBadGateway // Or 503?
         httpErrMessage = "Error communicating with a downstream service"
         logEntry.Errorf("Downstream service call failed: %+v", err)
         span.SetStatus(codes.Unavailable, httpErrMessage) // More specific code
    } else if errors.Is(err, commonErrors.ErrNotFound) { // Generic fallback
         code = http.StatusNotFound
         httpErrMessage = commonErrors.ErrNotFound.Error()
         span.SetStatus(codes.NotFound, httpErrMessage)
    } else {
        // Default case for unhandled errors
        code = http.StatusInternalServerError
        httpErrMessage = "An unexpected internal server error occurred"
        logEntry.Errorf("Unhandled internal server error: %+v", err)
        span.SetStatus(codes.Unknown, httpErrMessage) // Or codes.Internal
    }

    // Log the original error (already done)
    // Record error on span (already done)
    
    // Set span status (now done within the if/else block with more specific codes)
    // span.SetStatus(codes.Error, httpErrMessage) // REMOVE this generic one

    // Return response (already done)
    // ...
    ```

**Phase 2: Refine Error Creation**
1.  **Review `product-service/src/service.go` and `repository.go`:**
    *   Ensure that errors returned are specific enough (either sentinel errors from `common/errors` or wrapped standard errors) for the handler to map them correctly. Avoid returning generic `fmt.Errorf` where a more specific type or sentinel applies.

**Testing:** Introduce specific error conditions (e.g., simulate a DB error, return `ErrServiceCallFailed` from the service layer if applicable) and verify that the correct HTTP status code, response body, log message, and OTel span status are generated.

---

This concludes the analysis and planning phase. The next step would be to implement these changes, potentially using the `@implementer` agent. 