# Locust Simulations

This directory contains the Locust load testing simulations for the product service API and master-store API.

## Overview

This simulation uses two main user classes:
1. `SimulationUser` - Contains tasks for testing the product service endpoints
2. `MasterStoreUser` - Contains tasks for testing the master-store endpoints

Both classes can be configured with maximum execution counts through the Locust class picker UI, giving you fine-grained control over test scenarios.

## Features

- **Configurable Task Execution Limits**: Control exactly how many times each task is executed
- **Automatic Retry Logic**: All API calls automatically retry up to 3 times with exponential backoff for 5xx errors
- **Load Shape Selection**: Choose from different traffic patterns through the web UI
- **Real-time Task Counters**: Monitor execution counts through the web UI

## Running the Simulations

### Basic Run

To run the simulation with the class picker UI enabled:

```bash
cd locust-simulations
locust -f locustfile.py --class-picker
```

Then navigate to `http://localhost:8089` to access the Locust web UI.

### Docker Run

To run using Docker:

```bash
docker-compose up -d product-simulator
```

## Configuration

### Task Execution Limits

We use Locust's built-in class picker to configure maximum execution counts for each task:

1. Start Locust with the `--class-picker` flag
2. On the Locust web UI, you'll see a class picker that allows you to:
   - Set maximum execution counts for each task (e.g., `max_browse_all_products: 20`)
   - Task limits are interpreted as follows:
     - `-1`: Unlimited executions
     - `0`: Task disabled (default)
     - `Positive number`: Maximum number of times to execute
   - Configure the target host

**Important**: All tasks are disabled by default (limit=0). You must explicitly set a positive value or -1 to enable each task you want to run.

Each task will be executed up to its configured maximum count, after which it will be skipped. This allows you to precisely control how many times each API endpoint is called during the test.

### Error Handling and Retries

All API calls include automatic retry logic:

- Initial connections retry up to 3 times with increasing delays
- 5xx server errors trigger automatic retries (up to 3 attempts)
- Validation failures also trigger retries
- Exponential backoff increases the delay between retry attempts

This provides resilience against temporary network issues or service instability without manual intervention.

### Monitoring Task Execution Counts

You can monitor the current execution count for each task through the Task Counters page:

1. Click on the "Task Counters" link in the navigation bar
2. The page displays:
   - Current execution count for each task
   - Maximum limit for each task
   - Visual progress bar showing usage against limits

The counters automatically refresh every 2 seconds, giving you real-time visibility into your test execution.

### Load Shapes

The simulation supports several load shapes that control how users are spawned over time:

- **Standard (default)**: Constant number of users based on what you specify in the web UI
- **Stages**: Gradually ramps up, maintains steady load, then ramps down
- **Spike**: Sudden traffic surge to test system response to traffic spikes
- **Multiple Spikes**: Series of traffic spikes
- **Ramping**: Continuous ramping up of load

To select a load shape, click on the "Load Shape" link in the Locust web UI.

## Available Tasks

### SimulationUser Tasks
The `SimulationUser` class includes the following tasks, each with configurable execution limits:

### Browsing Tasks

- **Browse Product List** (`max_browse_all_products`): Browse all available products (GET /products)
- **Browse by Category** (`max_get_products_by_category`): Browse products filtered by category (GET /products/category)

### Product Tasks

- **Get Product by Name** (`max_get_product_by_name`): Get detailed information for a specific product by name (POST /products/details)
- **Buy Product** (`max_buy_product`): Purchase a product (POST /products/buy)

### Admin Tasks

- **Update Product Stock** (`max_update_product_stock`): Update product stock level (PATCH /products/stock)

### Utility Tasks

- **Health Check** (`max_health_check`): Perform a health check (GET /health)

### MasterStoreUser Tasks
The `MasterStoreUser` class includes the following tasks for testing the master-store API:

- **Master Health Check** (`max_master_health_check`): Perform a health check on master-store (GET /health)
- **Master Browse Products** (`max_master_browse_products`): Browse all products from master-store (GET /products)
- **Master Buy Product** (`max_master_buy_product`): Purchase a product from master-store (POST /products/update-stock)

### Master Store Configuration

The master-store tests can be configured with the following parameters:

- `--max_master_health_check`: Maximum number of health check requests to send to master-store (-1 for unlimited, 0 to disable)
- `--max_master_browse_products`: Maximum number of product browsing requests to send to master-store
- `--max_master_buy_product`: Maximum number of product purchase requests to send to master-store
- `--use_nginx_proxy`: When enabled, routes requests through the nginx proxy path (/master/)

By default, the environment variables in docker-compose.yml are set to:
- `MASTER_STORE_URL=http://nginx:80/master`
- `USE_NGINX_PROXY=true`

## Architecture

The simulation is built on a simplified, modular architecture:

- `locustfile.py`: Main entry point for Locust containing the SimulationUser class with all tasks
- `src/web_extension.py`: Custom web UI extension for load shape selection and task execution counters
- `src/load_shapes/`: Different load shapes for various testing scenarios
- `src/utils/`: Shared utilities for HTTP validation, data sharing, etc.
- `src/telemetry/`: OpenTelemetry integration for monitoring

## Examples

### Example 1: Fixed-Count API Test

To create a test with exact API call counts:

1. Start Locust with `--class-picker`
2. In the class picker UI, explicitly set the maximum count for each task you want to run:
   - `max_browse_all_products: 50`
   - `max_get_products_by_category: 30`
   - `max_buy_product: 10`
3. Leave other tasks at their default value of 0 to keep them disabled
4. Start the test

The API will receive exactly 50 product list requests, 30 category requests, and 10 product purchase requests, regardless of how long the test runs or how many users are simulated.

### Example 2: Mixed Test with Limited Admin Operations

To simulate a mixed workload with limited admin operations:

1. Start Locust with `--class-picker`
2. Set browsing and shopping tasks with high limits:
   - `max_browse_all_products: 100`
   - `max_get_product_by_name: 50`
   - `max_buy_product: 25`
3. Set admin tasks with low limits:
   - `max_update_product_stock: 5`
4. Leave other tasks at their default value of 0 to keep them disabled
5. Start the test

This configuration ensures the API receives a realistic mix of customer traffic with a controlled number of administrative operations.

### Example 3: Master Store Testing

To create a test that specifically targets the master-store API:

1. Start Locust with `--class-picker`
2. In the class picker UI, select the `MasterStoreUser` class
3. Set the maximum counts for master-store tasks:
   - `max_master_browse_products: 50`
   - `max_master_buy_product: 20`
   - `max_master_health_check: 5`
4. Start the test

The master-store API will receive exactly 50 product list requests, 20 product purchase requests, and 5 health check requests.

### Example 4: Simulating Both Services Simultaneously

To simulate load on both the product service and master-store:

1. Start Locust with `--class-picker`
2. In the class picker UI, select both `SimulationUser` and `MasterStoreUser` classes
3. Configure task limits for both classes
4. Start the test

This configuration allows testing both services simultaneously, simulating real-world traffic patterns where both services receive requests concurrently.

## Troubleshooting

If you encounter issues:

- Make sure the product service API is running and accessible
- Check that the host URL is correctly configured
- Verify that the initial product data load succeeds (check logs)
- If running in a container, ensure proper network connectivity 