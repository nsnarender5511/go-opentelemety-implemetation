# Building the Services

**Purpose:** Explain how to build the container images for the project services.
**Audience:** Developers
**Prerequisites:** Docker installed, ./Running_Locally_with_Docker_Compose.md
**Related Pages:** [`docker-compose.yml`](../../docker-compose.yml), `product-service/Dockerfile`, `tests/Dockerfile`

---

## 1. Overview

The primary method for building the services is using Docker Compose, which orchestrates the builds based on the `Dockerfile` definitions for each service.

---

## 2. Building with Docker Compose

To build all service images defined in `docker-compose.yml` without starting the containers:

```bash
# Navigate to the project root directory
cd /path/to/Signoz_assignment

# Build all services
docker compose build
```

To build a specific service (e.g., `product-service`):

```bash
docker compose build product-service
```

*   Docker Compose reads the `build:` context and `Dockerfile` specified for each service in `docker-compose.yml`.
*   It caches layers, so subsequent builds are typically faster unless code or dependencies have changed.
*   The `docker compose up --build` command (used in ../Quick_Start.md and ./Running_Locally_with_Docker_Compose.md) also performs a build automatically if images are missing or outdated before starting the containers.

---

## 3. Dockerfiles

*   **`product-service/Dockerfile`:** Defines a multi-stage build for the Go service. It first builds the binary in a Go build environment (using `golang:1.24-alpine`), copies necessary files (`go.work`, `go.mod`, source), downloads dependencies, builds the executable, and copies it along with `data.json` into the final stage. The current version uses the builder stage directly for easier debugging, with the lightweight final stage commented out.
*   **`tests/Dockerfile`:** Defines the image for the Python-based `product-simulator`. It copies the simulator script (`simulate_product_service.py`) and installs dependencies from `tests/requirements.txt`.
    *   **Dependency File:** A `tests/requirements.txt` file **must exist** in the `tests/` directory for the build to succeed. The simulator script (`simulate_product_service.py`) requires the `requests` library. Ensure `requirements.txt` contains at least: **// Fix Applied**
        ```txt
        requests>=2.0.0 # Example versioning
        ```

---

## 4. Manual Go Building (Optional)

While not the primary workflow, you can manually build the Go services locally if needed (requires Go installed):

```bash
# Build product-service
cd product-service
go build -o ../bin/product-service ./src/main.go
cd ..

# Build shared modules (if needed for testing outside Docker)
cd common
go build ./...
cd ..
```

This is generally not required when using the Docker-based workflow.

---

**Last Updated:** 2024-07-30
