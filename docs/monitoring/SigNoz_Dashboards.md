# SigNoz Dashboards

**Purpose:** Provide links to and explanations for relevant dashboards within the SigNoz UI for monitoring the application.
**Prerequisites:** [Running Locally with Docker Compose](../development/Running_Locally_with_Docker_Compose.md), [Monitoring Overview](./README.md), [Key Metrics](./Key_Metrics.md)
**Related Pages:** SigNoz UI ([http://localhost:3301](http://localhost:3301))

---

## 1. Overview

SigNoz allows creating custom dashboards to visualize key metrics and service performance. While default dashboards are available upon installation, creating application-specific dashboards provides the most targeted insights.

This page outlines potential dashboards valuable for this project. You should create these (or similar) dashboards within your SigNoz.


---

# 2. Dashboards

Here are Some dashboards that would be valuable for monitoring this application:
---


### 2.1 Product-Service Overview Dashboard

**Purpose:** Provide a single-pane snapshot of the health and performance of product-service.

**Key Panels / Metrics:**

- **Latency (P50, P90, P99)** – Track percentile latency trends
- **Request Rate** – Observe operations per second
- **Apdex** – Measure user satisfaction derived from latency
- **Key Operations Table** – View P50/P95/P99 latency, error rate, and call volume per operation
- **Database Call RPS & Avg Duration** – Monitor load and timings for repository operations

**Link:** [http://localhost:3301/dashboard/product-service-overview](http://localhost:3301/dashboard/product-service-overview)

**Screenshot:** (insert Product-Service-Overview screenshot)

---

### 2.2 Custom Dashboard

**Purpose:** Combine API traffic, error breakdowns, and business KPIs in a single view.

**Key Sections & Panels:**

#### API Wise Metrics
- API‑Wise Traffics (RPS line)
- Request Distribution (donut)

#### Error Details
- Success vs Failure
- Error Codes

#### Business Level Metrics
- Total New Products Created
- Product Updates Total
- Top 10 Products in Inventory

**Link:** [http://localhost:3301/dashboard/custom_dashboard](http://localhost:3301/dashboard/custom_dashboard)

**Screenshot:** (insert Custom-Dashboard screenshot)

---

### 2.3 Container Metrics Dashboard

**Purpose:** Visualize resource consumption and network activity for each container (via the docker_stats receiver).

**Key Panels / Metrics:**
- Container CPU Percent
- Container Memory Percent
- Network Bytes Received / Sent
- Packets Dropped
- Memory Usage vs Limit

**Link:** [http://localhost:3301/dashboard/container-metrics](http://localhost:3301/dashboard/container-metrics)

**Screenshot:** (insert Container-Metrics screenshot)

---

### 2.4 Trace Explorer View

**Purpose:** Drill into individual requests to find latency sources and call-stack bottlenecks.

**Key Elements:**
- All Traces List – Timestamp, route, client address, status code
- Flamegraph / Spans – Hierarchical span timings for a single trace

**Link:** [http://localhost:3301/trace](http://localhost:3301/trace)

**Screenshot:** (insert Trace-Explorer screenshot)

---

### 2.5 Logs Explorer View

**Purpose:** Search, filter, and visualize raw logs, then correlate them with traces for context-rich debugging.

**Key Elements:**
- Severity Filter & Frequency Chart – Spot bursts in WARN / ERROR levels
- Facet Filters – Environment, hostname, Kubernetes metadata, etc.
- Views – Toggle between List, Time Series, and Table views

**Link:** [http://localhost:3301/logs?service.name=product-service](http://localhost:3301/logs?service.name=product-service)

**Screenshot:** (insert Logs-Explorer screenshot)

---


**Last Updated:** 2024-07-30