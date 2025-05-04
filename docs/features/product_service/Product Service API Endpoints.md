**Purpose:** This page details the HTTP API endpoints exposed by the `product-service`.
**Audience:** Developers, Testers, API Consumers, Students
**Prerequisites:** [Product Service Features Overview](./Product%20Service%20Features%20Overview.md)
**Related Pages:** `product-service/src/main.go`, `product-service/src/handler.go`, `product-service/src/repository.go`, [Feature Update Product Stock](./Feature%20Update%20Product%20Stock.md)

---

## 1. Overview

The service exposes a REST-like API for managing products.

*   **Framework:** Fiber (`v2`)
*   **Base Path:** `/` (relative to the service port, default `8082`)
*   **Authentication:** None currently implemented.
*   **Instrumentation:** Requests are automatically traced via `otelfiber` middleware.

---

## 2. Endpoint Definitions

| Method | Path                           | Handler Function           | Description                                    |
| :----- | :----------------------------- | :------------------------- | :--------------------------------------------- |
| `GET`  | `/health`                      | (inline in `main.go`)      | Minimal health check, returns `{"status":"ok (minimal)"}`. |
| `GET`  | `/products`                    | `handler.GetAllProducts`   | Retrieves a list of all products.              |
| `GET`  | `/products/:productID`         | `handler.GetProductByID`   | Retrieves a single product by its ID.          |
| `GET`  | `/status`                      | `handler.HealthCheck`      | Returns `{"status": "ok"}`.                  |
| `PATCH`| `/products/:productID/stock`   | `handler.UpdateProductStock` | Updates the stock level for a specific product.|
| `POST` | `/products`                    | `handler.CreateProduct`    | Creates a new product.                         |

---

## 3. Request/Response Details

*   **`GET /products`:**
    *   **Success Response (200 OK):** `Content-Type: application/json`
        ```json
        [
          { "id": "...", "name": "...", ... }, // Updated to use 'id'
          ...
        ]
        ```
        (Array of Product objects)
    *   **Error Response:** Standard Fiber error (e.g., 500 if repository read fails).

*   **`GET /products/:productID`:**
    *   **Path Parameter:** `productID` (string) - The ID of the product to retrieve.
    *   **Success Response (200 OK):** `Content-Type: application/json`
        ```json
        {
          "id": "...", "name": "...", ... }
        ```
        (Single Product object)
    *   **Error Response:** Standard Fiber error (e.g., 500 if repository read fails, or likely 404/500 if product not found by service/repository).

*   **`GET /status`:**
    *   **Success Response (200 OK):** `Content-Type: application/json`
        ```json
        { "status": "ok" }
        ```

*   **`PATCH /products/:productID/stock`:**
    *   **Path Parameter:** `productID` (string) - The ID of the product to update.
    *   **Request Body:** `Content-Type: application/json`
        ```json
        { "stock": <integer> }
        ```
        (Uses `updateStockPayload` struct internally)
    *   **Validation:** Checks for non-negative `stock` value, primarily handled in the service layer ([Feature Update Product Stock](./Feature%20Update%20Product%20Stock.md)). **// Fix Applied**
    *   **Success Response (200 OK):** `Content-Type: application/json`
        ```json
        { "status": "ok" }
        ```
    *   **Error Response:** Standard Fiber error (e.g., 400 Bad Request if body parsing fails or stock is invalid, 404/500 if product not found, 500 if repository fails).
    *   **Telemetry Note:** The underlying repository operation adds the `product.old_stock` attribute to its trace span. See [Feature Update Product Stock](./Feature%20Update%20Product%20Stock.md). **// Fix Applied**

*   **`