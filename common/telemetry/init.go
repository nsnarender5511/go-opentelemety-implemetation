package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	// Import the main config package
	"example.com/product-service/common/config"
	// No need to import sub-packages like trace, metric, log here
)

// shutdownFunc defines the signature for shutdown functions returned by initializers.
type shutdownFunc func(context.Context) error

// InitTelemetry initializes OpenTelemetry Tracing, Metrics, and Logging.
// It loads configuration, creates resources, sets up providers/exporters,
// configures the Logrus hook, and returns a master shutdown function.
func InitTelemetry() (func(context.Context) error, error) {
	// Directly use variables from the config package
	log.Printf("Initializing Telemetry for service: %s, endpoint: %s, insecure: %t",
		config.OTEL_SERVICE_NAME, config.OTEL_EXPORTER_OTLP_ENDPOINT, config.OTEL_EXPORTER_INSECURE)

	// Use a timeout for the initial setup context.
	initCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second) // Increased timeout
	defer cancel()

	// --- 1. Create Resource ---
	// Pass service name directly. Assumes newResource signature is updated.
	res := newResource(initCtx, config.OTEL_SERVICE_NAME)

	shutdownFuncs := make([]shutdownFunc, 0, 3) // Store shutdown functions
	var initErr error                           // To capture the first error during init

	// --- 2. Initialize Trace Provider ---
	// Pass config variables directly. Assumes initTracerProvider signature is updated.
	tracerShutdown, err := initTracerProvider(initCtx, config.OTEL_EXPORTER_OTLP_ENDPOINT, config.OTEL_EXPORTER_INSECURE, config.OTEL_SAMPLE_RATIO, res)
	if err != nil {
		log.Printf("Error initializing TracerProvider: %v", err)
		initErr = errors.Join(initErr, fmt.Errorf("tracer init failed: %w", err))
	} else if tracerShutdown != nil {
		shutdownFuncs = append(shutdownFuncs, tracerShutdown)
		log.Println("TracerProvider initialization successful.")
	}

	// --- 3. Initialize Meter Provider ---
	// Pass config variables directly. Assumes initMeterProvider signature is updated.
	meterShutdown, err := initMeterProvider(initCtx, config.OTEL_EXPORTER_OTLP_ENDPOINT, config.OTEL_EXPORTER_INSECURE, res)
	if err != nil {
		log.Printf("Error initializing MeterProvider: %v", err)
		initErr = errors.Join(initErr, fmt.Errorf("meter init failed: %w", err))
	} else if meterShutdown != nil {
		shutdownFuncs = append(shutdownFuncs, meterShutdown)
		log.Println("MeterProvider initialization successful.")
	}

	// --- 4. Initialize Logger Provider ---
	// This MUST happen before ConfigureLogrus.
	// Pass config variables directly. Assumes initLoggerProvider signature is updated.
	loggerShutdown, err := initLoggerProvider(initCtx, config.OTEL_EXPORTER_OTLP_ENDPOINT, config.OTEL_EXPORTER_INSECURE, res)
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
