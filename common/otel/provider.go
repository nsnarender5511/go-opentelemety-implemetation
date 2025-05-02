package otel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/narender/common/config"
	logrustr "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// SetupLogrus configures the global Logrus logger based on the provided config.
func SetupLogrus(cfg *config.Config) *logrustr.Logger {
	logger := logrustr.New()

	// Set Log Level
	level, err := logrustr.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.Warnf("Invalid log level '%s', defaulting to 'info': %v", cfg.LogLevel, err)
		level = logrustr.InfoLevel
	}
	logger.SetLevel(level)

	// Set Log Format
	switch strings.ToLower(cfg.LogFormat) {
	case "json":
		logger.SetFormatter(&logrustr.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	case "text":
		logger.SetFormatter(&logrustr.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	default:
		logger.Warnf("Invalid log format '%s', defaulting to 'text'", cfg.LogFormat)
		logger.SetFormatter(&logrustr.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	}

	logger.SetOutput(os.Stderr) // Default output

	logrustr.SetLevel(logger.GetLevel()) // Set global level too, mainly for library logs
	logrustr.SetFormatter(logger.Formatter)
	logrustr.SetOutput(logger.Out)

	logger.Infof("Logrus initialized with level '%s' and format '%s'", logger.GetLevel(), cfg.LogFormat)
	return logger
}

// SetupOTelSDK initializes the OpenTelemetry SDK for tracing, metrics, and logging.
// It configures exporters based on the provided Config and registers global providers.
// Returns a configured Logrus logger, a shutdown function, and any error encountered.
func SetupOTelSDK(ctx context.Context, cfg *config.Config) (sdkLogger *logrustr.Logger, shutdown func(context.Context) error, err error) {
	// Setup Logrus first, as OTel setup logs things.
	logger := SetupLogrus(cfg)

	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var shutdownErr error
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			shutdownErr = errors.Join(shutdownErr, shutdownFuncs[i](ctx))
		}
		shutdownFuncs = nil // Clear funcs after execution
		// Join with any setup error that might have been assigned to the named return var `err`
		err = errors.Join(err, shutdownErr)
		return shutdownErr // Return the combined shutdown error
	}

	// Automatically call shutdown at the end of SetupOTelSDK if an error occurs during setup.
	// Note: The caller of SetupOTelSDK is still responsible for calling the returned shutdown function on graceful application exit.
	defer func() {
		if err != nil {
			// If setup failed, attempt to clean up any partial setup
			logger.Debug("Attempting OTel cleanup after setup error...")
			shutdownErr := shutdown(context.Background()) // Use background context for cleanup
			if shutdownErr != nil {
				logger.WithError(shutdownErr).Error("Error during OTel cleanup after setup failure")
			}
		} else {
			logger.Info("OTel SDK setup completed successfully.")
		}
	}()

	// --- Resource ---
	res, rErr := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithHost(),
		resource.WithOS(),
	)
	if rErr != nil {
		err = fmt.Errorf("failed to create OTel resource: %w", rErr)
		logger.WithError(err).Error("OTel setup failed")
		return logger, shutdown, err // Return immediately with logger and shutdown func
	}
	logger.Debug("OpenTelemetry resource created")

	// --- Propagator ---
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)
	logger.Debug("OpenTelemetry propagation configured")

	// --- Shared Exporter Options ---
	exporterOpts := []grpc.DialOption{
		// Set keepalive options to proactively check connection health
		// grpc.WithKeepaliveParams(keepalive.ClientParameters{...}),
	}
	var transportCreds credentials.TransportCredentials
	if cfg.OtelExporterInsecure {
		transportCreds = insecure.NewCredentials()
		logger.Warn("Using insecure connection for OTLP exporter")
	} else {
		// transportCreds = credentials.NewClientTLSFromCert(nil, "") // Use system cert pool
		// TODO: Add proper TLS configuration (loading certs, etc.)
		logger.Warn("TLS configuration for OTLP exporter not implemented, using insecure connection as fallback.")
		transportCreds = insecure.NewCredentials() // Fallback to insecure for now
	}
	exporterOpts = append(exporterOpts, grpc.WithTransportCredentials(transportCreds))

	// --- Trace Provider ---
	traceClientOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlptracegrpc.WithDialOption(exporterOpts...),
		otlptracegrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}
	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		traceClientOpts = append(traceClientOpts, otlptracegrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	traceExporter, texpErr := otlptracegrpc.New(ctx, traceClientOpts...)
	if texpErr != nil {
		err = fmt.Errorf("failed to create OTLP trace exporter: %w", texpErr)
		logger.WithError(err).Error("OTel setup failed")
		return logger, shutdown, err
	}
	logger.Info("OTLP trace exporter created")

	bspOpts := []trace.BatchSpanProcessorOption{
		trace.WithBatchTimeout(cfg.OtelBatchTimeout),
		// Consider adding MaxQueueSize, MaxExportBatchSize based on load
	}
	bsp := trace.NewBatchSpanProcessor(traceExporter, bspOpts...)

	sampler := trace.ParentBased(trace.TraceIDRatioBased(cfg.OtelSampleRatio))

	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(sampler),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)
	logger.Info("Trace provider registered globally")

	// --- Meter Provider ---
	metricClientOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
		otlpmetricgrpc.WithDialOption(exporterOpts...),
		otlpmetricgrpc.WithTimeout(cfg.OtelExporterOtlpTimeout),
	}
	if len(cfg.OtelExporterOtlpHeaders) > 0 {
		metricClientOpts = append(metricClientOpts, otlpmetricgrpc.WithHeaders(cfg.OtelExporterOtlpHeaders))
	}

	metricExporter, mexpErr := otlpmetricgrpc.New(ctx, metricClientOpts...)
	if mexpErr != nil {
		err = fmt.Errorf("failed to create OTLP metric exporter: %w", mexpErr)
		logger.WithError(err).Error("OTel setup failed")
		return logger, shutdown, err
	}
	logger.Info("OTLP metric exporter created")

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithTimeout(cfg.OtelExporterOtlpTimeout))), // Use exporter timeout for reader too
	)
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)
	logger.Info("Meter provider registered globally")

	// Return the configured logger, the shutdown function, and nil error
	return logger, shutdown, nil
}

// GetTracerProvider returns the global OpenTelemetry TracerProvider.
func GetTracerProvider() oteltrace.TracerProvider {
	return otel.GetTracerProvider()
}

// GetMeterProvider returns the global OpenTelemetry MeterProvider.
func GetMeterProvider() otelmetric.MeterProvider {
	return otel.GetMeterProvider()
}

// GetTracer returns a named Tracer instance from the global TracerProvider.
func GetTracer(instrumentationName string) oteltrace.Tracer {
	return otel.Tracer(instrumentationName)
}

// GetMeter returns a named Meter instance from the global MeterProvider.
func GetMeter(instrumentationName string) otelmetric.Meter {
	return otel.Meter(instrumentationName)
}

// Helper to parse OTEL_EXPORTER_INSECURE string to bool
// Deprecated: Config parsing now handles this directly.
func parseInsecure(insecureStr string) bool {
	val, err := strconv.ParseBool(strings.ToLower(insecureStr))
	if err != nil {
		logrustr.Warnf("Invalid boolean value for OTEL_EXPORTER_INSECURE: %s, defaulting to false", insecureStr)
		return false
	}
	return val
}
