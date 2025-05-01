package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	// Use alias to avoid collision
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TelemetryConfig struct definition removed - assumed to exist elsewhere or passed correctly.

// InitTelemetry initializes OpenTelemetry Tracing, Metrics, and Logging.
// It configures global providers and returns a master shutdown function.
// Application logging should use the configured global Logrus instance.
func InitTelemetry(ctx context.Context, config TelemetryConfig) (otelShutdownFunc, error) {
	// --- Initialize Temporary Logrus Logger for Setup ---
	setupLogger := logrus.New()
	setupLogger.SetOutput(os.Stdout)
	setupLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	// Set level based on config for setup visibility
	// --- Refactored Log Level Parsing (Annoying Issue 1) ---
	logLevelSetting := config.LogLevel
	parsedLevel, parseErr := logrus.ParseLevel(logLevelSetting)
	if parseErr != nil {
		parsedLevel = logrus.InfoLevel
		// Adjust warning message to only refer to the setup logger
		setupLogger.Warnf("Invalid log level '%s' in config, defaulting setup logger to 'info'. Standard logger level will be set during configuration.", logLevelSetting)
	} else {
		setupLogger.Infof("Using log level '%s' for setup logger", parsedLevel.String())
	}
	setupLogger.SetLevel(parsedLevel) // Use parsed level here

	// Log initial config details using the setup logger
	setupLogger.WithFields(logrus.Fields{
		"service":  config.ServiceName,
		"endpoint": config.Endpoint,
		"logLevel": parsedLevel.String(), // Log the *actual* parsed level being used
	}).Info("Initializing telemetry")

	// --- Central Shutdown Logic ---
	var shutdownFuncs []otelShutdownFunc
	var mu sync.Mutex
	addShutdownFunc := func(f otelShutdownFunc) {
		if f != nil {
			mu.Lock()
			shutdownFuncs = append(shutdownFuncs, f)
			mu.Unlock()
		}
	}

	masterShutdown := func(shutdownCtx context.Context) error {
		setupLogger.Info("Starting telemetry shutdown")
		mu.Lock()
		funcs := make([]otelShutdownFunc, len(shutdownFuncs))
		copy(funcs, shutdownFuncs)
		mu.Unlock()

		var combinedErr error
		// Shutdown in reverse order of initialization (LIFO)
		for i := len(funcs) - 1; i >= 0; i-- {
			if funcs[i] == nil {
				continue
			}
			// The context passed into masterShutdown (shutdownCtx) should already have the correct timeout
			// managed by the caller (e.g., WaitForGracefulShutdown). We execute the specific provider
			// shutdown function using this received context directly.
			if shutdownErr := funcs[i](shutdownCtx); shutdownErr != nil {
				// Use the setup logger for consistency during shutdown logging
				setupLogger.WithError(shutdownErr).Errorf("Error during telemetry provider shutdown step %d", i)
				combinedErr = errors.Join(combinedErr, shutdownErr)
			}
			// No need to cancel a sub-context here as we used shutdownCtx directly.
		}

		if combinedErr != nil {
			setupLogger.WithError(combinedErr).Error("Telemetry shutdown completed with errors")
		} else {
			setupLogger.Info("Telemetry shutdown completed successfully")
		}
		return combinedErr
	}

	// --- otelShutdownFunc type definition is now in log.go ---

	handleInitError := func(initErr error, component string) error {
		setupLogger.WithError(initErr).Errorf("Failed to initialize %s", component)
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownErr := masterShutdown(cleanupCtx)
		if shutdownErr != nil {
			setupLogger.WithError(shutdownErr).Error("Error during cleanup shutdown after initialization failure")
		}
		return fmt.Errorf("%s initialization failed: %w", component, initErr)
	}

	// --- Create Resource ---
	res, err := newResource(ctx, config.ServiceName)
	if err != nil {
		return nil, handleInitError(err, "resource")
	}
	setupLogger.Debug("Telemetry resource created")

	// --- Set up Propagator ---
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)
	setupLogger.Debug("Set global text map propagator (TraceContext, Baggage)")

	// --- Initialize Providers (Trace, Meter, Log) ---
	var tp *sdktrace.TracerProvider
	var traceShutdown otelShutdownFunc // Declare shutdown func var
	tp, traceShutdown, err = newTraceProvider(ctx, config, res, setupLogger)
	if err != nil {
		return nil, handleInitError(err, "trace provider")
	}
	addShutdownFunc(traceShutdown) // Register the shutdown func
	otel.SetTracerProvider(tp)
	setupLogger.Info("Trace provider registered globally")

	var mp *sdkmetric.MeterProvider
	var meterShutdown otelShutdownFunc // Declare shutdown func var
	mp, meterShutdown, err = newMeterProvider(ctx, config, res, setupLogger)
	if err != nil {
		return nil, handleInitError(err, "meter provider")
	}
	addShutdownFunc(meterShutdown) // Register the shutdown func
	otel.SetMeterProvider(mp)
	setupLogger.Info("Meter provider registered globally")

	var otelLogProvider *sdklog.LoggerProvider
	var logShutdown otelShutdownFunc
	// Pass endpoint for potential logging inside createOtlpLogProvider if needed
	otelLogProvider, logShutdown, err = createOtlpLogProvider(ctx, config.Endpoint, config.Insecure, res, setupLogger)
	if err != nil {
		// Log the error but don't make it fatal for InitTelemetry itself unless absolutely required.
		// The error from createOtlpLogProvider already logs details.
		// Allow init to continue so other telemetry might work, but OTel logging won't.
		setupLogger.WithError(err).Error("Failed to initialize OTLP logger provider. OTel log export will not function.")
		// Reset provider and shutdown func to nil as they are invalid
		otelLogProvider = nil
		logShutdown = nil
		// Do NOT return/handleInitError here - proceed to configure Logrus without the OTel hook.
	} else {
		// Only set global provider and add shutdown func if creation succeeded.
		setupLogger.Info("OTLP Logger Provider initialized successfully")
		global.SetLoggerProvider(otelLogProvider)
		addShutdownFunc(logShutdown) // Add shutdown func ONLY if successful
		setupLogger.Debug("Global OTel Logger Provider set")
	}

	// --- Configure Global Logrus Instance ---
	configureLogrus(parsedLevel, otelLogProvider, setupLogger) // Pass potentially nil provider

	setupLogger.Info("Telemetry initialization completed successfully")
	return masterShutdown, nil
}
