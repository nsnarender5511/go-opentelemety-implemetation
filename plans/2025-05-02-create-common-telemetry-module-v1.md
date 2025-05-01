
# Create Common Telemetry Module and Integrate with Product Service

## Objective
To create a reusable Go module named `common` containing OpenTelemetry initialization logic. This module will be integrated into the existing `product-service` to demonstrate distributed tracing with SigNoz, all running within a Docker Compose environment.

## Implementation Plan
1.  **Examine Existing Files**
    *   Read `go.work`, `product-service/go.mod`, `product-service/main.go`, and `docker-compose.yml` to understand the current project state.
    *   Dependencies: None
    *   Notes: Establishes a baseline before making changes.
    *   Files: `go.work`, `product-service/go.mod`, `product-service/main.go`, `docker-compose.yml`
    *   Status: Not Started
2.  **Design `common` Module Structure**
    *   Plan the directory layout (e.g., `common/telemetry/`) and define the public API (e.g., `InitTracerProvider`, `ShutdownTracerProvider`, `GetTracer`).
    *   Dependencies: None
    *   Notes: Focus on clarity and reusability. Consider potential future additions (logging, config).
    *   Files: (Conceptual design for files within `common/`)
    *   Status: Not Started
3.  **Create `common` Module Files (Conceptual)**
    *   Conceptually define `common/go.mod`, `common/telemetry/otel.go` (implementing OTel SDK setup: exporter, provider, resource attributes), and any other necessary files based on the design.
    *   Dependencies: Step 2
    *   Notes: Ensure proper OTel dependencies are added to `common/go.mod`. The implementation should be based on OpenTelemetry Go documentation and the chosen exporter protocol (gRPC/HTTP).
    *   Files: `common/go.mod`, `common/telemetry/otel.go`
    *   Status: Not Started
4.  **Update Go Workspace**
    *   Modify `go.work` to add `use ./common`. Conceptually run `go mod tidy` in relevant module directories.
    *   Dependencies: Step 3
    *   Notes: Integrates the new module into the build system.
    *   Files: `go.work`, `common/go.mod`, `product-service/go.mod`
    *   Status: Not Started
5.  **Integrate `common` Module into `product-service` (Conceptual)**
    *   Modify `product-service/main.go` to import and call initialization/shutdown functions from `common/telemetry`.
    *   Modify service handlers to obtain a tracer and create spans using the common module's API.
    *   Update `product-service/go.mod` if needed.
    *   Dependencies: Step 4
    *   Notes: Minimal changes should be required in the service logic itself, mainly boilerplate initialization and tracer usage.
    *   Files: `product-service/main.go`, `product-service/go.mod`, service handler files.
    *   Status: Not Started
6.  **Plan Compilation Check**
    *   Outline the command `go build ./...` to be run from the workspace root.
    *   Dependencies: Step 5
    *   Notes: Verifies successful integration at the code level.
    *   Files: All `.go` files, `go.mod`, `go.work`.
    *   Status: Not Started
7.  **Update `docker-compose.yml`**
    *   Add SigNoz services (otel-collector, query-service, frontend).
    *   Configure `product-service`: set build context, environment variables (e.g., `OTEL_EXPORTER_OTLP_ENDPOINT`), service dependencies (`depends_on`), and networking.
    *   Dependencies: Step 5
    *   Notes: Crucial for connecting the instrumented service to the SigNoz backend. Refer to SigNoz Docker Compose examples. Ensure endpoint hostname/port matches SigNoz collector configuration.
    *   Files: `docker-compose.yml`
    *   Status: Not Started
8.  **Plan Verification Steps**
    *   Outline running `docker compose up --build`.
    *   Describe sending test requests to `product-service`.
    *   Describe accessing the SigNoz UI (`http://localhost:3301` by default) to find and inspect traces from `product-service`.
    *   Dependencies: Step 7
    *   Notes: Confirms the end-to-end flow works as expected.
    *   Files: `docker-compose.yml`
    *   Status: Not Started

## Verification Criteria
-   The Go project (including `common` and `product-service`) compiles successfully using `go build ./...`.
-   `docker compose up --build` starts all services (product-service, SigNoz components) without errors.
-   Sending requests to `product-service` generates trace data.
-   Traces from `product-service` are visible and correctly correlated in the SigNoz UI, showing the expected service name and spans.

## Potential Risks and Mitigations
1.  **Incorrect OpenTelemetry Configuration:** The OTel setup in `common/telemetry/otel.go` might be misconfigured (wrong exporter endpoint, missing resource attributes, incorrect propagators).
    *   Mitigation: Double-check configuration against OpenTelemetry Go documentation and SigNoz examples. Start with a minimal configuration and add complexity incrementally. Check collector logs for connection errors.
2.  **Docker Networking Issues:** The `product-service` container might fail to reach the SigNoz OTLP collector container.
    *   Mitigation: Ensure all relevant services are on the same Docker network in `docker-compose.yml`. Verify the `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable uses the correct service name and port as defined in `docker-compose.yml`. Use `docker compose logs <service_name>` to debug.
3.  **Go Workspace Dependency Conflicts:** Issues might arise with dependency versions between `common`, `product-service`, and their transitive dependencies.
    *   Mitigation: Use `go mod tidy` within each module's directory after changes. Examine `go.mod` files and potentially use `replace` directives in the root `go.work` file if necessary to align versions, though this should be a last resort.

## Alternative Approaches
1.  **Direct Instrumentation:** Instrument `product-service` directly without a separate `common` module. (Less reusable if more services are added later).
2.  **Different Telemetry Backend:** Use a different OpenTelemetry backend instead of SigNoz (e.g., Jaeger, Prometheus/Grafana Tempo). (Requires changing the backend services in `docker-compose.yml` and potentially the exporter configuration).

