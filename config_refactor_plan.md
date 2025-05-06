# ✨ Feature Plan: Unified Configuration Refactoring ✨

**Goal:** To overhaul and unify the configuration management for both the Go application (`product-service` via `common` module) and the OpenTelemetry Collector. This involves migrating the Go application to an environment-variable-first approach with `.env` support, eliminating hardcoded values, and making specific OTEL Collector settings configurable via environment variables passed through Docker Compose.

**Libraries:** `github.com/joho/godotenv`, `github.com/caarlos0/env/v10`

---

## Affected Components

*   `common/config/config.go`
*   `common/globals/config.go` (or wherever `Init` is defined)
*   `common/go.mod`
*   `product-service/Dockerfile`
*   `docker-compose.yml`
*   `otel-collector-config.yaml`
*   New File: `.env.example` (at project root)

---

## 📋 Detailed Implementation Plan

**Phase 1: Go Application Configuration (`common` & `product-service`)**

1.  **Add Dependencies (common module):**
    *   **Where:** `common/go.mod`
    *   **What:** Add the necessary libraries for `.env` loading and environment variable parsing.
    *   **How:** Run `go get github.com/joho/godotenv github.com/caarlos0/env/v10` in the `common` directory.
    *   **Why:** To enable the new configuration loading mechanism.

2.  **Update `Config` Struct Definition:**
    *   **Where:** `common/config/config.go`
    *   **What:** Define the definitive structure for all application configuration. Replace `mapstructure` tags with `env` and `envDefault` tags. Ensure all required configuration points (ports, log levels, file paths, OTEL settings, simulation flags) are represented. Remove the environment-specific getter functions (`GetProductionConfig`, `GetDevelopmentConfig`, `commonConfig`).
    *   **How:**
        ```go
        package config

        // Config defines the application configuration structure.
        type Config struct {
            // Core App Settings
            ENVIRONMENT          string `env:"ENVIRONMENT,required" envDefault:"development"`
            PRODUCT_SERVICE_PORT string `env:"PRODUCT_SERVICE_PORT,required" envDefault:"8082"`
            LOG_LEVEL            string `env:"LOG_LEVEL" envDefault:"info"`
            PRODUCT_DATA_FILE_PATH string `env:"PRODUCT_DATA_FILE_PATH,required" envDefault:"./product-service/data.json"` // Sensible default for local dev

            // Telemetry Settings
            OTEL_ENDPOINT        string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required" envDefault:"localhost:4317"` // Default for local dev
            OTEL_SERVICE_NAME      string `env:"OTEL_SERVICE_NAME" envDefault:"product-service"`
            OTEL_RESOURCE_ATTRIBUTES string `env:"OTEL_RESOURCE_ATTRIBUTES" envDefault:"deployment.environment=development,service.version=0.1.0-local"`

            // Debug/Simulation Settings
            SimulateDelayEnabled bool `env:"SIMULATE_DELAY_ENABLED" envDefault:"false"` // Default to false
            SimulateDelayMinMs   int  `env:"SIMULATE_DELAY_MIN_MS" envDefault:"10"`
            SimulateDelayMaxMs   int  `env:"SIMULATE_DELAY_MAX_MS" envDefault:"100"`
        }

        // Remove GetProductionConfig, GetDevelopmentConfig, commonConfig functions
        ```
    *   **Why:** Centralizes config definition, uses declarative tags for loading, removes hardcoded defaults from functions.

3.  **Refactor Configuration Loading Logic:**
    *   **Where:** `common/globals/config.go` (or wherever `Init` is defined).
    *   **What:** Streamline the configuration loading process within the `Init` function. Use `godotenv` to load `.env` (ignoring errors) and then `env.Parse` to populate the `Config` struct from environment variables. Ensure the loaded `cfg` is stored globally (e.g., in the `globals` package). Remove the old `LoadConfig` function and its logic based on the `ENVIRONMENT` variable.
    *   **How:**
        ```go
        package globals

        import (
            "fmt"
            "log" // Use initial log before slog is ready
            "sync"

            "github.com/caarlos0/env/v10"
            "github.com/joho/godotenv"
            "github.com/narender/common/config"
            // ... other imports for logger/telemetry
        )

        var (
            cfg  *config.Config
            // logger *slog.Logger // Keep logger initialization
            // tracerProvider *sdktrace.TracerProvider // Keep telemetry initialization
            // meterProvider *sdkmetric.MeterProvider // Keep telemetry initialization
            once sync.Once
        )

        // Init loads configuration and sets up other globals
        func Init() error {
            var initErr error
            once.Do(func() {
                // 1. Load .env file (ignore file not found)
                err := godotenv.Load() // Loads .env from current or parent directories
                if err != nil {
                    log.Println("Info: .env file not found, loading config from environment variables.")
                }

                // 2. Parse environment variables into the Config struct
                currentCfg := &config.Config{} // Create instance
                if err := env.Parse(currentCfg); err != nil {
                    log.Printf("FATAL: Failed to parse configuration from environment: %+v\n", err)
                    initErr = fmt.Errorf("failed to parse configuration: %w", err)
                    return // Exit Do func
                }
                cfg = currentCfg // Assign to global var

                // 3. Initialize Logger (using cfg.LOG_LEVEL)
                if err := initLogger(cfg.LOG_LEVEL); err != nil {
                     log.Printf("FATAL: Failed to initialize logger: %+v\n", err)
                     initErr = fmt.Errorf("failed to initialize logger: %w", err)
                     return
                }
                logger := Logger() // Get the initialized logger
                logger.Info("Logger initialized", slog.String("level", cfg.LOG_LEVEL))

                // 4. Initialize Telemetry (using cfg.OTEL_ENDPOINT etc.)
                 if err := initOtel(cfg.OTEL_SERVICE_NAME, cfg.OTEL_ENDPOINT, cfg.ENVIRONMENT, cfg.OTEL_RESOURCE_ATTRIBUTES); err != nil {
                     logger.Error("Failed to initialize OpenTelemetry", slog.Any("error", err)) // Use slog now
                     initErr = fmt.Errorf("failed to initialize telemetry: %w", err)
                     return
                 }
                logger.Info("OpenTelemetry initialized", slog.String("endpoint", cfg.OTEL_ENDPOINT))

                 logger.Info("Application Globals Initialized Successfully.")
            }) // End of once.Do

            return initErr
        }

        // Cfg returns the loaded configuration
        func Cfg() *config.Config {
            if cfg == nil {
                panic("Configuration not initialized. Call globals.Init() first.")
            }
            return cfg
        }

        // Keep/Update initLogger, initOtel, Logger, TracerProvider, MeterProvider functions
        // Ensure they use the cfg variable correctly
        // ... (rest of globals package) ...
        ```
    *   **Why:** Implements the environment-first loading strategy with `.env` support, simplifies the logic, ensures required variables are checked by `env.Parse`.

4.  **Create `.env.example`:**
    *   **Where:** Project root directory (`Signoz_assignment/.env.example`).
    *   **What:** Create an example file documenting all expected environment variables.
    *   **How:** Populate with keys from the `Config` struct and example/default values (commenting out sensitive ones if any).
        ```dotenv
        # Example environment variables for local development (Copy to .env)
        ENVIRONMENT="development"
        PRODUCT_SERVICE_PORT="8082"
        LOG_LEVEL="debug"
        OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
        PRODUCT_DATA_FILE_PATH="./product-service/data.json"
        SIMULATE_DELAY_ENABLED="false"
        SIMULATE_DELAY_MIN_MS="50"
        SIMULATE_DELAY_MAX_MS="150"
        OTEL_SERVICE_NAME="product-service"
        OTEL_RESOURCE_ATTRIBUTES="deployment.environment=development,service.version=0.1.0-local"
        ```
    *   **Why:** Provides developers with a clear template for local configuration.

5.  **Update `product-service/Dockerfile`:**
    *   **Where:** `product-service/Dockerfile`
    *   **What:** Remove the line that explicitly copies `data.json`.
    *   **How:** Delete the line `COPY product-service/data.json /app/data.json`.
    *   **Why:** The data file path is now controlled by the `PRODUCT_DATA_FILE_PATH` environment variable, loaded at runtime.

**Phase 2: OTEL Collector Configuration**

6.  **Make OTEL `host.name` Configurable:**
    *   **Where:** `otel-collector-config.yaml`
    *   **What:** Replace the hardcoded `host.name` value with an environment variable placeholder.
    *   **How:** Change `value: "testing_metrics"` to `value: "${OTEL_RESOURCE_HOST_NAME:otel-collector-host}"`.
        ```yaml
        # ...
        processors:
          resource:
            attributes:
              - key: host.name
                value: "${OTEL_RESOURCE_HOST_NAME:otel-collector-host}" # Use env var w/ default
                action: upsert
        # ...
        ```
    *   **Why:** Allows the `host.name` attribute reported by the collector to be controlled externally.

**Phase 3: Docker Compose Integration**

7.  **Update `docker-compose.yml`:**
    *   **Where:** `docker-compose.yml`
    *   **What:**
        *   For `product-service`: Ensure environment variable names match the `env` tags in `config.Config`. Set `PRODUCT_DATA_FILE_PATH=/app/data.json`. Remove the old `DATA_FILE_PATH` variable.
        *   For `otel-collector`: Add an `environment` section and define `OTEL_RESOURCE_HOST_NAME`.
    *   **How:**
        ```yaml
        services:
          product-service:
            # ... build, ports ...
            volumes:
              - ./product-service/data.json:/app/data.json # Mount the file
                    env_file:
            env_file
                .env.compose

            # ... networks, deploy ...

          # ... product-simulator ...

          otel-collector:
            # ... image, command, volumes, ports, networks ...
            # Add environment section
            environment:
              - OTEL_RESOURCE_HOST_NAME=otel-collector-compose # Set the desired host name here
        ```
    *   **Why:** Correctly injects configuration into both the Go service and the OTEL collector containers at runtime.

**Phase 4: Finalize**

8.  **Tidy Workspace:**
    *   **Where:** Project root directory.
    *   **What:** Ensure Go module dependencies are consistent across the workspace.
    *   **How:** Run `go mod tidy` inside `common`, then run `go work sync` in the project root.
    *   **Why:** Resolves potential import issues and cleans up dependencies.

---

## 🧪 Testing Strategy

*   **Go Config:**
    *   Run `product-service` locally *without* a `.env` file; verify startup logs show default values being used (e.g., log level, port, OTEL endpoint `localhost:4317`).
    *   Create `.env` from `.env.example`, modify some values (e.g., `LOG_LEVEL=debug`, `PRODUCT_SERVICE_PORT=9090`). Run locally again; verify new values are used.
    *   Run `docker-compose up --build`. Check `product-service` logs; verify values from `docker-compose.yml` are used (e.g., `ENVIRONMENT=production`, `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317`). Ensure service starts and can read `/app/data.json`.
*   **OTEL Config:**
    *   With `docker-compose up`, send some requests to `product-service` to generate telemetry.
    *   Check the telemetry backend (Signoz) for metrics/traces exported by the *collector*. Verify the `host.name` resource attribute is set to the value specified in `docker-compose.yml` (e.g., `otel-collector-compose`).

---

## ✅ Expected Outcomes

*   Centralized configuration definition in `common/config/config.go`.
*   Simplified and robust configuration loading in `common/globals/config.go` (`Init`).
*   Environment variables (set via Docker Compose or system) are the primary configuration source.
*   `.env` file support for easy local development overrides.
*   Elimination of hardcoded config values in Go code.
*   Removal of complex environment-switching logic in `LoadConfig`.
*   `host.name` attribute in OTEL Collector telemetry is configurable via `docker-compose.yml`.

---

## ρίск Risks and Mitigations

*   **Dependency:** Adding new dependencies. Mitigation: Minimal risk, widely used libraries.
*   **Configuration Mapping:** Missing or incorrect `env` tags or `docker-compose.yml` variables. Mitigation: Careful review of `Config` struct against `docker-compose.yml` and `.env.example`. Testing.
*   **Missing Required Vars:** `env.Parse` will fail if a required var without a default is missing. Mitigation: Ensure all required vars have defaults or are set in `docker-compose.yml`. Robust startup error handling in `Init`.
*   **File Path Confusion:** `PRODUCT_DATA_FILE_PATH` needs correct values for local (`./...`) vs container (`/app/...`). Mitigation: Clear comments in `.env.example` and `docker-compose.yml`; testing both environments.
*   **OTEL Env Var Substitution:** Issues with `${...}` syntax or variable not being passed correctly. Mitigation: Check collector logs; test with simple values first. 