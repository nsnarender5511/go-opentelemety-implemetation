**Purpose:** This page explains how OpenTelemetry is configured and initialized within the Go services and how the OpenTelemetry Collector is set up to process and export telemetry data.
**Audience:** Developers, DevOps, Students
**Prerequisites:** Basic understanding of OpenTelemetry concepts (Traces, Metrics, Logs, OTLP). See [Glossary](../Glossary.md).
**Related Pages:** `common/telemetry/setup.go`, `otel-collector-config.yaml`, [Monitoring Overview](./README.md), [SigNoz Dashboards](./SigNoz_Dashboards.md)

---

## 1. Overview & Key Concepts

This project uses OpenTelemetry (OTel) for generating and collecting observability data (traces, metrics, logs). The setup involves two main parts:

1.  **Go Service Instrumentation:** The Go services (e.g., `product-service`) use the OpenTelemetry Go SDK libraries (via the `common/telemetry` modules) to generate telemetry data.
2.  **OpenTelemetry Collector:** A central OTel Collector service (`otel-collector` in `docker-compose.yml`) receives data from the services, processes it, and exports it to an external backend (SigNoz Cloud).

*   **Key Concept: OTLP:** The OpenTelemetry Protocol (OTLP) is used for communication between the Go services and the OTel Collector, and between the Collector and SigNoz Cloud.
*   **Core Responsibility:** Ensure that telemetry data from services is reliably generated, collected, and sent to the monitoring backend for analysis and visualization.
*   **Why it Matters:** This setup provides visibility into the application's performance, behavior, and errors, enabling debugging, monitoring, and performance analysis.

---

## 2. Configuration & Setup

### 2.1 Go Service SDK Initialization (`common/telemetry/setup.go`)

The Go OpenTelemetry SDK is initialized via the `telemetry.InitTelemetry` function, typically called once during application startup (e.g., via `common/globals.Init`).

**Relevant Files:**
*   `common/telemetry/setup.go`
*   `common/config/config.go` (for config values)
*   `common/telemetry/resource/resource.go`
*   `common/telemetry/trace/exporter.go`
*   `common/telemetry/metric/exporter.go`
*   `common/telemetry/log/exporter.go`

**Environment Variables (Read via `common/config` or OTel SDK defaults):**
*   `ENVIRONMENT`: Controls whether OTLP exporters are initialized (checked by `InitTelemetry`). Set to `production` to enable export.
*   `OTEL_ENDPOINT` / `OTEL_EXPORTER_OTLP_ENDPOINT`: Specifies the OTLP endpoint for the collector (e.g., `otel-collector:4317`). `InitTelemetry` uses the value from `config.Config`. OTel SDK might read the standard env var too.
*   `OTEL_SERVICE_NAME`: Defines the `service.name` resource attribute (read by OTel resource detectors).
*   `OTEL_RESOURCE_ATTRIBUTES`: Defines additional resource attributes, e.g., `deployment.environment=dev,service.version=1.0` (read by OTel resource detectors).

**Code Initialization Steps (`InitTelemetry`):**
1.  **Resource Creation:** An OTel Resource is created (`otelemetryResource.NewResource`) merging programmatically defined attributes (like process/SDK info) with attributes detected from environment variables (`OTEL_SERVICE_NAME`, `OTEL_RESOURCE_ATTRIBUTES`).
2.  **Environment Check:** If the `environment` string passed to `InitTelemetry` (which comes from `config.Config`) is `production`:
    *   **OTLP Exporter Configuration:** A gRPC connection is made to the `OTEL_ENDPOINT` from the config.
    *   Separate setup functions (`trace.SetupOtlpTraceExporter`, `metric.SetupOtlpMetricExporter`, `log.SetupOtlpLogExporter`) configure the respective signal exporters, processors, and global providers using this connection and the created Resource.
3.  **No-Op Setup (Else):** If `environment` is not `production`, No-Op providers are set up, effectively disabling telemetry export.

**Note on `globals.Init()`:** As mentioned in other documents, if `globals.Init()` is used, it currently hardcodes the environment to "production" when loading the config, meaning the OTLP exporters will always be initialized regardless of external `ENVIRONMENT` variables.

### 2.2 OpenTelemetry Collector (`otel-collector-config.yaml`)

The OTel Collector service handles receiving, processing, and exporting telemetry.

**Relevant Files:**
*   `otel-collector-config.yaml`
*   `docker-compose.yml` (defines the service, volumes, ports, environment variables like `SIGNOZ_INGESTION_KEY`)

**Key Configuration Sections (`otel-collector-config.yaml`):**

*   **Receivers:**
    *   `otlp`: Listens on `0.0.0.0:4317` (gRPC) for OTLP data from services.
    *   `docker_stats`: Collects container metrics from the Docker daemon socket (`/var/run/docker.sock` mounted as a volume).
*   **Processors:**
    *   `batch`: Batches telemetry data before export for efficiency.
    *   `resource`: Adds/updates resource attributes (Example adds `host.name=testing_metrics` - could be used for env tags, etc.).
*   **Exporters:**
    *   `otlp`: Exports data via OTLP/gRPC to SigNoz Cloud (`ingest.in.signoz.cloud:443`). Uses `headers` to send the SigNoz ingestion key (read from `SIGNOZ_INGESTION_KEY` env var passed from `docker-compose.yml`). **[Security Note: Manage the ingestion key securely]**.
    *   `debug`: Exports *metrics* to the collector's logs for debugging purposes.
*   **Service Pipelines:** Define how data flows from receivers through processors to exporters for each signal type:
    *   `traces`: `otlp` receiver -> `resource` processor -> `batch` processor -> `otlp` exporter
    *   `metrics`: `otlp`, `docker_stats` receivers -> `batch` processor -> `otlp`, `debug` exporters
    *   `logs`: `otlp` receiver -> `batch`, `resource` processors -> `otlp` exporter

---

## 3. Implementation Details & Usage

This page focuses on the setup. Refer to [Logging Details](./Logging_Details.md), [Tracing Details](./Tracing_Details.md), and [Key Metrics](./Key_Metrics.md) for how the initialized SDK components are used within the application code.

---

## 4. Monitoring & Observability Integration

This entire page describes the core integration setup.

---

## 5. Visuals & Diagrams

<!-- 
[USER ACTION REQUIRED]
Export the diagram from ../assets/diagrams/telemetry_pipeline.excalidraw to PNG or SVG,
place it in ../assets/images/ or ../assets/diagrams/, 
and update the link below.

Example Markdown:
![Telemetry Pipeline Diagram](../assets/images/telemetry_pipeline.png)
*Fig 1: Telemetry Pipeline Diagram.*
-->

*Placeholder for Telemetry Pipeline Diagram.*

---

## 6. Teaching Points & Demo Walkthrough

*   **Key Takeaway:** Telemetry involves instrumenting code (Go SDK) to *generate* data and using a collector to *receive, process, and export* that data to a backend like SigNoz. The collector provides flexibility (processing, batching, multiple backends) and offloads work from the application.
*   **Demo Steps:**
    1.  Show `common/telemetry/setup.go` explaining the SDK initialization logic (`InitTelemetry`).
    2.  Show `otel-collector-config.yaml` highlighting the receivers, processors, exporters, and pipelines. Explain the flow using the diagram above (once added).
    3.  Show how the `SIGNOZ_INGESTION_KEY` is passed from `docker-compose.yml` to the collector's environment and used in the OTLP exporter configuration.
    4.  Run `docker-compose up`.
    5.  Show collector logs (`docker compose logs otel-collector`) potentially including debug exporter output for metrics.
    6.  Briefly show the [SigNoz Dashboards](./SigNoz_Dashboards.md) page.
*   **Common Pitfalls / Questions:**
    *   Why use a Collector instead of exporting directly from the service? (Decoupling, batching, processing, multiple backends, potentially less resource usage in the service).
    *   What's the difference between SDK initialization and Collector configuration? (SDK is *in* the app code, generates data; Collector is *external* infrastructure, handles data flow).
    *   Why is my data not showing up? (Check `ENVIRONMENT` variable (and potential `globals.Init` override), Collector logs, SigNoz endpoint/key in collector config, network connectivity between services and collector, and collector and SigNoz Cloud).
*   **Simplification Analogy:** The Go SDK puts letters (telemetry data) in envelopes with specific addresses (OTLP endpoint). The OTel Collector is like a local post office that collects these letters, sorts them (processors like batch/resource), maybe adds a regional stamp, and sends them in bulk to the main sorting center (SigNoz Cloud) using a special delivery truck (OTLP exporter with API key).

---

**Last Updated:** 2024-07-30
