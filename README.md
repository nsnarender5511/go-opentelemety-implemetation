# Signoz Assignment Project

This project demonstrates a simple microservices application built with Go (using the Fiber framework) and instrumented with OpenTelemetry for observability using SigNoz.

## Overview

The system includes:
*   A `product-service` (Go/Fiber) providing a basic API for managing products.
*   A `product-simulator` (Python) to generate load against the API.
*   An OpenTelemetry Collector (`otel-collector`) to receive, process, and export telemetry.
*   Docker Compose configuration (`docker-compose.yml`) to run the entire stack locally.
*   Shared Go modules (`common/`) for configuration, logging, simple file-based persistence, and telemetry setup.

> ⚠️ **Important Note:** The current file-based persistence implementation (`common/db/file_database.go` and its usage in `product-service/src/repository.go`) lacks proper locking for concurrent write operations and is **not safe for concurrent use** without modifications. This is intentionally left as is for demonstration purposes related to observability.

## Documentation

Comprehensive documentation covering architecture, features, development setup, and monitoring details can be found within the `docs/` directory (intended for use within an Obsidian vault or compatible Markdown viewer).

**Start Here:** [**Signoz Assignment Project Documentation**](./docs/Signoz Assignment Project Documentation.md)

## Quick Start

To run the project locally using Docker Compose:

1.  Ensure Docker and Docker Compose are installed.
2.  Navigate to the project root directory (`Signoz_assignment`).
3.  Run the command:
    ```bash
    docker compose up --build
    ```
    (Add `-d` to run in detached mode).
4.  Wait for services to initialize.
5.  Access SigNoz UI at [http://localhost:3301](http://localhost:3301).

For more detailed steps, see the [Quick Start Guide](./docs/Quick Start.md) in the main documentation.

## Contributing

This is primarily a demo project. Please refer to the documentation for details on components. 