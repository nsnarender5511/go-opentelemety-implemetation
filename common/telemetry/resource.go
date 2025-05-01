package telemetry

import (
	"context"
	"log" // Use standard log for setup errors

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0" // Use the latest stable semantic conventions

	"github.com/narender/common-module/config" // Import config package
)

// newResource creates an OTel Resource describing this service.
// It now reads service name and version directly from the config package.
func newResource(ctx context.Context) *resource.Resource { // Remove serviceName parameter
	// Get service identity from config
	serviceName := config.OTEL_SERVICE_NAME // Use the specific OTel variable from config
	serviceVersion := config.SERVICE_VERSION

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// --- Essential Attributes ---
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion), // Add service version attribute

			// --- Optional Attributes (Add more as needed) ---
			// semconv.DeploymentEnvironment(config.ENVIRONMENT), // Example: Use config for environment
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

	log.Printf("OTel Resource created with service name: %s, version: %s", serviceName, serviceVersion)
	return mergedRes
}
