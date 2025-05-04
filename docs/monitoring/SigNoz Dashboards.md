# SigNoz Dashboards

**Purpose:** Provide links to and explanations for relevant dashboards within the SigNoz UI for monitoring the application.
**Audience:** Developers, DevOps, SREs, Students
**Prerequisites:** [Running Locally with Docker Compose](../../development/Running%20Locally%20with%20Docker%20Compose.md), [Monitoring Overview](./README.md), [Key Metrics](./Key%20Metrics.md)
**Related Pages:** SigNoz UI ([http://localhost:3301](http://localhost:3301))

---

## 1. Overview

SigNoz allows creating custom dashboards to visualize key metrics and service performance. While default dashboards are available upon installation, creating application-specific dashboards provides the most targeted insights.

This page outlines potential dashboards valuable for this project. You should create these (or similar) dashboards within your SigNoz instance running at [http://localhost:3301](http://localhost:3301).

<!-- 
[USER ACTION REQUIRED]
Create the dashboards described below in your SigNoz instance.
Optionally, replace the descriptive text with actual screenshots or embedded dashboard links if your environment supports it.
-->

---

## 2. Recommended Dashboards

Here are examples of dashboards that would be valuable for monitoring this application:

### 2.1 Product Service Overview Dashboard
*   **Purpose:** Provides a single pane of glass view into the health and performance of the `product-service`.
*   **Key Panels / Metrics:**
    *   **Request Rate (per endpoint):** `http.server.request_count` (from OTel instrumentation, potentially requires `http.route` attribute) or aggregate `app.operations.total` filtered by `app.layer=handler`.
    *   **Request Latency (P99, P95, P50):** `http.server.duration` (from OTel instrumentation) or `app.operations.duration_milliseconds` filtered by `app.layer=handler`.
    *   **Error Rate (HTTP 5xx):** `http.server.request_count` filtered by `http.status_code>=500`.
    *   **Repository Operation Rate:** `app.operations.total` filtered by `app.layer=repository`.
    *   **Repository Operation Duration (P95):** `app.operations.duration_milliseconds` filtered by `app.layer=repository`.
    *   **Repository Error Rate:** `app.operations.errors.total` filtered by `app.layer=repository`.
    *   **CPU / Memory Usage (`product-service` container):** Metrics from `docker_stats` receiver (e.g., `container.cpu.usage.total`, `container.memory.usage.bytes`).

### 2.2 Repository Performance Dashboard
*   **Purpose:** Deep dive into the performance of the file-based repository operations.
*   **Key Panels / Metrics (Filter all by `app.layer=repository`):**
    *   **Operation Count (Grouped by `app.operation`):** `app.operations.total`.
    *   **Operation Duration (P99, P95, P50) (Grouped by `app.operation`):** `app.operations.duration_milliseconds`.
    *   **Error Count (Grouped by `app.operation`):** `app.operations.errors.total`.
    *   **Duration Heatmap/Histogram:** `app.operations.duration_milliseconds`.

### 2.3 Infrastructure / Docker Stats Dashboard
*   **Purpose:** Monitor the resource consumption of the running Docker containers.
*   **Key Panels / Metrics (Using `docker_stats` receiver metrics):**
    *   **CPU Usage per Container:** e.g., `container.cpu.usage.total` (grouped by `container.name` or `service.instance.id`).
    *   **Memory Usage per Container:** e.g., `container.memory.usage.bytes` (grouped by `container.name` or `service.instance.id`).
    *   **Network I/O per Container:** e.g., `container.network.io.usage.tx_bytes`, `container.network.io.usage.rx_bytes`.
    *   **(Optional) Disk I/O per Container:** e.g., `container.disk.io.usage.total`.

---

## 3. Creating Dashboards in SigNoz

1.  Navigate to the **Dashboard List** section in the SigNoz UI.
2.  Click **New Dashboard**.
3.  Add panels (e.g., Time Series, Value, Table, Heatmap).
4.  Configure each panel:
    *   Select **Metrics** as the data source.
    *   Use the **Query Builder** or **Clickhouse Query** editor.
    *   Select the desired metric (e.g., `app.operations.duration_milliseconds`).
    *   Apply filters (e.g., `