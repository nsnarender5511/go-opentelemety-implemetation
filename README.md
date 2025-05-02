# OpenTelemetry with SigNoz Demo

## SigNoz Integration Features

This project has been enhanced with comprehensive OpenTelemetry integration for SigNoz, featuring:

1. **Custom Metrics Collection**
   - `product_requests_total` - Counter tracking API request frequency by endpoint
   - `product_stock_level` - Gauge showing current stock levels by product
   - `product_request_duration_ms` - Histogram measuring API response times
   - `product_error_total` - Counter tracking error frequency by type and endpoint

2. **Enhanced Trace Context**
   - Product attributes (ID, category, stock)
   - Request duration tracking
   - Error details (type, status code)
   - Endpoint information

3. **Structured Logging with Trace Context**
   - Log correlation with trace IDs (via `ContextLoggerMiddleware`)
   - JSON-formatted logs using Zap
   - Proper log levels for different scenarios


This project demonstrates the integration of OpenTelemetry with SigNoz for complete observability (traces, metrics, and logs) in a microservices architecture.

## Project Structure

- **common/** - Reusable module that contains OpenTelemetry integration and other common utilities
- **telemetry/** - Comprehensive OpenTelemetry integration (traces, metrics, logs)
  - **otel/** - OpenTelemetry setup and utilities for Traces, Metrics, and Logs.
    - **Note:** If this module is intended *only* for use within `product-service`,
      consider moving it to `product-service/internal/otel/` to adhere to Go's
      internal package conventions. If it's designed as a reusable library for
      multiple future services, keeping it in `common/` or moving to a dedicated
      `pkg/` directory is appropriate.
  - **config/** - Configuration management
  - **lifecycle/** - Application lifecycle management (graceful shutdown)
  - **errors/** - Error handling utilities

- **product-service/** - Sample microservice that uses the common module
  - **src/** - Service implementation
  - **Dockerfile** - Container definition

- **tests/** - Load simulators and testing utilities

## Features

- **Complete OpenTelemetry Integration** - Automatic instrumentation of HTTP requests, database calls, and service operations
- **Custom Span Creation** - Common wrappers (`trace.StartSpan`) for manual span creation and enrichment
- **Metric Recording** - Common wrappers (`metric.RecordOperationMetrics`) for capturing standard operational metrics
- **Structured Logging** - Using Zap, integrated with OpenTelemetry for correlation with traces via middleware
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
       "github.com/narender/common/logging" // For logger context access
       "github.com/narender/common/middleware" // For context logger middleware
       "github.com/narender/common/telemetry" // For initialization
       "github.com/narender/common/telemetry/manager" // For global accessors
       "github.com/narender/common/telemetry/trace" // For StartSpan wrapper
       "github.com/narender/common/telemetry/metric" // For RecordOperationMetrics wrapper

       "go.uber.org/zap" // Zap logger
   )
   ```

2. Initialize telemetry in your service:
   ```go
   // Load configuration
   cfg, err := config.LoadConfig(".") // Or appropriate path
   if err != nil {
       // Use a temporary basic logger or panic if config is critical
       log.Fatalf("Failed to load configuration: %v", err)
   }

   // Initialize the full telemetry stack (includes base Zap logger setup)
   shutdown, err := telemetry.InitTelemetry(context.Background(), cfg)
   if err != nil {
       // InitTelemetry handles its own logging on failure
       log.Fatalf("Failed to initialize telemetry: %v", err)
   }
   // Ensure graceful shutdown is handled (e.g., using common/lifecycle)
   defer func() {
       if err := shutdown(context.Background()); err != nil {
           log.Printf("Error shutting down telemetry: %v", err)
       }
   }()

   // Get the initialized base logger if needed directly (rarely)
   baseLogger := manager.GetLogger()
   baseLogger.Info("Telemetry initialized successfully")
   ```

3. Set up middleware (e.g., for Fiber):
   ```go
   import "github.com/gofiber/fiber/v2"
   import otelfiber "github.com/gofiber/contrib/otelfiber/v2"

   app := fiber.New()

   // 1. OTel middleware (adds trace info to context)
   app.Use(otelfiber.Middleware(otelfiber.WithServerName(cfg.ServiceName)))

   // 2. Context Logger middleware (adds trace-aware logger to context)
   //    Must run AFTER OTel middleware
   app.Use(middleware.ContextLoggerMiddleware(baseLogger))

   // 3. Request Logger (uses logger from context)
   app.Use(middleware.NewRequestLogger())

   // 4. Error Handler (uses logger from context)
   app.ErrorHandler = middleware.NewErrorHandler(baseLogger, nil) // Base logger for unhandled routes
   ```

4. Create custom spans using the common wrapper:
   ```go
   import "go.opentelemetry.io/otel/attribute"

   func myOperation(ctx context.Context) (err error) {
       // Get logger from context (it will have trace IDs!)
       logger := logging.LoggerFromContext(ctx)

       // Use common span wrapper
       ctx, span := trace.StartSpan(ctx, "myScope", "myOperation", attribute.String("key", "value"))
       defer span.End() // Ensure span is ended

       // Record metrics using common wrapper (defer AFTER span.End)
       startTime := time.Now()
       defer func() {
           // Pass the error (opErr) to the metrics recorder
           metric.RecordOperationMetrics(ctx, "myLayer", "myOperation", startTime, err, attribute.String("key", "value"))
       }()

       logger.Info("Starting operation", zap.String("some_field", "some_value"))

       // ... perform operation ...
       // if err != nil {
       //    span.RecordError(err) // Record error on span
       //    span.SetStatus(codes.Error, err.Error())
       //    logger.Error("Operation failed", zap.Error(err))
       //    return err // Return the error for metric recording
       // }

       logger.Info("Operation successful")
       return nil // Ensure opErr is nil for metric recording on success
   }
   ```

5. Logging with trace context:
   ```go
   func handleRequest(ctx context.Context) {
       // Retrieve the request-scoped logger from context
       logger := logging.LoggerFromContext(ctx)

       // This log automatically includes trace_id and span_id if available in ctx
       logger.Info("Processing request", zap.String("request_id", "xyz"))

       // ... call other functions passing ctx ...
   }
   ```

## Testing with the Simulator

The project includes a load simulator in the `tests/` directory to generate traffic to the product service. The simulator is configured to:

- Create 5 instances with Docker Compose
- Generate random product requests
- Vary request patterns to demonstrate different trace patterns

## Observability in SigNoz

After running the application with the simulator, you can observe:

1. **Traces** - End-to-end request flows with detailed span information, enriched by `trace.StartSpan`.
2. **Metrics** - Service performance metrics (`ops.count`, `ops.error.count`, `ops.duration`), custom business metrics, recorded via `metric.RecordOperationMetrics`.
3. **Logs** - Structured Zap logs correlated with traces via `ContextLoggerMiddleware` and `logging.LoggerFromContext`.
4. **Service Map** - Visual representation of service interactions

## Recommended SigNoz Dashboards

To get the most out of the telemetry data, create the following dashboards in SigNoz:

### 1. API Performance Dashboard
   - Panel: Request rate by endpoint (using `product_requests_total`)
   - Panel: Response time distribution (using `product_request_duration_ms`)
   - Panel: Error rate by endpoint (using `product_error_total`)
   - Panel: Response time percentiles (p50, p90, p99)

### 2. Product Insights Dashboard
   - Panel: Stock levels by product (using `product_stock_level`)
   - Panel: Most requested products (using `product_requests_total` with product ID attribution)
   - Panel: Zero stock alerts (using `product_stock_level` with threshold alert)

### 3. Error Analysis Dashboard
   - Panel: Error count by type (using `product_error_total`)
   - Panel: Error count by endpoint (using `product_error_total`)
   - Panel: Error rate change over time
   - Panel: Top errors with trace links

### 4. Health Overview Dashboard
   - Panel: Service health status
   - Panel: Response time trends
   - Panel: Error rate trends
   - Panel: Service dependencies health


## Troubleshooting

- **No data in SigNoz**: Ensure the OTEL_EXPORTER_OTLP_ENDPOINT in .env points to your SigNoz collector endpoint
- **Service fails to start**: Check the logs with `docker-compose logs product-service`
