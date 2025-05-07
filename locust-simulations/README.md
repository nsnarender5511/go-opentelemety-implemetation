# Product Service Load Testing with Locust

This directory contains a Locust-based load testing framework for the product-service API.

## Directory Structure

```
locust-simulations/
├── src/
│   ├── tasks/               # Task modules grouped by functionality
│   ├── users/               # User type definitions
│   │   ├── base_user.py     # Base user with common functionality
│   │   ├── browser_user.py  # Browsing-focused user
│   │   ├── shopper_user.py  # Purchase-focused user
│   │   └── admin_user.py    # Administrative user
│   ├── utils/               # Utility functions
│   ├── telemetry/           # OpenTelemetry integration
│   ├── load_shapes/         # Custom load patterns
│   └── data/                # Test data definitions
├── config/                  # Configuration files
├── scripts/                 # Utility scripts
├── locustfile.py            # Main entry point
└── Dockerfile               # Container configuration
```

## Configuration

Configuration is loaded from `config/config.yaml` and can be overridden with environment variables:

- `PRODUCT_SERVICE_URL`: Target service URL
- `LOAD_SHAPE`: Load shape to use (stages, spike, multiple_spikes, ramping)

### Telemetry Configuration

OpenTelemetry integration is **disabled by default**. To enable it:

1. In `config/config.yaml`, set `telemetry.enabled: true` and provide an endpoint
2. Or set environment variables:
   - `OTEL_ENABLED=true` - Enable telemetry
   - `OTEL_ENDPOINT=your-collector:4317` - Collector endpoint

When enabled, the simulation will send telemetry data to the specified collector for monitoring test performance alongside service performance.

## Running Tests

### Local Development

1. Install dependencies:
   ```
   pip install -r requirements.txt
   ```

2. Run with UI mode:
   ```
   ./scripts/run_tests.sh ui
   ```
   Then open http://localhost:8089 in your browser

3. Run headless mode:
   ```
   ./scripts/run_tests.sh headless 50 10 300 http://localhost:8082 stages
   ```
   Arguments: [mode] [users] [spawn_rate] [duration] [host] [shape]

### Docker

Build and run with Docker:

```bash
docker build -t product-service-simulator .
docker run -e PRODUCT_SERVICE_URL=http://host.docker.internal:8082 product-service-simulator
```

### Docker Compose

The service is already configured in the main `docker-compose.yml` as `product-simulator`. Telemetry is disabled by default in this configuration.

To enable telemetry, update the `OTEL_ENABLED` environment variable:

```yaml
environment:
  - PRODUCT_SERVICE_URL=http://nginx:80
  - LOAD_SHAPE=stages
  - OTEL_ENDPOINT=otel-collector:4317
  - OTEL_ENABLED=true  # Change to true to enable
  - SERVICE_NAME=product-simulator
```

## User Types

The simulation includes three user types with different behavior profiles:

1. **Browser User**: Primarily browses products, rarely purchases
2. **Shopper User**: Focuses on product purchasing
3. **Admin User**: Performs administrative tasks like stock updates

## Load Shapes

Several load patterns are available:

- **Stages**: Multi-stage ramp-up, plateau, and ramp-down
- **Spike**: Sudden traffic spike to test resilience
- **Multiple Spikes**: Series of traffic spikes
- **Ramping**: Gradual ramp-up to a maximum 