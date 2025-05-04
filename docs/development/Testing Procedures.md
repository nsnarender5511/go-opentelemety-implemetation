**Purpose:** Explain the testing strategies and tools used in the project, focusing on the `product-simulator`.
**Audience:** Developers, Testers
**Prerequisites:** [Running Locally with Docker Compose](./Running%20Locally%20with%20Docker%20Compose.md)
**Related Pages:** `tests/`, `docker-compose.yml`, `tests/simulate_product_service.py`

---

## 1. Overview

Testing currently focuses on **integration and load simulation** using the `product-simulator` service. **There are currently no automated unit tests** for the Go packages in `common/` or `product-service/`. Manual API testing is also possible.

---

## 2. Product Simulator (`product-simulator` service)

*   **Source Code:** `tests/simulate_product_service.py` (Python script)
*   **Purpose:** This service runs automatically when the stack is started with Docker Compose. It continuously sends various HTTP requests to the `product-service` API endpoints to simulate user traffic and generate telemetry data (traces, logs, metrics via OTel instrumentation within `product-service`).
*   **Functionality (Confirmed from `simulate_product_service.py`):**
    *   Makes `GET /products` to fetch all products.
    *   Makes `GET /products/{id}` for specific products (using IDs from the previous call).
    *   Makes `POST /products` to create new products.
    *   Makes `PATCH /products/{id}/stock` to update stock (using IDs from the previous call).
    *   Makes `GET /health` to check the health endpoint.
    *   Makes `GET /invalid-path` to test 404 handling.
    *   Includes delays between requests.
    *   Its primary role is to generate observable activity in the `product-service` and the SigNoz backend.

### Running the Simulator
The simulator is defined as a service in `docker-compose.yml` and starts automatically with `docker compose up`. It targets the `product-service` using its service name (`http://product-service:8082`).

### Observing the Simulator
You can view the simulator's activity by checking its logs:
```bash
docker compose logs -f product-simulator
```
This will show the requests it's making and any potential responses or errors it logs.

### Configuration
The target URL (`PRODUCT_SERVICE_URL`) is configured via environment variables in `docker-compose.yml`. The script itself does not appear to use other environment variables for configuration.

---

## 3. Unit Testing (Not Implemented)

*   **Status:** Currently, there are **no automated unit tests** for the Go code. **// Fix Applied**
*   **Future:** Go's built-in `testing` package *could* be used to add unit tests for functions and methods within the `common/` modules and `product-service` layers (handler, service, repository).
*   Example command if tests existed:
    ```bash
    # Example: Running tests in the common config package
    cd common/config
    go test ./...
    ```

---

## 4. Integration Testing (Manual)

Manual integration testing can be performed using tools like `curl`, Postman, or Insomnia to directly interact with the `product-service` API endpoints while the stack is running locally via Docker Compose.

---

**Last Updated:** 2024-07-30
