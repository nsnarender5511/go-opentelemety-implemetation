# 🚀 Simulation Module Revamp Plan Using Locust 🚀

**🎯 Goal:** Create a new simulation framework using Locust that is modular, configurable, and accurately reflects the `product-service` API. This includes diverse scenarios, comprehensive error condition handling, and support for concurrent load generation. This plan replaces the implementation in the current `simulations/src/` directory.

---
## ⓪ **Phase 0: Foundation & API Deep Dive (Confirmation)** 🕵️‍♀️
---

*   **🌟 Objective:** Achieve an exhaustive understanding of the `product-service` API to ensure the simulation accurately models its behavior. This is crucial *before* writing new Locust-based simulation code.
*   **Tasks:**
    1.  **Full API Surface Mapping:**
        *   **How:** Manually review the `product-service/src/handlers/` directory, corresponding service methods in `product-service/src/services/`, and data structures in `product-service/src/models/` as well as the shared `common/apirequests/`, `common/apiresponses/`, `common/apierrors/`, and `common/validator/` packages.
        *   **What to Document (for each endpoint):**
            *   HTTP Method (e.g., `GET`, `POST`, `PATCH`).
            *   Full Path (e.g., `/products`, `/products/stock`).
            *   Request Payload Structure: For `POST`/`PATCH`, detail all JSON fields, their data types (string, int, bool, nested objects), and any validation rules (e.g., required, min/max length, format). This is evident from Go struct tags in `common/apirequests/` and usage of `common/validator/`.
            *   Query Parameters: For `GET`, detail all parameters, types, and if they are optional/required (e.g., `GET /products/category?category=Electronics`).
            *   Success Response: Status code (e.g., `200 OK`, `201 Created`), and JSON payload structure (from `common/apiresponses/`).
            *   Error Responses: Document the various error responses (validation errors, not found errors, business logic errors, server errors) and how they are structured.
        *   **Where:** This documentation could be a new Markdown file (e.g., `product_service_api_spec.md`) or integrated into this plan.
        *   **Why:** This forms the blueprint for all subsequent Locust task development, ensuring our load test scenarios are based on the actual service behavior.
        *   **When:** This is the absolute first step.

    2.  **Identify Key Scenarios & User Flows:**
        *   **How:** Analyze the existing `simulations/src/actions.py` and `simulations/src/simulate.py` to identify the current scenarios being tested. Then expand and improve upon them based on knowledge of the service.
        *   **Examples:**
            *   **Happy Paths:**
                *   Fetch all products.
                *   Fetch products by a valid, existing category.
                *   Fetch details for a valid, existing product name.
                *   Successfully buy an in-stock product.
                *   Successfully update stock for an existing product.
            *   **Expected "Negative" Paths (Graceful Failures):**
                *   Fetch products by a non-existent category (expect empty list).
                *   Fetch details for a non-existent product name (expect 404).
                *   Attempt to buy a product that doesn't exist (expect 404 or relevant business error).
                *   Attempt to buy a product with insufficient stock (expect specific error, e.g., 409 or 400 with error code).
                *   Send invalid data (e.g., missing required field in `POST /products/buy`, negative quantity) and verify 400 validation errors.
                *   Hit an deliberately invalid API path (expect 404).
            *   **Concurrent Operations:** 
                *   Multiple users trying to buy the same product simultaneously.
                *   Updating stock while other users are reading/buying.
        *   **Where:** Document these scenarios in this plan or a dedicated scenarios document.
        *   **Why:** These scenarios will be translated into Locust tasks, ensuring comprehensive test coverage.
        *   **When:** After API mapping is mostly complete.

---
## ① **Phase 1: Locust Installation & Configuration** ⚙️
---

*   **🌟 Objective:** Set up Locust as the load testing framework and establish a flexible configuration system.
*   **Tasks:**
    1.  **Install Locust:**
        *   **How:** Update `simulations/requirements.txt` to include Locust and other necessary dependencies.
            ```
            # simulations/requirements.txt
            locust>=2.15.1
            PyYAML>=6.0
            ```
        *   **Why:** Locust will replace the custom threading and request management code with a proven, feature-rich framework.
        *   **When:** First step of implementation.

    2.  **Create Configuration File (`simulations/config.yaml`):**
        *   **How:** Create a YAML file that contains configuration options for Locust runs. Locust has its own CLI options, but we'll use this file for settings specific to our scenarios.
        *   **Why:** Provides a consistent way to configure test specifics across different environments.
        *   **When:** Right after Locust installation, before writing scenarios.
        *   **Example (`simulations/config.yaml`):**
            ```yaml
            # Service Configuration
            service:
              base_url: "http://localhost:8082"  # Target product-service URL (can be overridden by Locust CLI)
              request_timeout_seconds: 10

            # Test Data
            test_data:
              non_existent_product_name: "ProductThatWillNeverExist123"
              possible_categories:
                - "Electronics" 
                - "Apparel"
                - "Books"
                - "Kitchenware"
                - "Furniture"
                - "NonExistentCategory"  # For testing empty response

            # Scenario Weights (used within HttpUser classes to determine task frequency)
            scenario_weights:
              browse_all_products: 10
              get_products_by_category: 8
              get_product_details: 10
              buy_product: 7
              update_stock: 3
              hit_invalid_path: 1
              health_check: 2
            ```

    3.  **Configuration Loader:**
        *   **How:** Create a Python module to load and parse the configuration.
            ```python
            # simulations/config_loader.py
            import yaml
            import os
            import logging

            def load_config(config_path="config.yaml"):
                """Load configuration from the specified YAML file."""
                try:
                    if not os.path.exists(config_path):
                        logging.warning(f"Config file {config_path} not found, using defaults.")
                        return {
                            "service": {"base_url": "http://localhost:8082", "request_timeout_seconds": 10},
                            "test_data": {
                                "non_existent_product_name": "ProductThatWillNeverExist123",
                                "possible_categories": ["Electronics", "Apparel", "Books", "Kitchenware", "Furniture", "NonExistentCategory"]
                            },
                            "scenario_weights": {
                                "browse_all_products": 10,
                                "get_products_by_category": 8,
                                "get_product_details": 10,
                                "buy_product": 7,
                                "update_stock": 3,
                                "hit_invalid_path": 1,
                                "health_check": 2
                            }
                        }

                    with open(config_path, 'r') as f:
                        return yaml.safe_load(f)
                except Exception as e:
                    logging.error(f"Error loading config file: {e}")
                    raise
            ```
        *   **Where:** `simulations/config_loader.py`.
        *   **Why:** Provides a simple way to access configuration across all Locust task files.
        *   **When:** Implement after creating the configuration file structure.

---
## ② **Phase 2: Core Locust Implementation** 🦗
---

*   **🌟 Objective:** Create the core Locust test script that will handle users, tasks, and shared state.
*   **Tasks:**
    1.  **Main Locust File:**
        *   **How:** Create a `locustfile.py` in the `simulations` directory. This will be the entry point for Locust.
        *   **Why:** Locust requires a `locustfile.py` as its entry point. This file will define user types and their behaviors.
        *   **When:** After configuration is complete.
        *   **Example (`simulations/locustfile.py`):**
            ```python
            import logging
            import os
            import random
            from typing import List, Dict, Any, Optional
            
            from locust import HttpUser, task, between, events, tag
            from locust.clients import ResponseContextManager
            
            from config_loader import load_config
            from shared_data import SharedData
            
            # Load configuration
            config = load_config()
            
            # Set up logging
            logging.basicConfig(
                level=logging.INFO,
                format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
            )
            logger = logging.getLogger("product-service-simulation")
            
            # Initialize shared data provider for product information
            shared_data = SharedData()
            
            class ProductServiceUser(HttpUser):
                """Base user class for simulating interactions with the product service."""
                wait_time = between(1, 3)  # Wait between 1-3 seconds between tasks
                
                # Weights from config
                weights = config.get("scenario_weights", {})
                
                def on_start(self):
                    """Initialize user session."""
                    # Use shared data instance for all users
                    self.shared_data = shared_data
                    # Ensure we have product data before starting tasks
                    if not self.shared_data.get_products():
                        with self.client.get("/products", name="Initial_Fetch_Products") as response:
                            if response.status_code == 200:
                                products = self._extract_products(response)
                                if products:
                                    self.shared_data.update_products(products)
                
                def _extract_products(self, response: ResponseContextManager) -> List[Dict[str, Any]]:
                    """Extract valid products from API response."""
                    try:
                        data = response.json()
                        if isinstance(data, dict) and "data" in data:
                            products_list = data["data"]
                        elif isinstance(data, list):
                            products_list = data
                        else:
                            logger.warning(f"Unexpected response structure from GET /products: {type(data)}")
                            return []
                        
                        # Filter for valid product dictionaries
                        return [
                            item for item in products_list 
                            if isinstance(item, dict) and "productID" in item and "name" in item
                        ]
                    except Exception as e:
                        logger.error(f"Error extracting products: {e}")
                        return []
                
                @task(weights.get("browse_all_products", 10))
                @tag("browse")
                def browse_all_products(self):
                    """Browse all available products."""
                    with self.client.get("/products", name="Get_All_Products") as response:
                        if response.status_code == 200:
                            products = self._extract_products(response)
                            if products:
                                self.shared_data.update_products(products)
                                logger.debug(f"Found {len(products)} products")
                
                @task(weights.get("get_products_by_category", 8))
                @tag("browse", "category")
                def get_products_by_category(self):
                    """Search products by category."""
                    category = random.choice(config["test_data"]["possible_categories"])
                    with self.client.get(
                        f"/products/category",
                        params={"category": category},
                        name=f"Get_Products_By_Category_{category}"
                    ) as response:
                        if response.status_code == 200:
                            products = self._extract_products(response)
                            logger.debug(f"Found {len(products)} products in category '{category}'")
                
                @task(weights.get("get_product_details", 10))
                @tag("details")
                def get_product_details(self):
                    """Get details for a specific product by name."""
                    product = self.shared_data.get_random_product()
                    if not product:
                        logger.debug("No products available for details lookup")
                        return
                    
                    with self.client.post(
                        "/products/details",
                        json={"name": product["name"]},
                        name=f"Get_Product_Details"
                    ) as response:
                        if response.status_code == 200:
                            logger.debug(f"Got details for product: {product['name']}")
                
                @task(weights.get("buy_product", 7))
                @tag("purchase")
                def buy_product(self):
                    """Purchase a product."""
                    product = self.shared_data.get_random_product()
                    if not product:
                        logger.debug("No products available for purchase")
                        return
                    
                    quantity = random.randint(1, 5)
                    with self.client.post(
                        "/products/buy",
                        json={"name": product["name"], "quantity": quantity},
                        name="Buy_Product"
                    ) as response:
                        if response.status_code == 200:
                            logger.debug(f"Successfully bought {quantity} of {product['name']}")
                        else:
                            logger.warning(f"Failed to buy {quantity} of {product['name']}: {response.status_code}")
                
                @task(weights.get("update_stock", 3))
                @tag("admin")
                def update_stock(self):
                    """Update stock for a product."""
                    product = self.shared_data.get_random_product()
                    if not product:
                        logger.debug("No products available for stock update")
                        return
                    
                    new_stock = random.randint(0, 100)
                    with self.client.patch(
                        "/products/stock",
                        json={"name": product["name"], "stock": new_stock},
                        name="Update_Product_Stock"
                    ) as response:
                        if response.status_code == 200:
                            logger.debug(f"Updated stock for {product['name']} to {new_stock}")
                
                @task(weights.get("hit_invalid_path", 1))
                @tag("error")
                def hit_invalid_path(self):
                    """Hit an invalid API path to test error handling."""
                    import uuid
                    path = f"some/invalid/path/{uuid.uuid4()}"
                    with self.client.get(path, name="Invalid_Path") as response:
                        logger.debug(f"Invalid path returned {response.status_code}")
                
                @task(weights.get("health_check", 2))
                @tag("health")
                def health_check(self):
                    """Perform a health check."""
                    with self.client.get("/health", name="Health_Check") as response:
                        logger.debug(f"Health check returned {response.status_code}")
                
                # Specialized tasks for negative testing could be added here
                # @task(1)
                # def buy_non_existent_product(self):
                #     """Try to buy a product that doesn't exist."""
                #     ...
            ```

    2.  **Shared Data Manager:**
        *   **How:** Create a thread-safe data provider for sharing product data between tasks.
        *   **Why:** Tasks need to share data, such as the list of available products, to make informed API calls.
        *   **When:** Implement alongside the main Locust file.
        *   **Example (`simulations/shared_data.py`):**
            ```python
            import threading
            import random
            from typing import List, Dict, Any, Optional

            class SharedData:
                """Thread-safe container for shared simulation data."""
                
                def __init__(self):
                    self._lock = threading.RLock()
                    self._products = []
                    self._last_product_update = 0
                
                def update_products(self, products: List[Dict[str, Any]]) -> None:
                    """Thread-safe update of the products list."""
                    with self._lock:
                        self._products = products
                        self._last_product_update = time.time()
                
                def get_products(self) -> List[Dict[str, Any]]:
                    """Get all known products (thread-safe)."""
                    with self._lock:
                        return self._products.copy()
                
                def get_random_product(self) -> Optional[Dict[str, Any]]:
                    """Get a random product (thread-safe)."""
                    with self._lock:
                        if not self._products:
                            return None
                        return random.choice(self._products)
                
                def get_product_by_name(self, name: str) -> Optional[Dict[str, Any]]:
                    """Get a product by name (thread-safe)."""
                    with self._lock:
                        for product in self._products:
                            if product.get("name") == name:
                                return product
                        return None
            ```

    3.  **Specialized User Types (Optional):**
        *   **How:** Create multiple specialized user classes for different behavior patterns.
        *   **Why:** Simulates different types of users with distinct usage patterns.
        *   **When:** After the main user class is working properly.
        *   **Example (extending `locustfile.py`):**
            ```python
            class BrowserUser(ProductServiceUser):
                """User that primarily browses products without buying."""
                weight = 3  # 3x more browser users than admin users
                
                # Override task weights to focus on browsing
                browse_all_products = task(20)(ProductServiceUser.browse_all_products)
                get_products_by_category = task(15)(ProductServiceUser.get_products_by_category)
                get_product_details = task(10)(ProductServiceUser.get_product_details)
                # Keep other tasks with minimal weight
                buy_product = task(1)(ProductServiceUser.buy_product)
                update_stock = task(0)(ProductServiceUser.update_stock)  # Never update stock
                hit_invalid_path = task(1)(ProductServiceUser.hit_invalid_path)
                health_check = task(1)(ProductServiceUser.health_check)
            
            class AdminUser(ProductServiceUser):
                """User that performs administrative tasks like stock updates."""
                weight = 1  # Fewer admin users
                
                # Override task weights to focus on admin tasks
                update_stock = task(20)(ProductServiceUser.update_stock)
                browse_all_products = task(5)(ProductServiceUser.browse_all_products) 
                get_products_by_category = task(2)(ProductServiceUser.get_products_by_category)
                get_product_details = task(5)(ProductServiceUser.get_product_details)
                buy_product = task(0)(ProductServiceUser.buy_product)  # Admins don't buy
                hit_invalid_path = task(1)(ProductServiceUser.hit_invalid_path)
                health_check = task(5)(ProductServiceUser.health_check)
            ```

---
## ③ **Phase 3: Advanced Locust Features** 🔍
---

*   **🌟 Objective:** Enhance the basic Locust implementation with advanced features to better simulate real-world behavior and provide more actionable feedback.
*   **Tasks:**
    1.  **Custom Load Shape:**
        *   **How:** Implement a custom load shape to control user count over time.
        *   **Why:** Creates more realistic test scenarios such as ramp-up, sustained load, and ramp-down.
        *   **When:** After basic implementation is working.
        *   **Example (`simulations/custom_load_shape.py`):**
            ```python
            from locust import LoadTestShape

            class StagesLoadShape(LoadTestShape):
                """Custom load shape with stages for ramp-up, plateau, and ramp-down."""
                
                stages = [
                    {"duration": 60, "users": 5, "spawn_rate": 5},   # Ramp up to 5 users over 60 seconds
                    {"duration": 120, "users": 10, "spawn_rate": 5},  # Ramp up to 10 users over next 120 seconds
                    {"duration": 600, "users": 10, "spawn_rate": 5},  # Stay at 10 users for 600 seconds
                    {"duration": 120, "users": 0, "spawn_rate": 5},   # Ramp down to 0 users over 120 seconds
                ]
                
                def tick(self):
                    run_time = self.get_run_time()
                    
                    # Calculate current stage and time within the stage
                    current_stage_time = run_time
                    for stage in self.stages:
                        if current_stage_time < stage["duration"]:
                            return stage["users"], stage["spawn_rate"]
                        current_stage_time -= stage["duration"]
                    
                    # All stages complete, test is done
                    return None
            ```
            
            Add to `locustfile.py`:
            ```python
            # At the top, add:
            from custom_load_shape import StagesLoadShape

            # At the bottom of the file, add:
            class MyLoadTestShape(StagesLoadShape):
                # Can customize the stages here if needed
                pass
            ```

    2.  **Customized Reports and Event Hooks:**
        *   **How:** Use Locust's event hooks to capture additional data and generate custom reports.
        *   **Why:** Provides more detailed insights into API behavior and allows customized metrics.
        *   **When:** After the basic test is implemented and working.
        *   **Example (`simulations/event_hooks.py`):**
            ```python
            import time
            import csv
            from locust import events
            
            # Track custom metrics
            custom_stats = {
                "stock_updates": 0,
                "failed_purchases": 0,
                "successful_purchases": 0
            }
            
            @events.request.add_listener
            def request_handler(request_type, name, response_time, response_length, exception, **kwargs):
                """Track custom metrics based on request type."""
                if "Update_Product_Stock" in name and not exception:
                    custom_stats["stock_updates"] += 1
                elif "Buy_Product" in name:
                    if exception or (hasattr(kwargs.get("response", {}), "status_code") and kwargs["response"].status_code != 200):
                        custom_stats["failed_purchases"] += 1
                    else:
                        custom_stats["successful_purchases"] += 1
            
            @events.test_stop.add_listener
            def on_test_stop(environment, **kwargs):
                """Generate custom reports when test finishes."""
                # Generate custom CSV report
                with open("custom_report.csv", "w", newline="") as f:
                    writer = csv.writer(f)
                    writer.writerow(["Metric", "Value"])
                    for key, value in custom_stats.items():
                        writer.writerow([key, value])
                
                print("\n--- Custom Statistics ---")
                for key, value in custom_stats.items():
                    print(f"{key}: {value}")
            ```
            
            Add to `locustfile.py`:
            ```python
            # At the top, add:
            import event_hooks  # The import alone will register the hooks
            ```

    3.  **Data-Driven Testing:**
        *   **How:** Extend tests to use pre-loaded test data for specific scenarios.
        *   **Why:** Allows repeatable testing of edge cases or specific business scenarios.
        *   **When:** After basic functionality is solid.
        *   **Example (`simulations/test_data.py`):**
            ```python
            """Pre-defined test data for specific scenarios."""
            
            # Edge cases for product testing
            EDGE_CASE_PRODUCTS = [
                {"name": "Zero Stock Product", "quantity": 1},  # Should fail due to stock
                {"name": "Laptop Pro", "quantity": 1000},       # Very large order
                {"name": "Coffee Mug", "quantity": 0},          # Zero quantity
                {"name": "NonExistentProduct", "quantity": 1}   # Non-existent product
            ]
            
            # Categories with expected results
            CATEGORIES = [
                # Category name, expected min items, max items
                ("Electronics", 1, 100),
                ("NonExistentCategory", 0, 0),  # Should return empty list
                ("Furniture", 1, 100),
            ]
            ```
            
            Using this in tasks:
            ```python
            @task(1)
            @tag("edge_case")
            def test_edge_case_purchases(self):
                """Test various edge cases for purchases."""
                from test_data import EDGE_CASE_PRODUCTS
                
                edge_case = random.choice(EDGE_CASE_PRODUCTS)
                with self.client.post(
                    "/products/buy",
                    json={"name": edge_case["name"], "quantity": edge_case["quantity"]},
                    name=f"Edge_Case_Purchase_{edge_case['name']}_{edge_case['quantity']}"
                ) as response:
                    # The response may be a 4xx error in some cases, which is expected
                    logger.info(f"Edge case purchase: {edge_case['name']}, qty: {edge_case['quantity']}, " 
                                f"result: {response.status_code}")
            ```

---
## ④ **Phase 4: Test Verification & Assertions** ✅
---

*   **🌟 Objective:** Add response validation to ensure API behavior is correct, not just that calls complete successfully.
*   **Tasks:**
    1.  **Response Validation Framework:**
        *   **How:** Create utility functions to validate response content and structure.
        *   **Why:** Ensures the API is returning the expected data structure and content, not just HTTP 200.
        *   **When:** Once basic Locust implementation is working.
        *   **Example (`simulations/validators.py`):**
            ```python
            import logging
            from typing import Dict, Any, List, Optional, Callable, Union

            logger = logging.getLogger("validators")

            def validate_response(response, checks: List[Callable]) -> bool:
                """Run a series of validation checks on a response."""
                success = True
                for check in checks:
                    try:
                        if not check(response):
                            success = False
                    except Exception as e:
                        logger.error(f"Validation error: {e}")
                        success = False
                return success
            
            def check_status_code(expected_code: int) -> Callable:
                """Create a validator for status code."""
                def _check(response) -> bool:
                    if response.status_code != expected_code:
                        logger.warning(f"Expected status {expected_code}, got {response.status_code}")
                        return False
                    return True
                return _check
            
            def check_content_type(expected_type: str = "application/json") -> Callable:
                """Create a validator for content type."""
                def _check(response) -> bool:
                    content_type = response.headers.get('Content-Type', '')
                    if expected_type not in content_type:
                        logger.warning(f"Expected content type {expected_type}, got {content_type}")
                        return False
                    return True
                return _check
            
            def check_json_contains(expected_fields: List[str]) -> Callable:
                """Create a validator that checks if JSON response contains required fields."""
                def _check(response) -> bool:
                    try:
                        data = response.json()
                        for field in expected_fields:
                            if field not in data:
                                logger.warning(f"Expected field '{field}' not found in response")
                                return False
                        return True
                    except Exception as e:
                        logger.warning(f"JSON parsing error: {e}")
                        return False
                return _check
            
            def check_product_schema(response) -> bool:
                """Check if response contains a valid product schema."""
                try:
                    product = response.json()
                    required_fields = ["productID", "name", "price", "stock"]
                    for field in required_fields:
                        if field not in product:
                            logger.warning(f"Product missing required field: {field}")
                            return False
                    return True
                except Exception as e:
                    logger.warning(f"Product schema validation error: {e}")
                    return False
            ```

    2.  **Integrate Validators with Tasks:**
        *   **How:** Update the task methods to use these validators.
        *   **Why:** Makes test failures more informative by validating response quality.
        *   **When:** After the validation framework is created.
        *   **Example update for `get_product_details` task:**
            ```python
            @task(weights.get("get_product_details", 10))
            @tag("details")
            def get_product_details(self):
                """Get details for a specific product by name with validation."""
                from validators import validate_response, check_status_code, check_content_type, check_product_schema
                
                product = self.shared_data.get_random_product()
                if not product:
                    logger.debug("No products available for details lookup")
                    return
                
                with self.client.post(
                    "/products/details",
                    json={"name": product["name"]},
                    name=f"Get_Product_Details"
                ) as response:
                    # Validate the response
                    is_valid = validate_response(response, [
                        check_status_code(200),
                        check_content_type("application/json"),
                        check_product_schema
                    ])
                    
                    if is_valid:
                        logger.debug(f"Got valid details for product: {product['name']}")
                    else:
                        logger.warning(f"Invalid response for product details: {product['name']}")
            ```

---
## ⑤ **Phase 5: Test Execution & CI/CD Integration** 🔄
---

*   **🌟 Objective:** Set up flexible ways to run the Locust tests, including both UI mode and headless mode, and integrate with CI/CD pipelines.
*   **Tasks:**
    1.  **Command-Line Test Runner:**
        *   **How:** Create scripts to run Locust in different modes (UI, headless, distributed).
        *   **Why:** Makes it easy to execute tests consistently in different environments.
        *   **When:** After the Locust implementation is working.
        *   **Example (`simulations/run_tests.sh`):**
            ```bash
            #!/bin/bash
            
            # Simple script to run Locust in different modes
            # Usage: ./run_tests.sh [MODE] [USERS] [DURATION]
            # MODE: ui (default) or headless
            # USERS: number of users to simulate (default: 10)
            # DURATION: duration in seconds (default: 300)
            
            MODE=${1:-ui}
            USERS=${2:-10}
            DURATION=${3:-300}
            HOST=${4:-http://localhost:8082}
            
            echo "Running tests in $MODE mode with $USERS users for $DURATION seconds against $HOST"
            
            case $MODE in
              ui)
                # Start Locust with web UI
                cd "$(dirname "$0")" && locust -f locustfile.py --host=$HOST
                ;;
              headless)
                # Run Locust in headless mode
                cd "$(dirname "$0")" && locust -f locustfile.py --headless -u $USERS -t ${DURATION}s --host=$HOST --html=report.html
                echo "Test complete, results in report.html"
                ;;
              *)
                echo "Unknown mode: $MODE. Use 'ui' or 'headless'."
                exit 1
                ;;
            esac
            ```

    2.  **Docker Support:**
        *   **How:** Update the Dockerfile to support running Locust in different modes.
        *   **Why:** Provides a consistent environment for running tests.
        *   **When:** After the basic Locust implementation is working.
        *   **Example (Updated `simulations/Dockerfile`):**
            ```dockerfile
            FROM python:3.10-slim
            
            WORKDIR /app
            
            # Copy requirements and install dependencies
            COPY requirements.txt .
            RUN pip install --no-cache-dir -r requirements.txt
            
            # Copy test files
            COPY *.py .
            COPY config.yaml .
            COPY run_tests.sh .
            
            # Make run script executable
            RUN chmod +x run_tests.sh
            
            # Expose Locust web interface port
            EXPOSE 8089
            
            # Default command starts Locust in web UI mode
            CMD ["./run_tests.sh", "ui"]
            ```

    3.  **CI/CD Integration Examples:**
        *   **How:** Create example configurations for running Locust in CI pipelines.
        *   **Why:** Enables automated performance testing in CI/CD workflows.
        *   **When:** After the basic tests are working reliably.
        *   **Example (GitHub Actions Workflow - `.github/workflows/performance.yml`):**
            ```yaml
            name: Performance Tests
            
            on:
              workflow_dispatch:
              schedule:
                - cron: '0 0 * * 1'  # Weekly on Mondays
            
            jobs:
              performance-test:
                runs-on: ubuntu-latest
                
                steps:
                  - name: Checkout code
                    uses: actions/checkout@v3
                    
                  - name: Set up Python
                    uses: actions/setup-python@v4
                    with:
                      python-version: '3.10'
                      
                  - name: Install dependencies
                    run: |
                      cd simulations
                      pip install -r requirements.txt
                      
                  - name: Start product-service (in background)
                    run: |
                      # Start your service here
                      docker-compose up -d product-service
                      # Wait for service to be ready
                      sleep 10
                      
                  - name: Run performance tests
                    run: |
                      cd simulations
                      locust -f locustfile.py --headless -u 10 -t 60s --host=http://localhost:8082 --html=report.html
                      
                  - name: Upload test results
                    uses: actions/upload-artifact@v3
                    with:
                      name: performance-report
                      path: simulations/report.html
            ```

---
## ⑥ **Phase 6: Monitoring & Analysis** 📊
---

*   **🌟 Objective:** Enhance the ability to monitor test execution and analyze results.
*   **Tasks:**
    1.  **Integration with External Monitoring:**
        *   **How:** Use Locust's event hooks to send metrics to external systems.
        *   **Why:** Allows long-term tracking of performance metrics and integration with existing monitoring.
        *   **When:** After the basic tests are working reliably.
        *   **Example (`simulations/monitoring.py` - for integration with Prometheus/InfluxDB):**
            ```python
            """Integration with external monitoring systems."""
            import time
            from locust import events
            
            # Example for InfluxDB (would require influxdb-client package)
            try:
                from influxdb_client import InfluxDBClient, Point
                from influxdb_client.client.write_api import SYNCHRONOUS
                
                # Set up InfluxDB client (would come from config)
                INFLUXDB_URL = "http://localhost:8086"
                INFLUXDB_TOKEN = "your-token"
                INFLUXDB_ORG = "your-org"
                INFLUXDB_BUCKET = "locust-metrics"
                
                influx_client = InfluxDBClient(url=INFLUXDB_URL, token=INFLUXDB_TOKEN, org=INFLUXDB_ORG)
                write_api = influx_client.write_api(write_options=SYNCHRONOUS)
                
                @events.request.add_listener
                def report_to_influxdb(request_type, name, response_time, response_length, exception, **kwargs):
                    """Send request metrics to InfluxDB."""
                    if exception:
                        success = "0"
                    else:
                        success = "1"
                    
                    point = Point("locust_requests") \
                        .tag("request_name", name) \
                        .tag("request_type", request_type) \
                        .tag("success", success) \
                        .field("response_time", response_time) \
                        .field("response_length", response_length if response_length else 0)
                    
                    try:
                        write_api.write(bucket=INFLUXDB_BUCKET, org=INFLUXDB_ORG, record=point)
                    except Exception as e:
                        # Don't fail tests if monitoring fails
                        print(f"Error reporting to InfluxDB: {e}")
                
            except ImportError:
                # InfluxDB integration is optional
                pass
            ```

    2.  **Advanced Results Analysis:**
        *   **How:** Create scripts to process and visualize Locust's CSV output.
        *   **Why:** Provides deeper insights than what's available in the Locust UI.
        *   **When:** After multiple test runs have been completed.
        *   **Example (`simulations/analyze_results.py`):**
            ```python
            """Process Locust results for deeper analysis."""
            import pandas as pd
            import matplotlib.pyplot as plt
            import sys
            import os
            
            def analyze_csv(csv_file):
                """Analyze a Locust CSV result file."""
                if not os.path.exists(csv_file):
                    print(f"Error: File not found - {csv_file}")
                    return
                
                try:
                    # Load the CSV data
                    df = pd.read_csv(csv_file)
                    
                    # Basic statistics
                    print("\n=== Request Statistics ===")
                    stats = df.groupby('Name').agg({
                        'Total': 'sum',
                        'Failures': 'sum',
                        'Median': 'mean',
                        '95%': 'mean',
                        'Average': 'mean',
                    }).reset_index()
                    
                    stats['Success Rate'] = 1 - (stats['Failures'] / stats['Total'])
                    stats['Success Rate'] = stats['Success Rate'].apply(lambda x: f"{x:.2%}")
                    
                    print(stats)
                    
                    # Simple visualization
                    plt.figure(figsize=(12, 6))
                    plt.bar(stats['Name'], stats['Average'])
                    plt.xticks(rotation=45, ha='right')
                    plt.xlabel('Request Name')
                    plt.ylabel('Average Response Time (ms)')
                    plt.title('Average Response Time by Request Type')
                    plt.tight_layout()
                    plt.savefig('response_times.png')
                    print("\nGenerated visualization: response_times.png")
                    
                except Exception as e:
                    print(f"Error analyzing results: {e}")
            
            if __name__ == "__main__":
                if len(sys.argv) < 2:
                    print("Usage: python analyze_results.py <locust_csv_file>")
                    sys.exit(1)
                
                analyze_csv(sys.argv[1])
            ```

---
## 🗓️ **New Directory Structure for `simulations`**
---
```
simulations/
├── locustfile.py              # 🏁 Main Locust test script
├── config.yaml                # ⚙️ Configuration for tests
├── config_loader.py           # 🔧 Utility to load configuration
├── shared_data.py             # 🔄 Shared state management
├── validators.py              # ✅ Response validation utilities
├── custom_load_shape.py       # 📈 Custom load shape for tests
├── event_hooks.py             # 🪝 Locust event handlers
├── monitoring.py              # 📊 External monitoring integration
├── test_data.py               # 📋 Pre-defined test data
├── analyze_results.py         # 📉 Results analysis script
├── run_tests.sh               # 🚀 Script to run tests
├── requirements.txt           # 📦 Python dependencies (updated)
└── Dockerfile                 # 🐳 Updated Docker configuration
```

---
## 🔄 **Migration Path from Current Implementation**
---

1. **First Migration Step (Basic Locust Test):**
   * Convert `actions.py` functions to Locust tasks in `locustfile.py`
   * Map the weighted action selection in `simulate.py` to Locust's task weights
   * Implement the shared product list in `shared_data.py`
   * Test the basic functionality using Locust's web UI

2. **Second Migration Step (Enhanced Features):**
   * Add validation to ensure responses match expected patterns
   * Implement custom load shapes
   * Add negative testing scenarios for error cases

3. **Final Migration Step (Integration & Reporting):**
   * Set up CI/CD integration
   * Enhance reporting and monitoring
   * Document the new system for users

---
## 🛠️ **Running Tests with Locust**
---

After implementation, the tests can be run in several ways:

**1. Running with Web UI (Development Mode):**
```bash
cd simulations
locust -f locustfile.py --host=http://localhost:8082
```
Then open `http://localhost:8089` in your browser to control the test.

**2. Running Headless (CI/CD or Batch Testing):**
```bash
cd simulations
locust -f locustfile.py --headless -u 10 -r 10 -t 5m --host=http://localhost:8082 --html=report.html
```

**3. Running with Docker:**
```bash
# Build the Docker image
docker build -t product-service-simulation ./simulations

# Run in UI mode
docker run -p 8089:8089 product-service-simulation ./run_tests.sh ui http://product-service:8082

# Run in headless mode
docker run product-service-simulation ./run_tests.sh headless 10 300 http://product-service:8082
```

**4. Running with Tags (Specific Test Types):**
```bash
# Only run browse-related tasks
locust -f locustfile.py --host=http://localhost:8082 --tags browse
```

---
## 💡 **Advantages of Using Locust**
---

1. **Built-in Concurrency:** Handles thousands of users efficiently with gevent
2. **Real-time Web UI:** Monitor tests as they run with built-in dashboards
3. **Distributed Load Generation:** Scale across multiple machines for massive tests
4. **Extensibility:** Python-based, allowing for custom behavior and validation
5. **Active Community:** Regular updates, extensive documentation, and community support
6. **Production-Proven:** Used by major companies for large-scale load testing

This Locust-based approach replaces all the custom threading, request handling, and result collection code with a battle-tested framework, allowing you to focus on writing realistic test scenarios rather than infrastructure code. 