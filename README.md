# Go Microservice

  

This project demonstrates a simple microservices application built with Go (using the Fiber framework) and instrumented with OpenTelemetry for observability using SigNoz.

 
## Overview

  

The system includes:

* A `product-service` (Go/Fiber) providing a basic API for managing products.
*  Shared Go modules (`common/`) for configuration, logging, simple file-based persistence, and telemetry setup.
* *An OpenTelemetry Collector (`otel-collector`) to receive, process, and export telemetry.
* A `product-simulator` (Python) to generate load against the API.
* Docker Compose configuration (`docker-compose.yml`) to run the entire stack locally.

## Documentation

Comprehensive documentation covering architecture, features, development setup, and monitoring details can be found within the `docs/` directory (intended for use within an Obsidian vault or compatible Markdown viewer).

**Start Here:** [**Signoz Assignment Project Documentation**](./docs/Signoz_Assignment_Project_Documentation.md)

  

## Quick Start

  

To run the project locally using Docker Compose:

  

1. Ensure Docker and Docker Compose are installed.

2. Navigate to the project root directory (`Signoz_assignment`).

3. Run the command:

```bash
docker compose up --build
```
(This command builds the necessary images and starts the application services and the OpenTelemetry Collector. While the included `docker-compose.yml` also defines a local SigNoz instance for testing, using SigNoz Cloud is recommended for a full observability experience.)

(Add `-d` to run in detached mode).

4. Wait for services to initialize.
5. Configure your OpenTelemetry Collector (see `otel-collector-config.yaml`) to send data to your SigNoz Cloud endpoint. Find more details at [SigNoz Cloud](https://signoz.io/cloud/).

For more detailed steps, see the [Quick Start Guide](./docs/Quick Start.md) in the main documentation.

  

## Contributing

  

This is primarily a demo project. Please refer to the documentation for details on components.