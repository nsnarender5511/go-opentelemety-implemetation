Go OpenTelemetry Code Refactoring Plan
1. Final Goal
The primary goal of this refactoring is to reorganize the existing otel/ package (containing 12 Go files) into a new telemetry/ directory structure. This new structure will improve:
Readability: Code related to specific concerns (tracing, metrics, exporting) will be grouped together.
Maintainability: Changes to one aspect of telemetry (e.g., trace sampling) will be isolated to its specific package, reducing the risk of unintended side effects.
Scalability: Easier to add new telemetry features or support different backends in the future.
Testability: Smaller, focused packages are generally easier to unit test.
Adherence to Single Responsibility Principle (SRP): Each package will have a more clearly defined responsibility.
The final structure will look similar to this:
telemetry/
├── attributes/
│   └── attributes.go
├── exporter/
│   ├── otlp_grpc.go
│   ├── trace_exporter.go
│   ├── metric_exporter.go
│   └── log_exporter.go
├── instrumentation/
│   └── http.go
├── log/
│   ├── setup.go
│   └── processor.go
├── manager/
│   └── manager.go
├── metric/
│   ├── setup.go
│   ├── http_metrics.go
│   └── product_metrics.go
├── propagator/
│   └── propagator.go
├── resource/
│   └── resource.go
├── trace/
│   ├── setup.go
│   ├── processor.go
│   └── utils.go
└── setup.go

# Potentially outside telemetry/
logging/
└── setup.go


(Note: Some files like processor.go might be initially part of setup.go within their respective directories and extracted later if complexity warrants it.)
2. Prerequisites
Version Control: Ensure your project is under version control (e.g., Git). Create a new branch for this refactoring work.
git checkout -b feat/otel-refactor


Go Environment: A working Go development environment (Go 1.18+ recommended for generics, though the current code seems compatible with older versions).
Build System: Ensure your project builds and compiles correctly before starting.
Understanding of Go Packages: Familiarity with Go's package management and import paths.
3. Refactoring Phases and Steps
Phase 1: Preparation and Foundation
Goal: Create the new directory structure and move existing files as a starting point. Establish basic compilation.
Step 1.1: Create New Directory Structure
What: Create the target directories under a new top-level telemetry directory. Also create the separate logging directory if desired.
How: Use mkdir commands.
mkdir -p telemetry/{attributes,exporter,instrumentation,log,manager,metric,propagator,resource,trace}
mkdir logging # If moving Logrus setup out


Why: Establishes the skeleton for the new organization.
When: First step of the refactoring process.
Step 1.2: Move Existing Files (Initial Placement)
What: Move the existing 12 .go files from the otel/ directory into their most likely target directories within telemetry/ (and logging/). This is an initial placement; files will be broken down further later.
How: Use mv or your file explorer.
# Example movements (adjust based on final decision for logging.go)
mv otel/attributes.go telemetry/attributes/
mv otel/exporters.go telemetry/exporter/ # Will be split later
mv otel/global_manager.go telemetry/manager/
mv otel/http_metrics.go telemetry/metric/
mv otel/log_setup.go telemetry/log/
mv otel/logging.go logging/ # Or keep in telemetry temporarily
mv otel/metric_setup.go telemetry/metric/
mv otel/product_metrics.go telemetry/metric/
mv otel/resource.go telemetry/resource/
mv otel/setup.go telemetry/ # Top-level orchestrator
mv otel/trace_setup.go telemetry/trace/ # Will be split later
mv otel/trace_utils.go telemetry/trace/
# Delete the now empty otel/ directory
rmdir otel


Why: Gets the files into the new structure to begin organizing and updating imports.
When: After creating the directories.
Step 1.3: Update Go Package Declarations
What: Change the package otel declaration at the top of each moved file to reflect its new package name (directory name).
How: Edit each .go file.
Before (telemetry/resource/resource.go):
package otel // Old package name

import (
    // ...
)
// ...


After (telemetry/resource/resource.go):
package resource // New package name matches directory

import (
    // ...
)
// ...


Repeat for all moved files (attributes, exporter, log, manager, metric, resource, trace, setup, logging). The top-level telemetry/setup.go should likely be package telemetry.
Why: Aligns the package declaration with Go conventions and the new directory structure.
When: After moving the files.
Step 1.4: Initial Import Path Updates & Compile Check
What: Update internal import paths within the moved files to reference the new locations. Run go build ./... or go test ./... to identify initial compilation errors (mostly related to imports and visibility).
How: Search and replace import paths. Fix visibility issues (e.g., unexported functions/types needed across packages).
Before (e.g., in telemetry/setup.go):
// Assuming GetLogger was in the old 'otel' package
logger := GetLogger()
res, err := newResource(ctx, cfg) // Function in the same old 'otel' package


After (e.g., in telemetry/setup.go):
import (
    "your_module/logging" // Assuming Logrus setup moved here
    "your_module/telemetry/resource"
    // ... other new imports
)

// ...
logger := logging.GetLogger() // Assuming GetLogger is now exported from logging
res, err := resource.NewResource(ctx, cfg) // Assuming newResource is now exported as NewResource


Run go mod tidy to clean up dependencies.
Why: To make the code aware of the new structure and achieve a basic compilable state before splitting files further. Fixing visibility early is crucial.
When: After updating package declarations. Do not proceed until the project compiles, even if functionality is broken.
Phase 2: Core Component Isolation
Goal: Break down larger files and isolate core, reusable components into their dedicated packages.
Step 2.1: Isolate Resource Definition (telemetry/resource/)
What: Ensure telemetry/resource/resource.go only contains the logic for creating the OTel resource. Export the main function (newResource becomes NewResource).
How: Verify the file content. Update telemetry/setup.go and any other consumers to import "your_module/telemetry/resource" and call resource.NewResource.
Why: The resource definition is fundamental and used by multiple providers.
When: Beginning of Phase 2.
Step 2.2: Isolate Common Attributes (telemetry/attributes/)
What: Verify telemetry/attributes/attributes.go contains only attribute key definitions (both semantic and custom). Ensure keys are exported.
How: Review the file. Update consumers (like trace/utils.go, metric/http_metrics.go) to import "your_module/telemetry/attributes" and use the exported keys (e.g., attributes.HTTPMethodKey).
Why: Centralizes attribute definitions for consistency across traces, metrics, and logs.
When: After isolating the resource.
Step 2.3: Isolate Exporter Logic (telemetry/exporter/)
What: Split the monolithic telemetry/exporter/exporters.go into separate files for each signal type and the common gRPC connection logic.
How:
Create telemetry/exporter/otlp_grpc.go: Move newOTLPGrpcConnection here (make it unexported if only used within the exporter package, or exported if needed elsewhere).
Create telemetry/exporter/trace_exporter.go: Move newTraceExporter here (export as NewTraceExporter). Update it to call newOTLPGrpcConnection.
Create telemetry/exporter/metric_exporter.go: Move newMetricExporter here (export as NewMetricExporter). Update it to call newOTLPGrpcConnection.
Create telemetry/exporter/log_exporter.go: Move newLogExporter here (export as NewLogExporter). Update it to call newOTLPGrpcConnection.
Delete the original telemetry/exporter/exporters.go.
Update telemetry/setup.go to import "your_module/telemetry/exporter" and call the new exported functions (exporter.NewTraceExporter, etc.).
Why: Separates the configuration logic for each signal's exporter, adhering to SRP.
When: After isolating attributes.
Step 2.4: Isolate Propagator Setup (telemetry/propagator/)
What: Move the propagator setup logic from telemetry/setup.go to a new file telemetry/propagator/propagator.go.
How:
Create telemetry/propagator/propagator.go.
Define an exported function, e.g., SetupPropagators().
Move the propagation.NewCompositeTextMapPropagator(...) and otel.SetTextMapPropagator(prop) calls into this function.
Call propagator.SetupPropagators() from telemetry/setup.go.
Before (telemetry/setup.go):
// ...
prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
otel.SetTextMapPropagator(prop)
// ...


After (telemetry/propagator/propagator.go):
package propagator

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
    // Potentially import logging if needed
)

// SetupPropagators configures the global OTel propagators.
func SetupPropagators() {
    prop := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
    otel.SetTextMapPropagator(prop)
    // log.Debug("Global TextMapPropagator configured.") // Consider logging here or returning status
}


After (telemetry/setup.go):
import "your_module/telemetry/propagator"
// ...
propagator.SetupPropagators()
// ...


Why: Isolates propagator configuration, simplifying the main setup function.
When: After isolating exporters.
Phase 3: Signal-Specific Refactoring
Goal: Organize the setup and specific components for traces, metrics, and logs into their respective packages.
Step 3.1: Refactor Trace Setup (telemetry/trace/)
What: Organize trace-specific logic: provider setup, sampler setup, and span processor setup.
How:
Rename telemetry/trace/trace_setup.go to telemetry/trace/setup.go.
Ensure newSampler (export as NewSampler) and newTraceProvider (export as NewTraceProvider) are in telemetry/trace/setup.go.
Potentially create telemetry/trace/processor.go and move the sdktrace.NewBatchSpanProcessor logic into an exported function there (e.g., NewBatchSpanProcessor), which NewTraceProvider would call. This improves separation if processor options become complex.
Update telemetry/setup.go to call trace.NewSampler and trace.NewTraceProvider.
Why: Groups all core trace pipeline configuration (sampling, processing, provider) together.
When: Beginning of Phase 3.
Step 3.2: Refactor Metric Setup (telemetry/metric/)
What: Consolidate metric provider setup and ensure metric definitions (http_metrics.go, product_metrics.go) reside here.
How:
Ensure newMeterProvider (export as NewMeterProvider) is in telemetry/metric/setup.go.
Verify http_metrics.go and product_metrics.go are in telemetry/metric/ and use the correct package metric declaration. Ensure they use the global manager (manager.GetMeter) or accept a metric.MeterProvider / metric.Meter via parameters for better testability.
Update telemetry/setup.go to call metric.NewMeterProvider.
Why: Groups metric pipeline configuration and specific metric definitions.
When: After refactoring trace setup.
Step 3.3: Refactor Log Setup (telemetry/log/)
What: Organize OTel log provider and processor setup.
How:
Ensure newLoggerProvider (export as NewLoggerProvider) is in telemetry/log/setup.go.
Potentially create telemetry/log/processor.go and move the sdklog.NewBatchProcessor logic into an exported function there (e.g., NewBatchProcessor), which NewLoggerProvider would call.
Update telemetry/setup.go to call log.NewLoggerProvider.
Why: Groups OTel logging pipeline configuration.
When: After refactoring metric setup.
Phase 4: Utilities and Instrumentation
Goal: Place utility functions and instrumentation wrappers into appropriate packages.
Step 4.1: Relocate Trace Utilities (telemetry/trace/utils.go)
What: Ensure the RecordSpanError utility function resides in telemetry/trace/utils.go.
How: Verify the file location and package trace declaration. Ensure the function is exported. Update any callers if necessary.
Why: Keeps trace-specific helper functions within the trace package.
When: Beginning of Phase 4.
Step 4.2: Relocate HTTP Instrumentation (telemetry/instrumentation/http.go)
What: Move the NewHTTPHandler wrapper function from telemetry/trace/setup.go (its old location after the move) to telemetry/instrumentation/http.go.
How:
Create telemetry/instrumentation/http.go with package instrumentation.
Move the NewHTTPHandler function definition into this new file. Ensure it's exported.
Update any callers to import "your_module/telemetry/instrumentation" and use instrumentation.NewHTTPHandler.
Before (telemetry/trace/setup.go):
package trace
// ...
// NewHTTPHandler wraps an http.Handler with OpenTelemetry instrumentation.
func NewHTTPHandler(handler http.Handler, operationName string) http.Handler {
    return otelhttp.NewHandler(handler, operationName)
}


After (telemetry/instrumentation/http.go):
package instrumentation

import (
    "net/http"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewHTTPHandler wraps an http.Handler with OpenTelemetry instrumentation.
func NewHTTPHandler(handler http.Handler, operationName string) http.Handler {
    // Consider adding specific otelhttp options if needed later
    return otelhttp.NewHandler(handler, operationName)
}


Why: Groups instrumentation wrappers separately from core OTel setup logic. Makes it easier to add other instrumentation (e.g., gRPC, database) later.
When: After relocating trace utilities.
Step 4.3: Relocate Application Logging (logging/setup.go)
What: Move the Logrus setup code (SetupLogrus) from its temporary location (likely telemetry/ or logging/) definitively into logging/setup.go.
How:
Ensure logging/setup.go exists with package logging.
Move the SetupLogrus function here and ensure it's exported.
Update telemetry/setup.go (or wherever it's called initially) to import "your_module/logging" and call logging.SetupLogrus.
Update the global_manager.go and any other code using the logger to get it via the manager or potentially accept it via dependency injection in the future. For now, ensure the manager gets the logger instance from logging.SetupLogrus.
Why: Separates application logging configuration (Logrus) from OpenTelemetry SDK configuration.
When: After relocating HTTP instrumentation.
Phase 5: Global Manager and Orchestration
Goal: Refine the global access pattern and the main setup orchestration function.
Step 5.1: Refactor Global Manager (telemetry/manager/)
What: Update the TelemetryManager and its initialization/accessors to work with the new package structure.
How:
Ensure telemetry/manager/manager.go has package manager.
Update the initializeGlobalManager function (make it unexported, e.g., initManager) to accept providers from the new packages.
Ensure GetTracer, GetMeter, GetLoggerProvider, etc., are exported and correctly return the managed instances or appropriate NoOp versions if uninitialized.
Verify that GetLogger returns the Logrus instance configured in logging/setup.go.
Consider if GetTracer and GetMeter should use the service name/version stored in the manager by default.
Why: Adapts the global access point to the refactored structure.
When: Beginning of Phase 5.
Step 5.2: Refactor Main Setup Orchestration (telemetry/setup.go)
What: Simplify the main InitTelemetry function by having it call the setup functions from the specialized packages.
How:
Ensure telemetry/setup.go has package telemetry.
Modify InitTelemetry to:
Call logging.SetupLogrus (if applicable here).
Call resource.NewResource.
Call propagator.SetupPropagators.
Call exporter.NewTraceExporter, exporter.NewMetricExporter, exporter.NewLogExporter.
Call trace.NewSampler.
Call trace.NewTraceProvider (passing resource, exporter, sampler).
Call metric.NewMeterProvider (passing resource, exporter).
Call log.NewLoggerProvider (passing resource, exporter).
Call the manager's initialization function (e.g., manager.initManager) passing the created providers and logger.
Handle shutdown logic correctly, potentially getting shutdown functions from provider setup calls.
Conceptual Before (otel/setup.go):
func InitTelemetry(...) (shutdown func(...) error, err error) {
    // ... lots of inline setup for resource, exporters, providers, etc. ...
    res, err := newResource(...)
    traceExporter, err := newTraceExporter(...)
    sampler := newSampler(...)
    tp, bspShutdown := newTraceProvider(res, traceExporter, sampler)
    // ... etc ...
    initializeGlobalManager(...)
    return shutdown, nil
}


Conceptual After (telemetry/setup.go):
package telemetry

import (
    "your_module/logging"
    "your_module/telemetry/resource"
    "your_module/telemetry/propagator"
    "your_module/telemetry/exporter"
    "your_module/telemetry/trace"
    "your_module/telemetry/metric"
    logotel "your_module/telemetry/log" // Alias to avoid conflict
    "your_module/telemetry/manager"
    // ... other imports
)

func InitTelemetry(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
    logger := logging.SetupLogrus(cfg) // Or called earlier by application main

    var shutdownFuncs []func(context.Context) error
    // Simplified shutdown logic placeholder
    shutdown = func(ctx context.Context) error {
        // ... aggregate shutdown errors from shutdownFuncs ...
        manager.Shutdown(ctx) // Assuming manager handles provider shutdowns
        return nil
    }

    defer func() {
        if err != nil {
            // ... error handling ...
            manager.Initialize(nil, nil, nil, logger, cfg) // Init with NoOps on error
        }
    }()

    res, err := resource.NewResource(ctx, cfg)
    if err != nil { /* handle error */ }

    propagator.SetupPropagators()

    traceExporter, err := exporter.NewTraceExporter(ctx, cfg)
    if err != nil { /* handle error */ }
    // ... create metricExporter, logExporter ...

    sampler := trace.NewSampler(cfg)
    tp, tpShutdown := trace.NewTraceProvider(res, traceExporter, sampler)
    shutdownFuncs = append(shutdownFuncs, tpShutdown) // Collect shutdown funcs

    mp := metric.NewMeterProvider(cfg, res, metricExporter)
    lp, err := logotel.NewLoggerProvider(cfg, res, logExporter)
    if err != nil { /* handle error */ }

    // Initialize the global manager with configured providers
    manager.Initialize(tp, mp, lp, logger, cfg)

    logger.Info("OpenTelemetry SDK initialization completed successfully.")
    return shutdown, nil
}


Why: Makes the main initialization function a high-level orchestrator, improving readability and delegating details to specialized packages.
When: After refactoring the global manager.
Phase 6: Testing and Validation
Goal: Ensure the refactored code compiles, passes tests, and correctly emits telemetry data.
Step 6.1: Compile and Run Static Analysis
What: Perform a full build and run static analysis tools.
How: Run go build ./..., go vet ./..., staticcheck ./... (if used). Fix any reported errors or warnings. Run go mod tidy.
Why: Catch compilation errors, unused variables/imports, and potential bugs identified by static analysis.
When: First step of Phase 6.
Step 6.2: Update/Write Unit Tests
What: Update existing unit tests to reflect the new package structure and function signatures. Write new unit tests for newly created or significantly modified functions/packages.
How: Modify _test.go files. Use mocks/stubs for dependencies (like exporters) where appropriate. Test edge cases and error handling. Run go test ./....
Why: Verify the logic within individual packages works as expected.
When: After achieving successful compilation.
Step 6.3: Integration Testing (Verify Telemetry Data)
What: Run the application with the refactored telemetry code and verify that traces, metrics, and logs are being correctly generated and exported to your backend (or console exporter for testing).
How: Configure the OTLP exporter endpoint to point to a test collector/backend. Perform actions in your application that should generate telemetry. Check the backend to ensure:
Traces appear with correct spans, parent-child relationships, attributes, and error statuses.
Metrics (HTTP, product stock) are present with correct names, values, units, and attributes.
Logs (if using OTel logging exporter) appear with correct content and attributes.
Trace IDs are potentially correlated with logs (if implemented).
Why: Confirms that the end-to-end telemetry pipeline works correctly after the refactoring.
When: After unit tests pass. This is the final validation step.
4. Final Goal Recap
Upon successful completion of these phases, the Go OpenTelemetry instrumentation code will be significantly more organized, maintainable, and aligned with best practices for code structure. This provides a solid foundation for future enhancements and easier onboarding for new developers. Remember to merge the feature branch (feat/otel-refactor) back into your main development branch after thorough testing and review.
