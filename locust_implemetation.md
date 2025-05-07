# 🦗 Locust Implementation for Load Testing

## 📋 Table of Contents
- [Introduction to Locust](#introduction-to-locust)
- [Architecture Overview](#architecture-overview)
- [Implementation Guide](#implementation-guide)
- [Running and Configuring Tests](#running-and-configuring-tests)
- [Load Testing Strategies](#load-testing-strategies)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)
- [Project-Specific Implementation](#project-specific-implementation)
- [Example Test Runner Script](#example-test-runner-script)
- [Docker Integration](#docker-integration)
- [Project Docker Setup](#project-docker-setup)

---

## 🚀 Introduction to Locust

Locust is an open-source load testing tool written in Python that allows you to define user behavior with Python code. Unlike traditional load testing tools that use proprietary DSLs or GUIs, Locust enables you to write highly customizable tests using familiar Python syntax.

### ✨ Why Choose Locust?

| Feature | Description |
|---------|-------------|
| 🐍 **Python-based** | Define test scenarios in pure Python for maximum flexibility |
| 🌐 **Distributed architecture** | Scale tests across multiple machines |
| 📊 **Real-time metrics** | Monitor test progress via web UI with live statistics |
| ⚡ **Event-driven** | Non-blocking architecture allows simulating thousands of users on a single machine |
| 🧩 **Extensible** | Easily create custom behaviors, reporting, and integrations |
| 🔓 **Open source** | Active community and regular updates |
| 🔌 **Protocol-agnostic** | Not limited to HTTP - can test any system (MQTT, WebSockets, gRPC, etc.) |

---

## 🏗️ Architecture Overview

### 🧱 Locust Core Components

1. **User Classes**: Python classes that define user behavior
2. **Tasks**: Methods within User classes that represent user actions
3. **TaskSets**: Groups of related tasks for organizing complex behaviors
4. **Events**: Hooks for extending functionality at various points in the test lifecycle
5. **Web UI**: Real-time monitoring and control interface
6. **Runners**: Coordinate test execution (local or distributed)

### 🌐 Distributed Architecture

```
┌─────────────────┐
│                 │
│  Master Node    │◄───┐
│  (Web UI)       │    │
│                 │    │
└────────┬────────┘    │ Report stats
         │             │
         │ Coordinate  │
         ▼             │
┌─────────────────┐    │    ┌─────────────────┐
│                 │    │    │                 │
│  Worker Node 1  ├────┘    │  Worker Node N  │
│                 │         │                 │
└────────┬────────┘         └────────┬────────┘
         │                           │
         │ Generate load             │ Generate load
         ▼                           ▼
┌─────────────────────────────────────────────┐
│                                             │
│               Target System                 │
│                                             │
└─────────────────────────────────────────────┘
```

---

## 💻 Implementation Guide

### 📥 3.1 Installation

```bash
pip install locust
```

### 📄 3.2 Basic Locust File Structure

```python
# locustfile.py
import time
import random
from locust import HttpUser, task, between

class ProductServiceUser(HttpUser):
    # Wait between 1-5 seconds between tasks
    wait_time = between(1, 5)
    
    @task
    def get_all_products(self):
        self.client.get("/products")
    
    @task(3)  # 3x more frequent than other tasks
    def view_product_details(self):
        product_id = random.randint(1, 100)
        self.client.get(f"/products/{product_id}")
```

### 🔄 3.3 Advanced Implementation with Shared Data

```python
# locustfile.py
import time
import random
import json
from typing import List, Dict, Any
import threading
from locust import HttpUser, task, between, events, tag

# Thread-safe shared data container
class SharedData:
    def __init__(self):
        self.lock = threading.RLock()
        self.products = []
    
    def update_products(self, products: List[Dict[str, Any]]):
        with self.lock:
            self.products = products
    
    def get_random_product(self):
        with self.lock:
            if not self.products:
                return None
            return random.choice(self.products)

# Global shared data instance
shared_data = SharedData()

class ProductServiceUser(HttpUser):
    wait_time = between(1, 3)
    
    def on_start(self):
        """Initialize user session."""
        # Fetch products once at the start
        with self.client.get("/products", catch_response=True) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    products = data.get("data", [])
                    shared_data.update_products(products)
                except json.JSONDecodeError:
                    response.failure("Invalid JSON response")
    
    @task(10)
    @tag("browse")
    def browse_products(self):
        """Browse all products."""
        with self.client.get("/products", name="Get_All_Products") as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    products = data.get("data", [])
                    shared_data.update_products(products)
                except json.JSONDecodeError:
                    response.failure("Invalid JSON response")
    
    @task(7)
    @tag("purchase")
    def buy_product(self):
        """Purchase a product."""
        product = shared_data.get_random_product()
        if not product:
            return
        
        quantity = random.randint(1, 5)
        with self.client.post(
            "/products/buy",
            json={"name": product["name"], "quantity": quantity},
            name="Buy_Product"
        ) as response:
            if response.status_code != 200:
                response.failure(f"Failed with status {response.status_code}")
```

### ✅ 3.4 Response Validation

```python
@task(5)
@tag("details")
def view_product_details(self):
    """View details of a specific product with validation."""
    product = shared_data.get_random_product()
    if not product:
        return
    
    with self.client.post(
        "/products/details",
        json={"name": product["name"]},
        name="Product_Details",
        catch_response=True
    ) as response:
        if response.status_code != 200:
            response.failure(f"Status code: {response.status_code}")
            return
            
        try:
            data = response.json()
            # Validate response structure
            required_fields = ["productID", "name", "price", "stock"]
            for field in required_fields:
                if field not in data:
                    response.failure(f"Missing field in response: {field}")
                    return
            response.success()
        except json.JSONDecodeError:
            response.failure("Invalid JSON")
```

### 📈 3.5 Custom Load Shape

```python
# custom_shape.py
from locust import LoadTestShape

class StagesLoadShape(LoadTestShape):
    """
    A load shape that allows for defining stages with different user counts and durations
    """
    
    stages = [
        {"duration": 60, "users": 10, "spawn_rate": 10},  # Ramp up to 10 users over 1 minute
        {"duration": 300, "users": 50, "spawn_rate": 10},  # Ramp up to 50 users over 5 minutes
        {"duration": 600, "users": 50, "spawn_rate": 10},  # Stay at 50 users for 10 minutes
        {"duration": 120, "users": 0, "spawn_rate": 10},   # Ramp down to 0 over 2 minutes
    ]
    
    def tick(self):
        run_time = self.get_run_time()
        
        elapsed = 0
        for stage in self.stages:
            if elapsed + stage["duration"] > run_time:
                target_users = stage["users"]
                spawn_rate = stage["spawn_rate"]
                return target_users, spawn_rate
            elapsed += stage["duration"]
            
        return None  # Test is finished
```

---

## ⚙️ Running and Configuring Tests

### 🖥️ 4.1 Basic Command Line Usage

```bash
# Run with web UI (default)
locust -f locustfile.py --host=http://your-service:8080

# Run without web UI (headless mode)
locust -f locustfile.py --headless -u 100 -r 10 -t 5m --host=http://your-service:8080

# Run with specific tags
locust -f locustfile.py --tags purchase,browse --host=http://your-service:8080
```

### 🎛️ 4.2 Command Line Parameters

| Parameter | Description |
|-----------|-------------|
| `-f, --locustfile` | Python file to import (default: locustfile.py) |
| `--host` | Host to load test |
| `-u, --users` | Peak number of concurrent users |
| `-r, --spawn-rate` | Rate at which users are spawned (users per second) |
| `-t, --run-time` | Stop after the specified time (e.g., 1h30m, 60s, etc.) |
| `--headless` | Run without web UI |
| `--html` | Generate HTML report at test end |
| `--tags` | Only run tasks with specified tags |

### 🌐 4.3 Web UI

When running with the web UI (default mode), access the interface at:
```
http://localhost:8089
```

The web UI lets you:
- Start and stop tests
- Configure the number of users and spawn rate
- View real-time statistics and charts
- Download statistics in CSV format

![Locust Web UI](https://locust.io/static/img/screenshot.png)

### 🔄 4.4 Distributed Mode

```bash
# Start master node
locust -f locustfile.py --master --host=http://your-service:8080

# Start worker nodes (can be on different machines)
locust -f locustfile.py --worker --master-host=192.168.1.100
```

---

## 📊 Load Testing Strategies

### 🔝 5.1 Peak Load Testing

Simulate maximum expected load to verify system stability:

```python
class PeakLoadShape(LoadTestShape):
    def tick(self):
        run_time = self.get_run_time()
        
        if run_time < 60:
            # Ramp up to peak load over 1 minute
            return int(run_time * 5), 10  # Max 300 users
        elif run_time < 360:
            # Maintain peak load for 5 minutes
            return 300, 10
        elif run_time < 420:
            # Ramp down over 1 minute
            return int(300 - ((run_time - 360) * 5)), 10
        
        return None  # End test
```

### 💪 5.2 Stress Testing

Find breaking points by gradually increasing load:

```python
class StressTestShape(LoadTestShape):
    def tick(self):
        run_time = self.get_run_time()
        
        # Increase users by 10 every minute until system breaks
        if run_time < 3600:  # 1 hour max
            return int((run_time / 60) * 10), 10
        
        return None
```

### ⏱️ 5.3 Soak Testing

Test system stability over extended periods:

```python
class SoakTestShape(LoadTestShape):
    def tick(self):
        run_time = self.get_run_time()
        
        if run_time < 600:
            # Ramp up to 50 users over 10 minutes
            return int(run_time / 12), 5
        elif run_time < 7200:
            # Maintain 50 users for 2 hours
            return 50, 5
        
        return None
```

### ⚡ 5.4 Spike Testing

Test recovery from sudden load spikes:

```python
class SpikeTestShape(LoadTestShape):
    def tick(self):
        run_time = self.get_run_time()
        
        # Normal load for 5 minutes
        if run_time < 300:
            return 50, 10
        # Spike to 500 users for 2 minutes
        elif run_time < 420:
            return 500, 50
        # Back to normal for 5 minutes
        elif run_time < 720:
            return 50, 50
        
        return None
```

---

## 🛠️ Advanced Features

### 📏 6.1 Custom Metrics

```python
from locust import events

# Initialize custom counters
custom_stats = {
    "successful_purchases": 0,
    "failed_purchases": 0,
    "out_of_stock": 0
}

@events.request.add_listener
def on_request(request_type, name, response_time, response_length, exception, **kwargs):
    if name == "Buy_Product":
        if exception:
            custom_stats["failed_purchases"] += 1
        else:
            response = kwargs.get("response")
            if response and response.status_code == 200:
                custom_stats["successful_purchases"] += 1
            elif response and response.status_code == 409:  # Assume 409 is "out of stock"
                custom_stats["out_of_stock"] += 1

@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    print("\n=== Custom Statistics ===")
    for key, value in custom_stats.items():
        print(f"{key}: {value}")
```

### 🔁 6.2 Custom Request Handling

```python
from locust.clients import HttpSession
from urllib.parse import urljoin

class CustomClient:
    def __init__(self, base_url, user):
        self.session = HttpSession(base_url=base_url)
        self.user = user
    
    def request_with_retry(self, method, path, **kwargs):
        """Custom request with automatic retry logic"""
        max_retries = kwargs.pop("max_retries", 3)
        retry_delay = kwargs.pop("retry_delay", 1)
        
        url = urljoin(self.session.base_url, path)
        
        for attempt in range(max_retries):
            try:
                name = kwargs.pop("name", path)
                with self.session.request(method, url, name=name, **kwargs) as response:
                    if response.status_code < 500:  # Don't retry client errors
                        return response
                    
                    if attempt < max_retries - 1:
                        time.sleep(retry_delay)
            except Exception as e:
                if attempt < max_retries - 1:
                    time.sleep(retry_delay)
                else:
                    raise
        
        return response  # Return last response if all retries failed
```

### 📋 6.3 Data-Driven Testing

```python
# test_data.py
TEST_PRODUCTS = [
    {"name": "Laptop", "expected_price": 999.99},
    {"name": "Phone", "expected_price": 599.99},
    {"name": "Headphones", "expected_price": 149.99}
]

# In locustfile.py
from test_data import TEST_PRODUCTS

@task(3)
def verify_product_prices(self):
    """Verify prices match expected values."""
    product = random.choice(TEST_PRODUCTS)
    
    with self.client.post(
        "/products/details",
        json={"name": product["name"]},
        name=f"Verify_Price_{product['name']}",
        catch_response=True
    ) as response:
        if response.status_code != 200:
            response.failure(f"Status code: {response.status_code}")
            return
            
        try:
            data = response.json()
            actual_price = float(data.get("price", 0))
            expected_price = product["expected_price"]
            
            if abs(actual_price - expected_price) > 0.01:  # Allow small float differences
                response.failure(f"Price mismatch: expected {expected_price}, got {actual_price}")
            else:
                response.success()
        except Exception as e:
            response.failure(f"Error: {str(e)}")
```

---

## 📝 Best Practices

### 📚 7.1 Test Organization

- **Separate configuration from test code**: Use config files or environment variables
- **Use tags to categorize tasks**: Run subsets of tests with `--tags`
- **Create specialized user classes**: Different user types for different behaviors
- **Use weight to balance task frequency**: Higher weights for common operations

### ⚡ 7.2 Performance Considerations

- **Minimize resource usage in tasks**: Keep task code efficient and focused
- **Be careful with sleeps/waits**: Use Locust's built-in wait_time instead of time.sleep()
- **Use catch_response for validation**: Only mark responses as failures when truly necessary
- **Start with fewer users**: Begin with small tests and scale up gradually

### 👀 7.3 Monitoring and Analysis

- **Use the Locust UI during development**: Monitor real-time metrics
- **Generate HTML reports for sharing**: Use `--html` flag
- **Export CSV data for custom analysis**: Use UI export or `--csv` flag
- **Integrate with existing monitoring**: Use events to send metrics to external systems

### ⚙️ 7.4 Configuration Example

```yaml
# config.yaml
service:
  base_url: "http://localhost:8082"
  timeout_seconds: 30

load_profile:
  stages:
    - duration_seconds: 60
      users: 10
      spawn_rate: 10
    - duration_seconds: 300
      users: 50
      spawn_rate: 10
    - duration_seconds: 600
      users: 50
      spawn_rate: 10
    - duration_seconds: 120
      users: 0
      spawn_rate: 10

scenario_weights:
  browse_products: 10
  search_by_category: 7
  view_product_details: 8
  buy_product: 5
  update_stock: 1

test_data:
  categories:
    - "Electronics"
    - "Clothing"
    - "Books"
  edge_cases:
    - name: "NonExistentProduct"
      expected_status: 404
    - name: "ZeroStockItem"
      expected_status: 409
```

```python
# config_loader.py
import yaml
import os

def load_config(config_path="config.yaml"):
    """Load configuration file"""
    if not os.path.exists(config_path):
        print(f"Warning: Config file {config_path} not found, using defaults")
        return {
            "service": {"base_url": "http://localhost:8082", "timeout_seconds": 30},
            "scenario_weights": {
                "browse_products": 10,
                "buy_product": 5
            }
        }
    
    with open(config_path, 'r') as f:
        return yaml.safe_load(f)
```

---

## 🚀 Example Test Runner Script

```bash
#!/bin/bash
# run_tests.sh

MODE=${1:-"ui"}
USERS=${2:-10}
SPAWN_RATE=${3:-10}
RUNTIME=${4:-"5m"}
HOST=${5:-"http://localhost:8082"}

case $MODE in
    ui)
        echo "Starting Locust with UI on http://localhost:8089"
        locust -f locustfile.py --host=$HOST
        ;;
    headless)
        echo "Running headless test with $USERS users for $RUNTIME"
        locust -f locustfile.py --headless -u $USERS -r $SPAWN_RATE -t $RUNTIME --host=$HOST --html=report.html
        ;;
    distributed-master)
        echo "Starting distributed test (master node)"
        locust -f locustfile.py --master --host=$HOST
        ;;
    distributed-worker)
        MASTER_HOST=${6:-"localhost"}
        echo "Starting worker node connecting to $MASTER_HOST"
        locust -f locustfile.py --worker --master-host=$MASTER_HOST
        ;;
    *)
        echo "Unknown mode: $MODE"
        echo "Usage: $0 [ui|headless|distributed-master|distributed-worker] [users] [spawn_rate] [runtime] [host] [master_host]"
        exit 1
        ;;
esac
```

---

## 🐳 Docker Integration

```dockerfile
# Dockerfile
FROM python:3.9-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY *.py .
COPY *.yaml .
COPY run_tests.sh .

RUN chmod +x run_tests.sh

EXPOSE 8089

ENTRYPOINT ["./run_tests.sh"]
CMD ["ui"]
```

```yaml
# docker-compose.yml (generic example)
version: '3'

services:
  locust-master:
    build: ./simulations
    ports:
      - "8089:8089"
    command: distributed-master 100 10 30m http://product-service:8082
    volumes:
      - ./simulations:/app
  
  locust-worker:
    build: ./simulations
    command: distributed-worker 0 0 0 http://product-service:8082 locust-master
    volumes:
      - ./simulations:/app
    depends_on:
      - locust-master
    deploy:
      replicas: 3
  
  product-service:
    image: product-service:latest
    ports:
      - "8082:8082"
```

---

## 🔄 Project Docker Setup

Our project uses a complete deployment setup that includes Locust for load testing, integrated with OpenTelemetry for observability. The Docker setup includes multiple components:

### 🏗️ Docker Composition

```yaml
# Project docker-compose.yml (simplified)
networks:
  otel_internal-network:
    driver: bridge

services:
  # Load Testing Simulator (Locust implementation)
  product-simulator:
    container_name: simulator-service
    build:
      context: ./simulations
      dockerfile: Dockerfile
    environment:
      - PRODUCT_SERVICE_URL=http://nginx:80
    networks:
      - otel_internal-network
    depends_on:
      - nginx

  # OpenTelemetry Collector for observability
  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.99.0
    container_name: local-otel-collector
    # ... OpenTelemetry configuration ...
    networks:
      - otel_internal-network

  # Product Service Instances (Load Balanced)
  product-service-1:
    container_name: ${SERVICE_NAME}-1
    # ... configuration ...
    networks:
      - otel_internal-network

  product-service-2:
    container_name: ${SERVICE_NAME}-2
    # ... configuration ...
    networks:
      - otel_internal-network

  # Load Balancer
  nginx:
    container_name: nginx
    image: nginx:latest
    ports:
      - "8080:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    networks:
      - otel_internal-network
    depends_on:
      - product-service-1
      - product-service-2
```

### 📝 Key Integration Points

1. **Service Targeting**: The `product-simulator` container targets the load-balanced product services via nginx using the environment variable `PRODUCT_SERVICE_URL=http://nginx:80`.

2. **Networking**: All services share the same `otel_internal-network` network, allowing the load tester to communicate with other components.

3. **Dependencies**: The simulator depends on nginx, ensuring the load balancer is ready before starting load tests.

4. **Observability**: The setup includes OpenTelemetry collector for comprehensive monitoring of both the tested services and the load testing process.

### 🔧 Updated Dockerfile for Simulations

```dockerfile
# simulations/Dockerfile
FROM python:3.9-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy all simulation files
COPY . .

# Make run script executable
RUN chmod +x run_tests.sh

# Default command runs in headless mode with environment variables
CMD ["./run_tests.sh", "headless", "10", "10", "1h", "${PRODUCT_SERVICE_URL}"]
```

### 🧩 Locust Configuration for Docker

For our Docker environment, we need to update the `config_loader.py` to consider environment variables:

```python
# config_loader.py
import yaml
import os
import logging

def load_config(config_path="config.yaml"):
    """Load configuration from YAML file with fallback defaults"""
    try:
        # Default config
        config = {
            "service": {
                # Read service URL from environment if available, fallback to default
                "base_url": os.environ.get("PRODUCT_SERVICE_URL", "http://localhost:8082"),
                "request_timeout_seconds": 10
            },
            # Other default settings...
        }

        # Load from file if exists
        if os.path.exists(config_path):
            with open(config_path, 'r') as f:
                file_config = yaml.safe_load(f)
                if file_config:
                    # Deep merge configs, with file taking precedence except for base_url
                    # which should prioritize environment variable
                    deep_merge(config, file_config)
                    # Ensure environment variable takes precedence if set
                    if "PRODUCT_SERVICE_URL" in os.environ:
                        config["service"]["base_url"] = os.environ["PRODUCT_SERVICE_URL"]

        return config
    except Exception as e:
        logging.error(f"Error loading config: {e}")
        raise

def deep_merge(base, override):
    """Recursively merge dictionaries"""
    for key, value in override.items():
        if key in base and isinstance(base[key], dict) and isinstance(value, dict):
            deep_merge(base[key], value)
        else:
            base[key] = value
```

### 🚀 Running with Docker Compose

To run the complete system with Docker Compose:

```bash
# Start the entire system including load testing
docker-compose up -d

# To view Locust logs
docker-compose logs -f product-simulator

# To scale up worker nodes for distributed testing (if configured)
docker-compose up -d --scale locust-worker=5

# To shut down
docker-compose down
```

### ⚙️ Integration with OpenTelemetry

Our Locust implementation can export telemetry data to track test performance. We can add the following to `locustfile.py`:

```python
# Add OpenTelemetry support to locustfile.py
import os
from locust import events
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource

# Set up OpenTelemetry if OTEL_ENDPOINT is provided
if "OTEL_ENDPOINT" in os.environ:
    resource = Resource(attributes={
        SERVICE_NAME: "locust-load-tester"
    })
    
    tracer_provider = TracerProvider(resource=resource)
    trace.set_tracer_provider(tracer_provider)
    
    # Create OTLP exporter
    otlp_exporter = OTLPSpanExporter(endpoint=os.environ["OTEL_ENDPOINT"])
    span_processor = BatchSpanProcessor(otlp_exporter)
    tracer_provider.add_span_processor(span_processor)
    
    tracer = trace.get_tracer(__name__)
    
    # Create spans for each request
    @events.request.add_listener
    def create_spans(request_type, name, response_time, response_length, exception, **kwargs):
        with tracer.start_as_current_span(f"{request_type} {name}") as span:
            span.set_attribute("http.method", request_type)
            span.set_attribute("http.url", name)
            span.set_attribute("http.response_time_ms", response_time)
            
            if exception:
                span.set_attribute("error", True)
                span.set_attribute("error.message", str(exception))
            else:
                response = kwargs.get("response")
                if response:
                    span.set_attribute("http.status_code", response.status_code)
```

This setup allows for comprehensive monitoring of load testing behavior, making it easier to identify bottlenecks in both the load testing system and the services being tested.

---

## 📝 Summary

Locust provides a powerful, flexible framework for load testing applications with these key advantages:

- ✨ **Python-based**: Familiar language with full programming capabilities
- 🔌 **Extensible**: Highly customizable for any testing scenario
- 📊 **Real-time feedback**: Live metrics through the web UI
- 🚀 **Scalable**: Distributed architecture for large-scale tests
- 🧩 **Modular**: Compose tests from reusable components
- 📈 **Observable**: Integrates with OpenTelemetry for comprehensive monitoring

By following the implementation guidance in this document, you can create sophisticated load testing scenarios that accurately model real-world usage of your application and provide valuable insights into its performance characteristics. 