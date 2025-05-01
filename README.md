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
       "context"

       "github.com/narender/common/config"
       "github.com/narender/common/lifecycle"
       "github.com/narender/common/otel"

       "github.com/sirupsen/logrus"
   )
   ```

2. Initialize telemetry in your service using the builder pattern:
   ```go
   // Load configuration
   cfg, err := config.LoadConfig(".") // Or appropriate path
   if err != nil {
       logrus.Fatalf("Failed to load configuration: %v", err)
   }

   // Create a logger instance (consider using common/logging setup if available)
   logger := logrus.New()
   level, _ := logrus.ParseLevel(cfg.LogLevel)
   logger.SetLevel(level)
   logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

   // Use the OTel setup builder
   otelSetup := otel.NewSetup(cfg, logger)

   // Create the resource
   res, err := otelSetup.NewResource(context.Background(), cfg.ServiceName, cfg.ServiceVersion) // Add ServiceVersion to config if needed
   if err != nil {
       logger.Fatalf("Failed to create OpenTelemetry resource: %v", err)
   }

   // Build the telemetry stack (tracing, metrics, logging)
   shutdownFuncs, err := otelSetup.Build(context.Background(), res)
   if err != nil {
       logger.Fatalf("Failed to build OpenTelemetry stack: %v", err)
   }

   // Combine shutdown functions
   otelShutdown := func(ctx context.Context) error {
       var combinedErr error
       for _, fn := range shutdownFuncs {
           if err := fn(ctx); err != nil {
               combinedErr = fmt.Errorf("%w; %v", combinedErr, err) // Combine errors
           }
       }
       return combinedErr
   }
   ```

3. Set up graceful shutdown:
   ```go
   // For Fiber apps
   // lifecycle.WaitForGracefulShutdown(context.Background(), &lifecycle.FiberShutdownAdapter{App: app}, otelShutdown)

   // For standard HTTP server
   // server := &http.Server{...}
   // lifecycle.WaitForGracefulShutdown(context.Background(), &lifecycle.HTTPShutdownAdapter{Server: server}, otelShutdown)

   // Example: Simple wait
   lifecycle.WaitForSignal(context.Background(), otelShutdown)
   ```

4. Create custom spans for important operations:
   ```go
   // Get a tracer
   tracer := otel.GetTracer("your-instrumentation-name")
   ctx, span := tracer.Start(ctx, "operation-name")
   defer span.End()

   // Add attributes to the span
   span.SetAttributes(attribute.String("key", "value"))
   ```

5. Record metrics:
   ```go
   // Get a meter
   meter := otel.GetMeter("your-instrumentation-name")

   // Create a counter
   counter, err := meter.Int64Counter("requests.total")
   if err != nil {
       logger.Errorf("Failed to create counter: %v", err)
   }
   if counter != nil {
       counter.Add(ctx, 1, metric.WithAttributes(attribute.String("endpoint", "/api/products")))
   }

   // Record timing data (example using histogram)
   histogram, err := meter.Int64Histogram("operation.duration.ms")
   if err != nil {
        logger.Errorf("Failed to create histogram: %v", err)
   }
   start := time.Now()
   // ... do work ...
   durationMs := time.Since(start).Milliseconds()
   if histogram != nil {
       histogram.Record(ctx, durationMs, metric.WithAttributes(attribute.String("operation", "database-query")))
   }
   ```

6. Send logs via OpenTelemetry (using Logrus hook):
   ```go
   // The otelSetup.Build() function already configured the Logrus hook.
   // Standard logrus calls will be exported.
   logger.WithField("product_id", 123).Info("Product retrieved successfully")
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
