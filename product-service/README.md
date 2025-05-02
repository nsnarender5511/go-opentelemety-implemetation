# Product Service (Simplified)

This is a simplified version of the Product Service, refactored to focus solely on core API functionality using Go Fiber.

## Overview

*   **Framework:** Go Fiber
*   **Persistence:** Reads initial product data from `data.json` at startup. Updates are persisted back to `data.json`.
*   **Dependencies:** Minimal external dependencies (primarily Fiber).
*   **Features Removed:** OpenTelemetry (Tracing, Metrics, Logging), complex configuration, advanced error handling, custom middleware, graceful shutdown.

## Prerequisites

*   Go 1.24 or later
*   A `data.json` file in the `product-service` directory (an example is provided).

## Building

From the project root directory (`Signoz_assignment`):

```bash
# Tidy dependencies (if needed)
cd product-service
go mod tidy
cd ..

# Build the binary
go build -o product-service/product-service-app ./product-service/src
```

Alternatively, build using Docker (from the project root):

```bash
docker build -t product-service-app -f product-service/Dockerfile .
```

## Running

### Locally

Ensure `data.json` exists in the `product-service` directory.

From the project root directory:

```bash
./product-service/product-service-app
```

The service will start and listen on port `8080` by default.

### Using Docker

From the project root directory:

```bash
# Ensure data.json exists in product-service/
# Use docker run or docker-compose

# Example docker run (mounts data.json)
docker run --rm -p 8080:8080 -v $(pwd)/product-service/data.json:/app/data.json:ro product-service-app

# Or use the provided docker-compose.yml
docker-compose up product-service
```

## API Endpoints

The service exposes the following REST API endpoints under the `/api/v1` prefix:

*   **`GET /api/v1/products`**
    *   Retrieves a list of all available products.
    *   **Success Response (200 OK):** Array of product objects.
    *   **Error Response (500 Internal Server Error):** If there's an issue retrieving products.

*   **`GET /api/v1/products/{productId}`**
    *   Retrieves a specific product by its ID.
    *   **Success Response (200 OK):** Single product object.
    *   **Error Response (404 Not Found):** If the product ID does not exist.
    *   **Error Response (500 Internal Server Error):** If there's an internal issue.

*   **`GET /api/v1/healthz`**
    *   Simple health check endpoint.
    *   **Success Response (200 OK):** `{"status":"ok"}` 