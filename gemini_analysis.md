OpenTelemetry Integration Plan for Go Microservice
This document outlines the steps to integrate OpenTelemetry (OTel) into your existing Go application structure, specifically focusing on the product-service and leveraging the common directory for shared telemetry logic.
Goals:
Centralized Telemetry Setup: Create a common/telemetry package for reusable OTel initialization.
Tracing: Implement automatic request tracing (via Fiber middleware) and add custom spans for specific operations.
Metrics: Capture standard host/runtime metrics and implement custom application metrics.
Logging: Integrate Logrus with OTel to correlate logs with traces and export them.
Exporting: Configure OTel to export telemetry data via OTLP/gRPC (a common standard).
Prerequisites:
An OTel Collector running and accessible (e.g., SigNoz, Jaeger, Grafana Agent). We'll assume it's listening on localhost:4317 (standard OTLP gRPC port).
Go environment set up.
Step 1: Create the telemetry Package Structure
What: Define the directory structure for your shared telemetry code.
Where: Inside the common directory.
Why: To keep OTel configuration and initialization logic organized and reusable across different services if needed.
mkdir -p common/telemetry
touch common/telemetry/init.go
touch common/telemetry/config.go
touch common/telemetry/resource.go
touch common/telemetry/trace.go
touch common/telemetry/metric.go
touch common/telemetry/log.go


Step 2: Add OTel Dependencies
What: Add the necessary Go modules for OTel core, SDK, exporters, and instrumentation.
Where: In your project's root go.mod file (run these commands in the project root directory).
Why: To bring in the libraries required for OTel functionality.
# OTel Core & SDK
go get go.opentelemetry.io/otel \
       go.opentelemetry.io/otel/sdk \
       go.opentelemetry.io/otel/trace \
       go.opentelemetry.io/otel/metric \
       go.opentelemetry.io/otel/log \
       go.opentelemetry.io/otel/log/global \
       go.opentelemetry.io/otel/sdk/log \
       go.opentelemetry.io/otel/semconv/v1.25.0 # Semantic Conventions

# OTLP Exporters (gRPC)
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc \
       go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc \
       go.opentelemetry.io/otel/exporters/otlp/otlplogs/otlplogsgrpc

# Instrumentation Libraries
go get go.opentelemetry.io/contrib/instrumentation/runtime \
       go.opentelemetry.io/contrib/instrumentation/host \
       go.opentelemetry.io/contrib/instrumentation/github.com/gofiber/fiber/v2/otelgofiber

# Other necessary libraries (ensure grpc is present)
go get google.golang.org/grpc

# Tidy up dependencies
go mod tidy


Step 3: Configure Telemetry Settings
What: Define configuration options for OTel, primarily loaded from environment variables.
Where: common/telemetry/config.go and update common/config/config.go.
Why: To make the OTel setup adaptable without code changes (e.g., changing the exporter endpoint or service name).
1. Update common/config/config.go:
Add new variables for OTel settings.
package config

import (
	"github.com/spf13/viper"
)

// Add OTel defaults
var defaultConfigs = map[string]interface{}{
	"PRODUCT_SERVICE_PORT":        "8082",
	"LOG_LEVEL":                   "info",
	"LOG_FORMAT":                  "text",
	"OTEL_SERVICE_NAME":           "product-service", // Default service name
	"OTEL_EXPORTER_OTLP_ENDPOINT": "localhost:4317",  // Default OTLP gRPC endpoint
	"OTEL_EXPORTER_INSECURE":      "true",            // Use "false" for TLS
}

var (
	PRODUCT_SERVICE_PORT        string
	LOG_LEVEL                   string
	LOG_FORMAT                  string
	OTEL_SERVICE_NAME           string // New
	OTEL_EXPORTER_OTLP_ENDPOINT string // New
	OTEL_EXPORTER_INSECURE      bool   // New
)

func init() {
	viper.AutomaticEnv() // Reads from environment variables

	for key, value := range defaultConfigs {
		viper.SetDefault(key, value)
	}

	PRODUCT_SERVICE_PORT = viper.GetString("PRODUCT_SERVICE_PORT")
	LOG_LEVEL = viper.GetString("LOG_LEVEL")
	LOG_FORMAT = viper.GetString("LOG_FORMAT")

	// Read OTel config
	OTEL_SERVICE_NAME = viper.GetString("OTEL_SERVICE_NAME")
	OTEL_EXPORTER_OTLP_ENDPOINT = viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT")
	OTEL_EXPORTER_INSECURE = viper.GetBool("OTEL_EXPORTER_INSECURE") // Viper handles string "true"/"false"
}


2. Create common/telemetry/config.go:
This file will hold the telemetry-specific config structure, pulling values from the main config package.
package telemetry

import (
	// Use your actual module path for the common config
	"your_module_path/common/config" // <-- ADJUST THIS IMPORT PATH
)

// Config holds telemetry-specific configuration.
type Config struct {
	ServiceName          string
	ExporterEndpoint     string
	ExporterInsecure     bool
	// Add other relevant fields like ServiceNamespace, DeploymentEnvironment if needed
}

// LoadConfig creates a Telemetry Config from the global application config.
func LoadConfig() Config {
	return Config{
		ServiceName:      config.OTEL_SERVICE_NAME,
		ExporterEndpoint: config.OTEL_EXPORTER_OTLP_ENDPOINT,
		ExporterInsecure: config.OTEL_EXPORTER_INSECURE,
	}
}


Important: Replace "your_module_path/common/config" with the actual Go module path to your common/config package (e.g., github.com/youruser/signoz-assignment/common/config).
Step 4: Create the OTel Resource
What: Define the OTel resource, which describes the service emitting telemetry.
Where: common/telemetry/resource.go
Why: To identify your service (product-service) in the OTel backend and attach common attributes (like service name, version, environment) to all telemetry signals.
package telemetry

import (
	"context"
	"log" // Use standard log for setup errors

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0" // Use the latest stable semantic conventions
)

// newResource creates an OTel Resource describing this service.
func newResource(ctx context.Context, cfg Config) *resource.Resource {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			// --- Essential Attributes ---
			semconv.ServiceName(cfg.ServiceName), // From config

			// --- Optional Attributes (Add more as needed) ---
			// semconv.ServiceVersion("1.0.0"), // Set your service version
			// semconv.DeploymentEnvironment("production"), // e.g., production, staging
			// semconv.ServiceNamespace("your-namespace"),
		),
		// Automatically detect attributes from the environment (e.g., K8s pod name)
		resource.WithFromEnv(),
		// Detect host and OS attributes
		resource.WithHost(),
		// Detect process attributes (PID, executable name, etc.)
		resource.WithProcess(),
		// Detect runtime attributes (Go version)
		resource.WithProcessRuntimeDescription(),
		// Add other detectors if relevant (e.g., resource.WithContainer())
	)

	if err != nil {
		// Log error but return a default resource to avoid crashing
		log.Printf("Error creating OTel resource: %v. Using default.", err)
		// Merge with default to still get *some* basic attributes
		res, _ = resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(cfg.ServiceName)))
		return res
	}

	// Merge with default attributes (like schema URL) for completeness
	mergedRes, err := resource.Merge(resource.Default(), res)
	if err != nil {
		log.Printf("Error merging OTel resources: %v. Using created resource.", err)
		return res // Return the one we created if merge fails
	}

	log.Printf("OTel Resource created with service name: %s", cfg.ServiceName)
	return mergedRes
}


Step 5: Set Up Tracing
What: Configure the Trace Provider and OTLP Trace Exporter.
Where: common/telemetry/trace.go
Why: To enable the collection and export of distributed traces. The provider manages span creation, and the exporter sends them to the collector.
package telemetry

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure" // Use secure credentials in production
)

// initTracerProvider initializes and registers the OTLP Trace Provider.
func initTracerProvider(ctx context.Context, cfg Config, res *resource.Resource) (shutdownFunc func(context.Context) error, err error) {
	// --- Create OTLP Exporter ---
	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.ExporterEndpoint),
		// Add compression if desired: otlptracegrpc.WithCompressor(grpc.UseCompressor(gzip.Name)),
		// Add headers if needed: otlptracegrpc.WithHeaders(map[string]string{"api-key": "your-key"}),
		otlptracegrpc.WithDialOption(grpc.WithBlock()), // Wait for connection to be established
	}

	if cfg.ExporterInsecure {
		// Use insecure transport (suitable for local development)
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
	} else {
		// Use secure transport (TLS) - Recommended for production
		// Requires proper TLS configuration on the collector and potentially client certs
		// exporterOpts = append(exporterOpts, otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
		log.Println("Warning: Secure OTLP exporter not fully configured. Using insecure for now.")
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure()) // Fallback for now
	}

	traceExporter, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}
	log.Println("OTLP Trace Exporter created.")

	// --- Create Batch Span Processor ---
	// Processes spans in batches before exporting, more efficient than SimpleSpanProcessor.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter,
		// Adjust options as needed:
		// sdktrace.WithMaxQueueSize(2048),
		// sdktrace.WithMaxExportBatchSize(512),
		// sdktrace.WithExportTimeout(30*time.Second),
		// sdktrace.WithScheduledDelay(5*time.Second),
	)
	log.Println("Batch Span Processor created.")

	// --- Create Tracer Provider ---
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res), // Attach the resource information
		sdktrace.WithSpanProcessor(bsp),
		// Consider adding a sampler in production environments:
		// sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))), // Sample 10% of traces
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sample everything (good for dev/demo)
	)
	log.Println("Tracer Provider created.")

	// --- Set Global Tracer Provider and Propagator ---
	// Register the provider as the global default.
	otel.SetTracerProvider(tracerProvider)

	// Register the W3C Trace Context and Baggage propagators.
	// This allows context (trace IDs, baggage) to be passed across service boundaries.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // Standard W3C Trace Context
		propagation.Baggage{},      // Standard W3C Baggage
	))
	log.Println("Global Tracer Provider and Propagator set.")

	// Return the shutdown function for graceful cleanup.
	shutdown = func(shutdownCtx context.Context) error {
		log.Println("Shutting down Tracer Provider...")
		err := tracerProvider.Shutdown(shutdownCtx)
		if err != nil {
			log.Printf("Error shutting down Tracer Provider: %v", err)
		} else {
			log.Println("Tracer Provider shut down successfully.")
		}
		return err
	}

	return shutdown, nil
}

// GetTracer returns a named tracer instance.
func GetTracer(instrumentationName string) trace.Tracer {
    // otel.Tracer uses the globally registered TracerProvider.
	return otel.Tracer(instrumentationName)
}


Step 6: Set Up Metrics
What: Configure the Meter Provider, OTLP Metric Exporter, and basic host/runtime metrics.
Where: common/telemetry/metric.go
Why: To enable the collection and export of metrics. The provider manages metric instruments, the exporter sends data, and host/runtime metrics provide baseline system visibility.
package telemetry

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/host"     // Host metrics (CPU, memory)
	"go.opentelemetry.io/contrib/instrumentation/runtime" // Go runtime metrics (GC, goroutines)
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// initMeterProvider initializes and registers the OTLP Meter Provider.
func initMeterProvider(ctx context.Context, cfg Config, res *resource.Resource) (shutdownFunc func(context.Context) error, err error) {
	// --- Create OTLP Exporter ---
	exporterOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.ExporterEndpoint),
		// Add compression if desired
		// Add headers if needed
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	}

	if cfg.ExporterInsecure {
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithInsecure())
	} else {
		// Configure TLS for production
		log.Println("Warning: Secure OTLP metric exporter not fully configured. Using insecure.")
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithInsecure()) // Fallback
	}

	metricExporter, err := otlpmetricgrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}
	log.Println("OTLP Metric Exporter created.")

	// --- Create Periodic Reader ---
	// Exports metrics periodically (e.g., every 15 seconds).
	reader := sdkmetric.NewPeriodicReader(metricExporter,
		sdkmetric.WithInterval(15*time.Second), // Adjust interval as needed
	)
	log.Println("Periodic Metric Reader created.")

	// --- Create Meter Provider ---
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res), // Attach the resource information
		sdkmetric.WithReader(reader),
		// Add Views here to customize metric aggregation, naming, etc. if needed
		// sdkmetric.WithView(...)
	)
	log.Println("Meter Provider created.")

	// --- Set Global Meter Provider ---
	otel.SetMeterProvider(meterProvider)
	log.Println("Global Meter Provider set.")

	// --- Start Host & Runtime Metrics Collection ---
	// These instrumentations use the global MeterProvider we just set.
	err = runtime.Start(runtime.WithMeterProvider(meterProvider))
	if err != nil {
		log.Printf("Warning: Failed to start runtime metrics: %v", err)
		// Continue initialization even if this fails
	} else {
		log.Println("Runtime metrics collection started.")
	}

	err = host.Start(host.WithMeterProvider(meterProvider))
	if err != nil {
		log.Printf("Warning: Failed to start host metrics: %v", err)
		// Continue initialization
	} else {
		log.Println("Host metrics collection started.")
	}

	// Return the shutdown function.
	shutdown = func(shutdownCtx context.Context) error {
		log.Println("Shutting down Meter Provider...")
		// Timeout for shutdown, as it might involve flushing data.
		shutdownTimeoutCtx, cancel := context.WithTimeout(shutdownCtx, 5*time.Second)
		defer cancel()
		err := meterProvider.Shutdown(shutdownTimeoutCtx)
		if err != nil {
			log.Printf("Error shutting down Meter Provider: %v", err)
		} else {
			log.Println("Meter Provider shut down successfully.")
		}
		return err
	}

	return shutdown, nil
}

// GetMeter returns a named meter instance.
func GetMeter(instrumentationName string) metric.Meter {
    // otel.Meter uses the globally registered MeterProvider.
	return otel.Meter(instrumentationName)
}


Step 7: Set Up Logging (Logrus Hook)
What: Configure the OTel Log Provider, OTLP Log Exporter, and a custom Logrus hook to bridge logs.
Where: common/telemetry/log.go
Why: To send application logs (written via Logrus) to the OTel backend alongside traces and metrics. The hook automatically adds trace context (trace ID, span ID) to logs when available, enabling correlation.
package telemetry

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus" // Your logging library

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplogs/otlplogsgrpc"
	otelLogs "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global" // Global logger provider
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace" // To extract trace context
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var otelLogger otelLogs.Logger // Package-level variable for the OTel logger instance

// initLoggerProvider initializes the OTel LoggerProvider and sets the global provider.
func initLoggerProvider(ctx context.Context, cfg Config, res *resource.Resource) (shutdownFunc func(context.Context) error, err error) {
	// --- Create OTLP Exporter ---
	exporterOpts := []otlplogsgrpc.Option{
		otlplogsgrpc.WithEndpoint(cfg.ExporterEndpoint),
		otlplogsgrpc.WithDialOption(grpc.WithBlock()),
	}

	if cfg.ExporterInsecure {
		exporterOpts = append(exporterOpts, otlplogsgrpc.WithInsecure())
	} else {
		log.Println("Warning: Secure OTLP log exporter not fully configured. Using insecure.")
		exporterOpts = append(exporterOpts, otlplogsgrpc.WithInsecure()) // Fallback
	}

	logExporter, err := otlplogsgrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}
	log.Println("OTLP Log Exporter created.")

	// --- Create Batch Log Record Processor ---
	logProcessor := sdklog.NewBatchProcessor(logExporter,
		// Adjust options as needed
		// sdklog.WithExportTimeout(30*time.Second),
		// sdklog.WithMaxQueueSize(2048),
		// sdklog.WithMaxExportBatchSize(512),
	)
	log.Println("Batch Log Record Processor created.")

	// --- Create Logger Provider ---
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),        // Attach the resource
		sdklog.WithProcessor(logProcessor),
		// Add other options like sdklog.WithAttributeProvider() if needed
	)
	log.Println("Logger Provider created.")

	// --- Set Global Logger Provider ---
	// This makes the provider accessible via global.LoggerProvider().
	global.SetLoggerProvider(loggerProvider)
	log.Println("Global Logger Provider set.")

	// --- Get a Logger instance ---
	// Use a specific name for logs originating from this setup process or the hook itself.
	// The instrumentation scope name helps identify the source of the logs in the backend.
	// Replace 'your_module_path/common/telemetry' with your actual module path.
	otelLogger = loggerProvider.Logger("your_module_path/common/telemetry") // <-- ADJUST THIS
	log.Println("OTel Logger instance obtained.")

	// Return the shutdown function.
	shutdown = func(shutdownCtx context.Context) error {
		log.Println("Shutting down Logger Provider...")
		err := loggerProvider.Shutdown(shutdownCtx)
		if err != nil {
			log.Printf("Error shutting down Logger Provider: %v", err)
		} else {
			log.Println("Logger Provider shut down successfully.")
		}
		return err
	}

	return shutdown, nil
}

// --- Custom Logrus Hook for OpenTelemetry ---

// OtelHook implements logrus.Hook to send logs to OpenTelemetry.
type OtelHook struct {
	// Levels defines on which log levels this hook should fire.
	LogLevels []logrus.Level
}

// NewOtelHook creates a new hook for the specified levels.
func NewOtelHook(levels []logrus.Level) *OtelHook {
	return &OtelHook{LogLevels: levels}
}

// Levels returns the log levels this hook is registered for.
func (h *OtelHook) Levels() []logrus.Level {
	if h.LogLevels == nil {
		return logrus.AllLevels // Default to all levels if not specified
	}
	return h.LogLevels
}

// Fire is called by Logrus when a log entry is made.
func (h *OtelHook) Fire(entry *logrus.Entry) error {
	if otelLogger == nil {
		// This should not happen if InitTelemetry is called first, but it's a safeguard.
		// Avoid logging here to prevent infinite loops if logging itself fails.
		fmt.Fprintf(os.Stderr, "Error: OpenTelemetry logger not initialized in Logrus hook.\n")
		return nil // Don't block Logrus processing
	}

	// --- Get Context and Trace Information ---
	// Crucial step: Extract the context potentially passed via logrus.WithContext()
	ctx := entry.Context
	if ctx == nil {
		ctx = context.Background() // Use background context if none is provided
	}

	span := trace.SpanFromContext(ctx)
	spanContext := span.SpanContext()

	// --- Map Logrus Level to OTel Severity ---
	severity := mapLogLevel(entry.Level)

	// --- Prepare OTel Log Record ---
	record := otelLogs.Record{}
	record.SetTimestamp(entry.Time)           // Use Logrus timestamp
	record.SetObservedTimestamp(time.Now()) // When the hook observed the log
	record.SetSeverity(severity)
	record.SetSeverityText(entry.Level.String())
	record.SetBody(otelLogs.StringValue(entry.Message)) // Set the log message

	// --- Add Attributes from Logrus Fields ---
	// Reserve capacity: number of fields + trace/span IDs + potential caller info
	attrs := make([]attribute.KeyValue, 0, len(entry.Data)+5)

	// Add standard trace and span IDs if available
	if spanContext.IsValid() {
		record.SetTraceID(spanContext.TraceID())
		record.SetSpanID(spanContext.SpanID())
		record.SetTraceFlags(spanContext.TraceFlags())
		// Also add as attributes for easier searching/filtering in some backends
		attrs = append(attrs, semconv.TraceIDKey.String(spanContext.TraceID().String()))
		attrs = append(attrs, semconv.SpanIDKey.String(spanContext.SpanID().String()))
	}

	// Add all fields from the Logrus entry
	for k, v := range entry.Data {
		// Prefix Logrus fields to avoid potential clashes with OTel standard attributes
		attrs = append(attrs, attribute.Any(fmt.Sprintf("logrus.data.%s", k), v))
	}

	// Add Logrus-specific attributes
	attrs = append(attrs, attribute.String("log.iostream", "stdout")) // Assuming logs also go to stdout
	attrs = append(attrs, attribute.String("log.library", "logrus"))

	// Add caller information if available (requires logrus.SetReportCaller(true))
	if entry.HasCaller() {
		attrs = append(attrs, semconv.CodeFunctionKey.String(entry.Caller.Function))
		attrs = append(attrs, semconv.CodeFilepathKey.String(entry.Caller.File))
		attrs = append(attrs, semconv.CodeLineNumberKey.Int(entry.Caller.Line))
	}

	record.AddAttributes(attrs...) // Add all collected attributes

	// --- Emit the Log Record ---
	// Use the specific otelLogger instance obtained during initialization.
	otelLogger.Emit(ctx, record)

	return nil
}

// mapLogLevel converts Logrus level to OTel severity number.
func mapLogLevel(level logrus.Level) otelLogs.Severity {
	switch level {
	case logrus.TraceLevel:
		return otelLogs.SeverityTrace // SEVERITY_NUMBER_TRACE (1)
	case logrus.DebugLevel:
		return otelLogs.SeverityDebug // SEVERITY_NUMBER_DEBUG (5)
	case logrus.InfoLevel:
		return otelLogs.SeverityInfo // SEVERITY_NUMBER_INFO (9)
	case logrus.WarnLevel:
		return otelLogs.SeverityWarn // SEVERITY_NUMBER_WARN (13)
	case logrus.ErrorLevel:
		return otelLogs.SeverityError // SEVERITY_NUMBER_ERROR (17)
	case logrus.FatalLevel:
		return otelLogs.SeverityFatal // SEVERITY_NUMBER_FATAL (21)
	case logrus.PanicLevel:
		return otelLogs.SeverityFatal // SEVERITY_NUMBER_FATAL (21) - Map Panic to Fatal
	default:
		return otelLogs.SeverityInfo // Default to Info for unknown levels
	}
}

// ConfigureLogrus adds the OTel hook to the global Logrus instance.
// Call this *after* initLoggerProvider has run successfully.
func ConfigureLogrus() {
	// Determine which levels to send to OTel
	otelLogLevels := []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		// Add Debug/Trace if you want verbose logs in OTel backend
		// logrus.DebugLevel,
		// logrus.TraceLevel,
	}

	hook := NewOtelHook(otelLogLevels)

	// Add the hook to the standard Logrus logger
	logrus.AddHook(hook)

	// Optional: Keep Logrus writing to stdout/stderr as well
	// logrus.SetOutput(os.Stdout) // Or your desired output

	// Optional: Use JSON formatter for potentially better parsing by log collectors
	// logrus.SetFormatter(&logrus.JSONFormatter{
	// 	TimestampFormat: time.RFC3339Nano, // High precision timestamp
	// })

	// Optional: Enable reporting caller info (adds overhead)
	// logrus.SetReportCaller(true)

	log.Println("Logrus configured with OpenTelemetry hook.")
}


Important: Replace 'your_module_path/common/telemetry' with your actual module path in otelLogger = loggerProvider.Logger(...).
Step 8: Initialize Telemetry Components
What: Create a master initialization function that sets up tracing, metrics, and logging, and returns a combined shutdown function.
Where: common/telemetry/init.go
Why: To provide a single entry point for initializing all OTel components in the correct order and manage their graceful shutdown.
package telemetry

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
	// No need to import sub-packages like trace, metric, log here
)

// shutdownFunc defines the signature for shutdown functions returned by initializers.
type shutdownFunc func(context.Context) error

// InitTelemetry initializes OpenTelemetry Tracing, Metrics, and Logging.
// It loads configuration, creates resources, sets up providers/exporters,
// configures the Logrus hook, and returns a master shutdown function.
func InitTelemetry() (func(context.Context) error, error) {
	cfg := LoadConfig() // Load config using the function from telemetry/config.go
	log.Printf("Initializing Telemetry for service: %s, endpoint: %s, insecure: %t",
		cfg.ServiceName, cfg.ExporterEndpoint, cfg.ExporterInsecure)

	// Use a timeout for the initial setup context.
	initCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // Increased timeout
	defer cancel()

	// --- 1. Create Resource ---
	res := newResource(initCtx, cfg)

	shutdownFuncs := make([]shutdownFunc, 0, 3) // Store shutdown functions
	var initErr error                           // To capture the first error during init

	// --- 2. Initialize Trace Provider ---
	tracerShutdown, err := initTracerProvider(initCtx, cfg, res)
	if err != nil {
		log.Printf("Error initializing TracerProvider: %v", err)
		initErr = errors.Join(initErr, fmt.Errorf("tracer init failed: %w", err)) // Collect errors
	} else if tracerShutdown != nil {
		shutdownFuncs = append(shutdownFuncs, tracerShutdown)
		log.Println("TracerProvider initialization successful.")
	}

	// --- 3. Initialize Meter Provider ---
	meterShutdown, err := initMeterProvider(initCtx, cfg, res)
	if err != nil {
		log.Printf("Error initializing MeterProvider: %v", err)
		initErr = errors.Join(initErr, fmt.Errorf("meter init failed: %w", err))
	} else if meterShutdown != nil {
		shutdownFuncs = append(shutdownFuncs, meterShutdown)
		log.Println("MeterProvider initialization successful.")
	}

	// --- 4. Initialize Logger Provider ---
	// This MUST happen before ConfigureLogrus.
	loggerShutdown, err := initLoggerProvider(initCtx, cfg, res)
	if err != nil {
		log.Printf("Error initializing LoggerProvider: %v", err)
		initErr = errors.Join(initErr, fmt.Errorf("logger init failed: %w", err))
	} else if loggerShutdown != nil {
		shutdownFuncs = append(shutdownFuncs, loggerShutdown)
		log.Println("LoggerProvider initialization successful.")
	}

	// --- 5. Configure Logrus Hook ---
	// Only configure the hook if the logger provider initialized successfully.
	if loggerShutdown != nil {
		ConfigureLogrus() // Sets up the hook to use the initialized otelLogger
	} else {
		log.Println("Skipping Logrus hook configuration due to LoggerProvider init failure.")
	}

	if initErr != nil {
		log.Printf("OpenTelemetry initialization failed with errors: %v", initErr)
		// Attempt to shut down any components that *did* initialize successfully
		// We still return the master shutdown func, but also the init error.
		masterShutdownPartial := createMasterShutdown(shutdownFuncs)
		return masterShutdownPartial, initErr
	}

	log.Println("OpenTelemetry initialization complete.")

	// --- 6. Create Master Shutdown Function ---
	masterShutdown := createMasterShutdown(shutdownFuncs)

	// Return the master shutdown function and nil error if all initializations were okay.
	return masterShutdown, nil
}

// createMasterShutdown creates a function that calls all individual shutdown functions concurrently.
func createMasterShutdown(shutdownFuncs []shutdownFunc) func(context.Context) error {
	return func(shutdownCtx context.Context) error {
		log.Println("Starting OpenTelemetry master shutdown...")
		var wg sync.WaitGroup
		var multiErr error // Use errors.Join for better multiple error handling

		// Use a shorter timeout for individual shutdowns within the overall context.
		individualShutdownTimeout := 5 * time.Second

		wg.Add(len(shutdownFuncs))
		for _, fn := range shutdownFuncs {
			go func(shutdown shutdownFunc) {
				defer wg.Done()
				// Create a derived context with a timeout for this specific shutdown
				ctx, cancel := context.WithTimeout(shutdownCtx, individualShutdownTimeout)
				defer cancel()

				if err := shutdown(ctx); err != nil {
					log.Printf("Error during OTel component shutdown: %v", err)
					multiErr = errors.Join(multiErr, err) // Collect errors safely
				}
			}(fn)
		}

		wg.Wait() // Wait for all shutdowns to complete or time out

		if multiErr != nil {
			log.Printf("OpenTelemetry master shutdown finished with errors: %v", multiErr)
		} else {
			log.Println("OpenTelemetry master shutdown finished successfully.")
		}
		return multiErr
	}
}


Step 9: Integrate Telemetry into product-service
What: Call the initialization function, add Fiber middleware, and update logging calls.
Where: product-service/src/main.go and potentially handler.go, service.go.
Why: To activate OTel instrumentation for the service.
1. Update product-service/src/main.go:
package main

import (
	"context"
	"log" // Use standard log *only* for initial setup errors before Logrus is fully configured
	"os"
	"os/signal"
	"syscall"
	"time"

	// Use your actual module paths
	config "your_module_path/common/config" // <-- ADJUST
	// logger "your_module_path/common/logger" // We'll use Logrus directly now
	telemetry "your_module_path/common/telemetry" // <-- ADJUST

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover" // Good practice
	"github.com/sirupsen/logrus"                       // Use Logrus directly

	// OTel Fiber Middleware
	"go.opentelemetry.io/contrib/instrumentation/github.com/gofiber/fiber/v2/otelgofiber"
)

// Remove the old logger init() block if it exists

func main() {
	// --- 1. Initialize OpenTelemetry ---
	// This MUST be one of the first things to run.
	// It sets up global providers and configures the Logrus hook.
	otelShutdown, err := telemetry.InitTelemetry()
	if err != nil {
		// Use standard log here as Logrus might not be fully OTel-ready yet if init failed
		log.Fatalf("Failed to initialize OpenTelemetry: %v", err)
	}
	// Defer the master shutdown function for graceful cleanup on exit.
	defer func() {
		// Allow time for OTel exporters to flush data
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logrus.Info("Shutting down OpenTelemetry components...") // Use Logrus now
		if err := otelShutdown(shutdownCtx); err != nil {
			logrus.Errorf("Error during OpenTelemetry shutdown: %v", err)
		} else {
			logrus.Info("OpenTelemetry shutdown complete.")
		}
	}()

	// --- 2. Configure Application Logging (Logrus) ---
	// Set Logrus level and format from config AFTER OTel init (which adds the hook)
	logLevel, err := logrus.ParseLevel(config.LOG_LEVEL)
	if err != nil {
		logrus.Warnf("Invalid log level '%s', defaulting to 'info'. Error: %v", config.LOG_LEVEL, err)
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)
	logrus.SetOutput(os.Stdout) // Or your preferred output
	if config.LOG_FORMAT == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, TimestampFormat: time.RFC3339Nano})
	}
	// Optional: logrus.SetReportCaller(true) // If you want caller info in logs (added by hook too)

	// Now you can use Logrus for all subsequent logging
	logrus.Infof("Starting Product Service (service: %s, port: %s)", config.OTEL_SERVICE_NAME, config.PRODUCT_SERVICE_PORT)

	// --- 3. Setup Application Dependencies ---
	productRepo := NewProductRepository()       // Assuming this uses Logrus internally now
	productService := NewProductService(productRepo) // Assuming this uses Logrus internally now
	productHandler := NewProductHandler(productService)

	// --- 4. Setup Fiber App ---
	app := fiber.New(fiber.Config{
		// Optional: Configure Fiber error handler to potentially capture errors in spans
		// ErrorHandler: func(c *fiber.Ctx, err error) error { ... }
	})

	// --- 5. Add Middleware ---
	// IMPORTANT: OTel middleware should generally be among the FIRST middleware
	// It starts the trace span for the request.
	app.Use(otelgofiber.Middleware(otelgofiber.WithServerName(config.OTEL_SERVICE_NAME)))
	logrus.Info("Added OpenTelemetry Fiber middleware.")

	// Add other essential middleware AFTER OTel middleware
	app.Use(recover.New()) // Catches panics and recovers
	// app.Use(fiberlogger.New()) // You might remove Fiber's logger if Logrus covers it

	// --- 6. Define Routes ---
	api := app.Group("/products")
	api.Get("/", productHandler.GetAllProducts)
	api.Get("/:productId", productHandler.GetProductByID)
	api.Get("/:productId/stock", productHandler.GetProductStock)
	logrus.Info("API routes registered.")

	// --- 7. Start Server & Handle Graceful Shutdown ---
	go func() {
		logrus.Infof("Server listening on port %s", config.PRODUCT_SERVICE_PORT)
		if err := app.Listen(":" + config.PRODUCT_SERVICE_PORT); err != nil {
			logrus.Fatalf("Failed to start server: %v", err) // Use Fatalf to exit
		}
	}()

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Received termination signal. Shutting down server...")

	// Gracefully shut down the Fiber server
	// Give it a deadline to finish ongoing requests.
	shutdownCtxServer, cancelServer := context.WithTimeout(context.Background(), 15*time.Second) // Adjust timeout
	defer cancelServer()
	if err := app.ShutdownWithContext(shutdownCtxServer); err != nil {
		logrus.Errorf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server shutdown complete.")
	// Deferred OTel shutdown will run after this.
}


Important: Replace "your_module_path/..." with your actual Go module paths.
Note: We removed the direct dependency on common/logger and now use logrus directly, relying on the OTel hook configured during telemetry.InitTelemetry().
2. Update Logging Calls (handler.go, service.go, repository.go):
Modify existing logging calls to use logrus directly and importantly, pass the request context when available using logrus.WithContext(ctx).
Example in handler.go:
package main

import (
	"fmt"
	"net/http"
	// Use your actual module paths
	"your_module_path/common/errors" // <-- ADJUST
	// logger "your_module_path/common/logger" // Remove this
	"github.com/sirupsen/logrus" // Use Logrus

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute" // For adding attributes to spans
	"go.opentelemetry.io/otel/trace"     // For spans
)

// ... (ProductHandler struct) ...

// GetAllProducts handles GET /products
func (h *ProductHandler) GetAllProducts(c *fiber.Ctx) error {
	// Get context from Fiber (contains trace info from middleware)
	requestCtx := c.UserContext()

	// Use logrus.WithContext to link log to the current trace
	log := logrus.WithContext(requestCtx)
	log.Info("Handler: Received request to get all products")

	products, err := h.service.GetAll(requestCtx) // Pass context down
	if err != nil {
		// Error handled logs the error with context already
		return errors.HandleServiceError(c, err, "get all products")
	}

	log.Infof("Handler: Responding with %d products", len(products))
	return c.Status(http.StatusOK).JSON(products)
}

// GetProductByID handles GET /products/:productId
func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	requestCtx := c.UserContext()
	log := logrus.WithContext(requestCtx)

	productID, errResp := h.validatePathParam(c, "productId") // Keep validation as is
	if errResp != nil {
		return errResp // Validation error already logged if needed
	}
	log = log.WithField("product_id", productID) // Add product_id to subsequent logs
	log.Info("Handler: Received request to get product by ID")

	product, err := h.service.GetByID(requestCtx, productID) // Pass context down
	if err != nil {
		// Log within HandleServiceError will have context if called from here
		return errors.HandleServiceError(c, err, fmt.Sprintf("get product by ID %s", productID))
	}

	log.Info("Handler: Responding with product data")
	return c.Status(http.StatusOK).JSON(product)
}

// GetProductStock handles GET /products/:productId/stock
func (h *ProductHandler) GetProductStock(c *fiber.Ctx) error {
	requestCtx := c.UserContext()
	log := logrus.WithContext(requestCtx)

	productID, errResp := h.validatePathParam(c, "productId")
	if errResp != nil {
		return errResp
	}
	log = log.WithField("product_id", productID)
	log.Info("Handler: Received request to get product stock")

	// --- Example: Add Custom Span ---
	// Get the tracer instance (using the handler's package name as instrumentation scope)
	tracer := telemetry.GetTracer("your_module_path/product-service/handler") // <-- ADJUST MODULE PATH

	// Start a new child span for the service call
	var stock int
	var err error
	// The span automatically becomes a child of the span created by the otelgofiber middleware
	err = tracer.WithSpan(requestCtx, "GetStockServiceCall", func(ctx context.Context) error {
		// Add attributes specific to this operation within the span
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.String("product.id", productID))

		// Call the service layer, passing the context containing the new span
		var serviceErr error
		stock, serviceErr = h.service.GetStock(ctx, productID) // Pass context with span
		if serviceErr != nil {
			span.RecordError(serviceErr) // Record error on the span
			span.SetStatus(codes.Error, serviceErr.Error()) // Set span status to Error
		} else {
            span.SetAttributes(attribute.Int("product.stock", stock))
        }
		return serviceErr // Return the error to potentially end the span with an error status
	}) // tracer.WithSpan automatically ends the span

	if err != nil {
		return errors.HandleServiceError(c, err, fmt.Sprintf("get stock for product ID %s", productID))
	}

	response := fiber.Map{
		"productId": productID,
		"stock":     stock,
	}
	log.Infof("Handler: Responding with product stock %d", stock)
	return c.Status(http.StatusOK).JSON(response)
}

// ... (validatePathParam helper) ...



3. Update Service and Repository Layers:
Pass Context: Modify service and repository method signatures to accept context.Context as the first argument. Pass the context down from the handler through the service to the repository.
Update Logging: Replace logger.Info, logger.Error etc. with logrus.WithContext(ctx).Info, logrus.WithContext(ctx).Error.
Add Custom Spans (Optional but Recommended): Wrap potentially slow or important operations (like database calls in the repository) in their own spans using tracer.WithSpan(ctx, "spanName", func(ctx context.Context) error { ... }). Remember to get the tracer using telemetry.GetTracer("your_module_path/...").
Example in service.go:
package main

import (
	"context" // Add context import
	// logger "your_module_path/common/logger" // Remove
	"github.com/sirupsen/logrus" // Use Logrus
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
    "your_module_path/common/telemetry" // <-- ADJUST
    "go.opentelemetry.io/otel/codes"
)

// ProductService defines the interface for product business logic
type ProductService interface {
	GetAll(ctx context.Context) ([]Product, error) // Add context
	GetByID(ctx context.Context, productID string) (Product, error) // Add context
	GetStock(ctx context.Context, productID string) (int, error) // Add context
}

// ... (productService struct) ...

// NewProductService creates a new product service
func NewProductService(repo ProductRepository) ProductService {
	return &productService{
		repo: repo,
	}
}

// GetAll handles fetching all products
func (s *productService) GetAll(ctx context.Context) ([]Product, error) { // Add context
	log := logrus.WithContext(ctx) // Use context for logging
	log.Info("Service: Fetching all products")

    // Example: Add a span for the repository call
    tracer := telemetry.GetTracer("your_module_path/product-service/service") // <-- ADJUST
    var products []Product
    var err error

    err = tracer.WithSpan(ctx, "Repo.FindAll", func(ctx context.Context) error {
        var repoErr error
        products, repoErr = s.repo.FindAll(ctx) // Pass context down
        if repoErr != nil {
            span := trace.SpanFromContext(ctx)
            span.RecordError(repoErr)
            span.SetStatus(codes.Error, repoErr.Error())
        }
        return repoErr
    })

	if err != nil {
		log.WithError(err).Error("Service: Failed to fetch all products from repo")
		return nil, err // Return the original error
	}
	return products, nil
}

// GetByID handles fetching a product by its ID
func (s *productService) GetByID(ctx context.Context, productID string) (Product, error) { // Add context
	log := logrus.WithContext(ctx).WithField("product_id", productID) // Use context
	log.Info("Service: Fetching product by ID")

    tracer := telemetry.GetTracer("your_module_path/product-service/service") // <-- ADJUST
    var product Product
    var err error

    err = tracer.WithSpan(ctx, "Repo.FindByProductID", func(ctx context.Context) error {
        span := trace.SpanFromContext(ctx)
        span.SetAttributes(attribute.String("db.statement", "FindByProductID"), attribute.String("product.id", productID)) // Example DB attributes
        var repoErr error
        product, repoErr = s.repo.FindByProductID(ctx, productID) // Pass context down
        if repoErr != nil {
            span.RecordError(repoErr)
            span.SetStatus(codes.Error, repoErr.Error())
        }
        return repoErr
    })


	if err != nil {
		// Error already recorded on span if it happened in the repo call
		log.WithError(err).Error("Service: Failed to find product by ID in repo")
		return Product{}, err
	}
	return product, nil
}

// GetStock handles fetching stock for a product
func (s *productService) GetStock(ctx context.Context, productID string) (int, error) { // Add context
	log := logrus.WithContext(ctx).WithField("product_id", productID) // Use context
	log.Info("Service: Checking stock for product ID")

    tracer := telemetry.GetTracer("your_module_path/product-service/service") // <-- ADJUST
    var stock int
    var err error

    err = tracer.WithSpan(ctx, "Repo.FindStockByProductID", func(ctx context.Context) error {
        span := trace.SpanFromContext(ctx)
        span.SetAttributes(attribute.String("db.statement", "FindStockByProductID"), attribute.String("product.id", productID))
        var repoErr error
        stock, repoErr = s.repo.FindStockByProductID(ctx, productID) // Pass context down
         if repoErr != nil {
            span.RecordError(repoErr)
            span.SetStatus(codes.Error, repoErr.Error())
        } else {
             span.SetAttributes(attribute.Int("product.stock.result", stock))
        }
        return repoErr
    })

	if err != nil {
		log.WithError(err).Error("Service: Failed to get stock from repo")
		return 0, err
	}
	return stock, nil
}


Repeat similar context passing and logging updates in repository.go.
Step 10: Add Custom Metrics
What: Define and record custom metrics relevant to your application's logic.
Where: Typically in handler.go or service.go, using the Meter obtained via telemetry.GetMeter().
Why: To gain insights into specific business operations beyond standard metrics (e.g., how many times a specific product is looked up, stock check frequency).
1. Define Metric Instruments (e.g., in handler.go or a dedicated metrics.go):
It's good practice to define instruments once, perhaps globally or within the handler struct.
package main

import (
	// ... other imports
	"go.opentelemetry.io/otel/metric"
    "your_module_path/common/telemetry" // <-- ADJUST
)

var (
	meter = telemetry.GetMeter("your_module_path/product-service/handler") // Get meter once

	// Define a counter for product lookups
	productLookupCounter metric.Int64Counter
    // Define a counter for stock checks
    stockCheckCounter metric.Int64Counter
)

func init() {
	var err error
	productLookupCounter, err = meter.Int64Counter(
		"app.product.lookups",
		metric.WithDescription("Counts the number of product lookup requests by ID"),
		metric.WithUnit("{call}"), // Using UCUM units
	)
	if err != nil {
		logrus.Errorf("Failed to create product lookup counter: %v", err)
        // Handle error appropriately - maybe panic or log fatal
	}

    stockCheckCounter, err = meter.Int64Counter(
		"app.product.stock_checks",
		metric.WithDescription("Counts the number of product stock check requests by ID"),
		metric.WithUnit("{call}"),
	)
    if err != nil {
		logrus.Errorf("Failed to create stock check counter: %v", err)
	}
}

// ProductHandler struct definition...
// NewProductHandler function...

// ... handler methods ...


2. Record Metrics in Handler Methods:
Increment the counters within the relevant handler functions. Add attributes to provide dimensions.
Example in GetProductByID:
// GetProductByID handles GET /products/:productId
func (h *ProductHandler) GetProductByID(c *fiber.Ctx) error {
	requestCtx := c.UserContext()
	log := logrus.WithContext(requestCtx) // ... logging setup ...
	productID, errResp := h.validatePathParam(c, "productId")
    if errResp != nil { return errResp }
	log = log.WithField("product_id", productID)
	log.Info("Handler: Received request to get product by ID")

	product, err := h.service.GetByID(requestCtx, productID)

    // --- Record Metric ---
    lookupAttrs := attribute.NewSet(attribute.String("product.id", productID))
    if err != nil {
        // Add an attribute to indicate the lookup failed
        lookupAttrs = attribute.NewSet(
            attribute.String("product.id", productID),
            attribute.Bool("app.lookup.success", false),
            attribute.String("app.error.type", errors.GetType(err)), // Example error type attribute
        )
        productLookupCounter.Add(requestCtx, 1, metric.WithAttributeSet(lookupAttrs))
		return errors.HandleServiceError(c, err, fmt.Sprintf("get product by ID %s", productID))
	} else {
        // Indicate success
         lookupAttrs = attribute.NewSet(
            attribute.String("product.id", productID),
            attribute.Bool("app.lookup.success", true),
        )
        productLookupCounter.Add(requestCtx, 1, metric.WithAttributeSet(lookupAttrs))
    }
    // --- End Metric Recording ---


	log.Info("Handler: Responding with product data")
	return c.Status(http.StatusOK).JSON(product)
}


Example in GetProductStock:
// GetProductStock handles GET /products/:productId/stock
func (h *ProductHandler) GetProductStock(c *fiber.Ctx) error {
    requestCtx := c.UserContext()
	log := logrus.WithContext(requestCtx) // ... logging setup ...
	productID, errResp := h.validatePathParam(c, "productId")
    if errResp != nil { return errResp }
	log = log.WithField("product_id", productID)
	log.Info("Handler: Received request to get product stock")

    // ... (custom span creation as shown before) ...
    var stock int
	var err error
    // Call the service layer within the span...
    err = tracer.WithSpan(requestCtx, "GetStockServiceCall", func(ctx context.Context) error {
        // ... span logic ...
        stock, serviceErr = h.service.GetStock(ctx, productID)
        // ... error handling ...
		return serviceErr
	})

    // --- Record Metric ---
    stockCheckAttrs := attribute.NewSet(attribute.String("product.id", productID))
    if err != nil {
         stockCheckAttrs = attribute.NewSet(
            attribute.String("product.id", productID),
            attribute.Bool("app.lookup.success", false),
             attribute.String("app.error.type", errors.GetType(err)),
        )
        stockCheckCounter.Add(requestCtx, 1, metric.WithAttributeSet(stockCheckAttrs))
        return errors.HandleServiceError(c, err, fmt.Sprintf("get stock for product ID %s", productID))
    } else {
         stockCheckAttrs = attribute.NewSet(
            attribute.String("product.id", productID),
            attribute.Bool("app.lookup.success", true),
        )
        stockCheckCounter.Add(requestCtx, 1, metric.WithAttributeSet(stockCheckAttrs))
    }
     // --- End Metric Recording ---

	response := fiber.Map{ "productId": productID, "stock": stock, }
	log.Infof("Handler: Responding with product stock %d", stock)
	return c.Status(http.StatusOK).JSON(response)
}


Step 11: Running the Service
What: Set environment variables and run the service.
Where: Your terminal or deployment environment.
Why: To provide the necessary configuration for the OTel SDK to connect to the collector.
# Set environment variables (adjust if needed)
export OTEL_SERVICE_NAME="product-service-demo"
export OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317" # Your collector's gRPC endpoint
export OTEL_EXPORTER_INSECURE="true" # Use "false" if your collector uses TLS
export LOG_LEVEL="debug" # Set log level (optional)
export LOG_FORMAT="json" # Use JSON for structured logs (optional)
export PRODUCT_SERVICE_PORT="8082"

# Navigate to the service directory
cd path/to/your/project/product-service/src

# Run the service
go run .


Now, when you run your simulate_product_service.py script, traces, metrics (including host, runtime, and your custom ones), and logs (correlated with traces) should be sent to your OTel collector.
This detailed plan provides a robust foundation for instrumenting your Go service with OpenTelemetry. Remember to adapt module paths and potentially error handling details to your specific project setup.
