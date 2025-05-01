package telemetry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	// Import the main config package using the new module path

	// No need to import sub-packages like trace, metric, log here

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

// globalLogger is the shared logger instance used across the telemetry package
// It is set via SetLogger and used by the wrapper functions to log telemetry events
var globalLogger *logrus.Logger

// SetLogger sets the global logger for the telemetry package
// This should be called before any other telemetry functions
func SetLogger(logger *logrus.Logger) {
	globalLogger = logger
}

// getLogger gets the global logger or creates a default one if not set
// This ensures a logger is always available for internal telemetry operations
func getLogger() *logrus.Logger {
	if globalLogger == nil {
		globalLogger = logrus.New()
		globalLogger.SetLevel(logrus.InfoLevel)
	}
	return globalLogger
}

// shutdownFunc defines the signature for shutdown functions returned by initializers.
type shutdownFunc func(context.Context) error

// InitTelemetry initializes OpenTelemetry Tracing, Metrics, and Logging.
// It returns a shutdown function to cleanly terminate telemetry and
// any error encountered during initialization.
func InitTelemetry(ctx context.Context, config TelemetryConfig) (shutdown func(context.Context) error, err error) {
	logger := config.Logger
	if logger == nil {
		logger = getLogger()
	}

	// Create the shutdown function that will be returned
	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		logger.Info("Shutting down telemetry providers")

		// Shutdown providers in reverse order
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			if shutdownErr := shutdownFuncs[i](ctx); shutdownErr != nil {
				logger.WithError(shutdownErr).Error("Error during telemetry shutdown")
				err = errors.Join(err, shutdownErr)
			}
		}

		shutdownFuncs = nil
		if err != nil {
			logger.WithError(err).Error("Telemetry shutdown completed with errors")
		} else {
			logger.Info("Telemetry shutdown completed successfully")
		}
		return err
	}

	// Handle errors during initialization
	handleInitError := func(initErr error, component string) error {
		logger.WithError(initErr).Errorf("Failed to initialize %s", component)
		// Perform cleanup of already initialized components
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = shutdown(cleanupCtx) // Ignore any shutdown errors during initialization cleanup
		return fmt.Errorf("%s initialization failed: %w", component, initErr)
	}

	logger.WithFields(logrus.Fields{
		"service":  config.ServiceName,
		"endpoint": config.Endpoint,
	}).Info("Initializing telemetry")

	// Create and configure the Resource for all telemetry
	res, err := newResource(ctx, config.ServiceName)
	if err != nil {
		return nil, handleInitError(err, "resource")
	}

	// Set up propagator
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)

	// Initialize tracer provider
	tp, err := newTraceProvider(ctx, config, res)
	if err != nil {
		return nil, handleInitError(err, "trace provider")
	}
	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	otel.SetTracerProvider(tp)

	// Initialize meter provider
	mp, err := newMeterProvider(ctx, config, res)
	if err != nil {
		return nil, handleInitError(err, "meter provider")
	}
	shutdownFuncs = append(shutdownFuncs, mp.Shutdown)
	otel.SetMeterProvider(mp)

	// Initialize logger provider - Replace stub call with actual initialization
	// loggerShutdown, err := configureLoggerProvider(ctx, config, res)
	loggerShutdown, err := initLoggerProvider(ctx, config.Endpoint, config.Insecure, res) // Call the real init function
	if err != nil {
		// Logger is less critical, so we log the error but continue
		logger.WithError(err).Warn("Failed to initialize logger provider, continuing without telemetry logging")
	} else if loggerShutdown != nil {
		shutdownFuncs = append(shutdownFuncs, loggerShutdown)
	}

	logger.Info("Telemetry initialization completed successfully")
	return shutdown, nil
}

// newResource creates a resource with service information
// This resource is used by all telemetry signals (traces, metrics, logs)
func newResource(ctx context.Context, serviceName string) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	return res, nil
}

// createMasterShutdown creates a function that will shut down all provided shutdown functions
// in parallel, with a timeout. This is more efficient than shutting down sequentially.
func createMasterShutdown(logger *logrus.Logger, shutdownFuncs []func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		logger.Info("Starting telemetry shutdown")

		// Use WaitGroup for parallel shutdown
		var wg sync.WaitGroup
		var mu sync.Mutex
		var multiErr error

		for _, fn := range shutdownFuncs {
			if fn == nil {
				continue
			}

			wg.Add(1)
			go func(shutdownFn func(context.Context) error) {
				defer wg.Done()

				if err := shutdownFn(ctx); err != nil {
					mu.Lock()
					multiErr = errors.Join(multiErr, err)
					mu.Unlock()
				}
			}(fn)
		}

		wg.Wait()
		return multiErr
	}
}
