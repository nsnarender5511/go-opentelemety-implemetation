package telemetry

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/host"    // Host metrics (CPU, memory)
	"go.opentelemetry.io/contrib/instrumentation/runtime" // Go runtime metrics (GC, goroutines)
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric" // Alias for clarity
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
)

// initMeterProvider initializes and registers the OTLP Meter Provider.
func initMeterProvider(ctx context.Context, endpoint string, insecure bool, res *resource.Resource) (shutdownFunc func(context.Context) error, err error) {
	// --- Create OTLP Exporter ---
	exporterOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
		// otlpmetricgrpc.WithCompressor(grpc.UseCompressor(gzip.Name)),
		// otlpmetricgrpc.WithHeaders(map[string]string{"api-key": "your-key"}),
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
		// Use cumulative aggregation temporality by default unless specified otherwise.
		// Delta is often preferred for counters with Prometheus/Grafana, but OTLP typically defaults to cumulative.
		otlpmetricgrpc.WithTemporalitySelector(sdkmetric.DefaultTemporalitySelector),
	}

	if insecure {
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithInsecure())
	} else {
		// Use secure transport (TLS) - Recommended for production
		// Similar logic as trace exporter for TLS setup
		log.Println("Attempting to use secure OTLP metric exporter.")
		// No explicit credentials option means default secure gRPC
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
	shutdown := func(shutdownCtx context.Context) error {
		log.Println("Shutting down Meter Provider...")
		// Use a timeout for the shutdown context.
		ctx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		if err := meterProvider.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down Meter Provider: %v", err)
			return fmt.Errorf("failed to shutdown MeterProvider: %w", err)
		}
		log.Println("Meter Provider shut down successfully.")
		return nil
	}

	return shutdown, nil
}

// GetMeter returns a named meter instance.
func GetMeter(instrumentationName string) metric.Meter {
	// otel.Meter uses the globally registered MeterProvider.
	return otel.Meter(instrumentationName)
}
