# Configuration Management

**Purpose:** This page explains how application configuration is managed, loaded, and what settings are available.
**Audience:** Developers, DevOps, Students
**Prerequisites:** Basic understanding of Go structs and environment variables.
**Related Pages:** [[common/config/config.go]], [[Running Locally with Docker Compose]], [[Telemetry Setup]], [[docker-compose.yml]], [[common/globals/globals.go]]

---

## 1. Overview & Key Concepts

Configuration for services is managed via the `common/config` package, influenced by environment variables and potentially the `common/globals` initialization.

*   **Key Concept: Layered Configuration:** Configuration values often come from multiple sources: hardcoded defaults, environment variables, and potentially explicit overrides.
*   **Key Concept: Environment-Specific Settings:** The configuration logic provides different settings based on the runtime environment (e.g., "production" vs. "development"), particularly for external service endpoints like the OTel Collector.
*   **Key Concept: Centralized Defaults:** Common default values are defined centrally in `common/config`, reducing redundancy.
*   **Core Responsibility:** Provide a structured way to access configuration values needed by different parts of the application (e.g., server port, log level, OTel endpoint, data file paths).
*   **Why it Matters:** Proper configuration management allows deploying the same application code in different environments (dev, staging, prod) with appropriate settings without code changes.

---

## 2. Configuration & Setup

Configuration is defined by the `config.Config` struct and loaded primarily using logic within the `common/config` package, often triggered by `common/globals`.

**Relevant Files:**
*   `common/config/config.go`: Defines the `Config` struct and loading functions (`commonConfig`, `LoadConfig`).
*   `common/globals/globals.go`: Provides `Init()` which calls `config.LoadConfig` (potentially hardcoding environment) and accessors (`Cfg()`).
*   `docker-compose.yml`: Sets environment variables for services at runtime.

**`Config` Struct Definition (`common/config/config.go`):**
```go
package config

type Config struct {
    ENVIRONMENT          string
    PRODUCT_SERVICE_PORT string
    LOG_LEVEL            string
    OTEL_ENDPOINT        string
    PRODUCT_DATA_FILE_PATH string
    SimulateDelayEnabled bool
    SimulateDelayMinMs   int
    SimulateDelayMaxMs   int
    // Note: Original struct had `env`/`mapstructure` tags,
    // but current LoadConfig doesn't use them directly.
    // Loading relies on defaults + environment override in LoadConfig.
}
```

**Loading Logic:**
1.  **Defaults (`commonConfig()`):** Sets initial default values (e.g., `LOG_LEVEL="debug"`, `PRODUCT_DATA_FILE_PATH="/app/data.json"`, `OTEL_ENDPOINT="localhost:4317"`).
2.  **`LoadConfig(env string)`:**
    *   Takes an `env` string parameter.
    *   Starts with defaults from `commonConfig()`.
    *   Sets the `ENVIRONMENT` field based on the `env` parameter.
    *   **Overrides `OTEL_ENDPOINT`:** If `env == "production"`, sets it to `"otel-collector:4317"` (for Docker networking). Otherwise, keeps the default (likely `"localhost:4317"`).
3.  **`globals.Init()`:**
    *   Calls `config.LoadConfig("production")`, **hardcoding the environment string**. This means the configuration loaded via `globals.Init()` will *always* use the production OTel endpoint setting, regardless of external `ENVIRONMENT` variables.
    *   Stores the loaded config internally, accessible via `globals.Cfg()`.

**Key Configuration Parameters & Effective Values (when using `globals.Init()`):**
*   `ENVIRONMENT`: Always `"production"` within the loaded Go config struct.
*   `PRODUCT_SERVICE_PORT`: `"8082"` (from default)
*   `LOG_LEVEL`: `"debug"` (from default) - **Note:** `log.Init` might still default to `info` if it receives this and considers it invalid or prefers its own default.
*   `OTEL_ENDPOINT`: Always `"otel-collector:4317"` (overridden due to hardcoded "production" env).
*   `PRODUCT_DATA_FILE_PATH`: `"/app/data.json"` (from default)
*   `SimulateDelayEnabled`: `false` (from default)
*   `SimulateDelayMinMs`: `10` (from default)
*   `SimulateDelayMaxMs`: `10000` (from default)

**Environment Variables set in `docker-compose.yml`:**
*   `ENVIRONMENT=production` (or `development`): **This primarily affects the OTel SDK's own environment detection** for resource attributes and potentially other libraries, but **does not override** the hardcoded "production" used *within* `globals.Init -> config.LoadConfig` for setting the OTel endpoint in the Go config struct.
*   `LOG_LEVEL=debug` (or other): **This is likely ignored** by `log.Init` if `globals.Init` is called first, as `log.Init` receives the hardcoded "debug" from the config defaults loaded by `globals.Init`. If `log.Init` were called *manually* after reading this env var, it might take effect.
*   `OTEL_EXPORTER_OTLP_ENDPOINT=...`: **This is likely ignored** by the Go SDK's OTLP exporter setup if `globals.Init` is used, as the endpoint is taken from the `Config` struct which was loaded with the hardcoded "production" setting.
*   `OTEL_SERVICE_NAME=...`, `OTEL_RESOURCE_ATTRIBUTES=...`: These standard OTel env vars **are read** by the OTel SDK's resource detectors (`otelemetryResource.NewResource`) independently of the custom Go config loading. They correctly set the service name and other resource attributes.

---

## 3. Implementation Details & Usage

The standard pattern in `product-service/src/main.go` is:
1.  Call `globals.Init()` early.
2.  Access the loaded configuration via `globals.Cfg()`.
3.  Pass relevant config values (like `globals.Cfg().LOG_LEVEL`) to other initialization functions (like `log.Init`).

```go
// Simplified main.go
import (
    "github.com/narender/common/globals"
    "github.com/narender/common/log"
    // ...
)

func main() {
    // Initializes config (hardcoding "production" internally),
    // logging (using level from loaded config), and telemetry.
    globals.Init()

    // Access config via global accessor
    cfg := globals.Cfg()

    // Example: Use port from config
    addr := ":" + cfg.PRODUCT_SERVICE_PORT
    log.L.Info("Starting server", slog.String("address", addr))
    // ... fiber app setup using addr ...
}
```

---

## 4. Monitoring & Observability Integration

Configuration values directly impact observability:
*   `ENVIRONMENT` (Effectively hardcoded to "production" by `globals.Init`): Ensures OTLP exporters are always configured in Go code.
*   `LOG_LEVEL` (Effectively hardcoded to "debug" from defaults via `globals.Init`): Sets the logging level used by `log.Init`.
*   `OTEL_ENDPOINT` (Effectively hardcoded to "otel-collector:4317" via `globals.Init`): Tells the Go SDK OTLP exporters where to send data.
*   `OTEL_SERVICE_NAME` / `OTEL_RESOURCE_ATTRIBUTES` (Read from actual environment): Correctly define the service identity in SigNoz.

---

## 5. Visuals & Diagrams

```mermaid
graph TD
    subgraph Inputs
        Defaults[Code Defaults (config.go)]
        EnvVars[Environment Variables (docker-compose.yml)]
    end

    subgraph Loading Process
        GlobalsInit[globals.Init()]
        LoadConfig[config.LoadConfig("production")] -- Hardcoded "production" --> EnvProcessing{Overrides OTel Endpoint}
        EnvProcessing -- Uses --> Defaults
        GlobalsInit --> LoadConfig
    end

    subgraph OTel SDK Init
        OTelResource[OTel Resource Detector] -- Reads --> EnvVars
        GoExporterSetup[Go OTLP Exporter Setup] -- Reads Endpoint From --> LoadedConfig
    end

    subgraph Output
        LoadedConfig[Loaded config.Config Struct]
        OTelResourceAttrs[OTel Resource Attributes]
        FinalOTelEndpoint[Effective OTel Endpoint for Go SDK]
        FinalLogLevel[Effective Log Level]
    end

    LoadConfig --> LoadedConfig
    GlobalsInit -- Stores --> LoadedConfig

    EnvVars -- Sets --> OTelResourceAttrs
    LoadedConfig -- Provides --> FinalOTelEndpoint
    LoadedConfig -- Provides --> FinalLogLevel

    style GlobalsInit fill:#f9f,stroke:#333,stroke-width:2px
```
*Fig 1: Configuration Loading Flow (Highlighting `globals.Init` impact).*

---

## 6. Teaching Points & Demo Walkthrough

*   **Key Takeaway:** Configuration comes from defaults in Go code, overridden by logic within `config.LoadConfig`. Crucially, the `globals.Init()` function currently *hardcodes* the environment passed to `LoadConfig` as "production", meaning settings like `OTEL_ENDPOINT` in the Go struct are always set to the production value when using `globals.Init`. Standard OTel environment variables like `OTEL_SERVICE_NAME` are still read directly by the OTel SDK for resource attributes.
*   **Demo Steps:**
    1.  Show `common/config/config.go` (defaults, `LoadConfig`).
    2.  Show `common/globals/globals.go` (`Init` function calling `LoadConfig("production")`).
    3.  Show `docker-compose.yml` `environment:` section.
    4.  Explain which settings in the Go code are affected by `globals.Init` (OTel endpoint, log level from defaults) vs. which are read from the environment by the OTel SDK (service name, resource attributes).
    5.  Run the stack and verify the `service.name` in SigNoz matches `OTEL_SERVICE_NAME` from `docker-compose.yml`.
    6.  Show that changing `LOG_LEVEL` or `OTEL_EXPORTER_OTLP_ENDPOINT` in `docker-compose.yml` likely has no effect due to the `globals.Init` behavior.
*   **Common Pitfalls / Questions:**
    *   Why doesn't changing `LOG_LEVEL` in `docker-compose.yml` work? (Because `globals.Init` loads the config with hardcoded "production" env, using the default "debug" level from `commonConfig`, and passes *that* level to `log.Init`).
    *   How to run in a true development mode (e.g., with OTel endpoint `localhost:4317`)? (Avoid calling `globals.Init`. Manually call `config.LoadConfig("development")`, `log.Init` with desired level, and `telemetry.InitTelemetry` with the dev config).
*   **Simplification Analogy:** The `Config` struct is a settings form. `commonConfig` fills in defaults. `LoadConfig` changes the OTel endpoint *if* told the environment is "production". `globals.Init` *always* tells `LoadConfig` the environment is "production", gets the form back, and stores it. Later, things like the logger read the level off this stored form. Separately, the OTel SDK looks directly at environment variables (`OTEL_SERVICE_NAME`) for some specific settings.

---

**Last Updated:** [Current Date]
