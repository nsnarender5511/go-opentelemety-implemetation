

**Purpose:** This documentation provides a comprehensive overview of the Signoz Assignment project, including its architecture, services, development workflow, and observability setup using OpenTelemetry and SigNoz. It serves as the entry point for understanding the system.
**Audience:** Developers, DevOps, Students, anyone interested in the project.
**Prerequisites:** None

---

## 1. Overview

This project demonstrates a simple microservices application instrumented with OpenTelemetry for comprehensive observability using SigNoz as the backend. It includes:
*   A core `product-service` written in Go using the Fiber framework.
*   A `product-simulator` for generating load (Python).
*   An OpenTelemetry Collector for processing telemetry data.
*   Shared Go modules (`common/`) for configuration, logging, database access (file-based), and telemetry setup.
*   Docker Compose configuration for running the entire stack locally.

The primary goal is to showcase best practices in setting up and utilizing observability (traces, metrics, logs) in a realistic application context.

---

## 2. Documentation Structure

This documentation is organized into the following main sections:

*   **[Quick Start](./Quick Start.md)**: Minimal steps to get the project running and observe basic telemetry.
*   **[Glossary](./Glossary.md)**: Definitions of key terms and technologies used.
*   **[Architecture](./architecture/README.md)**: High-level view of the system components and their interactions.
    *   [Service Details](./architecture/Service Details.md): Detailed descriptions of individual services (`product-service`, `product-simulator`).
    *   [Data Model & Persistence](./architecture/Data Model & Persistence.md): Explanation of the file-based data storage.
*   **[Features](./features/README.md)**: Details about the application's functionality.
    *   [Product Service Features Overview](./features/product_service/Product Service Features Overview.md): Summary of features provided by the `product-service`.
    *   [Product Service API Endpoints](./features/product_service/Product Service API Endpoints.md): Specification of the HTTP API.
*   **[Development](./development/README.md)**: Information for developers working on the project.
    *   [Configuration Management](./development/Configuration Management.md): How application configuration is handled.
    *   [Building the Services](./development/Building the Services.md): Instructions for building the services.
    *   [Running Locally with Docker Compose](./development/Running Locally with Docker Compose.md): How to run the application stack using Docker Compose.
    *   [Testing Procedures](./development/Testing Procedures.md): Information on testing procedures, including the simulator.
*   **[Monitoring](./monitoring/README.md)**: Overview of the observability setup.
    *   [Telemetry Setup](./monitoring/Telemetry Setup.md): Detailed configuration of the OpenTelemetry SDK and Collector.
    *   [Logging Details](./monitoring/Logging Details.md): How logging is implemented and integrated with OTel.
    *   [Tracing Details](./monitoring/Tracing Details.md): How distributed tracing is implemented.
    *   [Key Metrics](./monitoring/Key Metrics.md): Description of custom application metrics.
    *   [SigNoz Dashboards](./monitoring/SigNoz Dashboards.md): Links and explanations for relevant SigNoz dashboards (once configured).

---

## 3. Navigating the Documentation

Use the links above or the file explorer to navigate through the different sections. Relative links (`./path/to/Page.md`) are used throughout to connect related topics.

---

**Last Updated:** 2024-07-30
