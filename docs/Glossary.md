

**Purpose:** Define key terminology, technologies, and concepts used within this project and its documentation.
**Prerequisites:** None

---

## Key Terms & Technologies

*   **Docker:** A platform for developing, shipping, and running applications in containers. Containers package code and dependencies together.
*   **Docker Compose:** A tool for defining and running multi-container Docker applications. Uses a `docker-compose.yml` file to configure services, networks, and volumes.
*   **Fiber:** A Go web framework inspired by Express.js, used in the `product-service` for building the HTTP API.
*   **Go (Golang):** The primary programming language used for the backend services (`product-service`, shared `common` modules).
*   **JSON (JavaScript Object Notation):** A lightweight data-interchange format used for the file-based database (`data.json`) and API request/response bodies.
*   **Microservices:** An architectural style that structures an application as a collection of small, independent, and loosely coupled services.
*   **Observability:** The ability to measure the internal states of a system by examining its outputs (logs, metrics, traces). Often referred to as the "three pillars".
    *   **Logs:** Records of discrete events that occurred at specific times. In this project, structured logs are generated using Go's `slog` and exported via OTel.
    *   **Metrics:** Numerical representations of system behavior or resources over time (e.g., request count, CPU usage, duration). Collected using the OTel SDK and potentially other sources (like `docker_stats`).
    *   **Traces (Distributed Tracing):** Records the path of a request as it travels through various components or services in a distributed system. Helps visualize flow, identify bottlenecks, and debug issues.
*   **OpenTelemetry (OTel):** An open-source observability framework comprising APIs, SDKs, and tools for instrumenting, generating, collecting, and exporting telemetry data (traces, metrics, logs). It provides a vendor-neutral standard.
    *   **OTel Collector:** A standalone service that receives telemetry data from various sources (using OTLP or other protocols), processes it, and exports it to one or more backends (like SigNoz). Used in this project.
    *   **OTel SDK:** Language-specific libraries (e.g., OTel Go SDK) used within application code to generate telemetry data.
    *   **OTLP (OpenTelemetry Protocol):** The native protocol for transmitting telemetry data between SDKs, Collectors, and backends.
    *   **Resource Attributes:** Key-value pairs describing the entity producing telemetry (e.g., `service.name`, `deployment.environment`).
    *   **Span:** The basic building block of a trace, representing a single unit of work (e.g., an HTTP request, a database call).
    *   **Span Attributes:** Key-value pairs providing context about a span (e.g., `http.method`, `db.statement`).
    *   **Trace Context:** Information (Trace ID, Span ID) propagated across process boundaries to correlate spans into a single trace (e.g., using W3C Trace Context headers).
*   **SigNoz:** An open-source observability platform used as the backend in this project. It provides storage, querying, visualization, and alerting for traces, metrics, and logs.
*   **`slog`:** Go's standard library package for structured logging.

---

**Last Updated:** 2024-07-30
