# Service Details: Product Service

**Purpose:** Provide detailed information about the `product-service`, including its responsibilities, API endpoints, dependencies, and key implementation notes.
**Prerequisites:** [Architecture Overview](./Architecture_Overview.md)
**Related Pages:** `docker-compose.yml`, `product-service/src/main.go`, [Testing Procedures](../../development/Testing_Procedures.md), [Data Model & Persistence](./Data_Model_&_Persistence.md)

---

## 1. `product-service`

**Source Code:** `product-service/`

### 1.1 Overview & Key Concepts

*   **Core Responsibility:** Provides an HTTP API for managing product information (retrieval, creation, stock updates).
*   **Technology:** Written in Go, uses the Fiber web framework (`github.com/gofiber/fiber/v2`).
*   **Architecture:** Follows a layered approach (Handler -> Service -> Repository).
    *   **Handler (`handler.go`):** Handles incoming HTTP requests using Fiber (`fiber.Ctx`), parses path parameters and JSON request bodies, calls the service layer, and returns JSON responses or errors. Explicitly creates trace spans for handler operations.
    *   **Service (`service.go`):** Contains the core business logic (e.g., validation, ID generation). Mostly acts as a pass-through to the repository layer. Creates its own trace spans and records operation metrics.
    *   **Repository (`repository.go`):** Implements `ProductRepository` interface. Interacts with the data persistence layer (`common/db.FileDatabase`), reads/writes `data.json`. Expects `data.json` to be a map keyed by product ID. Performs read-modify-write for updates/creates. **CRITICAL:** It **does not** implement the necessary locking (`sync.Mutex`) for concurrency control on these write operations, leading to potential race conditions. See [Data Model & Persistence](./Data_Model_&_Persistence.md). Well-instrumented with traces and metrics.
*   **Why it Matters:** This is the primary application service providing the core functionality, but its current persistence implementation has significant concurrency flaws.

### 1.2 Configuration & Setup

*   **Initialization:** The service entry point is `product-service/src/main.go`.
*   **Globals:** It calls `common/globals.Init()` on startup. This initializes shared config, logging, and telemetry by calling `config.LoadConfig("production")` internally. This means the **"production" configuration profile is loaded by default**, merging its settings (like OTel endpoint) with common defaults. While runtime environment variables (e.g., `LOG_LEVEL` set in `docker-compose.yml`) can override specific values after loading, the base configuration loaded is the "production" one. (See [Configuration Management](../../development/Configuration_Management.md) and [Telemetry Setup](../../monitoring/Telemetry_Setup.md)). **// Fix Applied**
*   **Dependencies:** Dependencies (Repository, Service, Handler) are manually instantiated and injected in `main.go`.
*   **Configuration Values:** Reads required values (e.g., `PRODUCT_SERVICE_PORT`) from the config provided by `globals.Cfg()`.

### 1.3 Implementation Details & Usage

*   **HTTP Server:** Uses `Fiber` to listen for HTTP requests on the configured port.
*   **Middleware:** Configured with:
    *   CORS (`cors.New`)
    *   Panic Recovery (`recover.New`)
    *   OpenTelemetry (`otelfiber.Middleware()`): Automatically creates spans for incoming HTTP requests, captures relevant HTTP attributes (URL, method, status code), and handles trace context propagation.
*   **Routing:** Defines routes for product operations and health checks (see [Product Service API Endpoints](../../features/product_service/Product_Service_API_Endpoints.md)).
*   **Concurrency Issue:** As noted, the repository layer lacks proper locking for write operations (`Create`, `UpdateStock`), making the service unsafe for concurrent modifications. See [Data Model & Persistence](./Data_Model_&_Persistence.md).

### 1.4 Monitoring & Observability Integration

*   **HTTP Instrumentation:** The `otelfiber.Middleware()` provides automatic tracing for all incoming requests.
*   **Logging/Tracing/Metrics:** Uses the shared `common/log`, `common/telemetry/trace`, and `common/telemetry/metric` packages for further instrumentation within the handler, service, and repository layers.
*   **Setup:** Relies on the telemetry setup initialized via `globals.Init()` (effectively always the "production" setup with OTLP export enabled).

### 1.5 Visuals & Diagrams

```mermaid
graph LR
    subgraph External
        HTTPClient[HTTP Client / Simulator]
    end
    subgraph ProductService
        direction TB
        Middleware[Fiber Middleware (OTel, CORS, Recover)] --> Handler
        Handler(handler.go) --> Service(service.go)
        Service --> Repository(repository.go \n <b style='color:red'>NO LOCKING!</b>)
        Repository --> CommonDB[common/db.FileDatabase \n <b style='color:red'>NO LOCKING!</b>]
        CommonDB --> JSONFile[data.json]
    end
    HTTPClient -- HTTP Request --> Middleware

    style ProductService fill:#f9f,stroke:#333,stroke-width:2px
    style Repository fill:#ffdddd,stroke:red
    style CommonDB fill:#ffdddd,stroke:red
    style JSONFile fill:#lightgrey,stroke:#333,stroke-width:1px
```
*Fig 1: Internal Layers and Data Flow of `product-service` (Highlighting lack of locking in persistence).*

### 1.6 Teaching Points & Demo Walkthrough

*   **Key Takeaway:** A standard layered Go web service using Fiber. Demonstrates dependency injection and integration with common modules for logging, config, and telemetry. **Crucially, it also serves as an example of an incorrect persistence implementation for concurrent environments due to the lack of locking.**
*   **Demo Steps:**
    1.  Show `main.go`, highlighting the `globals.Init()` call, dependency creation, Fiber app setup, middleware (`otelfiber`), and route definitions.
    2.  Explain the Handler -> Service -> Repository layering concept with the diagram.
    3.  **Explicitly** show the lack of `sync.Mutex` in `repository.go` and `common/db/file_database.go` and explain the race condition risk as documented in [Data Model & Persistence](./Data_Model_&_Persistence.md).
    4.  Run the service and make a request (e.g., `GET /products`).
    5.  Show the trace in SigNoz, pointing out the span created by `otelfiber` middleware for the HTTP request and the nested spans from handler/service/repository.
*   **Common Pitfalls:** Forgetting to initialize globals (`panic: configuration not initialized`), incorrect dependency injection, **ignoring concurrency issues in simple persistence layers.**

---

## 2. `product-simulator`

**Source Code:** `tests/` (`simulate_product_service.py`, `Dockerfile`)

### 2.1 Overview & Key Concepts

*   **Core Responsibility:** Generate continuous HTTP load against the `product-service` API to simulate traffic and produce telemetry data for observation in SigNoz.
*   **Technology:** Written in Python.
*   **Why it Matters:** Provides a simple way to exercise the `product-service` and demonstrate the observability features without manual intervention. Can also unintentionally expose concurrency issues in the `product-service` if it sends requests rapidly.

### 2.2 Configuration & Setup

*   **Dockerfile:** `tests/Dockerfile` defines how to build the Python image.
*   **Docker Compose:** Defined as the `product-simulator` service in `docker-compose.yml`. Depends on `product-service` being available.
*   **Target:** Configured via environment variable `PRODUCT_SERVICE_URL` (set to `http://product-service:8082` in `docker-compose.yml`) to target the `product-service` within the Docker network.
*   **Dependencies:** Requires Python libraries (likely `requests`). A `tests/requirements.txt` file is needed for the Docker build; its absence would cause the build to fail. See [Building the Services](../../development/Building_the_Services.md).

### 2.3 Implementation Details & Usage

*   Runs automatically when the stack is started via `docker compose up`.
*   Continuously loops, making various API calls (`GET`, `POST`, `PATCH`, `/health`, invalid paths) to the `PRODUCT_SERVICE_URL`.
*   Logs its actions to standard output (viewable with `docker compose logs product-simulator`).
*   **Note:** This service itself is *not* currently instrumented with OpenTelemetry. Its purpose is to generate activity in the *instrumented* `product-service`.

### 2.4 Monitoring & Observability Integration

*   Directly: None. Logs can be viewed via `docker compose logs`.
*   Indirectly: The requests it generates cause traces, metrics, and logs to be produced by the `product-service` and sent to SigNoz.

### 2.5 Teaching Points & Demo Walkthrough

*   **Key Takeaway:** Demonstrates a simple client service used for load generation in a microservices context.
*   **Demo Steps:**
    1.  Show the `product-simulator` service definition in `docker-compose.yml`.
    2.  While the stack is running, show its logs using `docker compose logs -f product-simulator`.
    3.  Correlate the requests seen in the simulator logs with traces appearing in SigNoz for the `product-service`.
    4.  If demonstrating the concurrency issue, point out how the simulator's rapid requests can trigger the race condition in the `product-service`'s repository layer.

---

## 3. `otel-collector`

This is the standard OpenTelemetry Collector.
*   **Configuration:** Defined in `otel-collector-config.yaml`.
*   **Role:** Receives OTLP data from `product-service`, processes it (batching, adding resource attributes), and exports it to SigNoz Cloud.
*   See [Telemetry Setup](../../monitoring/Telemetry_Setup.md) for detailed configuration.

---

**Last Updated:** 2024-07-30
