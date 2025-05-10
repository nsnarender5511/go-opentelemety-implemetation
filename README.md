## Project Overview
This project simulates a microservices environment for a product service, including load generation, observability, and various service configurations to mimic real-world scenarios. It utilizes Docker for containerization, Nginx as a reverse proxy, and OpenTelemetry for collecting telemetry data. The core components include multiple instances of a product service, a load simulator, and an OpenTelemetry collector.

## Directory Structure
- **`Signoz_assignment/`**: The root directory of the project.
    - **`common/`**: Contains shared Go packages for common functionalities like API error handling, requests/responses, configuration, database utilities, logging, middleware, telemetry, and general utils.
    - **`locust-simulations/`**: Holds the Locust-based load simulation setup.
        - `locustfile.py`: Defines the load testing scenarios.
        - `src/`: Source code for the simulation, potentially custom Python modules.
        - `results/`: Directory to store load testing results.
        - `Dockerfile`: Dockerfile for building the locust simulator service.
        - `requirements.txt`: Python dependencies for the load simulator.
    - **`product-service/`**: Contains the Go-based product microservice.
        - `src/`: Source code for the product service (handlers, models, repositories, services).
        - `data.json`: Sample product data used by the service.
        - `Dockerfile`: Dockerfile for building the product service.
        - `go.mod`, `go.sum`: Go module files.
    - **`.github/workflows/`**: GitHub Actions workflows for CI/CD or other automations (contents not inspected).
    - `docker-compose.yml`: Defines and configures the multi-container Docker application.
    - `nginx.conf`: Configuration file for the Nginx reverse proxy.
    - `otel-collector-config.yaml`: Configuration for the OpenTelemetry Collector.
    - `.env`, `.env.example`: Environment variable files for configuration.

## Technology Stack
- **Backend Services**: Go (`product-service`)
- **Load Testing**: Python, Locust (`locust-simulations`)
- **Containerization**: Docker, Docker Compose
- **Reverse Proxy/Load Balancer**: Nginx
- **Observability**: OpenTelemetry (Collector), potentially Signoz (based on environment variables in `docker-compose.yml`)
- **Configuration**: YAML, .env files

## Architecture Patterns
- **Microservices**: The application is structured as a set of loosely coupled services (`product-service` instances, `product-simulator`, `otel-collector`).
- **Reverse Proxy/Load Balancing**: Nginx is used to distribute incoming traffic among the different `product-service` instances.
- **Centralized Telemetry Collection**: An OpenTelemetry Collector (`otel-collector`) is used to gather traces, metrics, and logs from the services.
- **Container-based Deployment**: All services are containerized using Docker and orchestrated with Docker Compose.
- **Configuration via Environment Variables**: Services are configured using environment variables, allowing for flexibility across different environments.

## Core Components

### 1. `product-service` (Instances: `product-service-1` to `product-service-5`)
- **Description**: A Go-based microservice responsible for handling product-related requests.
- **Instances**: Multiple instances (`nsh-store-1` to `nsh-store-5`) are defined in `docker-compose.yml`, each with potentially different configurations related to:
    - `SERVICE_VERSION`: e.g., `v1.0.0`, `v1.1.0-beta`.
    - `SIMULATE_DELAY_ENABLED`: Boolean, to simulate network latency.
    - `SIMULATE_RANDOM_ERROR_ENABLED`: Boolean, to simulate failures.
    - Resource limits (CPU, memory).
- **Data Source**: Uses `product-service/data.json` for product information.
- **Telemetry**: Configured to send telemetry data to `otel-collector:4317`.

### 2. `product-simulator`
- **Description**: A Locust-based load generator designed to simulate user traffic to the `product-service` through Nginx.
- **Configuration**:
    - `PRODUCT_SERVICE_URL`: Points to `http://nginx:80`.
    - `OTEL_ENDPOINT`: `otel-collector:4317` (but `OTEL_ENABLED` is set to `false` by default).
- **Functionality**: Runs `locustfile.py` to generate load. Allows for class picking and different load shapes.

### 3. `otel-collector`
- **Description**: An OpenTelemetry Collector service responsible for receiving, processing, and exporting telemetry data (traces, metrics, logs).
- **Configuration**: Defined in `otel-collector-config.yaml`.
    - **Receivers**:
        - `otlp` (gRPC on `0.0.0.0:4317`) for application telemetry.
        - `docker_stats` for container metrics.
        - `hostmetrics` for host system metrics.
        - `nginx` for metrics from the Nginx server (`http://nginx:80/nginx_status`).
    - **Processors**:
        - `resourcedetection`: Detects resource attributes from `env` and `docker`.
    - **Exporters**:
        - `otlp`: Exports data to a Signoz backend (inferred from `${SIGNOZ_ENDPOINT}` and `${SIGNOZ_INGESTION_KEY}`).
        - `debug`: For detailed logging of telemetry data.
- **Pipelines**: Defines separate pipelines for traces, metrics, and logs.

### 4. `nginx`
- **Description**: Nginx service acting as a reverse proxy and load balancer.
- **Configuration**: Uses `nginx.conf` (details not inspected but typically defines upstream servers and proxy rules).
- **Ports**: Exposes port `8080` on the host, mapping to port `80` in the container.
- **Integration**:
    - `product-simulator` sends requests to `nginx`.
    - `nginx` distributes requests to the `product-service-*` instances.
    - Provides a status endpoint (`/nginx_status`) scraped by `otel-collector`.

## Data Models
- The `product-service` likely defines data models for products, which are loaded from `product-service/data.json`. The exact structure of these models would be found within the `product-service/src/models/` directory.
- Telemetry data (traces, metrics, logs) adheres to OpenTelemetry data models.

## Integration Points
- **User/Load Simulator -> Nginx**: `product-simulator` sends HTTP requests to `nginx:80`.
- **Nginx -> Product Services**: `nginx` forwards requests to one of the `product-service-*` instances based on its load-balancing configuration.
- **Services -> OTel Collector**: All instrumented services (`product-service-*`, potentially `product-simulator` if enabled) send telemetry data (traces, metrics, logs) to `otel-collector:4317` via OTLP.
- **OTel Collector -> Nginx**: `otel-collector` scrapes metrics from `nginx:80/nginx_status`.
- **OTel Collector -> Backend**: `otel-collector` exports telemetry data to an external backend, likely Signoz, as configured by `${SIGNOZ_ENDPOINT}` and `${SIGNOZ_INGESTION_KEY}`.

## Common Patterns
- **Containerization with Docker**: All components are packaged as Docker containers.
- **Orchestration with Docker Compose**: `docker-compose.yml` is used to define and manage the multi-container application.
- **Environment Variable Configuration**: Services are configured primarily through environment variables, promoting flexibility and adherence to 12-factor app principles.
- **Centralized Logging/Monitoring**: Telemetry data is routed through `otel-collector` for centralized observability.
- **Shared Code**: The `common/` directory suggests a pattern of reusing code for cross-cutting concerns across different Go services (if multiple distinct Go services were part of a larger system).

## Development Workflow
1.  **Setup**:
    *   Ensure Docker and Docker Compose are installed.
    *   Create a `.env` file from `.env.example` and populate necessary variables (e.g., `SIGNOZ_ENDPOINT`, `SIGNOZ_INGESTION_KEY`).
2.  **Running the Application**:
    *   Use `docker-compose up -d` to start all services in detached mode.
    *   Use `docker-compose logs -f [service_name]` to view logs for a specific service.
3.  **Load Testing**:
    *   Access the Locust web UI (typically `http://localhost:8089` as per `product-simulator` port mapping).
    *   Configure and start a new load test.
    *   Results are stored in `locust-simulations/results/`.
4.  **Stopping the Application**:
    *   Use `docker-compose down` to stop and remove containers, networks, and volumes (if not explicitly defined as persistent).
5.  **Development of Product Service**:
    *   Modify code in `product-service/src/`.
    *   Rebuild the service image: `docker-compose build product-service-1` (or whichever instance).
    *   Restart the services: `docker-compose up -d --no-deps product-service-1`.
6.  **Development of Load Simulator**:
    *   Modify `locustfile.py` or other files in `locust-simulations/src/`.
    *   Rebuild the simulator image: `docker-compose build product-simulator`.
    *   Restart the services: `docker-compose up -d --no-deps product-simulator`.

## Key Terminology
- **Product Service**: The core microservice handling product data and business logic.
- **Product Simulator (Locust)**: The tool used to generate load against the product service.
- **OTel Collector (OpenTelemetry Collector)**: A component that receives, processes, and exports telemetry data.
- **Nginx**: A web server used here as a reverse proxy and load balancer.
- **Signoz**: An open-source observability platform, likely the intended backend for telemetry data.
- **OTLP**: OpenTelemetry Protocol, used for transmitting telemetry data.
- **Docker Compose**: A tool for defining and running multi-container Docker applications.
- **Microservices**: An architectural style that structures an application as a collection of small, autonomous services.
- **Telemetry**: Data collected about the performance and behavior of applications (traces, metrics, logs). 