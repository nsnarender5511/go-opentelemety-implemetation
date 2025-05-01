package telemetry

import (
	"context"
	"log" // Use standard log for setup errors

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0" // Use the latest stable semantic conventions
)

// newResource creates an OTel Resource describing this service.
func newResource(ctx context.Context, serviceName string) *resource.Resource {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			// --- Essential Attributes ---
			semconv.ServiceName(serviceName), // Use serviceName parameter

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
		res, _ = resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(serviceName)))
		return res
	}

	// Merge with default attributes (like schema URL) for completeness
	mergedRes, err := resource.Merge(resource.Default(), res)
	if err != nil {
		log.Printf("Error merging OTel resources: %v. Using created resource.", err)
		return res // Return the one we created if merge fails
	}

	log.Printf("OTel Resource created with service name: %s", serviceName)
	return mergedRes
}
