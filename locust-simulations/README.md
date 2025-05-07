# Locust Simulations

This directory contains the Locust load testing simulations for the product service API.

## Overview

This simulation uses a single, unified `SimulationUser` class that contains all possible tasks. The tasks can be configured with maximum execution counts through the Locust class picker UI, giving you fine-grained control over test scenarios.

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
   - All task limits are initially set to zero, so you must explicitly enable the tasks you want to run
   - Configure the target host

Each task will be executed up to its configured maximum count, after which it will be skipped. This allows you to precisely control how many times each API endpoint is called during the test.

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

The `SimulationUser` class includes the following tasks, each with configurable execution limits:

### Browsing Tasks

- **Browse Product List** (`max_browse_all_products`): Browse all available products
- **Browse by Category** (`max_get_products_by_category`): Browse products filtered by category
- **Search Products** (`max_search_products`): Search for products by keyword

### Shopping Tasks

- **View Product Details** (`max_view_product_details`): View detailed information for a specific product
- **Add to Cart** (`max_add_to_cart`): Add a product to the shopping cart
- **Checkout** (`max_checkout`): Complete the checkout process

### Admin Tasks

- **Update Inventory** (`max_update_inventory`): Update product inventory (admin task)
- **View Analytics** (`max_view_analytics`): View analytics dashboard (admin task)

### Utility Tasks

- **Health Check** (`max_health_check`): Perform a health check

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
   - `max_search_products: 30`
   - `max_checkout: 10`
3. Leave other tasks at their default value of 0 to disable them
4. Start the test

The API will receive exactly 50 product list requests, 30 search requests, and 10 checkout requests, regardless of how long the test runs or how many users are simulated.

### Example 2: Mixed Test with Limited Admin Operations

To simulate a mixed workload with limited admin operations:

1. Start Locust with `--class-picker`
2. Set browsing and shopping tasks with high limits:
   - `max_browse_all_products: 100`
   - `max_add_to_cart: 50`
3. Set admin tasks with low limits:
   - `max_update_inventory: 5`
   - `max_view_analytics: 3`
4. Leave any tasks you don't want to run at their default value of 0
5. Start the test

This configuration ensures the API receives a realistic mix of customer traffic with a controlled number of administrative operations.

## Troubleshooting

If you encounter issues:

- Make sure the product service API is running and accessible
- Check that the host URL is correctly configured
- Verify that the initial product data load succeeds (check logs)
- If running in a container, ensure proper network connectivity 