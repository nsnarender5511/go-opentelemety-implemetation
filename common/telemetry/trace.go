package telemetry

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	// Use secure credentials in production
	// import "google.golang.org/grpc/credentials" // Example import if needed
	// import "crypto/tls" // Example import if needed
)

// initTracerProvider initializes and registers the OTLP Trace Provider.
func initTracerProvider(ctx context.Context, endpoint string, insecure bool, sampleRatio float64, res *resource.Resource) (shutdownFunc func(context.Context) error, err error) {
	// --- Create OTLP Exporter ---
	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		// Add compression if desired: otlptracegrpc.WithCompressor(grpc.UseCompressor(gzip.Name)),
		// Add headers if needed: otlptracegrpc.WithHeaders(map[string]string{"api-key": "your-key"}),
		otlptracegrpc.WithDialOption(grpc.WithBlock()), // Wait for connection to be established
	}

	if insecure {
		// Use insecure transport (suitable for local development)
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
	} else {
		// Use secure transport (TLS) - Recommended for production
		// Basic TLS configuration (requires collector to have TLS enabled)
		// For advanced TLS (client certs, custom CA), more complex credential setup is needed.
		log.Println("Attempting to use secure OTLP exporter.")
		// Note: This uses system default CAs. For custom CAs or client certs, use:
		// creds := credentials.NewTLS(&tls.Config{...}) // Provide TLS config
		// exporterOpts = append(exporterOpts, otlptracegrpc.WithTLSCredentials(creds))
		// Keep it simple for now, relying on default TLS verification:
		// otlptracegrpc.WithTLSCredentials() with no args implies default TLS config.
		// If no explicit credentials option is provided, gRPC default is secure.
		// So, simply *not* adding WithInsecure() should suffice for basic TLS.
		// exporterOpts = append(exporterOpts, otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))) // Example if cert needed
	}

	traceExporter, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}
	log.Println("OTLP Trace Exporter created.")

	// --- Create Batch Span Processor ---
	// Processes spans in batches before exporting, more efficient than SimpleSpanProcessor.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter) // Adjust options as needed:
	// sdktrace.WithMaxQueueSize(2048),
	// sdktrace.WithMaxExportBatchSize(512),
	// sdktrace.WithExportTimeout(30*time.Second),
	// sdktrace.WithScheduledDelay(5*time.Second),

	log.Println("Batch Span Processor created.")

	// --- Create Tracer Provider ---
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res), // Attach the resource information
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithSampler(getSampler(sampleRatio)), // Use helper func with direct sampleRatio
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
	shutdown := func(shutdownCtx context.Context) error {
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

// Helper function to select sampler based on sampleRatio
func getSampler(sampleRatio float64) sdktrace.Sampler {
	switch {
	case sampleRatio >= 1.0:
		log.Println("Using AlwaysSample sampler.")
		return sdktrace.AlwaysSample()
	case sampleRatio <= 0.0:
		log.Println("Using NeverSample sampler.")
		return sdktrace.NeverSample()
	default:
		log.Printf("Using ParentBased(TraceIDRatioBased(%.2f)) sampler.", sampleRatio)
		// ParentBased respects the sampling decision of the parent span, if any.
		// TraceIDRatioBased samples a fraction of traces based on the trace ID.
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio))
	}
}

// GetTracer returns a named tracer instance.
func GetTracer(instrumentationName string) trace.Tracer {
	// otel.Tracer uses the globally registered TracerProvider.
	return otel.Tracer(instrumentationName)
}
