# OpenTelemetry with SigNoz Demo

This project demonstrates the integration of OpenTelemetry with SigNoz for complete observability (traces, metrics, and logs) in a microservices architecture.

## Project Structure

- **common/** - Reusable module that contains OpenTelemetry integration and other common utilities
  - **telemetry/** - Comprehensive OpenTelemetry integration (traces, metrics, logs)
  - **config/** - Configuration management
  - **lifecycle/** - Application lifecycle management (graceful shutdown)
  - **errors/** - Error handling utilities

- **product-service/** - Sample microservice that uses the common module
  - **src/** - Service implementation
  - **Dockerfile** - Container definition

- **tests/** - Load simulators and testing utilities

## Features

- **Complete OpenTelemetry Integration** - Automatic instrumentation of HTTP requests, database calls, and service operations
- **Custom Span Creation** - Utilities for manual span creation and enrichment
- **Metric Recording** - Framework for capturing business and operational metrics
- **Structured Logging** - Integrated with OpenTelemetry for correlation with traces
- **Graceful Shutdown** - Clean termination of services with telemetry flushing
- **Containerization** - Docker and Docker Compose setup for easy deployment

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.20 or higher (for local development)
- SigNoz (running in a separate Docker Compose setup)

### Setup

1. Clone this repository:
   ```
   git clone https://github.com/yourusername/signoz-assignment.git
   cd signoz-assignment
   ```

2. Copy the example environment file and make any necessary changes:
   ```
   cp .env.example .env
   ```

3. Start the application:
   ```
   docker-compose up -d
   ```

4. Access the product service at http://localhost:8082/api/v1/products

5. View the telemetry data in SigNoz dashboard at http://localhost:3301

## Using the Common Module in New Services

The `common` module is designed to be easily integrated into any service. Here's how to use it:

1. Import the required packages:
   ```go
   import (
       "github.com/narender/common/config"
       "github.com/narender/common/lifecycle"
       "github.com/narender/common/telemetry"
   )
   ```

2. Initialize telemetry in your service:
   ```go
   // Load configuration
   if err := config.LoadConfig(); err != nil {
       logrus.Fatalf("Failed to load configuration: %v", err)
   }

   // Initialize telemetry
   telemetryConfig := telemetry.TelemetryConfig{
       ServiceName: config.ServiceName(),
       Endpoint:    config.OtelExporterEndpoint(),
       Insecure:    config.IsOtelExporterInsecure(),
       SampleRatio: config.OtelSampleRatio(),
       LogLevel:    config.LogLevel(),
   }
   otelShutdown, err := telemetry.InitTelemetry(context.Background(), telemetryConfig)
   if err != nil {
       logrus.WithError(err).Fatal("Failed to initialize telemetry")
   }
   ```

3. Set up graceful shutdown:
   ```go
   // For Fiber apps
   lifecycle.WaitForGracefulShutdown(context.Background(), &lifecycle.FiberShutdownAdapter{App: app}, otelShutdown)
   
   // For standard HTTP server
   server := &http.Server{...}
   lifecycle.WaitForGracefulShutdown(context.Background(), &lifecycle.HTTPShutdownAdapter{Server: server}, otelShutdown)
   ```

4. Create custom spans for important operations:
   ```go
   ctx, span := telemetry.StartSpan(ctx, "operation-name")
   defer span.End()
   
   // Add attributes to the span
   span.SetAttributes(telemetry.StringAttribute("key", "value"))
   ```

5. Record metrics:
   ```go
   // Record a counter
   telemetry.RecordCount(ctx, "requests.total", 1, telemetry.StringAttribute("endpoint", "/api/products"))
   
   // Record timing data
   start := time.Now()
   // ... do work ...
   telemetry.RecordDuration(ctx, "operation.duration", time.Since(start), telemetry.StringAttribute("operation", "database-query"))
   ```

## Testing with the Simulator

The project includes a load simulator in the `tests/` directory to generate traffic to the product service. The simulator is configured to:

- Create 5 instances with Docker Compose
- Generate random product requests
- Vary request patterns to demonstrate different trace patterns

## Observability in SigNoz

After running the application with the simulator, you can observe:

1. **Traces** - End-to-end request flows with detailed span information
2. **Metrics** - Service performance metrics, custom business metrics
3. **Logs** - Structured logs correlated with traces
4. **Service Map** - Visual representation of service interactions

## Troubleshooting

- **No data in SigNoz**: Ensure the OTEL_EXPORTER_OTLP_ENDPOINT in .env points to your SigNoz collector endpoint
- **Service fails to start**: Check the logs with `docker-compose logs product-service`
