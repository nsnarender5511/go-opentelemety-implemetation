
**Purpose:** Explain how to run the complete application stack locally using Docker Compose for development and testing.
**Audience:** Developers, Testers, Students
**Prerequisites:** Docker and Docker Compose installed, [Quick Start](../Quick Start.md), [Building the Services](./Building the Services.md)
**Related Pages:** [docker-compose.yml](), [Configuration Management](./Configuration Management.md)

---

## 1. Overview

Docker Compose is the standard way to run all the interconnected services (product service, simulator, OTel collector, SigNoz) in an isolated environment on your local machine.

---

## 2. Starting the Stack

1.  **Navigate:** Open a terminal in the project's root directory (`Signoz_assignment`).
2.  **Run Docker Compose:**
    ```bash
docker compose up --build
    ```
    *   `up`: Creates and starts containers based on `docker-compose.yml`.
    *   `--build`: Ensures images are built first if they don't exist or if changes have occurred since the last build.
    *   You can add `-d` to run in detached (background) mode: `docker compose up --build -d`.
3.  **Wait:** Allow a minute or two for all containers to start and initialize, especially SigNoz on the first run.

---

## 3. Accessing Services

Once the stack is running:

*   **SigNoz UI:** [http://localhost:3301](http://localhost:3301)
*   **Product Service API:** Base URL `http://localhost:8082`
    *   Examples:
        *   `curl http://localhost:8082/products`
        *   `curl http://localhost:8082/health`
*   **OTel Collector Ports (Internally used by services):** `4317` (gRPC)

---

## 4. Viewing Logs

You can view the logs for any running service:

```bash
# View logs for product-service (follow new logs with -f)
docker compose logs -f product-service

# View logs for otel-collector
docker compose logs otel-collector

# View logs for the simulator
docker compose logs product-simulator

# View logs for SigNoz frontend (example)
docker compose logs frontend
```

---

## 5. Configuration

*   Environment variables for services are primarily set within the `environment:` section of each service definition in `docker-compose.yml`.
*   These variables control aspects like log levels (`LOG_LEVEL`), application environment (`ENVIRONMENT`), and OTel endpoints (`OTEL_EXPORTER_OTLP_ENDPOINT`). See [Configuration Management](./Configuration Management.md).
*   Secrets like the SigNoz ingestion key (`SIGNOZ_INGESTION_KEY`) are also managed here. **Note:** For production, consider more secure secret management.

---

## 6. Stopping the Stack

To stop and remove the containers, networks, and volumes created by `docker compose up`:

```bash
docker compose down
```

To simply stop the containers without removing them:

```bash
docker compose stop
```

---

**Last Updated:** 2024-07-30
