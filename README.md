# Demo Microservices Application (Go/Fiber)

This project demonstrates a simple microservices application built with Go and the Fiber web framework. It includes five services designed to simulate a basic e-commerce backend flow, focusing on structure and simulated interactions suitable for developer tooling demonstrations.

## Services

The application consists of the following microservices:

1.  **UserService (`user-service/`)**
    *   Responsibility: Manages user creation and retrieval.
    *   Port (Default): `8081`
    *   Endpoints:
        *   `POST /users`
        *   `GET /users/:userId`

2.  **ProductService (`product-service/`)**
    *   Responsibility: Manages product catalog viewing and stock checks.
    *   Port (Default): `8082`
    *   Endpoints:
        *   `GET /products`
        *   `GET /products/:productId`
        *   `GET /products/:productId/stock`

3.  **CartService (`cart-service/`)**
    *   Responsibility: Manages user shopping carts (add/remove items, view cart). Uses in-memory storage.
    *   Port (Default): `8083`
    *   Endpoints:
        *   `POST /carts/:userId` (Get or Create)
        *   `POST /carts/:cartId/items`
        *   `GET /carts/:cartId`
        *   `DELETE /carts/:cartId/items/:productId`
        *   `DELETE /carts/:cartId`
    *   Interactions: Calls `ProductService` to check stock.

4.  **OrderService (`order-service/`)**
    *   Responsibility: Handles order creation, validation (user, stock), persistence (simulated), and event publishing (simulated).
    *   Port (Default): `8084`
    *   Endpoints:
        *   `POST /orders`
        *   `GET /orders/:orderId`
    *   Interactions: Calls `UserService`, `ProductService`, and `CartService`.

5.  **NotificationService (`notification-service/`)**
    *   Responsibility: Simulates consuming `OrderPlaced` events (from logs/timer) and sending notifications.
    *   Port: N/A (Runs as a standalone application)
    *   Interactions: Calls `UserService` to get user details.

## Technology Stack

*   Language: Go (1.21+ recommended)
*   Web Framework: Fiber (for API services)
*   Logging: Standard `log/slog`
*   Database: GORM with SQLite driver (`UserService`, `ProductService`, `CartService`, `OrderService`). Each service manages its own `data/<service>.db` file.
*   Inter-service Communication: Simulated via standard Go `net/http` client.

## Structure

Each API service (`user`, `product`, `cart`, `order`) follows a layered architecture:

*   `main.go`: Entry point, Fiber setup, DI, routing.
*   `handler.go`: HTTP request/response handling.
*   `service.go`: Business logic and orchestration.
*   `repository.go`: Data access layer (simulated).
*   `model.go`: Data structures.
*   `go.mod`/`go.sum`: Go module files.

The `notification-service` is a simpler standalone Go application.

## Running the Services

Each service is independent and needs to be run separately.

1.  **Navigate to a service directory:**
    ```bash
    cd user-service
    # or product-service, cart-service, etc.
    ```
2.  **Ensure dependencies are present:**
    ```bash
    go mod tidy
    ```
3.  **Run the service:**
    ```bash
    go run .
    # or go run *.go
    ```

Repeat these steps for each of the five services in separate terminal windows/tabs.

**Configuration:**

*   **Ports:** Services run on default ports (8081-8084). You can override these using environment variables (e.g., `USER_SERVICE_PORT=9081`).
*   **Service URLs:** Services that call others (`CartService`, `OrderService`, `NotificationService`) expect the URLs of their dependencies. Default `localhost` URLs are used if environment variables (`PRODUCT_SERVICE_URL`, `USER_SERVICE_URL`, `CART_SERVICE_URL`) are not set. See the `New...Service` functions for defaults and variable names.

## Simulation Notes

*   **Persistence:** `UserService`, `ProductService`, `CartService`, and `OrderService` now use GORM with SQLite for persistence. Database files (`data/<service>.db`) are created within each service's directory (or container filesystem when running with Docker). Data is persistent between runs *if run locally*, but **will be lost if the Docker container is removed** unless volumes are explicitly mounted.
*   **Cart Storage:** The `CartService` now uses SQLite for persistence via GORM, replacing the previous in-memory approach.
*   **Events:** The `OrderService` simulates publishing an `OrderPlaced` event by logging. The `NotificationService` simulates consuming this event via a timer.
*   **Inter-service Calls:** Direct HTTP calls are made between services using Go's standard library. Real-world scenarios might involve service discovery, load balancing, more robust clients, or asynchronous communication.

## Running with Docker

Each service can be built and run as a Docker container.

1.  **Build an image:**
    Navigate to the service directory (e.g., `user-service`) and run:
    ```bash
    docker build -t <your-dockerhub-username>/<service-name>:latest .
    # Example: docker build -t myuser/user-service:latest .
    ```
    Repeat for each service (`user-service`, `product-service`, `cart-service`, `order-service`, `notification-service`).

2.  **Create a Docker Network:**
    For services to communicate using their names, create a dedicated Docker network:
    ```bash
    docker network create demo-net
    ```

3.  **Run the containers:**
    Run each container, attaching it to the network. Expose ports for the API services.

    ```bash
    # User Service
    docker run -d --name user-service --network demo-net -p 8081:8081 myuser/user-service:latest

    # Product Service
    docker run -d --name product-service --network demo-net -p 8082:8082 myuser/product-service:latest

    # Cart Service
    docker run -d --name cart-service --network demo-net -p 8083:8083 \
      -e PRODUCT_SERVICE_URL=http://product-service:8082 \
      myuser/cart-service:latest

    # Order Service
    docker run -d --name order-service --network demo-net -p 8084:8084 \
      -e USER_SERVICE_URL=http://user-service:8081 \
      -e PRODUCT_SERVICE_URL=http://product-service:8082 \
      -e CART_SERVICE_URL=http://cart-service:8083 \
      myuser/order-service:latest

    # Notification Service (no port mapping needed)
    docker run -d --name notification-service --network demo-net \
      -e USER_SERVICE_URL=http://user-service:8081 \
      myuser/notification-service:latest
    ```
    *(Remember to replace `myuser` with your Docker Hub username if you plan to push the images)*

4.  **View Logs:**
    ```bash
    docker logs <container-name>
    # Example: docker logs order-service
    ```

5.  **Stopping Containers:**
    ```bash
    docker stop user-service product-service cart-service order-service notification-service
    docker rm user-service product-service cart-service order-service notification-service
    docker network rm demo-net
    ```

**Note:** Using Docker Compose is recommended for managing multi-container applications like this more easily. A `docker-compose.yml` file could simplify the build and run steps significantly. 