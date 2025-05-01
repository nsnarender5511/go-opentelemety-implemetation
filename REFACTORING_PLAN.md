# OpenTelemetry and SigNoz Integration Refactoring Plan

This document outlines a comprehensive plan to restructure and improve the current implementation of OpenTelemetry integration with SigNoz, focusing on creating a properly architected common module and demonstrating it with a product service.

## Project Goals

1. Create a properly architected common module with OpenTelemetry integration
2. Enable easy adoption in services with minimal boilerplate
3. Demonstrate complete observability (traces, metrics, logs) with SigNoz
4. Follow best practices for containerization and service communication

## Current Issues and Solutions

### 1. Monolithic Common Module with Poor Separation of Concerns

**Problem**: The common module combines unrelated responsibilities (telemetry, config, lifecycle, errors) into a single monolithic package, violating the Single Responsibility Principle.

**Solution**: Split into focused, independent packages that can be imported separately:

```
common/
├── otel/             # OpenTelemetry core functionality
│   ├── trace.go      # Tracing setup 
│   ├── metric.go     # Metrics setup
│   ├── log.go        # Logging setup
│   ├── propagation.go # Context propagation
│   └── provider.go   # Provider configuration
├── config/           # Configuration without global state
│   ├── option.go     # Configuration options pattern
│   ├── validator.go  # Configuration validation
│   └── defaults.go   # Sensible defaults
├── http/             # HTTP abstractions
│   ├── middleware/   # Framework-agnostic middleware
│   ├── server.go     # Server abstraction
│   └── client.go     # Instrumented client
└── lifecycle/        # Application lifecycle (simplified)
    └── shutdown.go   # Clean shutdown utilities
```

### 2. Global State in Configuration

**Problem**: The config package relies heavily on package-level variables and global state, making testing difficult and creating unpredictable side effects.

**Solution**: Implement a configuration object pattern with explicit dependencies:

```go
// New approach with configuration object pattern:
package config

type Config struct {
    ServiceName string
    OtelEndpoint string
    LogLevel string
    // Other fields...
}

// NewDefaultConfig provides sensible defaults that work without env vars
func NewDefaultConfig() *Config {
    return &Config{
        ServiceName: "service",
        OtelEndpoint: "http://localhost:4317",
        LogLevel: "info",
        // Set other sensible defaults
    }
}

// WithEnv loads from environment but keeps defaults for missing values
func (c *Config) WithEnv() *Config {
    if val := os.Getenv("OTEL_SERVICE_NAME"); val != "" {
        c.ServiceName = val
    }
    // Other env vars...
    return c
}

// Validate reports issues but doesn't crash
func (c *Config) Validate() []error {
    var errs []error
    if c.ServiceName == "" {
        errs = append(errs, errors.New("service name cannot be empty"))
    }
    // Other validations...
    return errs
}
```

### 3. Overly Complex Telemetry Initialization

**Problem**: Telemetry initialization logic is complex and tightly coupled, making it difficult to maintain and test.

**Solution**: Implement a builder pattern with clear options:

```go
// otel/setup.go
package otel

import (
    "context"
    "github.com/narender/common/config"
)

// Setup encapsulates OpenTelemetry setup
type Setup struct {
    cfg *config.Config
    tracerProvider trace.TracerProvider
    meterProvider metric.MeterProvider
    loggerProvider log.LoggerProvider
    // Other fields...
}

// NewSetup creates a new OpenTelemetry setup
func NewSetup(cfg *config.Config) *Setup {
    return &Setup{cfg: cfg}
}

// WithTracing enables tracing
func (s *Setup) WithTracing(ctx context.Context) (*Setup, error) {
    tp, err := createTracerProvider(ctx, s.cfg)
    if err != nil {
        return s, err
    }
    s.tracerProvider = tp
    return s, nil
}

// Start initializes all configured providers
func (s *Setup) Start(ctx context.Context) error {
    // Initialize enabled providers...
    return nil
}

// Shutdown properly cleans up resources
func (s *Setup) Shutdown(ctx context.Context) error {
    // Shutdown logic...
    return nil
}
```

### 4. Inconsistent Error Handling

**Problem**: Error handling is inconsistent throughout the codebase with no clear strategy.

**Solution**: Create a standardized error package with clear semantics:

```go
// errors/errors.go
package errors

import (
    "fmt"
    "github.com/pkg/errors"
)

// Application error types
var (
    ErrConfiguration = errors.New("configuration error")
    ErrTelemetry     = errors.New("telemetry error")
    ErrHTTP          = errors.New("HTTP error")
    // Other error types...
)

// Wrap adds context to an error while preserving the error type
func Wrap(err error, message string) error {
    return errors.Wrap(err, message)
}

// Configuration returns a configuration error
func Configuration(format string, args ...interface{}) error {
    return errors.Wrap(ErrConfiguration, fmt.Sprintf(format, args...))
}

// Other helper functions...
```

### 5. Framework Coupling

**Problem**: The service is tightly coupled to the Fiber framework, making it difficult to change or test.

**Solution**: Create a framework-agnostic HTTP abstraction layer:

```go
// http/server.go
package http

import (
    "context"
    "net/http"
)

// Server defines a generic HTTP server interface
type Server interface {
    Start() error
    Shutdown(ctx context.Context) error
}

// Handler is a framework-agnostic HTTP handler
type Handler interface {
    Handle(w http.ResponseWriter, r *http.Request)
}

// Framework-specific adapters
type FiberAdapter struct {
    // Fiber-specific fields
}

func NewFiberAdapter(cfg *config.Config) *FiberAdapter {
    // Initialize Fiber with OpenTelemetry middleware
    return &FiberAdapter{}
}
```

### 6. Containerization Improvements

**Problem**: Docker setup could be optimized for security and efficiency.

**Solution**: Improve Dockerfile and Docker Compose configurations:

```dockerfile
# Improved Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy and download dependencies
COPY go.* ./
RUN go mod download

# Copy source code
COPY . .

# Build with proper flags
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/narender/common/config.Version=$(git describe --tags --always)" \
    -o /app/service ./cmd/service

# Use distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /app/service /app/service

# Use nonroot user
USER nonroot:nonroot

ENTRYPOINT ["/app/service"]
```

### 7. Missing Tests

**Problem**: Lack of comprehensive testing makes changes risky.

**Solution**: Implement a full test suite with unit and integration tests:

```
common/
├── otel/
│   ├── trace_test.go
│   ├── metric_test.go
│   ├── provider_test.go
│   └── testdata/
├── config/
│   ├── config_test.go
│   └── testutil/
├── http/
│   ├── middleware/
│   │   └── middleware_test.go
│   └── server_test.go
└── testing/
    ├── mock_tracer.go     # Test doubles
    ├── mock_meter.go
    ├── mock_exporter.go
    └── integration/       # Integration tests
```

## Implementation Plan

### Phase 1: Core Architecture Refactoring (Week 1)

1. **Create new package structure**
   - Restructure the common module with focused packages
   - Set up new import paths and module structure

2. **Implement configuration object pattern**
   - Replace global variables with configuration objects
   - Add sensible defaults and validation
   - Create environment loading utilities

3. **Remove global state**
   - Refactor all consumers of global state to use explicit dependencies
   - Create factories and constructors where needed

4. **Add proper error handling**
   - Implement standardized error package
   - Update error handling throughout the codebase

### Phase 2: OpenTelemetry Integration (Week 2)

1. **Implement simplified telemetry setup**
   - Create builder pattern for telemetry initialization
   - Extract functionality into focused components

2. **Add tracing, metrics, and logging with proper separation**
   - Separate tracing, metrics, and logging concerns
   - Create clean interfaces for each

3. **Create HTTP abstraction layer**
   - Implement framework-agnostic interfaces
   - Create adapters for Fiber and standard library

4. **Add context propagation utilities**
   - Implement utilities for propagating context across services
   - Add helpers for distributed tracing

### Phase 3: Containerization and Service Implementation (Week 3)

1. **Update Dockerfiles and compose configuration**
   - Optimize for security and efficiency
   - Add health checks and resource limits

2. **Refactor product service to use new common module**
   - Adapt the product service to the new interfaces
   - Clean up service code

3. **Implement health checks and observability patterns**
   - Add health check endpoints
   - Implement standardized observability patterns

### Phase 4: Testing and Documentation (Week 4)

1. **Add unit and integration tests**
   - Create comprehensive test suite
   - Implement CI pipeline for testing

2. **Create examples and documentation**
   - Document common module usage
   - Create example applications

3. **Add deployment guidelines**
   - Document deployment process
   - Add monitoring and troubleshooting guides

## OpenTelemetry Best Practices

Following [OpenTelemetry best practices](https://opentelemetry.io/docs/languages/go/):

1. **Use Resources for service information**
   ```go
   resource := resource.NewWithAttributes(
       semconv.SchemaURL,
       semconv.ServiceNameKey.String(cfg.ServiceName),
       semconv.ServiceVersionKey.String(cfg.ServiceVersion),
   )
   ```

2. **Implement correct context propagation**
   ```go
   // Set up propagator
   prop := propagation.NewCompositeTextMapPropagator(
       propagation.TraceContext{},
       propagation.Baggage{},
   )
   otel.SetTextMapPropagator(prop)
   ```

3. **Use semantic conventions for naming**
   ```go
   meter.Counter(
       "http.server.request.count",
       metric.WithDescription("Total number of HTTP requests"),
   )
   ```

4. **Implement proper span attributes**
   ```go
   span.SetAttributes(
       semconv.HTTPMethodKey.String(r.Method),
       semconv.HTTPRouteKey.String(route),
       semconv.HTTPStatusCodeKey.Int(statusCode),
   )
   ```

## SigNoz Integration

Following [SigNoz documentation](https://signoz.io/docs/introduction/):

1. **Configure OTLP exporters for SigNoz**
   ```go
   exporter, err := otlptrace.New(
       context.Background(),
       otlptracegrpc.NewClient(
           otlptracegrpc.WithEndpoint(cfg.OtelEndpoint),
           otlptracegrpc.WithInsecure(),
       ),
   )
   ```

2. **Add custom metrics for business monitoring**
   ```go
   // Business metrics
   productCounter := meter.Int64Counter(
       "business.product.views",
       metric.WithDescription("Number of product views"),
   )
   ```

3. **Implement logging with trace correlation**
   ```go
   logger.WithFields(logrus.Fields{
       "trace_id": span.SpanContext().TraceID().String(),
       "span_id":  span.SpanContext().SpanID().String(),
   }).Info("Product requested")
   ```

## Containerization Best Practices

Following [Docker best practices](https://docs.docker.com/compose/):

1. **Use multi-stage builds**
2. **Implement proper health checks**
3. **Set resource limits**
4. **Use non-root users**
5. **Optimize layer caching**
6. **Implement proper logging configuration**

## Success Criteria

The refactoring will be considered successful when:

1. The common module can be used with minimal boilerplate in new services
2. Complete observability data (traces, metrics, logs) is visible in SigNoz
3. All identified architectural issues are resolved
4. Comprehensive test coverage is achieved
5. Documentation enables easy adoption

## References

- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/languages/go/)
- [SigNoz Documentation](https://signoz.io/docs/introduction/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Docker Best Practices](https://www.docker.com/)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
